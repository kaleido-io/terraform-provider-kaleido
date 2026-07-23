// Copyright © Kaleido, Inc. 2024-2026

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
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type KMSFolderResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Environment    types.String `tfsdk:"environment"`
	Service        types.String `tfsdk:"service"`
	Keystore       types.String `tfsdk:"keystore"`
	Name           types.String `tfsdk:"name"`
	ParentFolderID types.String `tfsdk:"parent_folder_id"`
}

type KMSFolderAPIModel struct {
	ID             string     `json:"id,omitempty"`
	Created        *time.Time `json:"created,omitempty"`
	Updated        *time.Time `json:"updated,omitempty"`
	Name           string     `json:"name"`
	KeystoreName   string     `json:"keystoreName"`
	ParentFolderID string     `json:"parentFolderId,omitempty"`
}

func KMSFolderResourceFactory() resource.Resource {
	return &kms_folderResource{}
}

type kms_folderResource struct {
	commonResource
}

func (r *kms_folderResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_kms_folder"
}

func (r *kms_folderResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A folder within a KMS keystore, used to organise keys into a hierarchy. Folders cannot be renamed or moved after creation — changes require replacement. A folder cannot be deleted while it still contains keys or sub-folders.",
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"environment": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Environment ID",
			},
			"service": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Key Manager Service ID",
			},
			"keystore": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Keystore (KMS Wallet) ID",
			},
			"name": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Folder name",
			},
			"parent_folder_id": &schema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "ID of the parent folder. Omit to create a root-level folder.",
			},
		},
	}
}

func (r *kms_folderResource) resolveKeystoreName(ctx context.Context, data *KMSFolderResourceModel) (string, bool) {
	var wallet KMSWalletAPIModel
	walletPath := fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/wallets/%s",
		data.Environment.ValueString(), data.Service.ValueString(), data.Keystore.ValueString())
	ok, _ := r.apiRequest(ctx, http.MethodGet, walletPath, nil, &wallet, nil)
	return wallet.Name, ok
}

func (r *kms_folderResource) apiPath(data *KMSFolderResourceModel) string {
	p := fmt.Sprintf("/endpoint/%s/%s/rest/api/v2/folders", data.Environment.ValueString(), data.Service.ValueString())
	if data.ID.ValueString() != "" {
		p = p + "/" + data.ID.ValueString()
	}
	return p
}

func (r *kms_folderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data KMSFolderResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	keystoreName, ok := r.resolveKeystoreName(ctx, &data)
	if !ok {
		resp.Diagnostics.AddError("Failed to resolve keystore", fmt.Sprintf("Could not resolve keystore ID %s to a name", data.Keystore.ValueString()))
		return
	}

	api := KMSFolderAPIModel{
		Name:         data.Name.ValueString(),
		KeystoreName: keystoreName,
	}
	if !data.ParentFolderID.IsNull() && data.ParentFolderID.ValueString() != "" {
		api.ParentFolderID = data.ParentFolderID.ValueString()
	}

	ok, _ = r.apiRequest(ctx, http.MethodPost, r.apiPath(&data), api, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	data.ID = types.StringValue(api.ID)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *kms_folderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data KMSFolderResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	var api KMSFolderAPIModel
	ok, status := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics, Allow404())
	if !ok {
		return
	}
	if status == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	data.ID = types.StringValue(api.ID)
	data.Name = types.StringValue(api.Name)
	if api.ParentFolderID != "" {
		data.ParentFolderID = types.StringValue(api.ParentFolderID)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *kms_folderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// All mutable fields are RequiresReplace — this method should never be called.
	var data KMSFolderResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *kms_folderResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data KMSFolderResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data), nil, nil, &resp.Diagnostics, Allow404())

	r.waitForRemoval(ctx, r.apiPath(&data), &resp.Diagnostics)
}
