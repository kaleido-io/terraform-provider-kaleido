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
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type ConnectorFlowConfigBindingResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Environment    types.String `tfsdk:"environment"`
	Service        types.String `tfsdk:"service"`
	Flow           types.String `tfsdk:"flow"`
	ConfigType     types.String `tfsdk:"config_type"`
	DynamicMapping types.Object `tfsdk:"dynamic_mapping"`
}

type ConnectorFlowConfigBindingDynamicMappingModel struct {
	NamePrefix types.String `tfsdk:"name_prefix"`
	JSONata    types.String `tfsdk:"jsonata"`
}

type ConnectorFlowConfigBindingDynamicMappingAPI struct {
	NamePrefix string `json:"namePrefix,omitempty"`
	JSONata    string `json:"JSONata,omitempty"`
}

type ConnectorFlowConfigBindingAPIModel struct {
	ID                    string                                       `json:"id,omitempty"`
	WorkflowConfigProfile string                                       `json:"workflowConfigProfile,omitempty"`
	DynamicMapping        *ConnectorFlowConfigBindingDynamicMappingAPI `json:"dynamicMapping,omitempty"`
}

type ConnectorFlowConfigBindingPatchAPIModel struct {
	DynamicMapping *ConnectorFlowConfigBindingDynamicMappingAPI `json:"dynamicMapping"`
}

func ConnectorFlowConfigBindingResourceFactory() resource.Resource {
	return &connectorFlowConfigBindingResource{}
}

type connectorFlowConfigBindingResource struct {
	commonResource
}

func (r *connectorFlowConfigBindingResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_connector_flow_config_binding"
}

var dynamicMappingAttrTypes = map[string]attr.Type{
	"name_prefix": types.StringType,
	"jsonata":     types.StringType,
}

func (r *connectorFlowConfigBindingResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the dynamic mapping on a config profile binding within a deployed connector flow. Use this resource to select a gas pricing profile (or any other config profile) dynamically per-transaction based on a JSONata expression evaluated against the transaction payload.",
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				Description:   "The binding ID (fcb:...) assigned by the server.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"environment": &schema.StringAttribute{
				Required:      true,
				Description:   "Environment ID.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"service": &schema.StringAttribute{
				Required:      true,
				Description:   "Connector service ID.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"flow": &schema.StringAttribute{
				Required:      true,
				Description:   "Name of the deployed connector flow (e.g. submission).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"config_type": &schema.StringAttribute{
				Required:      true,
				Description:   "Config type name of the binding to update (e.g. evm.gasPricing).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"dynamic_mapping": &schema.SingleNestedAttribute{
				Required:    true,
				Description: "Dynamic mapping that selects the config profile name at transaction submission time via a JSONata expression.",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"name_prefix": &schema.StringAttribute{
						Required:    true,
						Description: "Prefix prepended to the profile name returned by the JSONata expression (e.g. \"s:serviceId/\").",
					},
					"jsonata": &schema.StringAttribute{
						Required:    true,
						Description: "JSONata expression evaluated against the transaction state to produce a config profile short name.",
					},
				},
			},
		},
	}
}

func (r *connectorFlowConfigBindingResource) listPath(data *ConnectorFlowConfigBindingResourceModel) string {
	return fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/connector-flows/%s/config-profile-bindings",
		data.Environment.ValueString(), data.Service.ValueString(), data.Flow.ValueString())
}

func (r *connectorFlowConfigBindingResource) instancePath(data *ConnectorFlowConfigBindingResourceModel) string {
	return fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/connector-flows/%s/config-profile-bindings/%s",
		data.Environment.ValueString(), data.Service.ValueString(), data.Flow.ValueString(), data.ID.ValueString())
}

// findBindingByConfigType GETs the list of bindings filtered by config type and returns the first match.
func (r *connectorFlowConfigBindingResource) findBindingByConfigType(ctx context.Context, data *ConnectorFlowConfigBindingResourceModel, diagnostics *diag.Diagnostics) *ConnectorFlowConfigBindingAPIModel {
	configType := data.ConfigType.ValueString()
	profileSuffix := configType
	if i := strings.LastIndex(configType, "."); i >= 0 {
		profileSuffix = configType[i+1:]
	}
	listURL := r.listPath(data) + "?workflowconfigprofile=" + profileSuffix
	var result struct {
		Items []ConnectorFlowConfigBindingAPIModel `json:"items"`
	}
	ok, _ := r.apiRequest(ctx, http.MethodGet, listURL, nil, &result, diagnostics)
	if !ok {
		return nil
	}
	if len(result.Items) > 0 {
		return &result.Items[0]
	}
	diagnostics.AddError("Binding not found",
		fmt.Sprintf("no config-profile binding found for config type %q on flow %q", data.ConfigType.ValueString(), data.Flow.ValueString()))
	return nil
}

func (r *connectorFlowConfigBindingResource) toData(api *ConnectorFlowConfigBindingAPIModel, data *ConnectorFlowConfigBindingResourceModel, diagnostics *diag.Diagnostics) {
	data.ID = types.StringValue(api.ID)
	if api.DynamicMapping != nil {
		dm, diags := types.ObjectValueFrom(context.Background(), dynamicMappingAttrTypes, ConnectorFlowConfigBindingDynamicMappingModel{
			NamePrefix: types.StringValue(api.DynamicMapping.NamePrefix),
			JSONata:    types.StringValue(api.DynamicMapping.JSONata),
		})
		diagnostics.Append(diags...)
		data.DynamicMapping = dm
	}
}

func (r *connectorFlowConfigBindingResource) dynamicMappingFromData(ctx context.Context, data *ConnectorFlowConfigBindingResourceModel, diagnostics *diag.Diagnostics) *ConnectorFlowConfigBindingDynamicMappingAPI {
	if data.DynamicMapping.IsNull() || data.DynamicMapping.IsUnknown() {
		return nil
	}
	var dm ConnectorFlowConfigBindingDynamicMappingModel
	diagnostics.Append(data.DynamicMapping.As(ctx, &dm, basetypes.ObjectAsOptions{})...)
	if diagnostics.HasError() {
		return nil
	}
	return &ConnectorFlowConfigBindingDynamicMappingAPI{
		NamePrefix: dm.NamePrefix.ValueString(),
		JSONata:    dm.JSONata.ValueString(),
	}
}

func (r *connectorFlowConfigBindingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ConnectorFlowConfigBindingResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Look up the binding ID assigned by the server when the flow was deployed.
	binding := r.findBindingByConfigType(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	data.ID = types.StringValue(binding.ID)

	dm := r.dynamicMappingFromData(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	patch := ConnectorFlowConfigBindingPatchAPIModel{DynamicMapping: dm}
	var api ConnectorFlowConfigBindingAPIModel
	ok, _ := r.apiRequest(ctx, http.MethodPatch, r.instancePath(&data), &patch, &api, &resp.Diagnostics)
	if !ok {
		return
	}
	r.toData(&api, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *connectorFlowConfigBindingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ConnectorFlowConfigBindingResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var api ConnectorFlowConfigBindingAPIModel
	ok, status := r.apiRequest(ctx, http.MethodGet, r.instancePath(&data), nil, &api, &resp.Diagnostics, Allow404())
	if !ok {
		return
	}
	if status == 404 {
		resp.State.RemoveResource(ctx)
		return
	}
	r.toData(&api, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *connectorFlowConfigBindingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ConnectorFlowConfigBindingResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)
	if resp.Diagnostics.HasError() {
		return
	}
	dm := r.dynamicMappingFromData(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	patch := ConnectorFlowConfigBindingPatchAPIModel{DynamicMapping: dm}
	var api ConnectorFlowConfigBindingAPIModel
	ok, _ := r.apiRequest(ctx, http.MethodPatch, r.instancePath(&data), &patch, &api, &resp.Diagnostics)
	if !ok {
		return
	}
	r.toData(&api, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *connectorFlowConfigBindingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ConnectorFlowConfigBindingResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Clear the dynamic mapping by patching with null, restoring the static binding.
	patch := ConnectorFlowConfigBindingPatchAPIModel{DynamicMapping: nil}
	r.apiRequest(ctx, http.MethodPatch, r.instancePath(&data), &patch, nil, &resp.Diagnostics, Allow404())
}
