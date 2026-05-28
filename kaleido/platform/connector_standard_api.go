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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ConnectorStandardAPIResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Environment      types.String `tfsdk:"environment"`
	Service          types.String `tfsdk:"service"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	FlowTypeBindings types.Map    `tfsdk:"flow_type_bindings"`
	CurrentVersion   types.String `tfsdk:"current_version"`
}

type ConnectorStandardAPIDeployAPIModel struct {
	Name             string            `json:"name,omitempty"`
	Description      string            `json:"description,omitempty"`
	FlowTypeBindings map[string]string `json:"flowTypeBindings,omitempty"`
}

type ConnectorStandardAPIAPIModel struct {
	ID             string `json:"id,omitempty"`
	Name           string `json:"name,omitempty"`
	CurrentVersion string `json:"currentVersion,omitempty"`
}

func ConnectorStandardAPIResourceFactory() resource.Resource {
	return &connectorStandardAPIResource{}
}

type connectorStandardAPIResource struct {
	commonResource
}

func (r *connectorStandardAPIResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_connector_standard_api"
}

func (r *connectorStandardAPIResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Deploys a standard API from a connector service's embedded definitions, binding subflow types to deployed connector flows.",
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
				Description:   "Standard API template name (e.g. evm, bitcoin)",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"description": &schema.StringAttribute{
				Optional:    true,
				Description: "Optional description override.",
			},
			"flow_type_bindings": &schema.MapAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "Map of connector-flow type to deployed connector-flow name. The server iterates the standard API template's subflowBindingTypes (binding name -> flow type) and resolves each by looking up flow_type_bindings[flowType]; so multiple bindings sharing a flow type collapse to one entry here. Keys are the *values* of subflowBindingTypes (e.g. submission, query), not the binding names.",
			},
			"current_version": &schema.StringAttribute{
				Computed:    true,
				Description: "The currently deployed template version.",
			},
		},
	}
}

func (r *connectorStandardAPIResource) metadataPath(data *ConnectorStandardAPIResourceModel, suffix string) string {
	return fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/metadata/standard-apis/%s%s",
		data.Environment.ValueString(), data.Service.ValueString(), data.Name.ValueString(), suffix)
}

func (r *connectorStandardAPIResource) instancePath(data *ConnectorStandardAPIResourceModel) string {
	return fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/apis/%s",
		data.Environment.ValueString(), data.Service.ValueString(), data.Name.ValueString())
}

func (r *connectorStandardAPIResource) collectBindings(data *ConnectorStandardAPIResourceModel, diagnostics *diag.Diagnostics) map[string]string {
	bindings := map[string]string{}
	for k, v := range data.FlowTypeBindings.Elements() {
		s, ok := v.(types.String)
		if !ok {
			diagnostics.AddError("Invalid binding", fmt.Sprintf("flow_type_bindings[%s] is not a string", k))
			return nil
		}
		bindings[k] = s.ValueString()
	}
	return bindings
}

func (r *connectorStandardAPIResource) toData(api *ConnectorStandardAPIAPIModel, data *ConnectorStandardAPIResourceModel) {
	if api.ID != "" {
		data.ID = types.StringValue(api.ID)
	} else if data.ID.IsNull() || data.ID.IsUnknown() {
		data.ID = types.StringValue(api.Name)
	}
	data.CurrentVersion = types.StringValue(api.CurrentVersion)
}

func (r *connectorStandardAPIResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ConnectorStandardAPIResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	bindings := r.collectBindings(&data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	body := ConnectorStandardAPIDeployAPIModel{
		Name:             data.Name.ValueString(),
		FlowTypeBindings: bindings,
	}
	if !data.Description.IsNull() {
		body.Description = data.Description.ValueString()
	}
	var api ConnectorStandardAPIAPIModel
	ok, _ := r.apiRequest(ctx, http.MethodPost, r.metadataPath(&data, "/deploy"), &body, &api, &resp.Diagnostics)
	if !ok {
		return
	}
	r.toData(&api, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *connectorStandardAPIResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ConnectorStandardAPIResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var api ConnectorStandardAPIAPIModel
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

func (r *connectorStandardAPIResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ConnectorStandardAPIResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var api ConnectorStandardAPIAPIModel
	ok, _ := r.apiRequest(ctx, http.MethodPost, r.metadataPath(&data, "/upgrade"), map[string]any{}, &api, &resp.Diagnostics)
	if !ok {
		return
	}
	r.toData(&api, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *connectorStandardAPIResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ConnectorStandardAPIResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.apiRequest(ctx, http.MethodDelete, r.instancePath(&data), nil, nil, &resp.Diagnostics, Allow404())
}
