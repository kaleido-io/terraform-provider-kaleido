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

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	evidenceSourceBindingTypeApproval   = "approval"
	evidenceSourceBindingTypeAttachment = "attachment"
)

type PMSEvidenceSourceBindingResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Environment types.String `tfsdk:"environment"`
	Service     types.String `tfsdk:"service"`
	Policy      types.String `tfsdk:"policy"`
	Name        types.String `tfsdk:"name"`
	Type        types.String `tfsdk:"type"`
	Approval    types.Object `tfsdk:"approval"`
	Attachment  types.Object `tfsdk:"attachment"`
}

// PMSActionAPIModel configures the action a reviewer selects to approve or reject.
type PMSActionAPIModel struct {
	PayloadType     string             `json:"payloadType,omitempty"`
	PayloadTemplate *JSONataMappingAPI `json:"payloadTemplate,omitempty"`
}

type PMSIdentityListVersionReferenceAPIModel struct {
	ID      string `json:"id,omitempty"`
	Version string `json:"version,omitempty"`
	Hash    string `json:"hash,omitempty"`
}

type PMSApprovalEvidenceSourceBindingAPIModel struct {
	Approval            *PMSActionAPIModel                       `json:"approval,omitempty"`
	Rejection           *PMSActionAPIModel                       `json:"rejection,omitempty"`
	IdentityListVersion *PMSIdentityListVersionReferenceAPIModel `json:"identityListVersion,omitempty"`
}

type PMSAttachmentEvidenceSourceBindingAPIModel struct {
	PayloadMapping     *JSONataMappingAPI `json:"payloadMapping,omitempty"`
	AttestationMapping *JSONataMappingAPI `json:"attestationMapping,omitempty"`
}

type PMSEvidenceSourceBindingAPIModel struct {
	ID         string                                      `json:"id,omitempty"`
	PolicyID   string                                      `json:"policyId,omitempty"`
	Name       string                                      `json:"name,omitempty"`
	Type       string                                      `json:"type,omitempty"`
	Approval   *PMSApprovalEvidenceSourceBindingAPIModel   `json:"approval,omitempty"`
	Attachment *PMSAttachmentEvidenceSourceBindingAPIModel `json:"attachment,omitempty"`
	Created    *time.Time                                  `json:"created,omitempty"`
	Updated    *time.Time                                  `json:"updated,omitempty"`
}

// PMSEvidenceSourceBindingPatchAPIModel is the PATCH body - name and type are
// immutable after create.
type PMSEvidenceSourceBindingPatchAPIModel struct {
	Approval   *PMSApprovalEvidenceSourceBindingAPIModel   `json:"approval,omitempty"`
	Attachment *PMSAttachmentEvidenceSourceBindingAPIModel `json:"attachment,omitempty"`
}

var esbActionAttrTypes = map[string]attr.Type{
	"payload_type":             types.StringType,
	"payload_template_jsonata": types.StringType,
}

var esbIdentityListVersionAttrTypes = map[string]attr.Type{
	"id":      types.StringType,
	"version": types.StringType,
	"hash":    types.StringType,
}

var esbApprovalAttrTypes = map[string]attr.Type{
	"approval":              types.ObjectType{AttrTypes: esbActionAttrTypes},
	"rejection":             types.ObjectType{AttrTypes: esbActionAttrTypes},
	"identity_list_version": types.ObjectType{AttrTypes: esbIdentityListVersionAttrTypes},
}

var esbAttachmentAttrTypes = map[string]attr.Type{
	"payload_jsonata":     types.StringType,
	"attestation_jsonata": types.StringType,
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

func actionSchema(description string) *schema.SingleNestedAttribute {
	return &schema.SingleNestedAttribute{
		Optional:    true,
		Description: description,
		Attributes: map[string]schema.Attribute{
			"payload_type": &schema.StringAttribute{
				Optional:    true,
				Description: "The type of the action payload, e.g. 'TypedDataV4'",
			},
			"payload_template_jsonata": &schema.StringAttribute{
				Optional:    true,
				Description: "JSONata template used to build the action payload from the binding context",
			},
		},
	}
}

// evidenceSourceBindingApprovalSchema is the 'approval' configuration block, shared by
// the standalone binding resource and the inline blocks on kaleido_platform_pms_policy.
func evidenceSourceBindingApprovalSchema() *schema.SingleNestedAttribute {
	return &schema.SingleNestedAttribute{
		Optional:    true,
		Description: "Configuration for a binding of type 'approval', where evidence is gathered by asking the members of an identity list version to approve or reject",
		Attributes: map[string]schema.Attribute{
			"approval":  actionSchema("The action a reviewer selects to approve the request"),
			"rejection": actionSchema("The action a reviewer selects to reject the request"),
			"identity_list_version": &schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Reference to the identity list version whose members are notified and assigned tasks to approve or reject the request",
				Attributes: map[string]schema.Attribute{
					"id": &schema.StringAttribute{
						Optional:    true,
						Description: "ID of the identity list version - use the applied_version_id attribute of a kaleido_platform_pms_identity_list",
					},
					"version": &schema.StringAttribute{
						Optional:    true,
						Description: "The name of the identity list version",
					},
					"hash": &schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Description: "The hash of the identity list version, for irrefutable post hoc comparison",
					},
				},
			},
		},
	}
}

// evidenceSourceBindingAttachmentSchema is the 'attachment' configuration block, shared
// by the standalone binding resource and the inline blocks on kaleido_platform_pms_policy.
func evidenceSourceBindingAttachmentSchema() *schema.SingleNestedAttribute {
	return &schema.SingleNestedAttribute{
		Optional:    true,
		Description: "Configuration for a binding of type 'attachment', where evidence is mapped directly out of the transaction input",
		Attributes: map[string]schema.Attribute{
			"payload_jsonata": &schema.StringAttribute{
				Optional:    true,
				Description: "JSONata mapping from the transaction input to the evidence payload for the slot",
			},
			"attestation_jsonata": &schema.StringAttribute{
				Optional:    true,
				Description: "JSONata mapping from the transaction input to the evidence attestation for the slot",
			},
		},
	}
}

func (r *pms_evidenceSourceBindingResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an evidence source binding on a Policy Manager policy. The binding tells the policy where the evidence for a slot comes from. Types 'approval' and 'attachment' are supported; 'workflow' and 'serviceRequest' are not yet implemented.",
		Attributes: map[string]schema.Attribute{
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
			"name": &schema.StringAttribute{
				Required:      true,
				Description:   "The name of the evidence source binding within the policy. Referenced by the 'source' field of an evidence slot in the policy definition. Immutable after create.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"type": &schema.StringAttribute{
				Required:      true,
				Description:   "The type of evidence source binding: 'approval' or 'attachment'. Immutable after create.",
				Validators:    []validator.String{stringvalidator.OneOf(evidenceSourceBindingTypeApproval, evidenceSourceBindingTypeAttachment)},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"approval":   evidenceSourceBindingApprovalSchema(),
			"attachment": evidenceSourceBindingAttachmentSchema(),
		},
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

// validateEvidenceSourceBindingBlocks checks that the configuration block matching
// type is the one - and the only one - that is set.
func validateEvidenceSourceBindingBlocks(bindingType string, approval, attachment types.Object, diagnostics *diag.Diagnostics) {
	hasApproval := !approval.IsNull() && !approval.IsUnknown()
	hasAttachment := !attachment.IsNull() && !attachment.IsUnknown()

	if hasApproval && hasAttachment {
		diagnostics.AddError("Invalid configuration", "approval and attachment are mutually exclusive; set only the block matching type")
		return
	}
	switch bindingType {
	case evidenceSourceBindingTypeApproval:
		if !hasApproval {
			diagnostics.AddError("Invalid configuration", "the approval block must be set when type is \"approval\"")
		}
	case evidenceSourceBindingTypeAttachment:
		if !hasAttachment {
			diagnostics.AddError("Invalid configuration", "the attachment block must be set when type is \"attachment\"")
		}
	}
}

// esbApprovalToAPI builds the 'approval' half of an evidence source binding from its
// configuration block, whether that block is on the standalone binding resource or
// inline on the policy.
func esbApprovalToAPI(approval types.Object) *PMSApprovalEvidenceSourceBindingAPIModel {
	if approval.IsNull() || approval.IsUnknown() {
		return nil
	}
	attrs := approval.Attributes()
	result := &PMSApprovalEvidenceSourceBindingAPIModel{
		Approval:  actionToAPI(attrs["approval"]),
		Rejection: actionToAPI(attrs["rejection"]),
	}
	if obj, ok := objectAttr(attrs, "identity_list_version"); ok {
		ilvAttrs := obj.Attributes()
		result.IdentityListVersion = &PMSIdentityListVersionReferenceAPIModel{
			ID:      stringAttr(ilvAttrs, "id"),
			Version: stringAttr(ilvAttrs, "version"),
			Hash:    stringAttr(ilvAttrs, "hash"),
		}
	}
	return result
}

// esbAttachmentToAPI builds the 'attachment' half of an evidence source binding from
// its configuration block.
func esbAttachmentToAPI(attachment types.Object) *PMSAttachmentEvidenceSourceBindingAPIModel {
	if attachment.IsNull() || attachment.IsUnknown() {
		return nil
	}
	attrs := attachment.Attributes()
	result := &PMSAttachmentEvidenceSourceBindingAPIModel{}
	if jsonata := stringAttr(attrs, "payload_jsonata"); jsonata != "" {
		result.PayloadMapping = &JSONataMappingAPI{JSONata: jsonata}
	}
	if jsonata := stringAttr(attrs, "attestation_jsonata"); jsonata != "" {
		result.AttestationMapping = &JSONataMappingAPI{JSONata: jsonata}
	}
	return result
}

// objectAttr reads an optional nested object out of a terraform object's attribute map.
func objectAttr(attrs map[string]attr.Value, name string) (types.Object, bool) {
	val, ok := attrs[name]
	if !ok || val.IsNull() || val.IsUnknown() {
		return types.Object{}, false
	}
	obj, ok := val.(types.Object)
	return obj, ok
}

func actionToAPI(val attr.Value) *PMSActionAPIModel {
	if val == nil || val.IsNull() || val.IsUnknown() {
		return nil
	}
	obj, ok := val.(types.Object)
	if !ok {
		return nil
	}
	attrs := obj.Attributes()
	action := &PMSActionAPIModel{PayloadType: stringAttr(attrs, "payload_type")}
	if jsonata := stringAttr(attrs, "payload_template_jsonata"); jsonata != "" {
		action.PayloadTemplate = &JSONataMappingAPI{JSONata: jsonata}
	}
	return action
}

func actionToData(action *PMSActionAPIModel, diagnostics *diag.Diagnostics) types.Object {
	if action == nil || (action.PayloadType == "" && action.PayloadTemplate == nil) {
		return types.ObjectNull(esbActionAttrTypes)
	}
	obj, diags := types.ObjectValue(esbActionAttrTypes, map[string]attr.Value{
		"payload_type":             optionalString(action.PayloadType),
		"payload_template_jsonata": jsonataAttr(action.PayloadTemplate),
	})
	diagnostics.Append(diags...)
	return obj
}

func (r *pms_evidenceSourceBindingResource) toAPI(data *PMSEvidenceSourceBindingResourceModel, api *PMSEvidenceSourceBindingAPIModel, diagnostics *diag.Diagnostics) {
	evidenceSourceBindingToAPI(data.Name.ValueString(), data.Type.ValueString(), data.Approval, data.Attachment, api, diagnostics)
}

// evidenceSourceBindingToAPI populates the wire model from a binding's configuration,
// validating that the block supplied matches the declared type.
func evidenceSourceBindingToAPI(name, bindingType string, approval, attachment types.Object, api *PMSEvidenceSourceBindingAPIModel, diagnostics *diag.Diagnostics) {
	validateEvidenceSourceBindingBlocks(bindingType, approval, attachment, diagnostics)
	if diagnostics.HasError() {
		return
	}
	api.Name = name
	api.Type = bindingType
	api.Approval = esbApprovalToAPI(approval)
	api.Attachment = esbAttachmentToAPI(attachment)
}

// esbBlocksToData renders the type-specific blocks from the wire model. Only the block
// matching the binding type is populated: the API echoes back an empty object for the
// other one, which would otherwise turn a block the operator left unset into a non-null
// value in state.
func esbBlocksToData(api *PMSEvidenceSourceBindingAPIModel, diagnostics *diag.Diagnostics) (approval, attachment types.Object) {
	approval = types.ObjectNull(esbApprovalAttrTypes)
	attachment = types.ObjectNull(esbAttachmentAttrTypes)
	switch api.Type {
	case evidenceSourceBindingTypeApproval:
		if api.Approval == nil {
			return
		}
		obj, diags := types.ObjectValue(esbApprovalAttrTypes, map[string]attr.Value{
			"approval":              actionToData(api.Approval.Approval, diagnostics),
			"rejection":             actionToData(api.Approval.Rejection, diagnostics),
			"identity_list_version": identityListVersionToData(api.Approval.IdentityListVersion, diagnostics),
		})
		diagnostics.Append(diags...)
		approval = obj
	case evidenceSourceBindingTypeAttachment:
		if api.Attachment == nil {
			return
		}
		obj, diags := types.ObjectValue(esbAttachmentAttrTypes, map[string]attr.Value{
			"payload_jsonata":     jsonataAttr(api.Attachment.PayloadMapping),
			"attestation_jsonata": jsonataAttr(api.Attachment.AttestationMapping),
		})
		diagnostics.Append(diags...)
		attachment = obj
	}
	return
}

func (r *pms_evidenceSourceBindingResource) toData(api *PMSEvidenceSourceBindingAPIModel, data *PMSEvidenceSourceBindingResourceModel, diagnostics *diag.Diagnostics) {
	data.ID = types.StringValue(api.ID)
	if api.Name != "" {
		data.Name = types.StringValue(api.Name)
	}
	if api.Type != "" {
		data.Type = types.StringValue(api.Type)
	}

	data.Approval, data.Attachment = esbBlocksToData(api, diagnostics)
}

// identityListVersionToData renders the identity list version reference, treating a
// reference with no fields set as absent.
func identityListVersionToData(ilv *PMSIdentityListVersionReferenceAPIModel, diagnostics *diag.Diagnostics) types.Object {
	if ilv == nil || (ilv.ID == "" && ilv.Version == "" && ilv.Hash == "") {
		return types.ObjectNull(esbIdentityListVersionAttrTypes)
	}
	obj, diags := types.ObjectValue(esbIdentityListVersionAttrTypes, map[string]attr.Value{
		"id":      optionalString(ilv.ID),
		"version": optionalString(ilv.Version),
		"hash":    optionalString(ilv.Hash),
	})
	diagnostics.Append(diags...)
	return obj
}

func (r *pms_evidenceSourceBindingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PMSEvidenceSourceBindingResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var api PMSEvidenceSourceBindingAPIModel
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

	validateEvidenceSourceBindingBlocks(data.Type.ValueString(), data.Approval, data.Attachment, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	patch := PMSEvidenceSourceBindingPatchAPIModel{
		Approval:   esbApprovalToAPI(data.Approval),
		Attachment: esbAttachmentToAPI(data.Attachment),
	}

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
