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
	"gopkg.in/yaml.v3"
)

type PMSPolicyResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Environment    types.String `tfsdk:"environment"`
	Service        types.String `tfsdk:"service"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	DefinitionYAML types.String `tfsdk:"definition_yaml"`
	Version        types.String `tfsdk:"version"`
	AppliedVersion types.String `tfsdk:"applied_version"`
	Created        types.String `tfsdk:"created"`
	Updated        types.String `tfsdk:"updated"`

	IdentityListBindings   types.List `tfsdk:"identity_list_binding"`
	EvidenceSourceBindings types.List `tfsdk:"evidence_source_binding"`
}

type PMSPolicyAPIModel struct {
	ID             string     `json:"id,omitempty"`
	Name           string     `json:"name,omitempty"`
	Description    string     `json:"description,omitempty"`
	CurrentVersion string     `json:"currentVersion,omitempty"`
	Created        *time.Time `json:"created,omitempty"`
	Updated        *time.Time `json:"updated,omitempty"`
}

// PMSPolicyVersionAPIModel is a policy version. The definition fields (components,
// constants, evidence, decision, output, parameters, parameterValues, summaryTemplate)
// sit alongside the metadata at the top level of the object, and are passed through
// opaquely from definition_yaml.
type PMSPolicyVersionAPIModel struct {
	ID          string     `json:"id,omitempty"`
	Name        string     `json:"name,omitempty"`
	PolicyID    string     `json:"policyId,omitempty"`
	Description string     `json:"description,omitempty"`
	Hash        string     `json:"hash,omitempty"`
	Created     *time.Time `json:"created,omitempty"`
	Updated     *time.Time `json:"updated,omitempty"`
}

func PMSPolicyResourceFactory() resource.Resource {
	return &pms_policyResource{}
}

type pms_policyResource struct {
	commonResource
}

func (r *pms_policyResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_pms_policy"
}

func (r *pms_policyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a policy in the Policy Manager. A policy owns a set of immutable versions - each apply that changes definition_yaml creates and activates a new version. definition_yaml is optional, so a policy can be created as an empty container to hold bindings before its first version exists.",
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
				Description:   "Policy Manager service ID",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": &schema.StringAttribute{
				Required:      true,
				Description:   "Unique name of the policy",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"description": &schema.StringAttribute{
				Optional:    true,
				Description: "Description of the policy",
			},
			"definition_yaml": &schema.StringAttribute{
				Optional:    true,
				Description: "The policy definition as YAML, containing components, constants, evidence, decision, output, parameters, parameterValues and summaryTemplate. Omit it to create the policy as an empty container, so that bindings and a kaleido_platform_pms_policy_version resource can be declared separately. Matchers are never part of the definition - they are managed by kaleido_platform_pms_policy_matcher.",
			},
			"version": &schema.StringAttribute{
				Optional:    true,
				Description: "Name to give the policy version created by this resource. If omitted the server assigns a name based on the previous version.",
			},
			"applied_version": &schema.StringAttribute{
				Computed:    true,
				Description: "The currently activated version of the policy",
			},
			"created": &schema.StringAttribute{
				Computed:    true,
				Description: "Creation timestamp",
			},
			"updated": &schema.StringAttribute{
				Computed:    true,
				Description: "Last update timestamp",
			},
			"identity_list_binding": &schema.ListNestedAttribute{
				Optional:    true,
				Description: "Identity list bindings declared inline on the policy, each resolving an attester label used by the policy definition to a version of an identity list. They are written in the same call that creates the policy and its first version, which is what lets a definition reference a label on the very first apply. Only the labels listed here are managed - any other binding on the policy is left untouched, so bindings managed by a kaleido_platform_pms_policy_identity_list_binding resource can safely coexist.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": &schema.StringAttribute{
							Computed:    true,
							Description: "The binding ID assigned by the server",
						},
						"attester_label": &schema.StringAttribute{
							Required:    true,
							Description: "The attester label declared in a policy evidence attestation slot, e.g. 'treasuryOperations'",
						},
						"identity_list_version_id": &schema.StringAttribute{
							Required:    true,
							Description: "ID of the identity list version whose members may attest under this label. This is a version ID, not a version name - use the applied_version_id attribute of a kaleido_platform_pms_identity_list.",
						},
					},
				},
			},
			"evidence_source_binding": &schema.ListNestedAttribute{
				Optional:    true,
				Description: "Evidence source bindings declared inline on the policy, each telling the policy where the evidence for a slot comes from. As with identity list bindings these are written in the same call that creates the policy and its first version, and only the names listed here are managed, so bindings managed by a kaleido_platform_pms_policy_evidence_source_binding resource can safely coexist.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": &schema.StringAttribute{
							Computed:    true,
							Description: "The binding ID assigned by the server",
						},
						"name": &schema.StringAttribute{
							Required:    true,
							Description: "The name of the evidence source binding within the policy. Referenced by the 'source' field of an evidence slot in the policy definition.",
						},
						"type": &schema.StringAttribute{
							Required:    true,
							Description: "The type of evidence source binding: 'approval' or 'attachment'",
							Validators:  []validator.String{stringvalidator.OneOf(evidenceSourceBindingTypeApproval, evidenceSourceBindingTypeAttachment)},
						},
						"approval":   evidenceSourceBindingApprovalSchema(),
						"attachment": evidenceSourceBindingAttachmentSchema(),
					},
				},
			},
		},
	}
}

func (r *pms_policyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.commonResource.Configure(ctx, req, resp)
}

func (r *pms_policyResource) apiPath(data *PMSPolicyResourceModel, idOrName string) string {
	env := data.Environment.ValueString()
	service := data.Service.ValueString()
	return fmt.Sprintf("/endpoint/%s/%s/rest/api/v2/policies/%s", env, service, url.PathEscape(idOrName))
}

func (r *pms_policyResource) apiPolicyVersionPath(data *PMSPolicyResourceModel, idOrName string) string {
	return r.apiPath(data, idOrName) + "/versions"
}

func (r *pms_policyResource) apiIdentityListBindingPath(data *PMSPolicyResourceModel, policyIdOrName string) string {
	return r.apiPath(data, policyIdOrName) + "/identity-list-bindings"
}

func (r *pms_policyResource) apiEvidenceSourceBindingPath(data *PMSPolicyResourceModel, policyIdOrName string) string {
	return r.apiPath(data, policyIdOrName) + "/evidence-source-bindings"
}

// definitionFields parses definition_yaml into the fields the API carries at the top
// level of a policy version. It returns nil when no definition is configured, which is
// how a policy is created as an empty container.
func (r *pms_policyResource) definitionFields(data *PMSPolicyResourceModel, diagnostics *diag.Diagnostics) map[string]interface{} {
	if data.DefinitionYAML.IsNull() || data.DefinitionYAML.IsUnknown() {
		return nil
	}
	fields := map[string]interface{}{}
	if err := yaml.Unmarshal([]byte(data.DefinitionYAML.ValueString()), &fields); err != nil {
		diagnostics.AddError("Invalid YAML", fmt.Sprintf("Failed to parse policy definition YAML: %v", err))
		return nil
	}
	if len(fields) == 0 {
		diagnostics.AddError("Invalid policy definition", "definition_yaml is set but empty - omit it entirely to create the policy without a version")
		return nil
	}
	return fields
}

// toUpsertBody builds the flattened body of the policy upsert: the policy metadata, the
// inline binding maps for the kinds the configuration declares, and - when
// definition_yaml is set - the definition at the top level along with the version
// metadata. Sending all of it in one request means the bindings a definition references
// are written before the version resolves, which is what the API requires of a version
// whose constants initialize from an identity list binding.
func (r *pms_policyResource) toUpsertBody(data *PMSPolicyResourceModel, diagnostics *diag.Diagnostics) map[string]interface{} {
	definition := r.definitionFields(data, diagnostics)
	if diagnostics.HasError() {
		return nil
	}

	body := map[string]interface{}{}
	for field, value := range definition {
		body[field] = value
	}
	// On this endpoint the top level 'name' and 'description' belong to the policy, while
	// the version travels as 'version' and 'versionDescription'. A definition carrying its
	// own 'description' therefore has to be moved across, or it would silently become the
	// policy's description.
	if versionDescription, supplied := body["description"]; supplied {
		body["versionDescription"] = versionDescription
	}
	body["name"] = data.Name.ValueString()
	body["description"] = data.Description.ValueString()
	if definition != nil {
		if version := data.Version.ValueString(); version != "" {
			body["version"] = version
		}
	}

	r.addBindingMaps(data, body, diagnostics)
	if diagnostics.HasError() {
		return nil
	}
	return body
}

// addBindingMaps adds the inline binding maps to a policy write body. A map is only
// added for a kind the configuration declares: the API leaves a kind alone entirely
// when its key is absent from the body, which is what keeps separately managed bindings
// of that kind intact.
func (r *pms_policyResource) addBindingMaps(data *PMSPolicyResourceModel, body map[string]interface{}, diagnostics *diag.Diagnostics) {
	if !data.IdentityListBindings.IsNull() && !data.IdentityListBindings.IsUnknown() {
		desired := r.desiredIdentityListBindings(data, diagnostics)
		if diagnostics.HasError() {
			return
		}
		bindings := map[string]interface{}{}
		for label, versionID := range desired {
			bindings[label] = map[string]interface{}{"identityListVersionId": versionID}
		}
		body["identityListBindings"] = bindings
	}

	if !data.EvidenceSourceBindings.IsNull() && !data.EvidenceSourceBindings.IsUnknown() {
		desired := r.desiredEvidenceSourceBindings(data, diagnostics)
		if diagnostics.HasError() {
			return
		}
		bindings := map[string]interface{}{}
		for name, binding := range desired {
			bindings[name] = binding
		}
		body["evidenceSourceBindings"] = bindings
	}
}

// toVersionBody builds the body posted to the versions endpoint, where the version's own
// name and description are the top level 'name' and 'description'.
func (r *pms_policyResource) toVersionBody(data *PMSPolicyResourceModel, diagnostics *diag.Diagnostics) map[string]interface{} {
	body := r.definitionFields(data, diagnostics)
	if body == nil {
		return nil
	}
	if version := data.Version.ValueString(); version != "" {
		body["name"] = version
	}
	// 'description' is left as the definition supplied it: on this endpoint it is the
	// version's own description, not the policy's
	return body
}

func (r *pms_policyResource) toData(api *PMSPolicyAPIModel, data *PMSPolicyResourceModel) {
	data.ID = types.StringValue(api.ID)
	data.Name = types.StringValue(api.Name)
	data.AppliedVersion = types.StringValue(api.CurrentVersion)
	// Note: environment and service are not returned by the API, they remain as set in the resource

	if api.Description == "" {
		data.Description = types.StringNull()
	} else {
		data.Description = types.StringValue(api.Description)
	}
	data.Created = timeAttr(api.Created)
	data.Updated = timeAttr(api.Updated)
}

var policyIdentityListBindingAttrTypes = map[string]attr.Type{
	"id":                       types.StringType,
	"attester_label":           types.StringType,
	"identity_list_version_id": types.StringType,
}

var policyEvidenceSourceBindingAttrTypes = map[string]attr.Type{
	"id":         types.StringType,
	"name":       types.StringType,
	"type":       types.StringType,
	"approval":   types.ObjectType{AttrTypes: esbApprovalAttrTypes},
	"attachment": types.ObjectType{AttrTypes: esbAttachmentAttrTypes},
}

// desiredIdentityListBindings reads the inline blocks, keyed by attester label. The
// label is the identity of a binding: the server holds a unique index on
// (policy_id, attester_label).
func (r *pms_policyResource) desiredIdentityListBindings(data *PMSPolicyResourceModel, diagnostics *diag.Diagnostics) map[string]string {
	desired := map[string]string{}
	if data.IdentityListBindings.IsNull() || data.IdentityListBindings.IsUnknown() {
		return desired
	}
	for _, item := range data.IdentityListBindings.Elements() {
		obj, ok := item.(types.Object)
		if !ok {
			continue
		}
		attrs := obj.Attributes()
		label := stringAttr(attrs, "attester_label")
		if _, duplicate := desired[label]; duplicate {
			diagnostics.AddError("Duplicate identity list binding",
				fmt.Sprintf("attester_label %q is declared more than once; each label may be bound only once per policy", label))
			return nil
		}
		desired[label] = stringAttr(attrs, "identity_list_version_id")
	}
	return desired
}

// desiredEvidenceSourceBindings reads the inline blocks, keyed by binding name, which is
// how the API and the 'source' field of an evidence slot address them.
func (r *pms_policyResource) desiredEvidenceSourceBindings(data *PMSPolicyResourceModel, diagnostics *diag.Diagnostics) map[string]*PMSEvidenceSourceBindingAPIModel {
	desired := map[string]*PMSEvidenceSourceBindingAPIModel{}
	if data.EvidenceSourceBindings.IsNull() || data.EvidenceSourceBindings.IsUnknown() {
		return desired
	}
	for _, item := range data.EvidenceSourceBindings.Elements() {
		obj, ok := item.(types.Object)
		if !ok {
			continue
		}
		attrs := obj.Attributes()
		name := stringAttr(attrs, "name")
		if _, duplicate := desired[name]; duplicate {
			diagnostics.AddError("Duplicate evidence source binding",
				fmt.Sprintf("name %q is declared more than once; each evidence source binding name is unique within a policy", name))
			return nil
		}
		approval, _ := objectAttr(attrs, "approval")
		attachment, _ := objectAttr(attrs, "attachment")
		binding := &PMSEvidenceSourceBindingAPIModel{}
		evidenceSourceBindingToAPI(name, stringAttr(attrs, "type"), approval, attachment, binding, diagnostics)
		if diagnostics.HasError() {
			return nil
		}
		desired[name] = binding
	}
	return desired
}

// listIdentityListBindings returns the policy's bindings keyed by attester label.
func (r *pms_policyResource) listIdentityListBindings(ctx context.Context, data *PMSPolicyResourceModel, policyID string, diagnostics *diag.Diagnostics) map[string]PMSIdentityListBindingAPIModel {
	var result struct {
		Items []PMSIdentityListBindingAPIModel `json:"items"`
	}
	ok, _ := r.apiRequest(ctx, http.MethodGet, r.apiIdentityListBindingPath(data, policyID), nil, &result, diagnostics)
	if !ok {
		return nil
	}
	byLabel := make(map[string]PMSIdentityListBindingAPIModel, len(result.Items))
	for _, item := range result.Items {
		byLabel[item.AttesterLabel] = item
	}
	return byLabel
}

// listEvidenceSourceBindings returns the policy's bindings keyed by name.
func (r *pms_policyResource) listEvidenceSourceBindings(ctx context.Context, data *PMSPolicyResourceModel, policyID string, diagnostics *diag.Diagnostics) map[string]PMSEvidenceSourceBindingAPIModel {
	var result struct {
		Items []PMSEvidenceSourceBindingAPIModel `json:"items"`
	}
	ok, _ := r.apiRequest(ctx, http.MethodGet, r.apiEvidenceSourceBindingPath(data, policyID), nil, &result, diagnostics)
	if !ok {
		return nil
	}
	byName := make(map[string]PMSEvidenceSourceBindingAPIModel, len(result.Items))
	for _, item := range result.Items {
		byName[item.Name] = item
	}
	return byName
}

// bindingsToData refreshes both sets of inline blocks from the server. The write
// responses do not carry the binding IDs, so the collections are listed to pick them up.
func (r *pms_policyResource) bindingsToData(ctx context.Context, data *PMSPolicyResourceModel, policyID string, diagnostics *diag.Diagnostics) {
	r.identityListBindingsToData(ctx, data, policyID, diagnostics)
	if diagnostics.HasError() {
		return
	}
	r.evidenceSourceBindingsToData(ctx, data, policyID, diagnostics)
}

// identityListBindingsToData refreshes the inline blocks from the server, preserving the
// configured ordering and ignoring bindings this resource does not manage.
func (r *pms_policyResource) identityListBindingsToData(ctx context.Context, data *PMSPolicyResourceModel, policyID string, diagnostics *diag.Diagnostics) {
	listType := types.ObjectType{AttrTypes: policyIdentityListBindingAttrTypes}
	if data.IdentityListBindings.IsNull() || data.IdentityListBindings.IsUnknown() {
		data.IdentityListBindings = types.ListNull(listType)
		return
	}
	existing := r.listIdentityListBindings(ctx, data, policyID, diagnostics)
	if diagnostics.HasError() {
		return
	}
	elements := []attr.Value{}
	for _, item := range data.IdentityListBindings.Elements() {
		obj, ok := item.(types.Object)
		if !ok {
			continue
		}
		label := stringAttr(obj.Attributes(), "attester_label")
		current, found := existing[label]
		if !found {
			// Deleted outside terraform - drop it so the next plan recreates it
			continue
		}
		value, diags := types.ObjectValue(policyIdentityListBindingAttrTypes, map[string]attr.Value{
			"id":                       types.StringValue(current.ID),
			"attester_label":           types.StringValue(label),
			"identity_list_version_id": types.StringValue(current.IdentityListVersionID),
		})
		diagnostics.Append(diags...)
		elements = append(elements, value)
	}
	if len(elements) == 0 {
		data.IdentityListBindings = types.ListNull(listType)
		return
	}
	data.IdentityListBindings = types.ListValueMust(listType, elements)
}

// evidenceSourceBindingsToData refreshes the inline blocks from the server, preserving
// the configured ordering and ignoring bindings this resource does not manage.
func (r *pms_policyResource) evidenceSourceBindingsToData(ctx context.Context, data *PMSPolicyResourceModel, policyID string, diagnostics *diag.Diagnostics) {
	listType := types.ObjectType{AttrTypes: policyEvidenceSourceBindingAttrTypes}
	if data.EvidenceSourceBindings.IsNull() || data.EvidenceSourceBindings.IsUnknown() {
		data.EvidenceSourceBindings = types.ListNull(listType)
		return
	}
	existing := r.listEvidenceSourceBindings(ctx, data, policyID, diagnostics)
	if diagnostics.HasError() {
		return
	}
	elements := []attr.Value{}
	for _, item := range data.EvidenceSourceBindings.Elements() {
		obj, ok := item.(types.Object)
		if !ok {
			continue
		}
		name := stringAttr(obj.Attributes(), "name")
		current, found := existing[name]
		if !found {
			// Deleted outside terraform - drop it so the next plan recreates it
			continue
		}
		approval, attachment := esbBlocksToData(&current, diagnostics)
		value, diags := types.ObjectValue(policyEvidenceSourceBindingAttrTypes, map[string]attr.Value{
			"id":         types.StringValue(current.ID),
			"name":       types.StringValue(name),
			"type":       types.StringValue(current.Type),
			"approval":   approval,
			"attachment": attachment,
		})
		diagnostics.Append(diags...)
		elements = append(elements, value)
	}
	if len(elements) == 0 {
		data.EvidenceSourceBindings = types.ListNull(listType)
		return
	}
	data.EvidenceSourceBindings = types.ListValueMust(listType, elements)
}

// deleteDroppedBindings removes the bindings this resource previously managed that have
// since been dropped from the configuration. A binding that was never listed here is
// left alone, so one managed by its own resource is not destroyed.
func (r *pms_policyResource) deleteDroppedBindings(ctx context.Context, data, priorState *PMSPolicyResourceModel, policyID string, diagnostics *diag.Diagnostics) {
	previousLabels := r.desiredIdentityListBindings(priorState, diagnostics)
	desiredLabels := r.desiredIdentityListBindings(data, diagnostics)
	previousNames := r.desiredEvidenceSourceBindings(priorState, diagnostics)
	desiredNames := r.desiredEvidenceSourceBindings(data, diagnostics)
	if diagnostics.HasError() {
		return
	}

	var droppedLabels, droppedNames []string
	for label := range previousLabels {
		if _, stillDesired := desiredLabels[label]; !stillDesired {
			droppedLabels = append(droppedLabels, label)
		}
	}
	for name := range previousNames {
		if _, stillDesired := desiredNames[name]; !stillDesired {
			droppedNames = append(droppedNames, name)
		}
	}

	if len(droppedLabels) > 0 {
		existing := r.listIdentityListBindings(ctx, data, policyID, diagnostics)
		if diagnostics.HasError() {
			return
		}
		basePath := r.apiIdentityListBindingPath(data, policyID)
		for _, label := range droppedLabels {
			current, found := existing[label]
			if !found {
				continue
			}
			if ok, _ := r.apiRequest(ctx, http.MethodDelete, fmt.Sprintf("%s/%s", basePath, current.ID), nil, nil, diagnostics, Allow404()); !ok {
				return
			}
		}
	}

	if len(droppedNames) > 0 {
		existing := r.listEvidenceSourceBindings(ctx, data, policyID, diagnostics)
		if diagnostics.HasError() {
			return
		}
		basePath := r.apiEvidenceSourceBindingPath(data, policyID)
		for _, name := range droppedNames {
			current, found := existing[name]
			if !found {
				continue
			}
			if ok, _ := r.apiRequest(ctx, http.MethodDelete, fmt.Sprintf("%s/%s", basePath, current.ID), nil, nil, diagnostics, Allow404()); !ok {
				return
			}
		}
	}
}

func (r *pms_policyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PMSPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := r.toUpsertBody(&data, &resp.Diagnostics)
	if body == nil {
		return
	}

	// The policy, its inline bindings and its first version are all written by this one
	// call, so the version resolves against bindings that arrive with it
	var api PMSPolicyAPIModel
	ok, _ := r.apiRequest(ctx, http.MethodPut, r.apiPath(&data, data.Name.ValueString()), body, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	policyRef := api.ID
	if policyRef == "" {
		policyRef = data.Name.ValueString()
	}

	// currentVersion is returned by a plain GET. Asking for withCurrentVersion would
	// additionally embed the version body, and is rejected outright for a policy that has
	// no version yet.
	var updatedAPI PMSPolicyAPIModel
	ok, _ = r.apiRequest(ctx, http.MethodGet, r.apiPath(&data, policyRef), nil, &updatedAPI, &resp.Diagnostics)
	if !ok {
		return
	}

	r.toData(&updatedAPI, &data)
	r.bindingsToData(ctx, &data, policyRef, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *pms_policyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PMSPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var api PMSPolicyAPIModel
	ok, status := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data, data.ID.ValueString()), nil, &api, &resp.Diagnostics, Allow404())
	if !ok {
		return
	}
	if status == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	r.toData(&api, &data)
	r.bindingsToData(ctx, &data, api.ID, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *pms_policyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data PMSPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var priorState PMSPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &priorState)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policyID := data.ID.ValueString()

	// A PATCH merges the binding maps rather than treating them as the policy's complete
	// set, so bindings managed elsewhere survive. It cannot carry a version, which is
	// posted separately below.
	patch := map[string]interface{}{"description": data.Description.ValueString()}
	r.addBindingMaps(&data, patch, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	ok, _ := r.apiRequest(ctx, http.MethodPatch, r.apiPath(&data, policyID), patch, nil, &resp.Diagnostics)
	if !ok {
		return
	}

	// Versions are immutable, so a new one is cut only when the definition or the name
	// asked for it has actually changed
	definitionChanged := !data.DefinitionYAML.Equal(priorState.DefinitionYAML) || !data.Version.Equal(priorState.Version)
	if definitionChanged && !data.DefinitionYAML.IsNull() {
		versionBody := r.toVersionBody(&data, &resp.Diagnostics)
		if versionBody == nil {
			return
		}
		if ok, _ := r.apiRequest(ctx, http.MethodPost, r.apiPolicyVersionPath(&data, policyID), versionBody, nil, &resp.Diagnostics); !ok {
			return
		}
	}

	// Dropped bindings are removed after the new version is in place, so a binding the
	// outgoing version still needs is not taken away from under it
	r.deleteDroppedBindings(ctx, &data, &priorState, policyID, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	var updatedAPI PMSPolicyAPIModel
	ok, _ = r.apiRequest(ctx, http.MethodGet, r.apiPath(&data, policyID), nil, &updatedAPI, &resp.Diagnostics)
	if !ok {
		return
	}

	r.toData(&updatedAPI, &data)
	r.bindingsToData(ctx, &data, policyID, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *pms_policyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PMSPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data, data.ID.ValueString()), nil, nil, &resp.Diagnostics, Allow404())
	r.waitForRemoval(ctx, r.apiPath(&data, data.ID.ValueString()), &resp.Diagnostics)
}
