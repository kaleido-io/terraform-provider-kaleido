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

type CMSActionDeployResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Environment      types.String `tfsdk:"environment"`
	Service          types.String `tfsdk:"service"`
	Build            types.String `tfsdk:"build"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	FireFlyNamespace types.String `tfsdk:"firefly_namespace"`
	SigningKey       types.String `tfsdk:"signing_key"`
	TransactionID    types.String `tfsdk:"transaction_id"`
	IdempotencyKey   types.String `tfsdk:"idempotency_key"`
	OperationID      types.String `tfsdk:"operation_id"`
	ContractAddress  types.String `tfsdk:"contract_address"`
	BlockNumber      types.String `tfsdk:"block_number"`
}

type CMSActionDeployAPIModel struct {
	CMSActionBaseAPIModel
	Input  CMSDeployActionInputAPIModel   `json:"input,omitempty"`
	Output *CMSDeployActionOutputAPIModel `json:"output,omitempty"`
}

type CMSDeployActionInputAPIModel struct {
	Namespace         string                             `json:"namespace,omitempty"`
	Build             *CMSActionDeployBuildInputAPIModel `json:"build,omitempty"`
	SingingKey        string                             `json:"signingKey,omitempty"`
	ConstructorParams interface{}                        `json:"constructorParams,omitempty"`
}

type CMSDeployActionOutputAPIModel struct {
	CMSActionOutputBaseAPIModel
	TransactionID  string                                `json:"transactionId,omitempty"`
	IdempotencyKey string                                `json:"idempotencyKey,omitempty"`
	OperationID    string                                `json:"operationId,omitempty"`
	Location       CMSDeployActionOutputLocationAPIModel `json:"location,omitempty"`
	BlockNumber    string                                `json:"blockNumber,omitempty"`
}

type CMSDeployActionOutputLocationAPIModel struct {
	Address string `json:"address,omitempty"`
}

type CMSActionDeployBuildInputAPIModel struct {
	ID string `json:"id"`
}

func CMSActionDeployResourceFactory() resource.Resource {
	return &cms_action_deployResource{}
}

func (data *CMSActionDeployResourceModel) ResourceIdentifiers() (types.String, types.String, types.String) {
	return data.Environment, data.Service, data.ID
}

type cms_action_deployResource struct {
	cms_action_baseResource
}

func (a *CMSActionDeployAPIModel) OutputBase() *CMSActionOutputBaseAPIModel {
	if a.Output == nil {
		return nil
	}
	return &a.Output.CMSActionOutputBaseAPIModel
}

func (r *cms_action_deployResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_cms_action_deploy"
}

func (r *cms_action_deployResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"signing_key": &schema.StringAttribute{
				Required:      true,
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
			"contract_address": &schema.StringAttribute{
				Computed: true,
			},
			"block_number": &schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (data *CMSActionDeployResourceModel) toAPI(api *CMSActionDeployAPIModel) {
	api.Type = "deploy"
	api.Name = data.Name.ValueString()
	api.Description = data.Description.ValueString()
	api.Input = CMSDeployActionInputAPIModel{
		Namespace: data.FireFlyNamespace.ValueString(),
		Build: &CMSActionDeployBuildInputAPIModel{
			ID: data.Build.ValueString(),
		},
		SingingKey: data.SigningKey.ValueString(),
	}
}

func (api *CMSActionDeployAPIModel) toData(data *CMSActionDeployResourceModel) {
	data.ID = types.StringValue(api.ID)
	if api.Output != nil {
		data.TransactionID = types.StringValue(api.Output.TransactionID)
		data.IdempotencyKey = types.StringValue(api.Output.IdempotencyKey)
		data.OperationID = types.StringValue(api.Output.OperationID)
		data.ContractAddress = types.StringValue(api.Output.Location.Address)
		data.BlockNumber = types.StringValue(api.Output.BlockNumber)
	}
}

func (r *cms_action_deployResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var data CMSActionDeployResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	var api CMSActionDeployAPIModel
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

func (r *cms_action_deployResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

	var data CMSActionDeployResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)

	// Update from plan
	var api CMSActionDeployAPIModel
	data.toAPI(&api)
	if ok, _ := r.apiRequest(ctx, http.MethodPatch, r.apiPath(&data), api, &api, &resp.Diagnostics); !ok {
		return
	}

	api.toData(&data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *cms_action_deployResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CMSActionDeployResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	var api CMSActionDeployAPIModel
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

func (r *cms_action_deployResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CMSActionDeployResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data), nil, nil, &resp.Diagnostics, Allow404())

	r.waitForRemoval(ctx, r.apiPath(&data), &resp.Diagnostics)
}
