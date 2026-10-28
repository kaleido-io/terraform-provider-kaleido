// Copyright © Kaleido, Inc. 2026

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package platform

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Limit on the size of any single file extracted from an archive, to guard against
// matching a large binary by name. Compiled contract artifacts are well under this.
const contractArchiveMaxFileSize = 32 * 1024 * 1024

type ContractArchiveDatasourceModel struct {
	URL       types.String                            `tfsdk:"url"`
	Contracts []types.String                          `tfsdk:"contracts"`
	Files     []types.String                          `tfsdk:"files"`
	Artifacts map[string]ContractArchiveArtifactModel `tfsdk:"artifacts"`
	FileData  map[string]types.String                 `tfsdk:"file_contents"`
}

type ContractArchiveArtifactModel struct {
	ABI        types.String `tfsdk:"abi"`
	Bytecode   types.String `tfsdk:"bytecode"`
	SourceName types.String `tfsdk:"source_name"`
	Path       types.String `tfsdk:"path"`
}

// hardhatArtifact is the subset of a Hardhat/Foundry compiled artifact JSON file used here
type hardhatArtifact struct {
	ContractName   string                     `json:"contractName"`
	SourceName     string                     `json:"sourceName"`
	ABI            json.RawMessage            `json:"abi"`
	Bytecode       string                     `json:"bytecode"`
	LinkReferences map[string]json.RawMessage `json:"linkReferences"`
}

type contractArchiveEntry struct {
	path string
	data []byte
}

func ContractArchiveDatasourceModelFactory() datasource.DataSource {
	return &contractArchiveDatasource{}
}

type contractArchiveDatasource struct {
	commonDataSource
}

func (s *contractArchiveDatasource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_contract_archive"
}

func (s *contractArchiveDatasource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Read compiled contract artifacts (Hardhat format JSON files with `abi` and `bytecode`) from a .zip, .tar.gz, .tgz, .gz or .tar archive, for use in `precompiled` ContractManager builds. The archive is read on every plan; nothing is uploaded to the platform.",
		Attributes: map[string]schema.Attribute{
			"url": &schema.StringAttribute{
				Required:    true,
				Description: "Location of the archive: an `http://` or `https://` URL, or a local file as a `file:` URL (`file:///abs/path.tar.gz`, or `file:relative/path.tar.gz` relative to the working directory)",
			},
			"contracts": &schema.ListAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "Names of the contracts to extract. Each is matched to an artifact file named `<contract>.json` anywhere in the archive, and must be found exactly once (identical copies are allowed), be deployable (non-empty bytecode) and need no library linking",
			},
			"files": &schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Base names of additional text files to return the contents of (for example `anon-vkey.json`), matched anywhere in the archive",
			},
			"artifacts": &schema.MapNestedAttribute{
				Computed:    true,
				Description: "Extracted artifacts keyed by contract name",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"abi": &schema.StringAttribute{
							Computed:    true,
							Description: "Contract ABI as a JSON string",
						},
						"bytecode": &schema.StringAttribute{
							Computed:    true,
							Description: "0x-prefixed deployment bytecode",
						},
						"source_name": &schema.StringAttribute{
							Computed:    true,
							Description: "Solidity source file the contract was compiled from",
						},
						"path": &schema.StringAttribute{
							Computed:    true,
							Description: "Path of the artifact file within the archive",
						},
					},
				},
			},
			"file_contents": &schema.MapAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Contents of the requested `files`, keyed by base name",
			},
		},
	}
}

func (s *contractArchiveDatasource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ContractArchiveDatasourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Map each wanted base name to the name it is reported under
	wantedContracts := map[string]string{}
	for _, c := range data.Contracts {
		wantedContracts[c.ValueString()+".json"] = c.ValueString()
	}
	wantedFiles := map[string]bool{}
	for _, f := range data.Files {
		wantedFiles[f.ValueString()] = true
	}

	found, err := readContractArchive(ctx, data.URL.ValueString(), func(name string) bool {
		base := path.Base(name)
		return wantedFiles[base] || wantedContracts[base] != ""
	})
	if err != nil {
		resp.Diagnostics.AddError("failed to read contract archive", err.Error())
		return
	}

	data.Artifacts = map[string]ContractArchiveArtifactModel{}
	for base, contractName := range wantedContracts {
		artifact, err := parseContractArchiveArtifact(contractName, found[base])
		if err != nil {
			resp.Diagnostics.AddError("invalid contract archive", err.Error())
			continue
		}
		data.Artifacts[contractName] = *artifact
	}

	data.FileData = map[string]types.String{}
	for name := range wantedFiles {
		content, err := uniqueContractArchiveEntry(name, found[name])
		if err != nil {
			resp.Diagnostics.AddError("invalid contract archive", err.Error())
			continue
		}
		data.FileData[name] = types.StringValue(string(content.data))
	}
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

// uniqueContractArchiveEntry returns the single entry for a name, tolerating identical copies
// (archives commonly include the same artifact under more than one path)
func uniqueContractArchiveEntry(name string, entries []contractArchiveEntry) (*contractArchiveEntry, error) {
	if len(entries) == 0 {
		return nil, fmt.Errorf("'%s' not found in the archive", name)
	}
	for _, e := range entries[1:] {
		if !bytes.Equal(e.data, entries[0].data) {
			paths := make([]string, len(entries))
			for i, e := range entries {
				paths[i] = e.path
			}
			sort.Strings(paths)
			return nil, fmt.Errorf("'%s' is ambiguous: the archive has differing copies at %s", name, strings.Join(paths, ", "))
		}
	}
	return &entries[0], nil
}

func parseContractArchiveArtifact(contractName string, entries []contractArchiveEntry) (*ContractArchiveArtifactModel, error) {
	entry, err := uniqueContractArchiveEntry(contractName+".json", entries)
	if err != nil {
		return nil, err
	}
	var artifact hardhatArtifact
	if err := json.Unmarshal(entry.data, &artifact); err != nil {
		return nil, fmt.Errorf("artifact '%s' for contract '%s' is not valid JSON: %s", entry.path, contractName, err)
	}
	if artifact.ContractName != "" && artifact.ContractName != contractName {
		return nil, fmt.Errorf("artifact '%s' is for contract '%s', not '%s'", entry.path, artifact.ContractName, contractName)
	}
	if len(artifact.ABI) == 0 {
		return nil, fmt.Errorf("artifact '%s' for contract '%s' has no abi", entry.path, contractName)
	}
	if strings.TrimPrefix(artifact.Bytecode, "0x") == "" {
		return nil, fmt.Errorf("artifact '%s' for contract '%s' has no bytecode (interface or abstract contract)", entry.path, contractName)
	}
	if len(artifact.LinkReferences) > 0 {
		libs := make([]string, 0, len(artifact.LinkReferences))
		for lib := range artifact.LinkReferences {
			libs = append(libs, lib)
		}
		sort.Strings(libs)
		return nil, fmt.Errorf("contract '%s' requires linking libraries from %s, which is not supported", contractName, strings.Join(libs, ", "))
	}
	bytecode := artifact.Bytecode
	if !strings.HasPrefix(bytecode, "0x") {
		bytecode = "0x" + bytecode
	}
	return &ContractArchiveArtifactModel{
		ABI:        types.StringValue(string(artifact.ABI)),
		Bytecode:   types.StringValue(bytecode),
		SourceName: types.StringValue(artifact.SourceName),
		Path:       types.StringValue(entry.path),
	}, nil
}

// readContractArchive opens the archive at the URL and returns the contents of every
// regular file whose name the filter accepts, keyed by base name
func readContractArchive(ctx context.Context, archiveURL string, filter func(name string) bool) (map[string][]contractArchiveEntry, error) {
	f, err := openContractArchive(ctx, archiveURL)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.Close()
		if f.temp {
			_ = os.Remove(f.Name())
		}
	}()

	found := map[string][]contractArchiveEntry{}
	add := func(name string, size int64, r io.Reader) error {
		if size > contractArchiveMaxFileSize {
			return fmt.Errorf("'%s' in the archive is too large (%d bytes)", name, size)
		}
		data, err := io.ReadAll(io.LimitReader(r, contractArchiveMaxFileSize+1))
		if err != nil {
			return fmt.Errorf("failed reading '%s' from the archive: %s", name, err)
		}
		base := path.Base(name)
		found[base] = append(found[base], contractArchiveEntry{path: strings.TrimPrefix(name, "./"), data: data})
		return nil
	}

	// Detect the format from the content rather than the file extension
	header := make([]byte, 4)
	n, _ := io.ReadFull(f, header)
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	switch {
	case n == 4 && bytes.Equal(header, []byte("PK\x03\x04")):
		stat, err := f.Stat()
		if err != nil {
			return nil, err
		}
		zr, err := zip.NewReader(f, stat.Size())
		if err != nil {
			return nil, fmt.Errorf("invalid zip archive: %s", err)
		}
		for _, zf := range zr.File {
			if zf.FileInfo().IsDir() || !filter(zf.Name) {
				continue
			}
			rc, err := zf.Open()
			if err != nil {
				return nil, fmt.Errorf("failed reading '%s' from the archive: %s", zf.Name, err)
			}
			err = add(zf.Name, int64(zf.UncompressedSize64), rc)
			_ = rc.Close()
			if err != nil {
				return nil, err
			}
		}
	default:
		var r io.Reader = bufio.NewReader(f)
		if n >= 2 && header[0] == 0x1f && header[1] == 0x8b {
			gz, err := gzip.NewReader(r)
			if err != nil {
				return nil, fmt.Errorf("invalid gzip archive: %s", err)
			}
			defer gz.Close()
			r = gz
		}
		tr := tar.NewReader(r)
		for {
			hdr, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, fmt.Errorf("invalid archive (expected a .zip, .tar.gz, .tgz, .gz or .tar file): %s", err)
			}
			if hdr.Typeflag != tar.TypeReg || !filter(hdr.Name) {
				continue
			}
			if err := add(hdr.Name, hdr.Size, tr); err != nil {
				return nil, err
			}
		}
	}
	return found, nil
}

type contractArchiveFile struct {
	*os.File
	temp bool
}

// openContractArchive opens a local file: URL directly, or downloads an http(s) URL to a
// temporary file (zip archives need random access)
func openContractArchive(ctx context.Context, archiveURL string) (*contractArchiveFile, error) {
	u, err := url.Parse(archiveURL)
	if err != nil {
		return nil, fmt.Errorf("invalid archive URL '%s': %s", archiveURL, err)
	}
	switch u.Scheme {
	case "file":
		filePath := u.Opaque // file:relative/path
		if filePath == "" {
			if u.Host != "" && u.Host != "localhost" {
				return nil, fmt.Errorf("invalid archive URL '%s': file URLs must refer to the local host", archiveURL)
			}
			filePath = u.Path // file:///abs/path or file:/abs/path
		}
		if filePath == "" {
			return nil, fmt.Errorf("invalid archive URL '%s': no file path", archiveURL)
		}
		f, err := os.Open(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to open archive: %s", err)
		}
		return &contractArchiveFile{File: f}, nil
	case "http", "https":
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, archiveURL, nil)
		if err != nil {
			return nil, err
		}
		res, err := http.DefaultClient.Do(httpReq)
		if err != nil {
			return nil, fmt.Errorf("failed to download archive: %s", err)
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to download archive from '%s': HTTP status %d", archiveURL, res.StatusCode)
		}
		f, err := os.CreateTemp("", "kaleido-contract-archive-*")
		if err != nil {
			return nil, err
		}
		archive := &contractArchiveFile{File: f, temp: true}
		if _, err := io.Copy(f, res.Body); err != nil {
			_ = f.Close()
			_ = os.Remove(f.Name())
			return nil, fmt.Errorf("failed to download archive from '%s': %s", archiveURL, err)
		}
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			_ = f.Close()
			_ = os.Remove(f.Name())
			return nil, err
		}
		return archive, nil
	default:
		return nil, fmt.Errorf("invalid archive URL '%s': the scheme must be http, https or file", archiveURL)
	}
}
