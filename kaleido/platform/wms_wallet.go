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

type WMSWalletConfigKMSResourceModel struct {
	KeyID string `tfsdk:"key_id"`
}

type WMSWalletConfigReadonlyResourceModel struct {
	IdentifierMap types.Map `tfsdk:"identifier_map"`
}

type WMSWalletConfigResourceModel struct {
	Type     types.String                          `tfsdk:"type"`
	KMS      *WMSWalletConfigKMSResourceModel      `tfsdk:"kms"`
	Readonly *WMSWalletConfigReadonlyResourceModel `tfsdk:"readonly"`
}

type WMSWalletResourceModel struct {
	ID          types.String                  `tfsdk:"id"`
	Environment types.String                  `tfsdk:"environment"`
	Service     types.String                  `tfsdk:"service"`
	Name        types.String                  `tfsdk:"name"`
	Color       types.String                  `tfsdk:"color"`
	IconID      types.String                  `tfsdk:"icon_id"`
	Config      *WMSWalletConfigResourceModel `tfsdk:"config"`
}

type WMSWalletConfigKMSAPIModel struct {
	KeyID string `json:"keyId,omitempty"`
}

type WMSWalletConfigReadonlyAPIModel struct {
	IdentifierMap map[string]string `json:"identifierMap,omitempty"`
}

type WMSWalletConfigAPIModel struct {
	Type     string                           `json:"type"`
	KMS      *WMSWalletConfigKMSAPIModel      `json:"kms,omitempty"`
	Readonly *WMSWalletConfigReadonlyAPIModel `json:"readonly,omitempty"`
}

type WMSWalletAPIModel struct {
	ID      string                  `json:"id,omitempty"`
	Created string                  `json:"created,omitempty"`
	Updated string                  `json:"updated,omitempty"`
	Name    string                  `json:"name,omitempty"`
	Color   string                  `json:"color,omitempty"`
	Config  WMSWalletConfigAPIModel `json:"config,omitempty"`
	IconID  string                  `json:"iconId,omitempty"`
}

func WMSWalletResourceFactory() resource.Resource {
	return &wms_walletResource{}
}

type wms_walletResource struct {
	commonResource
}

func (r *wms_walletResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_wms_wallet"
}

func (r *wms_walletResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"service": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
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
			"config": &schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"type": &schema.StringAttribute{
						Required: true,
					},
					"kms": &schema.SingleNestedAttribute{
						Optional: true,
						Attributes: map[string]schema.Attribute{
							"key_id": &schema.StringAttribute{
								Required: true,
							},
						},
					},
					"readonly": &schema.SingleNestedAttribute{
						Optional: true,
						Attributes: map[string]schema.Attribute{
							"identifier_map": &schema.MapAttribute{
								Required:    true,
								ElementType: types.StringType,
							},
						},
					},
				},
			},
		},
	}
}

func (api *WMSWalletAPIModel) toData(ctx context.Context, data *WMSWalletResourceModel, diagnostics *diag.Diagnostics) {
	data.ID = types.StringValue(api.ID)
	data.Name = types.StringValue(api.Name)
	data.Config = &WMSWalletConfigResourceModel{}
	data.Config.Type = types.StringValue(api.Config.Type)
	if api.Config.KMS != nil {
		data.Config.KMS = &WMSWalletConfigKMSResourceModel{}
		data.Config.KMS.KeyID = api.Config.KMS.KeyID
	} else {
		data.Config.KMS = nil
	}
	if api.Config.Readonly != nil {
		data.Config.Readonly = &WMSWalletConfigReadonlyResourceModel{}

		identityMap := make(map[string]string)
		for account, value := range api.Config.Readonly.IdentifierMap {
			identityMap[account] = value
		}
		mapValue, diag := types.MapValueFrom(ctx, types.StringType, identityMap)
		diagnostics.Append(diag...)
		data.Config.Readonly.IdentifierMap = mapValue
	}

	data.Color = types.StringValue(api.Color)
	if api.IconID == "" {
		data.IconID = types.StringNull()
	} else {
		data.IconID = types.StringValue(api.IconID)
	}
}

func (data *WMSWalletResourceModel) toAPI(ctx context.Context, api *WMSWalletAPIModel, diagnostics *diag.Diagnostics) bool {

	api.Name = data.Name.ValueString()
	api.Color = data.Color.ValueString()
	api.ID = data.ID.ValueString()
	if !data.Color.IsNull() {
		api.Color = data.Color.ValueString()
	}
	if !data.IconID.IsNull() {
		api.IconID = data.IconID.ValueString()
	}

	api.Config.Type = data.Config.Type.ValueString()
	if data.Config.KMS != nil {
		api.Config.KMS = &WMSWalletConfigKMSAPIModel{}
		api.Config.KMS.KeyID = data.Config.KMS.KeyID
	} else {
		api.Config.KMS = nil
	}
	if data.Config.Readonly != nil {
		api.Config.Readonly = &WMSWalletConfigReadonlyAPIModel{}
		identityMap := make(map[string]string)
		diag := data.Config.Readonly.IdentifierMap.ElementsAs(ctx, &identityMap, false)
		diagnostics.Append(diag...)
		api.Config.Readonly.IdentifierMap = identityMap
	} else {
		api.Config.Readonly = nil
	}

	return true

}

func (r *wms_walletResource) apiPath(data *WMSWalletResourceModel) string {
	path := fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/wallets", data.Environment.ValueString(), data.Service.ValueString())
	if data.ID.ValueString() != "" {
		path = path + "/" + data.ID.ValueString()
	}
	return path
}

func (r *wms_walletResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var data WMSWalletResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	var api WMSWalletAPIModel
	ok := data.toAPI(ctx, &api, &resp.Diagnostics)
	if ok {
		ok, _ = r.apiRequest(ctx, http.MethodPost, r.apiPath(&data), &api, &api, &resp.Diagnostics)
	}
	if !ok {
		return
	}

	api.toData(ctx, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)

}

func (r *wms_walletResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

	var data WMSWalletResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)

	var api WMSWalletAPIModel
	ok := data.toAPI(ctx, &api, &resp.Diagnostics)
	if ok {
		ok, _ = r.apiRequest(ctx, http.MethodPatch, r.apiPath(&data), &api, &api, &resp.Diagnostics)
	}
	if !ok {
		return
	}

	api.toData(ctx, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *wms_walletResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data WMSWalletResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	var api WMSWalletAPIModel
	api.ID = data.ID.ValueString()
	ok, status := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics, Allow404())
	if !ok {
		return
	}
	if status == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	api.toData(ctx, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *wms_walletResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data WMSWalletResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data), nil, nil, &resp.Diagnostics, Allow404())

	r.waitForRemoval(ctx, r.apiPath(&data), &resp.Diagnostics)
}
