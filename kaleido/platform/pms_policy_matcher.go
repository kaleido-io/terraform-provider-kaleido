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
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type PMSPolicyMatcherResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Environment      types.String `tfsdk:"environment"`
	Service          types.String `tfsdk:"service"`
	Policy           types.String `tfsdk:"policy"`
	EnforcementPoint types.String `tfsdk:"enforcement_point"`
	MatchJSON        types.String `tfsdk:"match_json"`
	ParametersJSON   types.String `tfsdk:"parameters_json"`
	Evidence         types.List   `tfsdk:"evidence"`
}

type PMSMatcherEvidenceMappingAPIModel struct {
	Slot        string             `json:"slot,omitempty"`
	Payload     *JSONataMappingAPI `json:"payload,omitempty"`
	Attestation *JSONataMappingAPI `json:"attestation,omitempty"`
}

// JSONataMappingAPI is the {jsonata: "..."} wrapper the policy manager uses for
// payload, attestation and action payload template mappings.
type JSONataMappingAPI struct {
	JSONata string `json:"jsonata,omitempty"`
}

type PMSPolicyMatcherAPIModel struct {
	ID               string                              `json:"id,omitempty"`
	PolicyID         string                              `json:"policyId,omitempty"`
	EnforcementPoint string                              `json:"enforcementPoint,omitempty"`
	Match            map[string]interface{}              `json:"match,omitempty"`
	Parameters       map[string]interface{}              `json:"parameters,omitempty"`
	Evidence         []PMSMatcherEvidenceMappingAPIModel `json:"evidence,omitempty"`
	Created          *time.Time                          `json:"created,omitempty"`
	Updated          *time.Time                          `json:"updated,omitempty"`
}

// PMSPolicyMatcherPatchAPIModel is the PATCH body - enforcementPoint and evidence
// are immutable after create.
type PMSPolicyMatcherPatchAPIModel struct {
	Match      map[string]interface{} `json:"match,omitempty"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

var matcherEvidenceAttrTypes = map[string]attr.Type{
	"slot":                types.StringType,
	"payload_jsonata":     types.StringType,
	"attestation_jsonata": types.StringType,
}

func PMSPolicyMatcherResourceFactory() resource.Resource {
	return &pms_policy_matcherResource{}
}

type pms_policy_matcherResource struct {
	commonResource
}

func (r *pms_policy_matcherResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_pms_policy_matcher"
}

func (r *pms_policy_matcherResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a matcher on a Policy Manager policy. A matcher decides which objects at an enforcement point the policy applies to, and seeds evidence slots from the matched object.",
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				Description:   "The matcher ID assigned by the server",
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
				Description:   "Name or ID of the policy this matcher belongs to",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"enforcement_point": &schema.StringAttribute{
				Required:      true,
				Description:   "The enforcement point the matcher applies at. 'wfe-hook' matches against a workflow engine transaction. Immutable after create.",
				Validators:    []validator.String{stringvalidator.OneOf("wfe-hook")},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"match_json": &schema.StringAttribute{
				Optional:    true,
				Description: "A JSON query expression (use jsonencode) evaluated against the fields and 'label.<name>' labels of the object at the enforcement point",
			},
			"parameters_json": &schema.StringAttribute{
				Optional:    true,
				Description: "Values (use jsonencode) for the parameters declared by the policy definition",
			},
			"evidence": &schema.ListNestedAttribute{
				Optional:      true,
				Description:   "Mappings that seed evidence slots from the matched object when the matcher fires. Immutable after create.",
				PlanModifiers: []planmodifier.List{listplanmodifier.RequiresReplace()},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"slot": &schema.StringAttribute{
							Required:    true,
							Description: "The evidence slot name the mapped payload and attestation belong to",
						},
						"payload_jsonata": &schema.StringAttribute{
							Optional:    true,
							Description: "JSONata mapping from the transaction input to the evidence payload for the slot",
						},
						"attestation_jsonata": &schema.StringAttribute{
							Optional:    true,
							Description: "JSONata mapping from the transaction input to the evidence attestation for the slot",
						},
					},
				},
			},
		},
	}
}

func (r *pms_policy_matcherResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.commonResource.Configure(ctx, req, resp)
}

func (r *pms_policy_matcherResource) listPath(data *PMSPolicyMatcherResourceModel) string {
	return fmt.Sprintf("/endpoint/%s/%s/rest/api/v2/policies/%s/matchers",
		data.Environment.ValueString(), data.Service.ValueString(), url.PathEscape(data.Policy.ValueString()))
}

func (r *pms_policy_matcherResource) instancePath(data *PMSPolicyMatcherResourceModel) string {
	return fmt.Sprintf("%s/%s", r.listPath(data), data.ID.ValueString())
}

func (r *pms_policy_matcherResource) toAPI(data *PMSPolicyMatcherResourceModel, api *PMSPolicyMatcherAPIModel, diagnostics *diag.Diagnostics) {
	api.EnforcementPoint = data.EnforcementPoint.ValueString()
	api.Match = jsonObjectFromString(data.MatchJSON, "match_json", diagnostics)
	api.Parameters = jsonObjectFromString(data.ParametersJSON, "parameters_json", diagnostics)
	if diagnostics.HasError() {
		return
	}

	if data.Evidence.IsNull() || data.Evidence.IsUnknown() {
		return
	}
	var evidence []PMSMatcherEvidenceMappingAPIModel
	for _, item := range data.Evidence.Elements() {
		obj, ok := item.(types.Object)
		if !ok {
			continue
		}
		attrs := obj.Attributes()
		mapping := PMSMatcherEvidenceMappingAPIModel{Slot: stringAttr(attrs, "slot")}
		if jsonata := stringAttr(attrs, "payload_jsonata"); jsonata != "" {
			mapping.Payload = &JSONataMappingAPI{JSONata: jsonata}
		}
		if jsonata := stringAttr(attrs, "attestation_jsonata"); jsonata != "" {
			mapping.Attestation = &JSONataMappingAPI{JSONata: jsonata}
		}
		evidence = append(evidence, mapping)
	}
	api.Evidence = evidence
}

// jsonObjectFromString unmarshals an optional JSON-string attribute into a map.
func jsonObjectFromString(v types.String, attrName string, diagnostics *diag.Diagnostics) map[string]interface{} {
	if v.IsNull() || v.IsUnknown() || v.ValueString() == "" {
		return nil
	}
	obj := make(map[string]interface{})
	if err := json.Unmarshal([]byte(v.ValueString()), &obj); err != nil {
		diagnostics.AddError("Invalid JSON", fmt.Sprintf("Failed to parse %s: %v.  %s", attrName, err, v.ValueString()))
		return nil
	}
	return obj
}

// jsonStringFromObject renders a map back to a JSON-string attribute value.
func jsonStringFromObject(obj map[string]interface{}, attrName string, diagnostics *diag.Diagnostics) types.String {
	if len(obj) == 0 {
		return types.StringNull()
	}
	b, err := json.Marshal(obj)
	if err != nil {
		diagnostics.AddError("Invalid JSON", fmt.Sprintf("Failed to serialize %s: %v", attrName, err))
		return types.StringNull()
	}
	return types.StringValue(string(b))
}

func (r *pms_policy_matcherResource) toData(api *PMSPolicyMatcherAPIModel, data *PMSPolicyMatcherResourceModel, diagnostics *diag.Diagnostics) {
	data.ID = types.StringValue(api.ID)
	if api.EnforcementPoint != "" {
		data.EnforcementPoint = types.StringValue(api.EnforcementPoint)
	}
	data.MatchJSON = jsonStringFromObject(api.Match, "match_json", diagnostics)
	data.ParametersJSON = jsonStringFromObject(api.Parameters, "parameters_json", diagnostics)

	if len(api.Evidence) == 0 {
		data.Evidence = types.ListNull(types.ObjectType{AttrTypes: matcherEvidenceAttrTypes})
		return
	}
	evidence := make([]attr.Value, len(api.Evidence))
	for i, mapping := range api.Evidence {
		obj, diags := types.ObjectValue(matcherEvidenceAttrTypes, map[string]attr.Value{
			"slot":                types.StringValue(mapping.Slot),
			"payload_jsonata":     jsonataAttr(mapping.Payload),
			"attestation_jsonata": jsonataAttr(mapping.Attestation),
		})
		diagnostics.Append(diags...)
		evidence[i] = obj
	}
	data.Evidence = types.ListValueMust(types.ObjectType{AttrTypes: matcherEvidenceAttrTypes}, evidence)
}

// jsonataAttr unwraps an optional {jsonata: "..."} mapping to a terraform string.
func jsonataAttr(mapping *JSONataMappingAPI) types.String {
	if mapping == nil || mapping.JSONata == "" {
		return types.StringNull()
	}
	return types.StringValue(mapping.JSONata)
}

func (r *pms_policy_matcherResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PMSPolicyMatcherResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var api PMSPolicyMatcherAPIModel
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

func (r *pms_policy_matcherResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PMSPolicyMatcherResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var api PMSPolicyMatcherAPIModel
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

func (r *pms_policy_matcherResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data PMSPolicyMatcherResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	patch := PMSPolicyMatcherPatchAPIModel{
		Match:      jsonObjectFromString(data.MatchJSON, "match_json", &resp.Diagnostics),
		Parameters: jsonObjectFromString(data.ParametersJSON, "parameters_json", &resp.Diagnostics),
	}
	if resp.Diagnostics.HasError() {
		return
	}

	var api PMSPolicyMatcherAPIModel
	ok, _ := r.apiRequest(ctx, http.MethodPatch, r.instancePath(&data), &patch, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	r.toData(&api, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *pms_policy_matcherResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PMSPolicyMatcherResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.instancePath(&data), nil, nil, &resp.Diagnostics, Allow404())
}
