// Copyright © Kaleido, Inc. 2026

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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ConnectorFlowResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Environment        types.String `tfsdk:"environment"`
	Service            types.String `tfsdk:"service"`
	Name               types.String `tfsdk:"name"`
	Description        types.String `tfsdk:"description"`
	ConfigTypeBindings types.Map    `tfsdk:"config_type_bindings"`
	FlowType           types.String `tfsdk:"flow_type"`
	CurrentVersion    types.String `tfsdk:"current_version"`
}

type ConfigProfileBindingTargetInput struct {
	ConfigProfile   string `json:"configProfile,omitempty"`
	ConfigProfileID string `json:"configProfileId,omitempty"`
}

type ConnectorFlowDeployAPIModel struct {
	Name               string                                     `json:"name,omitempty"`
	Description        string                                     `json:"description,omitempty"`
	ConfigTypeBindings map[string]ConfigProfileBindingTargetInput `json:"configTypeBindings,omitempty"`
}

type ConnectorFlowUpgradeAPIModel struct {
	ConfigTypeBindings map[string]ConfigProfileBindingTargetInput `json:"configTypeBindings,omitempty"`
}

type ConnectorFlowAPIModel struct {
	ID             string `json:"id,omitempty"`
	Name           string `json:"name,omitempty"`
	FlowType       string `json:"flowType,omitempty"`
	CurrentVersion string `json:"currentVersion,omitempty"`
	Description    string `json:"description,omitempty"`
}

func ConnectorFlowResourceFactory() resource.Resource {
	return &connectorFlowResource{}
}

type connectorFlowResource struct {
	commonResource
}

func (r *connectorFlowResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_connector_flow"
}

func (r *connectorFlowResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Deploys a connector flow (workflow template) from a connector service's embedded definitions, binding it to user-supplied config profiles.",
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"environment": &schema.StringAttribute{
				Required:      true,
				Description:   "Environment ID",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"service": &schema.StringAttribute{
				Required:      true,
				Description:   "Connector service ID",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": &schema.StringAttribute{
				Required:      true,
				Description:   "Name of the connector flow template (e.g. submission, query)",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"description": &schema.StringAttribute{
				Optional:    true,
				Description: "Optional description override for the deployed workflow.",
			},
			"config_type_bindings": &schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Map of config type name to config profile name-or-ID. Each entry binds a config type referenced by the flow to a concrete config profile.",
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
			},
			"flow_type": &schema.StringAttribute{
				Computed:    true,
				Description: "The flow type as reported by the deployed workflow (e.g. submission, query).",
			},
			"current_version": &schema.StringAttribute{
				Computed:    true,
				Description: "The currently deployed template version.",
			},
		},
	}
}

func (r *connectorFlowResource) metadataPath(data *ConnectorFlowResourceModel, suffix string) string {
	return fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/metadata/connector-flows/%s%s",
		data.Environment.ValueString(), data.Service.ValueString(), data.Name.ValueString(), suffix)
}

func (r *connectorFlowResource) instancePath(data *ConnectorFlowResourceModel) string {
	return fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/connector-flows/%s",
		data.Environment.ValueString(), data.Service.ValueString(), data.Name.ValueString())
}

func (r *connectorFlowResource) collectBindings(data *ConnectorFlowResourceModel, diagnostics *diag.Diagnostics) map[string]ConfigProfileBindingTargetInput {
	bindings := map[string]ConfigProfileBindingTargetInput{}
	if data.ConfigTypeBindings.IsNull() || data.ConfigTypeBindings.IsUnknown() {
		return bindings
	}
	for k, v := range data.ConfigTypeBindings.Elements() {
		s, ok := v.(types.String)
		if !ok {
			diagnostics.AddError("Invalid binding", fmt.Sprintf("config_type_bindings[%s] is not a string", k))
			return nil
		}
		bindings[k] = ConfigProfileBindingTargetInput{ConfigProfile: s.ValueString()}
	}
	return bindings
}

func (r *connectorFlowResource) toData(api *ConnectorFlowAPIModel, data *ConnectorFlowResourceModel) {
	if api.ID != "" {
		data.ID = types.StringValue(api.ID)
	} else if data.ID.IsNull() || data.ID.IsUnknown() {
		data.ID = types.StringValue(api.Name)
	}
	data.FlowType = types.StringValue(api.FlowType)
	data.CurrentVersion = types.StringValue(api.CurrentVersion)
}

func (r *connectorFlowResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ConnectorFlowResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	bindings := r.collectBindings(&data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	body := ConnectorFlowDeployAPIModel{
		Name:               data.Name.ValueString(),
		ConfigTypeBindings: bindings,
	}
	if !data.Description.IsNull() {
		body.Description = data.Description.ValueString()
	}
	var api ConnectorFlowAPIModel
	ok, _ := r.apiRequest(ctx, http.MethodPost, r.metadataPath(&data, "/deploy"), &body, &api, &resp.Diagnostics)
	if !ok {
		return
	}
	r.toData(&api, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *connectorFlowResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ConnectorFlowResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var api ConnectorFlowAPIModel
	ok, status := r.apiRequest(ctx, http.MethodGet, r.instancePath(&data), nil, &api, &resp.Diagnostics, Allow404())
	if !ok {
		return
	}
	if status == 404 {
		resp.State.RemoveResource(ctx)
		return
	}
	r.toData(&api, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *connectorFlowResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ConnectorFlowResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)
	if resp.Diagnostics.HasError() {
		return
	}
	bindings := r.collectBindings(&data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	// Use /upgrade which accepts the same configTypeBindings and is idempotent on the
	// current template version. The connector-manager merges bindings server-side.
	// See .ai/plan.md for the open question on whether to switch to PATCH for
	// binding-only edits once that's verified end-to-end.
	body := ConnectorFlowUpgradeAPIModel{ConfigTypeBindings: bindings}
	var api ConnectorFlowAPIModel
	ok, _ := r.apiRequest(ctx, http.MethodPost, r.metadataPath(&data, "/upgrade"), &body, &api, &resp.Diagnostics)
	if !ok {
		return
	}
	r.toData(&api, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *connectorFlowResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ConnectorFlowResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.apiRequest(ctx, http.MethodDelete, r.instancePath(&data), nil, nil, &resp.Diagnostics, Allow404())
}
