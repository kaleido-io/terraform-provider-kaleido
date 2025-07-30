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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type AMSFFListenerResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Environment types.String `tfsdk:"environment"`
	Service     types.String `tfsdk:"service"`
	Name        types.String `tfsdk:"name"`
	Disabled    types.Bool   `tfsdk:"disabled"`
	ConfigJSON  types.String `tfsdk:"config_json"`
}

type AMSFFListenerAPIModel struct {
	ID       string      `json:"id,omitempty"`
	Name     string      `json:"name,omitempty"`
	Created  string      `json:"created,omitempty"`
	Updated  string      `json:"updated,omitempty"`
	Disabled bool        `json:"disabled"`
	Config   interface{} `json:"config,omitempty"`
}

func AMSFFListenerResourceFactory() resource.Resource {
	return &ams_fflistenerResource{}
}

type ams_fflistenerResource struct {
	commonResource
}

func (r *ams_fflistenerResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_ams_fflistener"
}

func (r *ams_fflistenerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"name": &schema.StringAttribute{
				Required: true,
			},
			"disabled": &schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			// TODO what is this and can be it properly typed ?
			"config_json": &schema.StringAttribute{
				Required: true,
			},
		},
	}
}

func (data *AMSFFListenerResourceModel) toAPI(api *AMSFFListenerAPIModel, diagnostics *diag.Diagnostics) bool {
	api.Name = data.Name.ValueString()
	api.Disabled = data.Disabled.ValueBool()
	err := json.Unmarshal([]byte(data.ConfigJSON.ValueString()), &api.Config)
	if err != nil {
		diagnostics.AddError("failed to serialize config JSON", err.Error())
		return false
	}
	return true
}

func (api *AMSFFListenerAPIModel) toData(data *AMSFFListenerResourceModel) {
	data.ID = types.StringValue(api.ID)
	data.Disabled = types.BoolValue(api.Disabled)
}

func (r *ams_fflistenerResource) apiPath(data *AMSFFListenerResourceModel, idOrName string) string {
	return fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/listeners/firefly/%s", data.Environment.ValueString(), data.Service.ValueString(), idOrName)
}

func (r *ams_fflistenerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var data AMSFFListenerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	var api AMSFFListenerAPIModel
	ok := data.toAPI(&api, &resp.Diagnostics)
	if ok {
		ok, _ = r.apiRequest(ctx, http.MethodPut, r.apiPath(&data, data.Name.ValueString()), &api, &api, &resp.Diagnostics)
	}
	if !ok {
		return
	}

	api.toData(&data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)

}

func (r *ams_fflistenerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

	var data AMSFFListenerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)

	var api AMSFFListenerAPIModel
	ok := data.toAPI(&api, &resp.Diagnostics)
	if ok {
		ok, _ = r.apiRequest(ctx, http.MethodPut, r.apiPath(&data, data.ID.ValueString()), &api, &api, &resp.Diagnostics)
	}
	if !ok {
		return
	}

	api.toData(&data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *ams_fflistenerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AMSFFListenerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	var api AMSFFListenerAPIModel
	api.ID = data.ID.ValueString()
	ok, status := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data, data.ID.ValueString()), nil, &api, &resp.Diagnostics, Allow404())
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

func (r *ams_fflistenerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AMSFFListenerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data, data.ID.ValueString()), nil, nil, &resp.Diagnostics, Allow404())

	r.waitForRemoval(ctx, r.apiPath(&data, data.ID.ValueString()), &resp.Diagnostics)
}
