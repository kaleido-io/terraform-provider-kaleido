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

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type PMSOutputFormatterBindingResourceModel struct {
	ID                    types.String `tfsdk:"id"`
	Environment           types.String `tfsdk:"environment"`
	Service               types.String `tfsdk:"service"`
	Policy                types.String `tfsdk:"policy"`
	PolicyOutputFormatter types.String `tfsdk:"policy_output_formatter"`
	OutputFormatterID     types.String `tfsdk:"output_formatter_id"`
}

// PMSOutputFormatterBindingTargetAPIModel is what a policy's output formatter binding
// resolves to.
type PMSOutputFormatterBindingTargetAPIModel struct {
	OutputFormatterID string `json:"outputFormatterId,omitempty"`
}

type PMSOutputFormatterBindingAPIModel struct {
	ID                    string     `json:"id,omitempty"`
	PolicyID              string     `json:"policyId,omitempty"`
	PolicyOutputFormatter string     `json:"policyOutputFormatter,omitempty"`
	Created               *time.Time `json:"created,omitempty"`
	Updated               *time.Time `json:"updated,omitempty"`
	PMSOutputFormatterBindingTargetAPIModel
}

func PMSPolicyOutputFormatterBindingResourceFactory() resource.Resource {
	return &pms_outputFormatterBindingResource{}
}

type pms_outputFormatterBindingResource struct {
	commonResource
}

func (r *pms_outputFormatterBindingResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_pms_policy_output_formatter_binding"
}

// outputFormatterBindingTargetSchema is the set of attributes describing what a binding
// resolves to, shared by the standalone binding resource and the inline blocks on
// kaleido_platform_pms_policy.
func outputFormatterBindingTargetSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"output_formatter_id": &schema.StringAttribute{
			Required:    true,
			Description: "ID of the kaleido_platform_pms_output_formatter the policy emits its output through",
		},
	}
}

func (r *pms_outputFormatterBindingResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := map[string]schema.Attribute{
		"id": &schema.StringAttribute{
			Computed:      true,
			Description:   "The binding ID assigned by the server",
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
		"policy": &schema.StringAttribute{
			Required:      true,
			Description:   "Name or ID of the policy this binding belongs to",
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		"policy_output_formatter": &schema.StringAttribute{
			Required:      true,
			Description:   "The name the policy uses for this binding: the 'output.formatter' field of the policy definition. Immutable after create.",
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
	}
	for name, attribute := range outputFormatterBindingTargetSchema() {
		attributes[name] = attribute
	}
	resp.Schema = schema.Schema{
		Description: "Manages an output formatter binding on a Policy Manager policy. A binding gives the policy definition's 'output.formatter' a name that resolves to a kaleido_platform_pms_output_formatter, so the formatter can be swapped without editing the definition. An output formatter cannot be deleted while a binding refers to it.",
		Attributes:  attributes,
	}
}

func (r *pms_outputFormatterBindingResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.commonResource.Configure(ctx, req, resp)
}

func (r *pms_outputFormatterBindingResource) listPath(data *PMSOutputFormatterBindingResourceModel) string {
	return fmt.Sprintf("/endpoint/%s/%s/rest/api/v2/policies/%s/output-formatter-bindings",
		data.Environment.ValueString(), data.Service.ValueString(), url.PathEscape(data.Policy.ValueString()))
}

func (r *pms_outputFormatterBindingResource) instancePath(data *PMSOutputFormatterBindingResourceModel) string {
	return fmt.Sprintf("%s/%s", r.listPath(data), data.ID.ValueString())
}

// outputFormatterBindingTargetToAPI builds the wire target from the target attributes,
// whether they sit on the standalone binding resource or inline on the policy.
func outputFormatterBindingTargetToAPI(attrs map[string]attr.Value) PMSOutputFormatterBindingTargetAPIModel {
	return PMSOutputFormatterBindingTargetAPIModel{
		OutputFormatterID: stringAttr(attrs, "output_formatter_id"),
	}
}

// outputFormatterBindingTargetToData renders the wire target as terraform attribute
// values, keyed as the schema names them.
func outputFormatterBindingTargetToData(target *PMSOutputFormatterBindingTargetAPIModel) map[string]attr.Value {
	return map[string]attr.Value{
		"output_formatter_id": optionalString(target.OutputFormatterID),
	}
}

func (r *pms_outputFormatterBindingResource) toAPI(data *PMSOutputFormatterBindingResourceModel, api *PMSOutputFormatterBindingAPIModel) {
	api.PolicyOutputFormatter = data.PolicyOutputFormatter.ValueString()
	api.PMSOutputFormatterBindingTargetAPIModel = outputFormatterBindingTargetToAPI(map[string]attr.Value{
		"output_formatter_id": data.OutputFormatterID,
	})
}

func (r *pms_outputFormatterBindingResource) toData(api *PMSOutputFormatterBindingAPIModel, data *PMSOutputFormatterBindingResourceModel) {
	data.ID = types.StringValue(api.ID)
	if api.PolicyOutputFormatter != "" {
		data.PolicyOutputFormatter = types.StringValue(api.PolicyOutputFormatter)
	}
	values := outputFormatterBindingTargetToData(&api.PMSOutputFormatterBindingTargetAPIModel)
	data.OutputFormatterID = values["output_formatter_id"].(types.String)
}

func (r *pms_outputFormatterBindingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PMSOutputFormatterBindingResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var api PMSOutputFormatterBindingAPIModel
	r.toAPI(&data, &api)

	ok, _ := r.apiRequest(ctx, http.MethodPost, r.listPath(&data), &api, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	r.toData(&api, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *pms_outputFormatterBindingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PMSOutputFormatterBindingResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var api PMSOutputFormatterBindingAPIModel
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

func (r *pms_outputFormatterBindingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data PMSOutputFormatterBindingResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	patch := outputFormatterBindingTargetToAPI(map[string]attr.Value{
		"output_formatter_id": data.OutputFormatterID,
	})

	var api PMSOutputFormatterBindingAPIModel
	ok, _ := r.apiRequest(ctx, http.MethodPatch, r.instancePath(&data), &patch, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	r.toData(&api, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *pms_outputFormatterBindingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PMSOutputFormatterBindingResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.instancePath(&data), nil, nil, &resp.Diagnostics, Allow404())
}
