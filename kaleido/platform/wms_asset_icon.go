// Copyright © Kaleido, Inc. 2024

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
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type WMSAssetIconResourceModel struct {
	Environment types.String `tfsdk:"environment"`
	Service     types.String `tfsdk:"service"`
	AssetName   types.String `tfsdk:"asset_name"`
	FilePath    types.String `tfsdk:"file_path"`
	FileType    types.String `tfsdk:"file_type"` // e.g. image/png or image/jpeg
}

type wmsAssetIconResource struct {
	commonResource
}

func NewWMSAssetIconResource() resource.Resource {
	return &wmsAssetIconResource{}
}

func WMSAssetIconResourceFactory() resource.Resource {
	return &wmsAssetIconResource{}
}

func (r *wmsAssetIconResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_wms_asset_icon"
}

func (r *wmsAssetIconResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"environment": &schema.StringAttribute{
				Required:    true,
				Description: "The environment ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"service": &schema.StringAttribute{
				Required:    true,
				Description: "The wallet manager service ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"asset_name": &schema.StringAttribute{
				Required:    true,
				Description: "The name of the asset",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"file_path": &schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The path to the PNG file to upload",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"file_type": &schema.StringAttribute{
				Required:    true,
				Description: "The type of the file to upload. e.g. image/png or image/jpeg",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *wmsAssetIconResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.commonResource.Configure(ctx, req, resp)
}

func (r *wmsAssetIconResource) apiPath(data *WMSAssetIconResourceModel) string {
	return fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/assets/%s/icon",
		data.Environment.ValueString(),
		data.Service.ValueString(),
		data.AssetName.ValueString())
}

func (r *wmsAssetIconResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data WMSAssetIconResourceModel
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate file exists and is a PNG
	if data.FilePath.IsNull() || data.FilePath.IsUnknown() {
		resp.Diagnostics.AddError("File path required", "file_path must be specified")
		return
	}

	filePath := data.FilePath.ValueString()
	file, err := os.Open(filePath)
	if err != nil {
		resp.Diagnostics.AddError("File not found", fmt.Sprintf("Could not open file %s: %v", filePath, err))
		return
	}
	defer file.Close()

	// Create multipart form data
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	// Add the "type" field to the multipart form data
	if err := w.WriteField("type", data.FileType.ValueString()); err != nil {
		resp.Diagnostics.AddError("Form creation error", fmt.Sprintf("Could not add type field: %v", err))
		return
	}

	// Add the file field
	fw, err := w.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		resp.Diagnostics.AddError("Form creation error", fmt.Sprintf("Could not create form file: %v", err))
		return
	}
	_, err = io.Copy(fw, file)
	if err != nil {
		resp.Diagnostics.AddError("File copy error", fmt.Sprintf("Could not copy file data: %v", err))
		return
	}

	w.Close()

	// Make the API request using the common method but with custom multipart handling
	path := r.apiPath(&data)

	tflog.Debug(ctx, fmt.Sprintf("Uploading icon for asset %s from file %s", data.AssetName.ValueString(), filePath))

	// Use the common requestor but with multipart form data
	req_resty := r.Platform.R().
		SetContext(ctx).
		SetDoNotParseResponse(true).
		SetHeader("Content-Type", w.FormDataContentType()).
		SetBody(b.Bytes())

	res, err := req_resty.Post(path)
	if err != nil {
		resp.Diagnostics.AddError("API request failed", fmt.Sprintf("Could not upload icon: %v", err))
		return
	}

	if !res.IsSuccess() || res.StatusCode() != 204 {
		body, _ := io.ReadAll(res.RawBody())
		resp.Diagnostics.AddError("API error", fmt.Sprintf("Upload failed with status %d: %s", res.StatusCode(), string(body)))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)

}

func (r *wmsAssetIconResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data WMSAssetIconResourceModel
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// For icon resources, we can't really "read" the current state from the API
	// since the API doesn't provide a way to get icon information.
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *wmsAssetIconResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Icon updates are not supported - requires replacement
	resp.Diagnostics.AddError("Update not supported", "Asset icons cannot be updated. Use replace instead.")
}

func (r *wmsAssetIconResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data WMSAssetIconResourceModel
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Make DELETE request to remove the icon
	path := r.apiPath(&data)

	tflog.Debug(ctx, fmt.Sprintf("Deleting icon for asset %s", data.AssetName.ValueString()))

	_, _ = r.apiRequest(ctx, http.MethodDelete, path, nil, nil, &resp.Diagnostics, Allow404())

	r.waitForRemoval(ctx, path, &resp.Diagnostics)
}

func (r *wmsAssetIconResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: environment/service/asset_name
	parts := strings.Split(req.ID, "/")
	if len(parts) != 3 {
		resp.Diagnostics.AddError("Invalid import ID", "Import ID must be in format: environment/service/asset_name")
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("service"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("asset_name"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
