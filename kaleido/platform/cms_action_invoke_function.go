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
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	SigningKey       types.String `tfsdk:"signing_key"`
	ParamsJSON       types.String `tfsdk:"params_json"`
	MethodPath       types.String `tfsdk:"method_path"`
	ContractAddress  types.String `tfsdk:"contract_address"`
	TransactionID    types.String `tfsdk:"transaction_id"`
	IdempotencyKey   types.String `tfsdk:"idempotency_key"`
	OperationID      types.String `tfsdk:"operation_id"`
	BlockNumber      types.String `tfsdk:"block_number"`
}

type CMSActionInvokeFunctionAPIModel struct {
	CMSActionBaseAPIModel
	Input  CMSInvokeFunctionActionInputAPIModel   `json:"input,omitempty"`
	Output *CMSInvokeFunctionActionOutputAPIModel `json:"output,omitempty"`
}

type CMSInvokeFunctionActionInputAPIModel struct {
	Namespace  string                                        `json:"namespace,omitempty"`
	MethodPath string                                        `json:"methodPath,omitempty"`
	SigningKey string                                        `json:"signingKey,omitempty"`
	Params     interface{}                                   `json:"params,omitempty"`
	Build      *CMSActionInvokeFunctionBuildInputAPIModel    `json:"build,omitempty"`
	Location   *CMSInvokeFunctionActionInputLocationAPIModel `json:"location,omitempty"`
}

type CMSInvokeFunctionActionInputLocationAPIModel struct {
	Address string `json:"address,omitempty"`
}

type CMSInvokeFunctionActionOutputAPIModel struct {
	CMSActionOutputBaseAPIModel
	TransactionID  string `json:"transactionId,omitempty"`
	IdempotencyKey string `json:"idempotencyKey,omitempty"`
	OperationID    string `json:"operationId,omitempty"`
	BlockNumber    string `json:"blockNumber,omitempty"`
}

type CMSActionInvokeFunctionBuildInputAPIModel struct {
	ID string `json:"id"`
}

func CMSActionInvokeFunctionResourceFactory() resource.Resource {
	return &cms_action_invokefunctionResource{}
}

func (data *CMSActionInvokeFunctionResourceModel) ResourceIdentifiers() (types.String, types.String, types.String) {
	return data.Environment, data.Service, data.ID
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
	resp.TypeName = "kaleido_platform_cms_action_invoke_function"
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
			"build": &schema.StringAttribute{
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
			"signing_key": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"method_path": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"contract_address": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"params_json": &schema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"transaction_id": &schema.StringAttribute{
				Computed: true,
			},
			"idempotency_key": &schema.StringAttribute{
				Computed: true,
			},
			"operation_id": &schema.StringAttribute{
				Computed: true,
			},
			"block_number": &schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (data *CMSActionInvokeFunctionResourceModel) toAPI(api *CMSActionInvokeFunctionAPIModel, diagnostics *diag.Diagnostics) bool {
	api.Type = "invoke"
	api.Name = data.Name.ValueString()
	api.Description = data.Description.ValueString()
	api.Input = CMSInvokeFunctionActionInputAPIModel{
		Namespace:  data.FireFlyNamespace.ValueString(),
		MethodPath: data.MethodPath.ValueString(),
		SigningKey: data.SigningKey.ValueString(),
		Build: &CMSActionInvokeFunctionBuildInputAPIModel{
			ID: data.Build.ValueString(),
		},
		Location: &CMSInvokeFunctionActionInputLocationAPIModel{
			Address: data.ContractAddress.ValueString(),
		},
	}

	if data.ParamsJSON.ValueString() != "" {
		err := json.Unmarshal([]byte(data.ParamsJSON.ValueString()), &api.Input.Params)
		if err != nil {
			diagnostics.AddError("failed to serialize params JSON", err.Error())
			return false
		}
	}
	return true
}

func (api *CMSActionInvokeFunctionAPIModel) toData(data *CMSActionInvokeFunctionResourceModel) {
	data.ID = types.StringValue(api.ID)
	if api.Output != nil {
		data.TransactionID = types.StringValue(api.Output.TransactionID)
		data.IdempotencyKey = types.StringValue(api.Output.IdempotencyKey)
		data.OperationID = types.StringValue(api.Output.OperationID)
		data.BlockNumber = types.StringValue(api.Output.BlockNumber)
	}
}

func (r *cms_action_invokefunctionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var data CMSActionInvokeFunctionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	var api CMSActionInvokeFunctionAPIModel
	ok := data.toAPI(&api, &resp.Diagnostics)
	if ok {
		ok, _ = r.apiRequest(ctx, http.MethodPost, r.apiPath(&data), api, &api, &resp.Diagnostics)
	}
	if !ok {
		return
	}

	api.toData(&data) // need the ID copied over
	r.waitForActionStatus(ctx, &data, &api, &resp.Diagnostics)
	api.toData(&data) // capture the transaction info
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)

}

func (r *cms_action_invokefunctionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

	var data CMSActionInvokeFunctionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)

	// Update from plan
	var api CMSActionInvokeFunctionAPIModel
	ok := data.toAPI(&api, &resp.Diagnostics)
	// PATCH only supports editing the Name/Description fields
	patchAPI := CMSActionBaseAPIModel{
		Name:        api.Name,
		Description: api.Description,
	}
	if ok {
		ok, _ = r.apiRequest(ctx, http.MethodPatch, r.apiPath(&data), patchAPI, &api, &resp.Diagnostics)
	}
	if !ok {
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
