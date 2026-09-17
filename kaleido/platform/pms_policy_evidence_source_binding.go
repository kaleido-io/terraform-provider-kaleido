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
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type PMSEvidenceSourceBindingResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	Environment          types.String `tfsdk:"environment"`
	Service              types.String `tfsdk:"service"`
	Policy               types.String `tfsdk:"policy"`
	PolicyEvidenceSource types.String `tfsdk:"policy_evidence_source"`
	EvidenceSourceID     types.String `tfsdk:"evidence_source_id"`
	Attesters            types.String `tfsdk:"attesters"`
	RunAs                types.String `tfsdk:"run_as"`
	PayloadJSONata       types.String `tfsdk:"payload_jsonata"`
	AttestationJSONata   types.String `tfsdk:"attestation_jsonata"`
}

// PMSEvidenceSourceBindingTargetAPIModel is what a policy evidence slot is bound to: the
// evidence source that gathers it (none for a sourceless slot) plus the per-policy inputs
// that source needs.
type PMSEvidenceSourceBindingTargetAPIModel struct {
	EvidenceSourceID   string             `json:"evidenceSourceId,omitempty"`
	Attesters          string             `json:"attesters,omitempty"`
	RunAs              string             `json:"runAs,omitempty"`
	PayloadMapping     *JSONataMappingAPI `json:"payloadMapping,omitempty"`
	AttestationMapping *JSONataMappingAPI `json:"attestationMapping,omitempty"`
}

type PMSEvidenceSourceBindingAPIModel struct {
	ID                   string     `json:"id,omitempty"`
	PolicyID             string     `json:"policyId,omitempty"`
	PolicyEvidenceSource string     `json:"policyEvidenceSource,omitempty"`
	Created              *time.Time `json:"created,omitempty"`
	Updated              *time.Time `json:"updated,omitempty"`
	PMSEvidenceSourceBindingTargetAPIModel
}

func PMSPolicyEvidenceSourceBindingResourceFactory() resource.Resource {
	return &pms_evidenceSourceBindingResource{}
}

type pms_evidenceSourceBindingResource struct {
	commonResource
}

func (r *pms_evidenceSourceBindingResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_pms_policy_evidence_source_binding"
}

// evidenceSourceBindingTargetSchema is the set of attributes describing what a slot is
// bound to, shared by the standalone binding resource and the inline blocks on
// kaleido_platform_pms_policy.
func evidenceSourceBindingTargetSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"evidence_source_id": &schema.StringAttribute{
			Optional:    true,
			Description: "ID of the kaleido_platform_pms_evidence_source that gathers this slot. Omit for a slot with no source, whose evidence is seeded by a matcher, attached manually, or supplied late-bound by whatever builds the transaction.",
		},
		"attesters": &schema.StringAttribute{
			Optional:    true,
			Description: "The attester label of one of the policy's identity list bindings; its identity list version supplies the identities the source addresses (the approvers of an approval source). Required when bound to an approval source.",
		},
		"run_as": &schema.StringAttribute{
			Optional:    true,
			Description: "Application ID the source acts as when it calls out. Required when bound to a serviceRequest or workflow source.",
		},
		"payload_jsonata": &schema.StringAttribute{
			Optional:    true,
			Description: "JSONata selecting the evidence payload out of a message POSTed to the slot's attach endpoint",
		},
		"attestation_jsonata": &schema.StringAttribute{
			Optional:    true,
			Description: "JSONata selecting the attestation out of a message POSTed to the slot's attach endpoint",
		},
	}
}

func (r *pms_evidenceSourceBindingResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
		"policy_evidence_source": &schema.StringAttribute{
			Required:      true,
			Description:   "The name the policy uses for this binding: the 'source' field of an evidence slot in the policy definition. Immutable after create.",
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
	}
	for name, attribute := range evidenceSourceBindingTargetSchema() {
		attributes[name] = attribute
	}
	resp.Schema = schema.Schema{
		Description: "Manages an evidence source binding on a Policy Manager policy. A binding ties one of the policy's evidence slots to a kaleido_platform_pms_evidence_source, plus the inputs that source needs from this policy: who it acts as (run_as) and whose attestations it seeks (attesters). A binding with no evidence_source_id is a slot with no source, whose evidence is seeded by a matcher, attached manually, or supplied late-bound; it may still carry the ingress mappings.",
		Attributes:  attributes,
	}
}

func (r *pms_evidenceSourceBindingResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.commonResource.Configure(ctx, req, resp)
}

func (r *pms_evidenceSourceBindingResource) listPath(data *PMSEvidenceSourceBindingResourceModel) string {
	return fmt.Sprintf("/endpoint/%s/%s/rest/api/v2/policies/%s/evidence-source-bindings",
		data.Environment.ValueString(), data.Service.ValueString(), url.PathEscape(data.Policy.ValueString()))
}

func (r *pms_evidenceSourceBindingResource) instancePath(data *PMSEvidenceSourceBindingResourceModel) string {
	return fmt.Sprintf("%s/%s", r.listPath(data), data.ID.ValueString())
}

// evidenceSourceBindingTargetToAPI builds the wire target from the target attributes,
// whether they sit on the standalone binding resource or inline on the policy.
func evidenceSourceBindingTargetToAPI(attrs map[string]attr.Value) PMSEvidenceSourceBindingTargetAPIModel {
	target := PMSEvidenceSourceBindingTargetAPIModel{
		EvidenceSourceID: stringAttr(attrs, "evidence_source_id"),
		Attesters:        stringAttr(attrs, "attesters"),
		RunAs:            stringAttr(attrs, "run_as"),
	}
	if jsonata := stringAttr(attrs, "payload_jsonata"); jsonata != "" {
		target.PayloadMapping = &JSONataMappingAPI{JSONata: jsonata}
	}
	if jsonata := stringAttr(attrs, "attestation_jsonata"); jsonata != "" {
		target.AttestationMapping = &JSONataMappingAPI{JSONata: jsonata}
	}
	return target
}

// evidenceSourceBindingTargetToData renders the wire target as terraform attribute
// values, keyed as the schema names them.
func evidenceSourceBindingTargetToData(target *PMSEvidenceSourceBindingTargetAPIModel) map[string]attr.Value {
	return map[string]attr.Value{
		"evidence_source_id":  optionalString(target.EvidenceSourceID),
		"attesters":           optionalString(target.Attesters),
		"run_as":              optionalString(target.RunAs),
		"payload_jsonata":     jsonataAttr(target.PayloadMapping),
		"attestation_jsonata": jsonataAttr(target.AttestationMapping),
	}
}

func (r *pms_evidenceSourceBindingResource) targetAttrs(data *PMSEvidenceSourceBindingResourceModel) map[string]attr.Value {
	return map[string]attr.Value{
		"evidence_source_id":  data.EvidenceSourceID,
		"attesters":           data.Attesters,
		"run_as":              data.RunAs,
		"payload_jsonata":     data.PayloadJSONata,
		"attestation_jsonata": data.AttestationJSONata,
	}
}

func (r *pms_evidenceSourceBindingResource) toAPI(data *PMSEvidenceSourceBindingResourceModel, api *PMSEvidenceSourceBindingAPIModel) {
	api.PolicyEvidenceSource = data.PolicyEvidenceSource.ValueString()
	api.PMSEvidenceSourceBindingTargetAPIModel = evidenceSourceBindingTargetToAPI(r.targetAttrs(data))
}

func (r *pms_evidenceSourceBindingResource) toData(api *PMSEvidenceSourceBindingAPIModel, data *PMSEvidenceSourceBindingResourceModel, _ *diag.Diagnostics) {
	data.ID = types.StringValue(api.ID)
	if api.PolicyEvidenceSource != "" {
		data.PolicyEvidenceSource = types.StringValue(api.PolicyEvidenceSource)
	}
	values := evidenceSourceBindingTargetToData(&api.PMSEvidenceSourceBindingTargetAPIModel)
	data.EvidenceSourceID = values["evidence_source_id"].(types.String)
	data.Attesters = values["attesters"].(types.String)
	data.RunAs = values["run_as"].(types.String)
	data.PayloadJSONata = values["payload_jsonata"].(types.String)
	data.AttestationJSONata = values["attestation_jsonata"].(types.String)
}

func (r *pms_evidenceSourceBindingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PMSEvidenceSourceBindingResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var api PMSEvidenceSourceBindingAPIModel
	r.toAPI(&data, &api)

	ok, _ := r.apiRequest(ctx, http.MethodPost, r.listPath(&data), &api, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	r.toData(&api, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *pms_evidenceSourceBindingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PMSEvidenceSourceBindingResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var api PMSEvidenceSourceBindingAPIModel
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

func (r *pms_evidenceSourceBindingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data PMSEvidenceSourceBindingResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// PATCH replaces each field it carries and cannot clear one, so a field removed from
	// the configuration is left as the server has it.
	patch := evidenceSourceBindingTargetToAPI(r.targetAttrs(&data))

	var api PMSEvidenceSourceBindingAPIModel
	ok, _ := r.apiRequest(ctx, http.MethodPatch, r.instancePath(&data), &patch, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	r.toData(&api, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *pms_evidenceSourceBindingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PMSEvidenceSourceBindingResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.instancePath(&data), nil, nil, &resp.Diagnostics, Allow404())
}
