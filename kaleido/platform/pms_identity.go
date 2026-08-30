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
	"reflect"
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

// NotificationMethod is read-only on the v2 API - identities return their notification
// methods but there is no v2 write path for them.
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
	"public_key_jwk_json":  types.StringType,
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
	"value_json": types.StringType,
}

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
							Description: "The key format: Multikey (use public_key_multibase) or JsonWebKey (use public_key_jwk_json)",
							Validators:  []validator.String{stringvalidator.OneOf("Multikey", "JsonWebKey")},
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
							Optional:    true,
							Description: "JWK-encoded public key as a JSON string (use jsonencode), for type JsonWebKey (RFC 7517). For Ethereum signing: {kty:EC, crv:secp256k1, x:..., y:...}",
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
			KeyURI:             stringAttr(attrs, "key_uri"),
		}
		if jwk := stringAttr(attrs, "public_key_jwk_json"); jwk != "" {
			if !json.Valid([]byte(jwk)) {
				diagnostics.AddError("Invalid JSON", fmt.Sprintf("Failed to parse verification method public_key_jwk_json: %s", jwk))
				return
			}
			vm.PublicKeyJwk = json.RawMessage(jwk)
		}
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

	priorVMs := data.VerificationMethods.Elements()
	verificationMethods := make([]attr.Value, len(api.VerificationMethods))
	for i, vm := range api.VerificationMethods {
		var priorJwk types.String
		if i < len(priorVMs) {
			if obj, ok := priorVMs[i].(types.Object); ok {
				if val, ok := obj.Attributes()["public_key_jwk_json"]; ok {
					if str, ok := val.(types.String); ok {
						priorJwk = str
					}
				}
			}
		}
		publicKeyJwk := preserveJSONFormatting(priorJwk, vm.PublicKeyJwk)
		attrs := map[string]attr.Value{
			"id":                   types.StringValue(vm.ID),
			"identity_id":          types.StringValue(vm.IdentityID),
			"name":                 optionalString(vm.Name),
			"type":                 optionalString(vm.Type),
			"controller":           optionalString(vm.Controller),
			"public_key_multibase": optionalString(vm.PublicKeyMultibase),
			"public_key_jwk_json":  publicKeyJwk,
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
		valueJSON := types.StringNull()
		if nm.Value != nil {
			valueJSON = types.StringValue(string(nm.Value))
		}
		obj, diags := types.ObjectValue(notificationMethodAttrTypes, map[string]attr.Value{
			"id":         types.StringValue(nm.ID),
			"name":       optionalString(nm.Name),
			"type":       optionalString(nm.Type),
			"value_json": valueJSON,
		})
		diagnostics.Append(diags...)
		notificationMethods[i] = obj
	}
	data.NotificationMethods = types.ListValueMust(types.ObjectType{AttrTypes: notificationMethodAttrTypes}, notificationMethods)
}

// preserveJSONFormatting keeps the configured JSON string when the API returns the same
// value formatted differently. The server re-serializes a JWK with its keys sorted, so a
// byte comparison against the configured string would otherwise report a change that is
// not one.
func preserveJSONFormatting(configured types.String, apiValue json.RawMessage) types.String {
	if len(apiValue) == 0 {
		return types.StringNull()
	}
	if !configured.IsNull() && !configured.IsUnknown() &&
		jsonSemanticallyEqual([]byte(configured.ValueString()), apiValue) {
		return configured
	}
	return types.StringValue(string(apiValue))
}

// jsonSemanticallyEqual reports whether two JSON documents differ only in key order or
// whitespace.
func jsonSemanticallyEqual(a, b []byte) bool {
	var aVal, bVal interface{}
	if json.Unmarshal(a, &aVal) != nil || json.Unmarshal(b, &bVal) != nil {
		return false
	}
	return reflect.DeepEqual(aVal, bVal)
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
