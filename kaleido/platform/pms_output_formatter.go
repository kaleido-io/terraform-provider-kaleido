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
	"net/url"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type PMSOutputFormatterResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Environment types.String `tfsdk:"environment"`
	Service     types.String `tfsdk:"service"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Type        types.String `tfsdk:"type"`
	MappingRego types.String `tfsdk:"mapping_rego"`
	Parameters  types.List   `tfsdk:"parameter"`
	Created     types.String `tfsdk:"created"`
	Updated     types.String `tfsdk:"updated"`
}

// PMSOutputFormatterMappingAPIModel is the Rego expression that builds the output value
// from the formatter's parameters.
type PMSOutputFormatterMappingAPIModel struct {
	Rego string `json:"rego,omitempty"`
}

// PMSOutputFormatterAPIModel is an output formatter on the wire: the envelope type
// consumers key on, the parameters a policy must feed it, and the Rego mapping that turns
// those parameters into the output value.
type PMSOutputFormatterAPIModel struct {
	ID          string                             `json:"id,omitempty"`
	Name        string                             `json:"name,omitempty"`
	Description string                             `json:"description,omitempty"`
	Type        string                             `json:"type,omitempty"`
	Parameters  []PMSParameterAPIModel             `json:"parameters,omitempty"`
	Mapping     *PMSOutputFormatterMappingAPIModel `json:"mapping,omitempty"`
	Created     *time.Time                         `json:"created,omitempty"`
	Updated     *time.Time                         `json:"updated,omitempty"`
}

func PMSOutputFormatterResourceFactory() resource.Resource {
	return &pms_outputFormatterResource{}
}

type pms_outputFormatterResource struct {
	commonResource
}

func (r *pms_outputFormatterResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_pms_output_formatter"
}

func (r *pms_outputFormatterResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Policy Manager output formatter: a reusable, policy-independent description of a policy's output payload. It declares the envelope type consumers discriminate on, the parameters a policy must feed it, and the Rego mapping that turns those parameters into the output value. A policy uses one through a kaleido_platform_pms_policy_output_formatter_binding, which its definition's 'output.formatter' names; 'output.values' then supplies a Rego expression for each parameter. The formatter is resolved when a policy version is created and pinned to it, so a later change here does not alter existing versions.",
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				Description:   "The output formatter ID assigned by the server. This is the value to bind to: the output_formatter_id of a kaleido_platform_pms_policy_output_formatter_binding.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"environment": &schema.StringAttribute{
				Required:      true,
				Description:   "Environment ID",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"service": &schema.StringAttribute{
				Required:      true,
				Description:   "Policy Manager service ID",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": &schema.StringAttribute{
				Required:      true,
				Description:   "Unique name of the output formatter. Immutable after create.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"description": &schema.StringAttribute{
				Optional:    true,
				Description: "Description of the output formatter",
			},
			"type": &schema.StringAttribute{
				Required:      true,
				Description:   "Stable identifier for the shape of the output payload, emitted as the type of the output envelope for consumers to discriminate on, e.g. 'kaleido.policy.evm.v1'. Immutable after create.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"mapping_rego": &schema.StringAttribute{
				Required:    true,
				Description: "A single Rego expression over parameters.<name> that evaluates to the output value. It may reference only the parameters declared here.",
			},
			"parameter": pmsParameterNestedSchema("The parameters the mapping reads as parameters.<name>. A policy bound to the formatter supplies a Rego expression for each in its 'output.values'; a parameter with a default may be left out."),
			"created": &schema.StringAttribute{
				Computed:    true,
				Description: "Creation timestamp",
			},
			"updated": &schema.StringAttribute{
				Computed:    true,
				Description: "Last update timestamp",
			},
		},
	}
}

func (r *pms_outputFormatterResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.commonResource.Configure(ctx, req, resp)
}

func (r *pms_outputFormatterResource) listPath(data *PMSOutputFormatterResourceModel) string {
	return fmt.Sprintf("/endpoint/%s/%s/rest/api/v2/output-formatters", data.Environment.ValueString(), data.Service.ValueString())
}

func (r *pms_outputFormatterResource) instancePath(data *PMSOutputFormatterResourceModel) string {
	return fmt.Sprintf("%s/%s", r.listPath(data), url.PathEscape(data.ID.ValueString()))
}

func (r *pms_outputFormatterResource) toAPI(data *PMSOutputFormatterResourceModel, api *PMSOutputFormatterAPIModel, diagnostics *diag.Diagnostics) {
	api.Name = data.Name.ValueString()
	api.Description = data.Description.ValueString()
	api.Type = data.Type.ValueString()
	api.Mapping = &PMSOutputFormatterMappingAPIModel{Rego: data.MappingRego.ValueString()}
	api.Parameters = pmsParametersToAPI(data.Parameters, diagnostics)
}

func (r *pms_outputFormatterResource) toData(api *PMSOutputFormatterAPIModel, data *PMSOutputFormatterResourceModel, diagnostics *diag.Diagnostics) {
	data.ID = types.StringValue(api.ID)
	if api.Name != "" {
		data.Name = types.StringValue(api.Name)
	}
	data.Description = optionalString(api.Description)
	if api.Type != "" {
		data.Type = types.StringValue(api.Type)
	}
	if api.Mapping != nil {
		data.MappingRego = types.StringValue(api.Mapping.Rego)
	}
	data.Parameters = pmsParametersToData(api.Parameters, data.Parameters, diagnostics)
	data.Created = timeAttr(api.Created)
	data.Updated = timeAttr(api.Updated)
}

func (r *pms_outputFormatterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PMSOutputFormatterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var api PMSOutputFormatterAPIModel
	r.toAPI(&data, &api, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	ok, _ := r.apiRequest(ctx, http.MethodPost, r.listPath(&data), &api, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	r.toData(&api, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *pms_outputFormatterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PMSOutputFormatterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var api PMSOutputFormatterAPIModel
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

func (r *pms_outputFormatterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data PMSOutputFormatterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// A PATCH cannot clear a field, so the whole formatter is replaced with a PUT: an
	// attribute removed from the configuration is then removed from the server too.
	var desired PMSOutputFormatterAPIModel
	r.toAPI(&data, &desired, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	var api PMSOutputFormatterAPIModel
	ok, _ := r.apiRequest(ctx, http.MethodPut, r.instancePath(&data), &desired, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	r.toData(&api, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *pms_outputFormatterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PMSOutputFormatterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.instancePath(&data), nil, nil, &resp.Diagnostics, Allow404())
}
