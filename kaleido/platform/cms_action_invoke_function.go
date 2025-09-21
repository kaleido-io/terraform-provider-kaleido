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

type CMSActionInvokeFunctionResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Environment      types.String `tfsdk:"environment"`
	Service          types.String `tfsdk:"service"`
	Build            types.String `tfsdk:"build"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	FireFlyNamespace types.String `tfsdk:"firefly_namespace"`
	SigningKey         types.String `tfsdk:"signing_key"`
	ParamsJSON         types.String `tfsdk:"params_json"`
	TransactionID      types.String `tfsdk:"transaction_id"`
	IdempotencyKey     types.String `tfsdk:"idempotency_key"`
	OperationID        types.String `tfsdk:"operation_id"`
	ContractAddress    types.String `tfsdk:"contract_address"`
	MethodPath       types.String `tfsdk:"method_path"`
}

type CMSActionInvokeFunctionAPIModel struct {
	CMSActionBaseAPIModel
	Input  CMSInvokeFunctionActionInputAPIModel   `json:"input,omitempty"`
	Output *CMSInvokeFunctionActionOutputAPIModel `json:"output,omitempty"`
}

type CMSInvokeFunctionActionInputAPIModel struct {
	Namespace string                                   `json:"namespace,omitempty"`
	MethodPath string `json:"methodPath,omitempty"`
	SigningKey string `json:"signingKey,omitempty"`
	Params map[string]interface{} `json:"params,omitempty"`
	Build *CMSActionInvokeFunctionBuildInputAPIModel `json:"build,omitempty"`
	Location  *CMSInvokeFunctionActionInputLocationAPIModel `json:"location,omitempty"`
}

type CMSInvokeFunctionActionInputLocationAPIModel struct {
	Address string `json:"address,omitempty"`
	BlockNumber string `json:"blockNumber,omitempty"`
}

type CMSInvokeFunctionActionOutputAPIModel struct {
	CMSActionOutputBaseAPIModel
	APIID string `json:"apiId,omitempty"`
}

type CMSActionInvokeFunctionBuildInputAPIModel struct {
	ID string `json:"id"`
}

func CMSActionInvokeFunctionResourceFactory() resource.Resource {
	return &cms_action_createapiResource{}
}

func (data *CMSActionInvokeFunctionResourceModel) ResourceIdentifiers() (types.String, types.String, types.String) {
	//return data.Environment, data.Service, data.ID
	return "", "", ""
}

type cms_action_invokefunctionResource struct {
	cms_action_baseResource
}

func (a *CMSActionInvokeFunctionAPIModel) OutputBase() *CMSActionOutputBaseAPIModel {
	if a.Output == nil {
		return nil
	}
	return &a.Output.CMSActionOutputBaseAPIModel
}

func (r *cms_action_invokefunctionResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_cms_action_createapi"
}

func (r *cms_action_invokefunctionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"params_json": &schema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},
	}
}

func (data *CMSActionInvokeFunctionResourceModel) toAPI(api *CMSActionInvokeFunctionAPIModel) {
	api.Type = "invoke"
	api.Name = data.Name.ValueString()
	api.Description = data.Description.ValueString()
	api.Input = CMSInvokeFunctionActionInputAPIModel{
		Namespace: data.FireFlyNamespace.ValueString(),
		// Build: &CMSActionCreateAPIBuildInputAPIModel{
		// 	ID: data.Build.ValueString(),
		// },
		APIName: data.APIName.ValueString(),
	}
	if data.ContractAddress.ValueString() != "" {
		api.Input.Location = &CMSInvokeFunctionActionInputLocationAPIModel{
			Address: data.ContractAddress.ValueString(),
		}
	}
}

func (api *CMSActionInvokeFunctionAPIModel) toData(data *CMSActionInvokeFunctionResourceModel) {
	data.ID = types.StringValue(api.ID)
	if api.Output != nil {
		data.APIID = types.StringValue(api.Output.APIID)
	}
}

func (r *cms_action_invokefunctionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var data CMSActionInvokeFunctionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	var api CMSActionInvokeFunctionAPIModel
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

func (r *cms_action_invokefunctionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

	var data CMSActionInvokeFunctionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)

	// Update from plan
	var api CMSActionInvokeFunctionAPIModel
	data.toAPI(&api)
	if ok, _ := r.apiRequest(ctx, http.MethodPatch, r.apiPath(&data), api, &api, &resp.Diagnostics); !ok {
		return
	}

	api.toData(&data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *cms_action_invokefunctionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CMSActionInvokeFunctionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	var api CMSActionInvokeFunctionAPIModel
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

func (r *cms_action_invokefunctionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CMSActionInvokeFunctionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data), nil, nil, &resp.Diagnostics, Allow404())

	r.waitForRemoval(ctx, r.apiPath(&data), &resp.Diagnostics)
}
