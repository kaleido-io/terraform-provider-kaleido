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
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type PolicyIdentityResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Environment         types.String `tfsdk:"environment"`
	Service             types.String `tfsdk:"service"`
	Name                types.String `tfsdk:"name"`
	Description         types.String `tfsdk:"description"`
	Controller          types.String `tfsdk:"controller"`
	VerificationMethods types.List   `tfsdk:"verification_method"`
	NotificationMethods types.List   `tfsdk:"notification_method"`
}

type PolicyIdentityAPIModel struct {
	ID                  string               `json:"id,omitempty"`
	Created             *time.Time           `json:"created,omitempty"`
	Updated             *time.Time           `json:"updated,omitempty"`
	Name                string               `json:"name,omitempty"`
	Description         string               `json:"description,omitempty"`
	Controller          string               `json:"controller,omitempty"`
	VerificationMethods []VerificationMethod `json:"verificationMethods,omitempty"`
	NotificationMethods []NotificationMethod `json:"notificationMethods,omitempty"`
}

type NotificationMethod struct {
	ID         string          `json:"id,omitempty"`
	IdentityID string          `json:"identityId,omitempty"`
	Name       string          `json:"name,omitempty"`
	Type       string          `json:"type,omitempty"`
	Value      json.RawMessage `json:"value,omitempty"`
}

type VerificationMethod struct {
	ID                 string          `json:"id,omitempty"`
	IdentityID         string          `json:"identityId,omitempty"`
	Name               string          `json:"name,omitempty"`
	Type               string          `json:"type,omitempty"`
	Controller         string          `json:"controller,omitempty"`
	PublicKeyMultibase string          `json:"publicKeyMultibase,omitempty"`
	PublicKeyJwk       json.RawMessage `json:"publicKeyJwk,omitempty"`
	EthereumAddress    string          `json:"ethereumAddress,omitempty"`
	KeyURI             string          `json:"keyUri,omitempty"`
	Created            *time.Time      `json:"created,omitempty"`
	Updated            *time.Time      `json:"updated,omitempty"`
	Expires            *time.Time      `json:"expires,omitempty"`
	Revoked            *time.Time      `json:"revoked,omitempty"`
}

// verificationMethodAttrTypes is the object type of the verification_method list elements.
var verificationMethodAttrTypes = map[string]attr.Type{
	"id":                   types.StringType,
	"identity_id":          types.StringType,
	"name":                 types.StringType,
	"type":                 types.StringType,
	"controller":           types.StringType,
	"public_key_multibase": types.StringType,
	"public_key_jwk_json":  jsonStringType{},
	"ethereum_address":     types.StringType,
	"created":              types.StringType,
	"expires":              types.StringType,
	"revoked":              types.StringType,
	"key_uri":              types.StringType,
}

// notificationMethodAttrTypes is the object type of the notification_method list elements.
var notificationMethodAttrTypes = map[string]attr.Type{
	"id":         types.StringType,
	"name":       types.StringType,
	"type":       types.StringType,
	"value_json": jsonStringType{},
}

var _ resource.ResourceWithUpgradeState = &policyIdentityResource{}

func PMSIdentityResourceFactory() resource.Resource {
	return &policyIdentityResource{}
}

type policyIdentityResource struct {
	commonResource
}

func (r *policyIdentityResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_pms_identity"
}

func (r *policyIdentityResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     1,
		Description: "Manages Policy Manager identities. An identity is a subject that can make attestations, and carries the verification methods (public keys) used to prove those attestations.",
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
				Description:   "Unique name of the identity",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"description": &schema.StringAttribute{
				Optional:      true,
				Description:   "Description of the identity.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"controller": &schema.StringAttribute{
				Optional:      true,
				Description:   "Optional controller (KID) of the identity, e.g. a user or application. If set, attestations against this identity are only accepted from that controller. This is the field called `owner` on the v1 API. Distinct from the `controller` of an individual verification method, which identifies who controls that particular key.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"verification_method": &schema.ListNestedAttribute{
				Optional:      true,
				Description:   "Array of verification methods (cryptographic keys) that can be used to prove statements made by this identity",
				PlanModifiers: []planmodifier.List{listplanmodifier.RequiresReplace()},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": &schema.StringAttribute{
							Computed:      true,
							PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
						},
						"identity_id": &schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "ID of the identity this verification method belongs to",
						},
						"name": &schema.StringAttribute{
							Optional:    true,
							Description: "A human-readable label for this verification method, e.g. 'primary-signing-key'",
						},
						"type": &schema.StringAttribute{
							Optional:    true,
							Description: "The key format: Multikey (use public_key_multibase), JsonWebKey (use public_key_jwk_json) or EthereumAddress (use ethereum_address)",
							Validators:  []validator.String{stringvalidator.OneOf("Multikey", "JsonWebKey", "EthereumAddress")},
						},
						"controller": &schema.StringAttribute{
							Optional:    true,
							Description: "Optional URI, KID, or DID identifying who controls this specific key (may differ from the identity subject)",
						},
						"public_key_multibase": &schema.StringAttribute{
							Optional:    true,
							Description: "Multibase-encoded public key, for type Multikey. For secp256k1/Ethereum: 0xe701 varint prefix + 33-byte compressed key, base58btc-encoded with a 'z' header.",
						},
						"public_key_jwk_json": &schema.StringAttribute{
							CustomType:  jsonStringType{},
							Optional:    true,
							Description: "JWK-encoded public key as a JSON string (use jsonencode), for type JsonWebKey (RFC 7517). For Ethereum signing: {kty:EC, crv:secp256k1, x:..., y:...}",
						},
						"ethereum_address": &schema.StringAttribute{
							Optional:    true,
							Description: "0x-prefixed Ethereum address, for type EthereumAddress. Matched only when verifying EIP-712 attestations, by recovering the signer's address from the signature.",
						},
						"created": &schema.StringAttribute{
							Computed:    true,
							Description: "Creation timestamp",
						},
						"expires": &schema.StringAttribute{
							Optional:    true,
							Description: "Expiration timestamp",
						},
						"revoked": &schema.StringAttribute{
							Optional:    true,
							Description: "Revocation timestamp",
						},
						"key_uri": &schema.StringAttribute{
							Optional:    true,
							Description: "URI of the Key Manager key backing this verification method, e.g. 'kld:///keystore/<id>/key/<name>'. Required to route a signing request to the Key Manager, which addresses keys by URI rather than by public key.",
						},
					},
				},
			},
			"notification_method": &schema.ListNestedAttribute{
				Computed:      true,
				Description:   "Notification methods (e.g. workflow) associated with this identity. Read-only: the v2 Policy Manager API returns notification methods but provides no way to create them.",
				PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown()},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": &schema.StringAttribute{
							Computed:    true,
							Description: "ID of the notification method",
						},
						"name": &schema.StringAttribute{
							Computed:    true,
							Description: "Name of the notification method",
						},
						"type": &schema.StringAttribute{
							Computed:    true,
							Description: "Type of the notification method, e.g. 'workflow' or 'email'",
						},
						"value_json": &schema.StringAttribute{
							CustomType:  jsonStringType{},
							Computed:    true,
							Description: "The type-specific configuration of the notification method, as a JSON string",
						},
					},
				},
			},
		},
	}
}

func (r *policyIdentityResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.commonResource.Configure(ctx, req, resp)
}

func (r *policyIdentityResource) apiPath(data *PolicyIdentityResourceModel) string {
	env := data.Environment.ValueString()
	service := data.Service.ValueString()

	if data.ID.IsNull() || data.ID.IsUnknown() {
		return fmt.Sprintf("/endpoint/%s/%s/rest/api/v2/identities", env, service)
	}
	return r.apiInstancePath(data, data.ID.ValueString())
}

func (r *policyIdentityResource) apiInstancePath(data *PolicyIdentityResourceModel, id string) string {
	return fmt.Sprintf("/endpoint/%s/%s/rest/api/v2/identities/%s?fetchDetails=true",
		data.Environment.ValueString(), data.Service.ValueString(), id)
}

func (r *policyIdentityResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PolicyIdentityResourceModel
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var api PolicyIdentityAPIModel
	r.toAPI(&data, &api, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	ok, _ := r.apiRequest(ctx, "POST", r.apiPath(&data), api, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	// The create response carries only the identity itself - its verification methods
	// are inserted separately and are not attached to it. Re-read with fetchDetails so
	// they are not nulled out of state straight after apply.
	var created PolicyIdentityAPIModel
	ok, _ = r.apiRequest(ctx, "GET", r.apiInstancePath(&data, api.ID), nil, &created, &resp.Diagnostics)
	if !ok {
		return
	}

	r.toData(&created, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *policyIdentityResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PolicyIdentityResourceModel
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var api PolicyIdentityAPIModel
	ok, status := r.apiRequest(ctx, "GET", r.apiPath(&data), nil, &api, &resp.Diagnostics, Allow404())
	if !ok {
		return
	}
	if status == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	r.toData(&api, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *policyIdentityResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// The API has no identity PATCH - every attribute requires replacement
	resp.Diagnostics.AddError("Update not supported", "Policy identities cannot be updated. Use replace instead.")
}

func (r *policyIdentityResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PolicyIdentityResourceModel
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, _ = r.apiRequest(ctx, "DELETE", r.apiPath(&data), nil, nil, &resp.Diagnostics, Allow404())
}

func (r *policyIdentityResource) toAPI(data *PolicyIdentityResourceModel, api *PolicyIdentityAPIModel, diagnostics *diag.Diagnostics) {
	api.Name = data.Name.ValueString()
	api.Description = data.Description.ValueString()
	api.Controller = data.Controller.ValueString()

	if data.VerificationMethods.IsNull() || data.VerificationMethods.IsUnknown() {
		return
	}
	var verificationMethods []VerificationMethod
	for _, item := range data.VerificationMethods.Elements() {
		obj, ok := item.(types.Object)
		if !ok {
			continue
		}
		attrs := obj.Attributes()
		vm := VerificationMethod{
			ID:                 stringAttr(attrs, "id"),
			IdentityID:         stringAttr(attrs, "identity_id"),
			Name:               stringAttr(attrs, "name"),
			Type:               stringAttr(attrs, "type"),
			Controller:         stringAttr(attrs, "controller"),
			PublicKeyMultibase: stringAttr(attrs, "public_key_multibase"),
			EthereumAddress:    stringAttr(attrs, "ethereum_address"),
			KeyURI:             stringAttr(attrs, "key_uri"),
		}
		vm.PublicKeyJwk = jsonAttrToAPI(attrs, "public_key_jwk_json")
		if expires := stringAttr(attrs, "expires"); expires != "" {
			t, err := time.Parse(time.RFC3339, expires)
			if err != nil {
				diagnostics.AddError("Invalid timestamp", fmt.Sprintf("Failed to parse verification method expires %q: %v", expires, err))
				return
			}
			vm.Expires = &t
		}
		if revoked := stringAttr(attrs, "revoked"); revoked != "" {
			t, err := time.Parse(time.RFC3339, revoked)
			if err != nil {
				diagnostics.AddError("Invalid timestamp", fmt.Sprintf("Failed to parse verification method revoked %q: %v", revoked, err))
				return
			}
			vm.Revoked = &t
		}
		verificationMethods = append(verificationMethods, vm)
	}
	api.VerificationMethods = verificationMethods
}

// stringAttr reads an optional string out of a terraform object's attribute map.
func stringAttr(attrs map[string]attr.Value, name string) string {
	val, ok := attrs[name]
	if !ok || val.IsNull() || val.IsUnknown() {
		return ""
	}
	s, ok := val.(types.String)
	if !ok {
		return ""
	}
	return s.ValueString()
}

func (r *policyIdentityResource) toData(api *PolicyIdentityAPIModel, data *PolicyIdentityResourceModel, diagnostics *diag.Diagnostics) {
	data.ID = types.StringValue(api.ID)
	data.Name = types.StringValue(api.Name)
	// Note: environment and service are not returned by the API, they remain as set in the resource

	if api.Description != "" {
		data.Description = types.StringValue(api.Description)
	} else {
		data.Description = types.StringNull()
	}

	data.Controller = optionalString(api.Controller)

	r.notificationMethodsToData(api, data, diagnostics)

	if len(api.VerificationMethods) == 0 {
		data.VerificationMethods = types.ListNull(types.ObjectType{AttrTypes: verificationMethodAttrTypes})
		return
	}

	verificationMethods := make([]attr.Value, len(api.VerificationMethods))
	for i, vm := range api.VerificationMethods {
		attrs := map[string]attr.Value{
			"id":                   types.StringValue(vm.ID),
			"identity_id":          types.StringValue(vm.IdentityID),
			"name":                 optionalString(vm.Name),
			"type":                 optionalString(vm.Type),
			"controller":           optionalString(vm.Controller),
			"public_key_multibase": optionalString(vm.PublicKeyMultibase),
			"public_key_jwk_json":  jsonFromAPI(vm.PublicKeyJwk),
			"ethereum_address":     optionalString(vm.EthereumAddress),
			"key_uri":              optionalString(vm.KeyURI),
			"created":              timeAttr(vm.Created),
			"expires":              timeAttr(vm.Expires),
			"revoked":              timeAttr(vm.Revoked),
		}
		obj, diags := types.ObjectValue(verificationMethodAttrTypes, attrs)
		diagnostics.Append(diags...)
		verificationMethods[i] = obj
	}
	data.VerificationMethods = types.ListValueMust(types.ObjectType{AttrTypes: verificationMethodAttrTypes}, verificationMethods)
}

// notificationMethodsToData renders the read-only notification methods returned by the API.
func (r *policyIdentityResource) notificationMethodsToData(api *PolicyIdentityAPIModel, data *PolicyIdentityResourceModel, diagnostics *diag.Diagnostics) {
	if len(api.NotificationMethods) == 0 {
		data.NotificationMethods = types.ListNull(types.ObjectType{AttrTypes: notificationMethodAttrTypes})
		return
	}
	notificationMethods := make([]attr.Value, len(api.NotificationMethods))
	for i, nm := range api.NotificationMethods {
		obj, diags := types.ObjectValue(notificationMethodAttrTypes, map[string]attr.Value{
			"id":         types.StringValue(nm.ID),
			"name":       optionalString(nm.Name),
			"type":       optionalString(nm.Type),
			"value_json": jsonFromAPI(nm.Value),
		})
		diagnostics.Append(diags...)
		notificationMethods[i] = obj
	}
	data.NotificationMethods = types.ListValueMust(types.ObjectType{AttrTypes: notificationMethodAttrTypes}, notificationMethods)
}

// optionalString renders an API string as a terraform string, mapping empty to null
// so that attributes left unset in configuration stay null in state.
func optionalString(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}

// timeAttr renders an optional API timestamp as an RFC3339 terraform string.
func timeAttr(t *time.Time) types.String {
	if t == nil {
		return types.StringNull()
	}
	return types.StringValue(t.Format(time.RFC3339))
}

// policyIdentityStateV0 is the state of schema version 0 as stored. Two shapes carry that
// version: state written against the v1 Policy Manager API (owner, assertion_method,
// preferred_assertion_method), and state written against the v2 API before the schema
// was versioned (controller, verification_method). Every value in either is a string or
// null.
type policyIdentityStateV0 struct {
	ID                 *string              `json:"id"`
	Environment        *string              `json:"environment"`
	Service            *string              `json:"service"`
	Name               *string              `json:"name"`
	Description        *string              `json:"description"`
	Owner              *string              `json:"owner"`
	Controller         *string              `json:"controller"`
	AssertionMethod    []map[string]*string `json:"assertion_method"`
	VerificationMethod []map[string]*string `json:"verification_method"`
	NotificationMethod []map[string]*string `json:"notification_method"`
}

// UpgradeState migrates state of schema version 0 to version 1. The v1 and v2 Policy
// Manager APIs read and write the same identities and verification methods, so the
// identity keeps its ID either way, and the refresh that follows the upgrade fills in
// anything the old state did not hold. There is no prior schema: the raw state is read
// directly, because a prior schema can describe only one of the two shapes and would
// silently drop the attributes of the other.
func (r *policyIdentityResource) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {StateUpgrader: upgradePolicyIdentityStateV0},
	}
}

// upgradePolicyIdentityStateV0 carries v2 attributes across unchanged, keeping the stored
// JSON text so the next plan compares equal to the configuration, and maps v1 attributes
// onto their v2 equivalents:
//   - owner is the identity's controller (the same stored value, renamed on the v2 API)
//   - each assertion_method is a verification_method, its verification_material the
//     ethereum_address of an EthereumAddress method (the only type that ever had one)
//   - preferred_assertion_method and signing_method have no v2 equivalent and are dropped
//   - notification_method, now read-only, keeps its values until the refresh adds IDs
func upgradePolicyIdentityStateV0(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
	if req.RawState == nil {
		resp.Diagnostics.AddError("Unable to upgrade state", "no prior state was supplied")
		return
	}
	var old policyIdentityStateV0
	if err := json.Unmarshal(req.RawState.JSON, &old); err != nil {
		resp.Diagnostics.AddError("Unable to upgrade state", fmt.Sprintf("the stored state of the identity could not be read: %s", err))
		return
	}

	controller := old.Controller
	if controller == nil {
		controller = old.Owner
	}

	var verificationMethods []map[string]attr.Value
	if old.VerificationMethod != nil {
		for _, vm := range old.VerificationMethod {
			verificationMethods = append(verificationMethods, map[string]attr.Value{
				"id":                   types.StringPointerValue(vm["id"]),
				"identity_id":          types.StringPointerValue(vm["identity_id"]),
				"name":                 types.StringPointerValue(vm["name"]),
				"type":                 types.StringPointerValue(vm["type"]),
				"controller":           types.StringPointerValue(vm["controller"]),
				"public_key_multibase": types.StringPointerValue(vm["public_key_multibase"]),
				"public_key_jwk_json":  jsonStringPointer(vm["public_key_jwk_json"]),
				"ethereum_address":     types.StringPointerValue(vm["ethereum_address"]),
				"created":              types.StringPointerValue(vm["created"]),
				"expires":              types.StringPointerValue(vm["expires"]),
				"revoked":              types.StringPointerValue(vm["revoked"]),
				"key_uri":              types.StringPointerValue(vm["key_uri"]),
			})
		}
	} else if old.AssertionMethod != nil {
		for _, am := range old.AssertionMethod {
			ethereumAddress := types.StringNull()
			if am["type"] != nil && *am["type"] == "EthereumAddress" {
				ethereumAddress = types.StringPointerValue(am["verification_material"])
			}
			verificationMethods = append(verificationMethods, map[string]attr.Value{
				"id":                   types.StringPointerValue(am["id"]),
				"identity_id":          types.StringPointerValue(am["identity_id"]),
				"name":                 types.StringPointerValue(am["name"]),
				"type":                 types.StringPointerValue(am["type"]),
				"controller":           types.StringNull(),
				"public_key_multibase": types.StringNull(),
				"public_key_jwk_json":  jsonStringNull(),
				"ethereum_address":     ethereumAddress,
				"created":              types.StringPointerValue(am["created"]),
				"expires":              types.StringPointerValue(am["expires"]),
				"revoked":              types.StringPointerValue(am["revoked"]),
				"key_uri":              types.StringNull(),
			})
		}
	}

	var notificationMethods []map[string]attr.Value
	for _, nm := range old.NotificationMethod {
		notificationMethods = append(notificationMethods, map[string]attr.Value{
			"id":         types.StringPointerValue(nm["id"]),
			"name":       types.StringPointerValue(nm["name"]),
			"type":       types.StringPointerValue(nm["type"]),
			"value_json": jsonStringPointer(nm["value_json"]),
		})
	}

	data := PolicyIdentityResourceModel{
		ID:                  types.StringPointerValue(old.ID),
		Environment:         types.StringPointerValue(old.Environment),
		Service:             types.StringPointerValue(old.Service),
		Name:                types.StringPointerValue(old.Name),
		Description:         types.StringPointerValue(old.Description),
		Controller:          types.StringPointerValue(controller),
		VerificationMethods: objectListOrNull(verificationMethodAttrTypes, verificationMethods, old.VerificationMethod != nil || old.AssertionMethod != nil, &resp.Diagnostics),
		NotificationMethods: objectListOrNull(notificationMethodAttrTypes, notificationMethods, old.NotificationMethod != nil, &resp.Diagnostics),
	}
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// jsonStringPointer is a *_json attribute holding exactly the stored text, or null.
func jsonStringPointer(s *string) jsonStringValue {
	if s == nil {
		return jsonStringNull()
	}
	return jsonStringOf(*s)
}

// objectListOrNull builds a list of objects, or a null list when the stored state held
// no list at all.
func objectListOrNull(attrTypes map[string]attr.Type, elements []map[string]attr.Value, present bool, diagnostics *diag.Diagnostics) types.List {
	elemType := types.ObjectType{AttrTypes: attrTypes}
	if !present {
		return types.ListNull(elemType)
	}
	values := make([]attr.Value, 0, len(elements))
	for _, element := range elements {
		obj, diags := types.ObjectValue(attrTypes, element)
		diagnostics.Append(diags...)
		values = append(values, obj)
	}
	list, diags := types.ListValue(elemType, values)
	diagnostics.Append(diags...)
	return list
}
