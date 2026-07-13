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
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var SupportedFileArtifactTypes = []string{
	"typescript",
	"json",
	"yaml",
	"blob",
	"dar",
	"abi",
	"bytecode",
}

// Mirrors the server-side validation in the Artifact Registry: slash-separated
// lowercase OCI remoteName components, max 200 chars, with "content" reserved
// as a path component (it would collide with the /content route suffix).
var (
	arsFileNameComponent      = `[a-z0-9]+(?:(?:[._]|__|[-]+)[a-z0-9]+)*`
	arsFileNamePattern        = regexp.MustCompile(`^` + arsFileNameComponent + `(?:/` + arsFileNameComponent + `)*$`)
	arsFileTagPattern         = regexp.MustCompile(`^[\w][\w.-]{0,127}$`)
	arsFileNameMaxLength      = 200
	arsFileReservedComponents = []string{"content"}
)

type ARSFileArtifactResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Environment       types.String `tfsdk:"environment"`
	Service           types.String `tfsdk:"service"`
	Namespace         types.String `tfsdk:"namespace"`
	Name              types.String `tfsdk:"name"`
	FilePath          types.String `tfsdk:"file_path"`
	Type              types.String `tfsdk:"type"`
	Version           types.String `tfsdk:"version"`
	Tag               types.String `tfsdk:"tag"`
	RemoveOldVersions types.Bool   `tfsdk:"remove_old_versions"`
	ContentSHA256     types.String `tfsdk:"content_sha256"`
	Size              types.Int64  `tfsdk:"size"`
}

// FileVersion (POST response) / FileMetadata (GET response) from the Artifact Registry
type ARSFileArtifactAPIModel struct {
	Namespace      string `json:"namespace,omitempty"`
	Repository     string `json:"repository,omitempty"`
	Tag            string `json:"tag,omitempty"`
	Kind           string `json:"kind,omitempty"`
	ArtifactType   string `json:"artifactType,omitempty"`
	LayerDigest    string `json:"layerDigest,omitempty"`
	ManifestDigest string `json:"manifestDigest,omitempty"`
	Size           int64  `json:"size,omitempty"`
}

func ARSFileArtifactResourceFactory() resource.Resource {
	return &arsFileArtifactResource{}
}

type arsFileArtifactResource struct {
	commonResource
}

var _ resource.ResourceWithModifyPlan = &arsFileArtifactResource{}
var _ resource.ResourceWithImportState = &arsFileArtifactResource{}

type arsFileNameValidator struct{}

func (v arsFileNameValidator) Description(_ context.Context) string {
	return "name must be slash-separated lowercase components (letters, digits, '.', '_', '-' separators), max 200 characters, and must not use the reserved path component 'content'"
}

func (v arsFileNameValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v arsFileNameValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	name := req.ConfigValue.ValueString()
	if len(name) > arsFileNameMaxLength || !arsFileNamePattern.MatchString(name) {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid artifact name",
			fmt.Sprintf("'%s' is not a valid artifact name: must be slash-separated lowercase components matching %s, max %d characters", name, arsFileNameComponent, arsFileNameMaxLength))
		return
	}
	for _, part := range strings.Split(name, "/") {
		for _, reserved := range arsFileReservedComponents {
			if part == reserved {
				resp.Diagnostics.AddAttributeError(req.Path, "Reserved path component",
					fmt.Sprintf("'%s' is a reserved path component and cannot be used in an artifact name", reserved))
				return
			}
		}
	}
}

func (r *arsFileArtifactResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_ars_file_artifact"
}

func (r *arsFileArtifactResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A file artifact in the Kaleido Artifact Registry, pushed from a local file and addressed as '{name}:{tag}' within a namespace, where the tag can be derived from a version and a checksum.",
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:    true,
				Description: "Composite ID: environment/service/namespace/name:tag",
			},
			"environment": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Environment ID",
			},
			"service": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Artifact Registry service ID",
			},
			"namespace": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Namespace name (the namespace must use a file-capable artifact family, e.g. 'file')",
			},
			"name": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:    []validator.String{arsFileNameValidator{}},
				Description:   "Path-style artifact name in the registry, e.g. 'path/to/myfilename.ext'. Slash-separated lowercase components; 'content' is reserved.",
			},
			"file_path": &schema.StringAttribute{
				Required:    true,
				Description: "Local path to the file to upload. With 'version' set the file is hashed at plan time and must exist. With an explicit 'tag' the file is only read at create.",
			},
			"type": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:    []validator.String{stringvalidator.OneOf(SupportedFileArtifactTypes...)},
				Description:   fmt.Sprintf("The file type (one of: %s)", strings.Join(SupportedFileArtifactTypes, ", ")),
			},
			"version": &schema.StringAttribute{
				Optional: true,
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.MatchRoot("version"), path.MatchRoot("tag")),
				},
				Description: "Version prefix for the content-addressed tag '{version}-{sha8}'. Exactly one of 'version' or 'tag' must be set. Changing the file content uploads the artifact under a new tag.",
			},
			"tag": &schema.StringAttribute{
				Optional: true,
				Computed: true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(arsFileTagPattern, "must be a valid OCI tag (word character followed by up to 127 word/'.'/'-' characters)"),
				},
				Description: "Explicit immutable tag. When set, the tag's existence is trusted: local file changes are not detected and the file is only uploaded when the tag changes. Pushing to an existing tag with different content fails (tags are immutable).",
			},
			"remove_old_versions": &schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "When true, moving to a new tag (e.g. a file content change with 'version' set) deletes the previously tracked version from the registry after the new one uploads. Defaults to false: old versions are retained in the registry on upgrade.",
			},
			"content_sha256": &schema.StringAttribute{
				Computed:    true,
				Description: "SHA-256 digest of the uploaded content in 'sha256:<hex>' form (matches the server layerDigest)",
			},
			"size": &schema.Int64Attribute{
				Computed:    true,
				Description: "Size of the uploaded content in bytes",
			},
		},
	}
}

func (api *ARSFileArtifactAPIModel) toData(data *ARSFileArtifactResourceModel) {
	if api.Repository != "" {
		data.Name = types.StringValue(api.Repository)
	}
	if api.Tag != "" {
		data.Tag = types.StringValue(api.Tag)
	}
	// Only fill 'type' when unset (import) - the server 'kind' echoes the requested type
	if (data.Type.IsNull() || data.Type.ValueString() == "") && api.Kind != "" {
		data.Type = types.StringValue(api.Kind)
	}
	data.ContentSHA256 = types.StringValue(api.LayerDigest)
	data.Size = types.Int64Value(api.Size)
	data.ID = types.StringValue(fmt.Sprintf("%s/%s/%s/%s:%s",
		data.Environment.ValueString(),
		data.Service.ValueString(),
		data.Namespace.ValueString(),
		data.Name.ValueString(),
		data.Tag.ValueString()))
}

func (r *arsFileArtifactResource) apiPath(data *ARSFileArtifactResourceModel) string {
	return fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/namespaces/%s/files/%s:%s",
		data.Environment.ValueString(),
		data.Service.ValueString(),
		data.Namespace.ValueString(),
		data.Name.ValueString(),
		data.Tag.ValueString())
}

func deriveFileArtifactTag(version, hexDigest string) string {
	return fmt.Sprintf("%s-%s", version, hexDigest[:8])
}

func (r *arsFileArtifactResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		// Destroy plan - never reads the file
		return
	}

	var plan ARSFileArtifactResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	var config ARSFileArtifactResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state *ARSFileArtifactResourceModel
	if !req.State.Raw.IsNull() {
		state = &ARSFileArtifactResourceModel{}
		resp.Diagnostics.Append(req.State.Get(ctx, state)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	autoMode := !config.Version.IsNull()
	if autoMode {
		if plan.FilePath.IsUnknown() || plan.Version.IsUnknown() {
			// Cannot derive the tag yet - all derived attributes are unknown
			resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("tag"), types.StringUnknown())...)
			resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("content_sha256"), types.StringUnknown())...)
			resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("size"), types.Int64Unknown())...)
			resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("id"), types.StringUnknown())...)
			return
		}

		fileBytes, err := os.ReadFile(plan.FilePath.ValueString())
		if err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("file_path"), "File not readable at plan time",
				fmt.Sprintf("With 'version' set the file is hashed at plan time to derive the tag, so it must exist when planning: %v", err))
			return
		}
		sum := sha256.Sum256(fileBytes)
		hexDigest := hex.EncodeToString(sum[:])
		derivedTag := deriveFileArtifactTag(plan.Version.ValueString(), hexDigest)

		plan.Tag = types.StringValue(derivedTag)
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("tag"), derivedTag)...)
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("content_sha256"), "sha256:"+hexDigest)...)
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("size"), int64(len(fileBytes)))...)
	} else if state != nil && (plan.Tag.IsUnknown() || plan.Tag.ValueString() != state.Tag.ValueString()) {
		// Explicit-tag mode with a tag change: the file is re-uploaded at apply time,
		// so the content attributes cannot be known until then
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("content_sha256"), types.StringUnknown())...)
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("size"), types.Int64Unknown())...)
	}

	// Composite ID is derivable at plan time once all its parts are known
	if !plan.Environment.IsUnknown() && !plan.Service.IsUnknown() && !plan.Namespace.IsUnknown() &&
		!plan.Name.IsUnknown() && !plan.Tag.IsUnknown() && !plan.Tag.IsNull() {
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("id"), fmt.Sprintf("%s/%s/%s/%s:%s",
			plan.Environment.ValueString(),
			plan.Service.ValueString(),
			plan.Namespace.ValueString(),
			plan.Name.ValueString(),
			plan.Tag.ValueString()))...)
	} else {
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("id"), types.StringUnknown())...)
	}
}

// uploadFile reads the local file, re-derives the tag in auto mode, pushes the content
// to the registry, and populates data from the server response.
func (r *arsFileArtifactResource) uploadFile(ctx context.Context, data *ARSFileArtifactResourceModel, diagnostics *diag.Diagnostics) bool {
	filePath := data.FilePath.ValueString()
	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		diagnostics.AddError("File not readable", fmt.Sprintf("Could not read file %s: %v", filePath, err))
		return false
	}
	sum := sha256.Sum256(fileBytes)
	hexDigest := hex.EncodeToString(sum[:])

	// In auto mode the tag is re-derived from the file as read at apply time. If the
	// content changed since plan, the framework surfaces the tag mismatch as an error.
	if !data.Version.IsNull() {
		data.Tag = types.StringValue(deriveFileArtifactTag(data.Version.ValueString(), hexDigest))
	}

	apiPath := r.apiPath(data)
	tflog.Debug(ctx, fmt.Sprintf("Uploading file artifact %s from %s", apiPath, filePath))

	// The upload route accepts multipart/form-data: the 'type' field must
	// precede the file part
	var form bytes.Buffer
	w := multipart.NewWriter(&form)
	if err := w.WriteField("type", data.Type.ValueString()); err != nil {
		diagnostics.AddError("Form creation error", fmt.Sprintf("Could not add type field: %v", err))
		return false
	}
	fw, err := w.CreateFormFile("file", filepath.Base(data.Name.ValueString()))
	if err != nil {
		diagnostics.AddError("Form creation error", fmt.Sprintf("Could not create form file: %v", err))
		return false
	}
	if _, err := fw.Write(fileBytes); err != nil {
		diagnostics.AddError("Form creation error", fmt.Sprintf("Could not write file data: %v", err))
		return false
	}
	if err := w.Close(); err != nil {
		diagnostics.AddError("Form creation error", fmt.Sprintf("Could not finalize form: %v", err))
		return false
	}

	res, err := r.Platform.R().
		SetContext(ctx).
		SetDoNotParseResponse(true).
		SetHeader("Content-Type", w.FormDataContentType()).
		SetBody(form.Bytes()).
		Post(apiPath)
	if err != nil {
		diagnostics.AddError("POST failed", fmt.Sprintf("POST %s failed with error: %v", apiPath, err))
		return false
	}
	defer res.RawResponse.Body.Close()
	body, _ := io.ReadAll(res.RawBody())

	if res.StatusCode() == http.StatusConflict {
		diagnostics.AddError("Tag is immutable",
			fmt.Sprintf("POST %s returned 409: %s. The tag already exists with different content - tags are immutable. "+
				"Push under a new tag, or use 'version' instead of 'tag' to derive content-addressed tags automatically.", apiPath, body))
		return false
	}
	if !res.IsSuccess() {
		diagnostics.AddError("POST failed", fmt.Sprintf("POST %s returned status code %d: %s", apiPath, res.StatusCode(), body))
		return false
	}

	var api ARSFileArtifactAPIModel
	if err := json.Unmarshal(body, &api); err != nil {
		diagnostics.AddError("POST failed", fmt.Sprintf("POST %s returned unparsable body: %v", apiPath, err))
		return false
	}
	api.toData(data)
	return true
}

func (r *arsFileArtifactResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ARSFileArtifactResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !r.uploadFile(ctx, &data, &resp.Diagnostics) {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *arsFileArtifactResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ARSFileArtifactResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var api ARSFileArtifactAPIModel
	ok, status := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics, Allow404())
	if !ok {
		return
	}
	if status == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	api.toData(&data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *arsFileArtifactResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ARSFileArtifactResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	var state ARSFileArtifactResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// A known planned tag equal to the current one means no new version is being
	// pushed - the update only touches attributes with no server-side effect (e.g.
	// 'file_path' in explicit-tag mode, or 'remove_old_versions'). Read and re-set state.
	if !data.Tag.IsUnknown() && data.Tag.ValueString() == state.Tag.ValueString() {
		var api ARSFileArtifactAPIModel
		ok, status := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics, Allow404())
		if !ok || status == 404 {
			return
		}
		api.toData(&data)
		resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
		return
	}

	// Upgrade to a new tag: upload the new version first, then optionally remove the
	// old one. Old versions are retained unless 'remove_old_versions' is set.
	if !r.uploadFile(ctx, &data, &resp.Diagnostics) {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.RemoveOldVersions.ValueBool() && data.Tag.ValueString() != state.Tag.ValueString() {
		oldPath := r.apiPath(&state)
		tflog.Debug(ctx, fmt.Sprintf("Removing old file artifact version %s", oldPath))
		// The new version is already uploaded and recorded in state, so a failure to
		// untag the old version is surfaced as a warning rather than failing the apply
		var deleteDiags diag.Diagnostics
		if ok, _ := r.apiRequest(ctx, http.MethodDelete, oldPath, nil, nil, &deleteDiags, Allow404()); !ok {
			for _, d := range deleteDiags.Errors() {
				resp.Diagnostics.AddWarning(
					fmt.Sprintf("Failed to remove old version '%s'", state.Tag.ValueString()), d.Detail())
			}
		}
	}
}

func (r *arsFileArtifactResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ARSFileArtifactResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Idempotent delete: Allow404 treats an already-removed tag as success.
	ok, _ := r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data), nil, nil, &resp.Diagnostics, Allow404())
	if !ok {
		return
	}
}

func (r *arsFileArtifactResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: environment/service/namespace/name:tag - the name itself may
	// contain slashes, so only the first three parts are positional.
	parts := strings.SplitN(req.ID, "/", 4)
	colonIdx := -1
	if len(parts) == 4 {
		colonIdx = strings.LastIndex(parts[3], ":")
	}
	if colonIdx <= 0 || colonIdx == len(parts[3])-1 {
		resp.Diagnostics.AddError("Invalid import ID", "Import ID must be in format: environment/service/namespace/name:tag")
		return
	}

	// Imported artifacts land in explicit-tag mode: the tag is trusted and content is
	// not re-verified. 'file_path' and 'type' are then populated from configuration.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("service"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("namespace"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), parts[3][:colonIdx])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("tag"), parts[3][colonIdx+1:])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("remove_old_versions"), false)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
