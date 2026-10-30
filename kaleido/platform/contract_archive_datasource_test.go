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
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testVerifierArtifact = `{
  "_format": "hh-sol-artifact-1",
  "contractName": "Groth16Verifier_Anon",
  "sourceName": "contracts/verifiers/verifier_anon.sol",
  "abi": [{"type":"function","name":"verifyProof","inputs":[],"outputs":[{"name":"","type":"bool"}]}],
  "bytecode": "0x6080604052",
  "linkReferences": {}
}`

const testTokenArtifact = `{
  "contractName": "Zeto_Anon",
  "sourceName": "contracts/zeto_anon.sol",
  "abi": [],
  "bytecode": "0x60a0604052",
  "linkReferences": {}
}`

const testLinkedArtifact = `{
  "contractName": "Zeto_AnonNullifier",
  "sourceName": "contracts/zeto_anon_nullifier.sol",
  "abi": [],
  "bytecode": "0x6080__$aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa$__6001__$bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb$____$bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb$__00",
  "linkReferences": {
    "@iden3/contracts/contracts/lib/Poseidon.sol": {"PoseidonUnit2L": [{"start": 2, "length": 20}]},
    "@iden3/contracts/contracts/lib/SmtLib.sol": {"SmtLib": [{"start": 24, "length": 20}, {"start": 44, "length": 20}]}
  }
}`

const testInterfaceArtifact = `{
  "contractName": "IZeto",
  "sourceName": "contracts/lib/interfaces/izeto.sol",
  "abi": [],
  "bytecode": "0x",
  "linkReferences": {}
}`

// testArchiveFiles mirrors the nested Hardhat layout of the Zeto release archive, with a
// duplicate identical copy of one artifact and a debug file that must be ignored
var testArchiveFiles = map[string]string{
	"artifacts/contracts/verifiers/verifier_anon.sol/Groth16Verifier_Anon.json":     testVerifierArtifact,
	"artifacts/contracts/verifiers/verifier_anon.sol/Groth16Verifier_Anon.dbg.json": `{"buildInfo":"x"}`,
	"artifacts/contracts/zeto_anon.sol/Zeto_Anon.json":                              testTokenArtifact,
	"artifacts/copy/Zeto_Anon.json":                                                 testTokenArtifact,
	"artifacts/contracts/zeto_anon_nullifier.sol/Zeto_AnonNullifier.json":           testLinkedArtifact,
	"artifacts/contracts/lib/interfaces/izeto.sol/IZeto.json":                       testInterfaceArtifact,
	"anon-vkey.json":       `{"protocol":"groth16"}`,
	"anon.zkey":            "binary proving key",
	"conflict/Dup.json":    `{"contractName":"Dup","abi":[],"bytecode":"0x01"}`,
	"conflict/v2/Dup.json": `{"contractName":"Dup","abi":[],"bytecode":"0x02"}`,
}

func writeTestTar(t *testing.T, path string, gzipped bool) {
	var buf bytes.Buffer
	var tw *tar.Writer
	var gw *gzip.Writer
	if gzipped {
		gw = gzip.NewWriter(&buf)
		tw = tar.NewWriter(gw)
	} else {
		tw = tar.NewWriter(&buf)
	}
	require.NoError(t, tw.WriteHeader(&tar.Header{Name: "./artifacts/", Typeflag: tar.TypeDir, Mode: 0755}))
	for name, content := range testArchiveFiles {
		require.NoError(t, tw.WriteHeader(&tar.Header{Name: "./" + name, Typeflag: tar.TypeReg, Mode: 0644, Size: int64(len(content))}))
		_, err := tw.Write([]byte(content))
		require.NoError(t, err)
	}
	require.NoError(t, tw.Close())
	if gw != nil {
		require.NoError(t, gw.Close())
	}
	require.NoError(t, os.WriteFile(path, buf.Bytes(), 0644))
}

func writeTestZip(t *testing.T, path string) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	_, err := zw.Create("artifacts/")
	require.NoError(t, err)
	for name, content := range testArchiveFiles {
		w, err := zw.Create(name)
		require.NoError(t, err)
		_, err = w.Write([]byte(content))
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())
	require.NoError(t, os.WriteFile(path, buf.Bytes(), 0644))
}

func contractArchiveConfig(url string, contracts []string, files []string) string {
	quote := func(items []string) string {
		quoted := make([]string, len(items))
		for i, s := range items {
			quoted[i] = fmt.Sprintf("%q", s)
		}
		return "[" + strings.Join(quoted, ", ") + "]"
	}
	filesAttr := ""
	if files != nil {
		filesAttr = "files = " + quote(files)
	}
	return fmt.Sprintf(`
data "kaleido_platform_contract_archive" "zeto" {
    url = %q
    contracts = %s
    %s
}
`, url, quote(contracts), filesAttr)
}

// contractArchiveLibrariesConfig adds a libraries map to the data source config
func contractArchiveLibrariesConfig(url string, contracts []string, libraries map[string]string) string {
	entries := make([]string, 0, len(libraries))
	for name, address := range libraries {
		entries = append(entries, fmt.Sprintf("%q = %q", name, address))
	}
	sort.Strings(entries)
	return strings.Replace(contractArchiveConfig(url, contracts, nil), "contracts =",
		"libraries = {\n      "+strings.Join(entries, "\n      ")+"\n    }\n    contracts =", 1)
}

func TestContractArchiveDatasource(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer mp.server.Close()

	dir := t.TempDir()
	writeTestTar(t, filepath.Join(dir, "contracts.tar.gz"), true)
	writeTestTar(t, filepath.Join(dir, "contracts.tar"), false)
	writeTestZip(t, filepath.Join(dir, "contracts.zip"))

	srv := httptest.NewServer(http.FileServer(http.Dir(dir)))
	defer srv.Close()

	const ds = "data.kaleido_platform_contract_archive.zeto"
	check := resource.ComposeAggregateTestCheckFunc(
		resource.TestCheckResourceAttr(ds, "artifacts.%", "2"),
		resource.TestCheckResourceAttr(ds, "artifacts.Groth16Verifier_Anon.bytecode", "0x6080604052"),
		resource.TestCheckResourceAttr(ds, "artifacts.Groth16Verifier_Anon.source_name", "contracts/verifiers/verifier_anon.sol"),
		resource.TestCheckResourceAttr(ds, "artifacts.Groth16Verifier_Anon.path", "artifacts/contracts/verifiers/verifier_anon.sol/Groth16Verifier_Anon.json"),
		resource.TestCheckResourceAttrWith(ds, "artifacts.Groth16Verifier_Anon.abi", func(value string) error {
			assert.JSONEq(t, `[{"type":"function","name":"verifyProof","inputs":[],"outputs":[{"name":"","type":"bool"}]}]`, value)
			return nil
		}),
		resource.TestCheckResourceAttr(ds, "artifacts.Zeto_Anon.bytecode", "0x60a0604052"),
		resource.TestCheckResourceAttr(ds, "file_contents.%", "1"),
		resource.TestCheckResourceAttr(ds, "file_contents.anon-vkey.json", `{"protocol":"groth16"}`),
	)
	contracts := []string{"Groth16Verifier_Anon", "Zeto_Anon"}
	files := []string{"anon-vkey.json"}

	steps := []resource.TestStep{}
	for _, url := range []string{
		"file://" + filepath.Join(dir, "contracts.tar.gz"),
		"file:" + filepath.Join(dir, "contracts.zip"),
		"file:" + filepath.Join(dir, "contracts.tar"),
		srv.URL + "/contracts.tar.gz",
		srv.URL + "/contracts.zip",
	} {
		steps = append(steps, resource.TestStep{
			Config: providerConfig + contractArchiveConfig(url, contracts, files),
			Check:  check,
		})
	}

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps:                    steps,
	})
}

func TestContractArchiveDatasourceErrors(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer mp.server.Close()

	dir := t.TempDir()
	archive := filepath.Join(dir, "contracts.tgz")
	writeTestTar(t, archive, true)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "notarchive.gz"), []byte("plain text"), 0644))

	srv := httptest.NewServer(http.FileServer(http.Dir(dir)))
	defer srv.Close()

	fileURL := "file://" + archive
	for _, tc := range []struct {
		name      string
		url       string
		contracts []string
		files     []string
		err       string
	}{
		{"missing contract", fileURL, []string{"Zeto_AnonEnc"}, nil, `'Zeto_AnonEnc.json' not found in the archive`},
		{"missing file", fileURL, []string{"Zeto_Anon"}, []string{"missing-vkey.json"}, `'missing-vkey.json' not found in the archive`},
		{"differing copies", fileURL, []string{"Dup"}, nil, `'Dup.json' is ambiguous`},
		{"unlinked libraries", fileURL, []string{"Zeto_AnonNullifier"}, nil, `links libraries that have no address in libraries: @iden3/contracts/contracts/lib/Poseidon.sol:PoseidonUnit2L, @iden3/contracts/contracts/lib/SmtLib.sol:SmtLib`},
		{"interface", fileURL, []string{"IZeto"}, nil, `has no bytecode`},
		{"not an archive", "file://" + filepath.Join(dir, "notarchive.gz"), []string{"Zeto_Anon"}, nil, `invalid archive \(expected a .zip`},
		{"missing local file", "file://" + filepath.Join(dir, "nope.tar.gz"), []string{"Zeto_Anon"}, nil, `failed to open archive`},
		{"http not found", srv.URL + "/nope.tar.gz", []string{"Zeto_Anon"}, nil, `HTTP status 404`},
		{"unsupported scheme", "s3://bucket/contracts.tar.gz", []string{"Zeto_Anon"}, nil, `scheme must be http, https or file`},
		{"remote file host", "file://otherhost/contracts.tar.gz", []string{"Zeto_Anon"}, nil, `must refer to the local host`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// error text is wrapped across lines, so match any whitespace between words
			step := resource.TestStep{
				Config:      providerConfig + contractArchiveConfig(tc.url, tc.contracts, tc.files),
				ExpectError: regexp.MustCompile(strings.ReplaceAll(tc.err, " ", `\s+`)),
			}
			resource.Test(t, resource.TestCase{
				IsUnitTest:               true,
				ProtoV6ProviderFactories: testAccProviders,
				Steps:                    []resource.TestStep{step},
			})
		})
	}
}

func TestContractArchiveDatasourceLinking(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer mp.server.Close()

	dir := t.TempDir()
	archive := filepath.Join(dir, "contracts.tgz")
	writeTestTar(t, archive, true)
	fileURL := "file://" + archive

	const ds = "data.kaleido_platform_contract_archive.zeto"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				// The fully qualified name takes precedence, addresses are lower-cased, and
				// libraries no contract links (Unused) and unlinked contracts are fine
				Config: providerConfig + contractArchiveLibrariesConfig(fileURL, []string{"Zeto_AnonNullifier", "Zeto_Anon"}, map[string]string{
					"PoseidonUnit2L": "0x1111111111111111111111111111111111111111",
					"SmtLib":         "0x2222222222222222222222222222222222222222",
					"@iden3/contracts/contracts/lib/SmtLib.sol:SmtLib": "0xABCDEFabcdef0000000000000000000000000003",
					"Unused": "0x4444444444444444444444444444444444444444",
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(ds, "artifacts.Zeto_AnonNullifier.bytecode",
						"0x6080"+strings.Repeat("11", 20)+"6001"+"abcdefabcdef0000000000000000000000000003"+"abcdefabcdef0000000000000000000000000003"+"00"),
					resource.TestCheckResourceAttr(ds, "artifacts.Zeto_Anon.bytecode", "0x60a0604052"),
				),
			},
			{
				Config: providerConfig + contractArchiveLibrariesConfig(fileURL, []string{"Zeto_AnonNullifier"}, map[string]string{
					"PoseidonUnit2L": "0x1111111111111111111111111111111111111111",
				}),
				ExpectError: regexp.MustCompile(`no\s+address\s+in\s+libraries:\s+@iden3/contracts/contracts/lib/SmtLib.sol:SmtLib`),
			},
			{
				Config: providerConfig + contractArchiveLibrariesConfig(fileURL, []string{"Zeto_AnonNullifier"}, map[string]string{
					"PoseidonUnit2L": "0x1234",
				}),
				ExpectError: regexp.MustCompile(`library\s+'PoseidonUnit2L'\s+address\s+'0x1234'\s+is\s+not`),
			},
		},
	})
}

func TestLinkContractArchiveLibrariesBadReference(t *testing.T) {
	refs := map[string]map[string][]hardhatLinkReference{"lib.sol": {"Lib": {{Start: 1, Length: 20}}}}
	_, err := linkContractArchiveLibraries("C", "6080"+strings.Repeat("00", 20), refs, map[string]string{"Lib": strings.Repeat("11", 20)})
	assert.ErrorContains(t, err, "invalid link reference for library 'Lib' at offset 1")

	_, err = linkContractArchiveLibraries("C", "6080", refs, map[string]string{"Lib": strings.Repeat("11", 20)})
	assert.ErrorContains(t, err, "invalid link reference")
}

func TestContractArchiveRelativeFileURL(t *testing.T) {
	dir := t.TempDir()
	writeTestTar(t, filepath.Join(dir, "contracts.tar.gz"), true)
	t.Chdir(dir)

	found, err := readContractArchive(context.Background(), "file:contracts.tar.gz", func(name string) bool {
		return strings.HasSuffix(name, "/Zeto_Anon.json")
	})
	require.NoError(t, err)
	assert.Len(t, found["Zeto_Anon.json"], 2)
}
