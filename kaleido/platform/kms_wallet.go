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
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type KMSWalletResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Environment types.String `tfsdk:"environment"`
	Service     types.String `tfsdk:"service"`
	Type        types.String `tfsdk:"type"`
	Name        types.String `tfsdk:"name"`
	ConfigJSON  types.String `tfsdk:"config_json"`
}

type KMSWalletAPIModel struct {
	ID            string                 `json:"id,omitempty"`
	Created       *time.Time             `json:"created,omitempty"`
	Updated       *time.Time             `json:"updated,omitempty"`
	Type          string                 `json:"type"`
	Name          string                 `json:"name"`
	Configuration map[string]interface{} `json:"config"`
}

func KMSWalletResourceFactory() resource.Resource {
	return &kms_walletResource{}
}

type kms_walletResource struct {
	commonResource
}

func (r *kms_walletResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_kms_wallet"
}

func (r *kms_walletResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"type": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": &schema.StringAttribute{
				Required: true,
			},
			"config_json": &schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
		},
	}
}

func (data *KMSWalletResourceModel) toAPI(api *KMSWalletAPIModel) {
	// required fields
	api.Type = data.Type.ValueString()
	api.Name = data.Name.ValueString()
	api.Configuration = map[string]interface{}{}
	if !data.ConfigJSON.IsNull() {
		_ = json.Unmarshal([]byte(data.ConfigJSON.ValueString()), &api.Configuration)
	}
}

func (api *KMSWalletAPIModel) toData(data *KMSWalletResourceModel) {
	var config string
	if api.Configuration != nil {
		d, err := json.Marshal(api.Configuration)
		if err == nil {
			config = string(d)
		}
	} else {
		config = `{}`
	}
	data.ID = types.StringValue(api.ID)
	data.ConfigJSON = types.StringValue(config)
}

func (r *kms_walletResource) apiPath(data *KMSWalletResourceModel) string {
	path := fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/wallets", data.Environment.ValueString(), data.Service.ValueString())
	if data.ID.ValueString() != "" {
		path = path + "/" + data.ID.ValueString()
	}
	return path
}

func (r *kms_walletResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var data KMSWalletResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	var api KMSWalletAPIModel
	data.toAPI(&api)
	ok, _ := r.apiRequest(ctx, http.MethodPost, r.apiPath(&data), api, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	api.toData(&data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)

}

func (r *kms_walletResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

	var data KMSWalletResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)

	// Read full current object
	var api KMSWalletAPIModel
	if ok, _ := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics); !ok {
		return
	}

	// Update from plan
	data.toAPI(&api)
	if ok, _ := r.apiRequest(ctx, http.MethodPut, r.apiPath(&data), api, &api, &resp.Diagnostics); !ok {
		return
	}

	api.toData(&data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *kms_walletResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data KMSWalletResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	var api KMSWalletAPIModel
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

func (r *kms_walletResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data KMSWalletResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data), nil, nil, &resp.Diagnostics, Allow404())

	r.waitForRemoval(ctx, r.apiPath(&data), &resp.Diagnostics)
}
