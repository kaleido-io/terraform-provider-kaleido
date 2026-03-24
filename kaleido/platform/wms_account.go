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

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type WMSAccountResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Environment    types.String `tfsdk:"environment"`
	Service        types.String `tfsdk:"service"`
	Asset          types.String `tfsdk:"asset"`
	Wallet         types.String `tfsdk:"wallet"`
	Identifier     types.String `tfsdk:"identifier"`
	IdentifierType types.String `tfsdk:"identifier_type"`
}

type WMSAccountAPIModel struct {
	ID             string `json:"id,omitempty"`
	Created        string `json:"created,omitempty"`
	Updated        string `json:"updated,omitempty"`
	AssetID        string `json:"assetId,omitempty"`
	WalletID       string `json:"walletId,omitempty"`
	Identifier     string `json:"identifier,omitempty"`
	IdentifierType string `json:"identifierType,omitempty"`
}

func WMSAccountResourceFactory() resource.Resource {
	return &wms_accountResource{}
}

type wms_accountResource struct {
	commonResource
}

func (r *wms_accountResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_wms_account"
}

func (r *wms_accountResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"service": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"asset": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"wallet": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"identifier": &schema.StringAttribute{
				Computed: true,
			},
			"identifier_type": &schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (r *wms_accountResource) apiPathConnect(data *WMSAccountResourceModel) string {
	path := fmt.Sprintf(
		"/endpoint/%s/%s/rest/api/v1/assets/%s/connect/%s",
		data.Environment.ValueString(),
		data.Service.ValueString(),
		data.Asset.ValueString(),
		data.Wallet.ValueString(),
	)

	return path
}

func (r *wms_accountResource) apiPath(data *WMSAccountResourceModel) string {
	path := fmt.Sprintf(
		"/endpoint/%s/%s/rest/api/v1/accounts/%s",
		data.Environment.ValueString(),
		data.Service.ValueString(),
		data.ID.ValueString(),
	)

	return path
}

func (api *WMSAccountAPIModel) toData(data *WMSAccountResourceModel) {
	data.ID = types.StringValue(api.ID)
	data.Wallet = types.StringValue(api.WalletID)
	data.Asset = types.StringValue(api.AssetID)
	data.Identifier = types.StringValue(api.Identifier)
	data.IdentifierType = types.StringValue(api.IdentifierType)

}

func (r *wms_accountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var data WMSAccountResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	var api WMSAccountAPIModel

	ok, _ := r.apiRequest(ctx, http.MethodPost, r.apiPathConnect(&data), &api, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	api.toData(&data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)

}

func (r *wms_accountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// all fields are marked as RequiresReplace, so we should not expect an Update

}

func (r *wms_accountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data WMSAccountResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	var api WMSAccountAPIModel
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

func (r *wms_accountResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data WMSAccountResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data), nil, nil, &resp.Diagnostics, Allow404())

	r.waitForRemoval(ctx, r.apiPath(&data), &resp.Diagnostics)
}
