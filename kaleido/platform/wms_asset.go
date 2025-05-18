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
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type WMSAssetResourceModel struct {
	ID                    types.String `tfsdk:"id"`
	Environment           types.String `tfsdk:"environment"`
	Service               types.String `tfsdk:"service"`
	Name                  types.String `tfsdk:"name"`
	Description           types.String `tfsdk:"description"`
	Symbol                types.String `tfsdk:"symbol"`
	ProtocolID            types.String `tfsdk:"protocol_id"`
	AccountIdentifierType types.String `tfsdk:"account_identifier_type"`
	Color                 types.String `tfsdk:"color"`
	IconID                types.String `tfsdk:"icon_id"`
	ConfigJSON            types.String `tfsdk:"config_json"`
}

type WMSAssetAPIModel struct {
	ID                    string      `json:"id,omitempty"`
	Created               string      `json:"created,omitempty"`
	Updated               string      `json:"updated,omitempty"`
	Name                  string      `json:"name,omitempty"`
	Symbol                string      `json:"symbol,omitempty"`
	ProtocolID            string      `json:"protocolId,omitempty"`
	AccountIdentifierType string      `json:"accountIdentifierType,omitempty"`
	Description           string      `json:"description,omitempty"`
	Color                 string      `json:"color,omitempty"`
	Config                interface{} `json:"config,omitempty,omitempty"`
	IconID                string      `json:"iconId,omitempty"`
}

func WMSAssetResourceFactory() resource.Resource {
	return &wms_assetResource{}
}

type wms_assetResource struct {
	commonResource
}

func (r *wms_assetResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_wms_asset"
}

func (r *wms_assetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"environment": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": &schema.StringAttribute{
				Required: true,
			},
			"description": &schema.StringAttribute{
				Optional: true,
			},
			"service": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"symbol": &schema.StringAttribute{
				Required:    true,
				Description: "The symbol for the asset",
			},
			"protocol_id": &schema.StringAttribute{
				Required:    true,
				Description: "The protocol / blockchain identifier for this asset - such as its ethereum address",
			},
			"account_identifier_type": &schema.StringAttribute{
				Required:    true,
				Description: "The type of account identifier required for a wallet to hold this asset - such as an eth_address",
			},
			"color": &schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "A HTML color to associate with the asset - randomly allocated if not supplied",
			},
			"icon_id": &schema.StringAttribute{
				Optional:    true,
				Description: "The id of the icon associated with the asset, if one has been uploaded",
			},
			"config_json": &schema.StringAttribute{
				Required:    true,
				Description: "The asset configuration",
			},
		},
	}
}

func (api *WMSAssetAPIModel) toData(data *WMSAssetResourceModel) {
	data.ID = types.StringValue(api.ID)
	data.Name = types.StringValue(api.Name)
	data.Symbol = types.StringValue(api.Symbol)
	data.ProtocolID = types.StringValue(api.ProtocolID)
	data.AccountIdentifierType = types.StringValue(api.AccountIdentifierType)

	if api.Description == "" {
		data.Description = types.StringNull()
	} else {
		data.Description = types.StringValue(api.Description)
	}
	data.Color = types.StringValue(api.Color)
	if api.IconID == "" {
		data.IconID = types.StringNull()
	} else {
		data.IconID = types.StringValue(api.IconID)
	}
	//TODO do we need to set data.ConfigJSON here?
}

func (data *WMSAssetResourceModel) toAPI(api *WMSAssetAPIModel, diagnostics *diag.Diagnostics) bool {

	api.Name = data.Name.ValueString()
	api.Symbol = data.Symbol.ValueString()
	api.ProtocolID = data.ProtocolID.ValueString()
	api.AccountIdentifierType = data.AccountIdentifierType.ValueString()
	api.Color = data.Color.ValueString()
	api.ID = data.ID.ValueString()
	if !data.Color.IsNull() {
		api.Color = data.Color.ValueString()
	}
	if !data.Description.IsNull() {
		api.Description = data.Description.ValueString()
	}
	if !data.IconID.IsNull() {
		api.IconID = data.IconID.ValueString()
	}

	err := json.Unmarshal([]byte(data.ConfigJSON.ValueString()), &api.Config)
	if err != nil {
		diagnostics.AddError("failed to serialize config JSON", err.Error())
		return false
	}
	return true

}

func (r *wms_assetResource) apiPath(data *WMSAssetResourceModel) string {
	path := fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/assets", data.Environment.ValueString(), data.Service.ValueString())
	if data.ID.ValueString() != "" {
		path = path + "/" + data.ID.ValueString()
	}
	return path
}

func (r *wms_assetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var data WMSAssetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	var api WMSAssetAPIModel
	ok := data.toAPI(&api, &resp.Diagnostics)
	if ok {
		ok, _ = r.apiRequest(ctx, http.MethodPost, r.apiPath(&data), &api, &api, &resp.Diagnostics)
	}
	if !ok {
		return
	}

	api.toData(&data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)

}

func (r *wms_assetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

	var data WMSAssetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)

	var api WMSAssetAPIModel
	ok := data.toAPI(&api, &resp.Diagnostics)
	if ok {
		ok, _ = r.apiRequest(ctx, http.MethodPatch, r.apiPath(&data), &api, &api, &resp.Diagnostics)
	}
	if !ok {
		return
	}

	api.toData(&data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *wms_assetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data WMSAssetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	var api WMSAssetAPIModel
	api.ID = data.ID.ValueString()
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

func (r *wms_assetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data WMSAssetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data), nil, nil, &resp.Diagnostics, Allow404())

	r.waitForRemoval(ctx, r.apiPath(&data), &resp.Diagnostics)
}
