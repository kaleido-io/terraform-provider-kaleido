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
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type CMSActionCreateAPIResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Environment      types.String `tfsdk:"environment"`
	Service          types.String `tfsdk:"service"`
	Build            types.String `tfsdk:"build"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	FireFlyNamespace types.String `tfsdk:"firefly_namespace"`
	APIName          types.String `tfsdk:"api_name"`
	ContractAddress  types.String `tfsdk:"contract_address"`
	APIID            types.String `tfsdk:"api_id"`
	Publish          types.Bool   `tfsdk:"publish"`
	IgnoreDestroy    types.Bool   `tfsdk:"ignore_destroy"`
}

type CMSActionCreateAPIAPIModel struct {
	CMSActionBaseAPIModel
	Input  CMSCreateAPIActionInputAPIModel   `json:"input,omitempty"`
	Output *CMSCreateAPIActionOutputAPIModel `json:"output,omitempty"`
}

type CMSCreateAPIActionInputAPIModel struct {
	Namespace string                                   `json:"namespace,omitempty"`
	Build     *CMSActionCreateAPIBuildInputAPIModel    `json:"build,omitempty"`
	APIName   string                                   `json:"apiName,omitempty"`
	Location  *CMSCreateAPIActionInputLocationAPIModel `json:"location,omitempty"`
	Publish   *bool                                    `json:"publish,omitempty"`
}

type CMSCreateAPIActionInputLocationAPIModel struct {
	Address string `json:"address,omitempty"`
}

type CMSCreateAPIActionOutputAPIModel struct {
	CMSActionOutputBaseAPIModel
	APIID string `json:"apiId,omitempty"`
}

type CMSActionCreateAPIBuildInputAPIModel struct {
	ID string `json:"id"`
}

func CMSActionCreateAPIResourceFactory() resource.Resource {
	return &cms_action_createapiResource{}
}

func (data *CMSActionCreateAPIResourceModel) ResourceIdentifiers() (types.String, types.String, types.String) {
	return data.Environment, data.Service, data.ID
}

type cms_action_createapiResource struct {
	cms_action_baseResource
}

func (a *CMSActionCreateAPIAPIModel) OutputBase() *CMSActionOutputBaseAPIModel {
	if a.Output == nil {
		return nil
	}
	return &a.Output.CMSActionOutputBaseAPIModel
}

func (r *cms_action_createapiResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_cms_action_createapi"
}

func (r *cms_action_createapiResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"description": &schema.StringAttribute{
				Optional: true,
			},
			"firefly_namespace": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"build": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"api_name": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"contract_address": &schema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"api_id": &schema.StringAttribute{
				Computed: true,
			},
			"publish": &schema.BoolAttribute{
				Optional: true,
			},
			"ignore_destroy": &schema.BoolAttribute{
				Optional: true,
			},
		},
	}
}

func (data *CMSActionCreateAPIResourceModel) toAPI(api *CMSActionCreateAPIAPIModel) {
	api.Type = "createapi"
	api.Name = data.Name.ValueString()
	api.Description = data.Description.ValueString()
	api.Input = CMSCreateAPIActionInputAPIModel{
		Namespace: data.FireFlyNamespace.ValueString(),
		Build: &CMSActionCreateAPIBuildInputAPIModel{
			ID: data.Build.ValueString(),
		},
		APIName: data.APIName.ValueString(),
	}
	if data.ContractAddress.ValueString() != "" {
		api.Input.Location = &CMSCreateAPIActionInputLocationAPIModel{
			Address: data.ContractAddress.ValueString(),
		}
	}

	if !data.Publish.IsNull() && !data.Publish.IsUnknown() {
		api.Input.Publish = data.Publish.ValueBoolPointer()
	}
}

func (api *CMSActionCreateAPIAPIModel) toData(data *CMSActionCreateAPIResourceModel) {
	data.ID = types.StringValue(api.ID)
	if api.Output != nil {
		data.APIID = types.StringValue(api.Output.APIID)
	}
}

func (r *cms_action_createapiResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var data CMSActionCreateAPIResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	var api CMSActionCreateAPIAPIModel
	data.toAPI(&api)
	ok, _ := r.apiRequest(ctx, http.MethodPost, r.apiPath(&data), api, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	api.toData(&data) // need the ID copied over
	r.waitForActionStatus(ctx, &data, &api, &resp.Diagnostics)
	api.toData(&data) // capture the build info
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)

}

func (r *cms_action_createapiResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

	var data CMSActionCreateAPIResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)

	// Update from plan
	var api CMSActionCreateAPIAPIModel
	data.toAPI(&api)
	if ok, _ := r.apiRequest(ctx, http.MethodPatch, r.apiPath(&data), api, &api, &resp.Diagnostics); !ok {
		return
	}

	api.toData(&data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *cms_action_createapiResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CMSActionCreateAPIResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	var api CMSActionCreateAPIAPIModel
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

func (r *cms_action_createapiResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CMSActionCreateAPIResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if !data.IgnoreDestroy.IsNull() && data.IgnoreDestroy.ValueBool() {
		return
	}

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data), nil, nil, &resp.Diagnostics, Allow404())

	r.waitForRemoval(ctx, r.apiPath(&data), &resp.Diagnostics)
}
