// Copyright © Kaleido, Inc. 2024-2026

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
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kaleido-io/terraform-provider-kaleido/kaleido/planmodifiers"
)

type KMSKeyResourceModel struct {
	ID                    types.String `tfsdk:"id"`
	Environment           types.String `tfsdk:"environment"`
	Service               types.String `tfsdk:"service"`
	Wallet                types.String `tfsdk:"wallet"`
	Name                  types.String `tfsdk:"name"`
	Path                  types.String `tfsdk:"path"`
	FolderPath            types.String `tfsdk:"folder_path"`
	URI                   types.String `tfsdk:"uri"`
	Address               types.String `tfsdk:"address"`
	Attributes            types.Map    `tfsdk:"attributes"`
	PublicIdentifierTypes types.List   `tfsdk:"public_identifier_types"`
	PublicIdentifiers     types.Map    `tfsdk:"public_identifiers"`
	KeystoreName          types.String `tfsdk:"keystore_name"`
	Spec                  types.String `tfsdk:"spec"`
}

type KMSKeyAPIModel struct {
	ID                    string            `json:"id,omitempty"`
	KeystoreName          string            `json:"keystoreName,omitempty"` // v2 only
	Spec                  string            `json:"spec,omitempty"`         // v2 only
	Created               *time.Time        `json:"created,omitempty"`
	Updated               *time.Time        `json:"updated,omitempty"`
	Name                  string            `json:"name"`
	Path                  string            `json:"path,omitempty"`      // v1 request/response field
	KeyHandle             string            `json:"keyHandle,omitempty"` // v2 only
	URI                   string            `json:"uri,omitempty"`
	Address               string            `json:"address,omitempty"` // v1 only; v2 never returns this — see PublicIdentifiers
	Attributes            map[string]string `json:"attributes,omitempty"`
	PublicIdentifierTypes []string          `json:"publicIdentifierTypes,omitempty"`

	// v2 create request/response only.
	ReturnPublicIdentifiers bool                   `json:"returnPublicIdentifiers,omitempty"`
	PublicIdentifiers       []PublicIdentifierWire `json:"publicIdentifiers,omitempty"`
}

// PublicIdentifierWire is one entry of the publicIdentifiers v2 returns inline
// on a create request when returnPublicIdentifiers is set.
type PublicIdentifierWire struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

func KMSKeyResourceFactory() resource.Resource {
	return &kms_keyResource{}
}

type kms_keyResource struct {
	commonResource
}

func (r *kms_keyResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_kms_key"
}

func (r *kms_keyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	const typeName = "kaleido_platform_kms_key"
	resp.Schema = schema.Schema{
		Version:     1, // bumped for keystore_name/spec/public_identifiers — see UpgradeState
		Description: "A reference to a signing key (also known as a key mapping) that is directly/indirectly derived from a piece of key material, and can be used for signing.",
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"environment": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{planmodifiers.RequireRecreate(typeName)},
				Description:   "Environment ID. Immutable after create — changing this value is not supported; create a new, separate key instead.",
			},
			"service": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{planmodifiers.RequireRecreate(typeName)},
				Description:   "Key Manager Service ID. Immutable after create — changing this value is not supported; create a new, separate key instead.",
			},
			"wallet": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{planmodifiers.RequireRecreate(typeName)},
				Description:   "Wallet ID. Immutable after create — changing this value is not supported; create a new, separate key instead.",
			},
			"name": &schema.StringAttribute{
				Required:    true, // technically optional in Kaleido service, but it is an anti-pattern we do not support in the terraform provider
				Description: "Key Display Name",
			},
			"uri": &schema.StringAttribute{
				Computed:    true,
				Description: "The canonical URI of the key, assigned by the server after creation. Updates when the key is renamed.",
			},
			"address": &schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Description:   "The key's address_ethereum public identifier value. Empty if address_ethereum isn't one of public_identifier_types — see public_identifiers for every identifier value the key has.",
			},
			"path": &schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{planmodifiers.RequireRecreate(typeName), stringplanmodifier.UseStateForUnknown()},
				Description:   "A unique identifier for a piece of key material that is understood by the associated signing technology for a wallet. Each key that exists must have a path to associate the key with the key material that is used for signing. Immutable after create — changing this value is not supported; create a new, separate key instead. Not supported together with keystore_name/spec — pin a path on a v2-created key via attributes[\"bip44_path\"] instead.",
			},
			"folder_path": &schema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{planmodifiers.RequireRecreate(typeName)},
				Description:   "Slash-separated folder hierarchy to place this key in, e.g. \"treasury\" or \"ops/hot\". Folders are automatically created if they do not exist. Immutable after create — changing this value is not supported; create a new, separate key instead.",
			},
			"attributes": &schema.MapAttribute{
				Optional:      true,
				Computed:      true,
				ElementType:   types.StringType,
				PlanModifiers: []planmodifier.Map{mapplanmodifier.UseStateForUnknown(), planmodifiers.RequireRecreateMap(typeName)},
				Description:   "Optional attributes of the key for key creation. Merged server-side with the wallet's default_key_attributes (per-key values take precedence), so the value read back may include additional entries even when none are set here. Immutable after create — changing this value is not supported; create a new, separate key instead.",
			},
			"public_identifier_types": &schema.ListAttribute{
				Optional:      true,
				ElementType:   types.StringType,
				PlanModifiers: []planmodifier.List{planmodifiers.RequireRecreateList(typeName)},
				Description:   "Public identifier types to create for the key. Recommended: also set spec (or keystore_name) so every entry here is honoured — without either, only address_ethereum is ever created. Applied only at create — immutable after create; changing this value is not supported; create a new, separate key instead.",
			},
			"public_identifiers": &schema.MapAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Value of each public identifier on the key, keyed by type (e.g. address_ethereum, address_ethereum_checksum). Read via the v2 API regardless of which API created the key.",
			},
			"keystore_name": &schema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{planmodifiers.RequireRecreate(typeName)},
				Description:   "Recommended: set this (or spec) so every entry in public_identifier_types is honoured — without either, only address_ethereum is ever created. Defaults to the wallet's name when unset. Immutable after create.",
			},
			"spec": &schema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{planmodifiers.RequireRecreate(typeName)},
				Description:   "Key algorithm/spec (e.g. secp256k1). Recommended: set this (or keystore_name) — see keystore_name. Immutable after create.",
			},
		},
	}
}

// KMSKeyResourceModelV0 is the schema-version-0 resource model, frozen as of
// the last release without keystore_name/spec/public_identifiers. Kept only
// so UpgradeState can decode state written before those attributes existed;
// never add fields here.
type KMSKeyResourceModelV0 struct {
	ID                    types.String `tfsdk:"id"`
	Environment           types.String `tfsdk:"environment"`
	Service               types.String `tfsdk:"service"`
	Wallet                types.String `tfsdk:"wallet"`
	Name                  types.String `tfsdk:"name"`
	Path                  types.String `tfsdk:"path"`
	FolderPath            types.String `tfsdk:"folder_path"`
	URI                   types.String `tfsdk:"uri"`
	Address               types.String `tfsdk:"address"`
	Attributes            types.Map    `tfsdk:"attributes"`
	PublicIdentifierTypes types.List   `tfsdk:"public_identifier_types"`
}

func (r *kms_keyResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	const typeName = "kaleido_platform_kms_key"
	priorSchema := &schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"environment": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{planmodifiers.RequireRecreate(typeName)},
			},
			"service": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{planmodifiers.RequireRecreate(typeName)},
			},
			"wallet": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{planmodifiers.RequireRecreate(typeName)},
			},
			"name": &schema.StringAttribute{
				Required: true,
			},
			"uri": &schema.StringAttribute{
				Computed: true,
			},
			"address": &schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"path": &schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{planmodifiers.RequireRecreate(typeName), stringplanmodifier.UseStateForUnknown()},
			},
			"folder_path": &schema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{planmodifiers.RequireRecreate(typeName)},
			},
			"attributes": &schema.MapAttribute{
				Optional:      true,
				Computed:      true,
				ElementType:   types.StringType,
				PlanModifiers: []planmodifier.Map{mapplanmodifier.UseStateForUnknown(), planmodifiers.RequireRecreateMap(typeName)},
			},
			"public_identifier_types": &schema.ListAttribute{
				Optional:      true,
				ElementType:   types.StringType,
				PlanModifiers: []planmodifier.List{planmodifiers.RequireRecreateList(typeName)},
			},
		},
	}

	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: priorSchema,
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var prior KMSKeyResourceModelV0
				resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
				if resp.Diagnostics.HasError() {
					return
				}

				resp.Diagnostics.Append(resp.State.Set(ctx, KMSKeyResourceModel{
					ID:                    prior.ID,
					Environment:           prior.Environment,
					Service:               prior.Service,
					Wallet:                prior.Wallet,
					Name:                  prior.Name,
					Path:                  prior.Path,
					FolderPath:            prior.FolderPath,
					URI:                   prior.URI,
					Address:               prior.Address,
					Attributes:            prior.Attributes,
					PublicIdentifierTypes: prior.PublicIdentifierTypes,
					PublicIdentifiers:     types.MapNull(types.StringType),
					KeystoreName:          types.StringNull(),
					Spec:                  types.StringNull(),
				})...)
			},
		},
	}
}

func (data *KMSKeyResourceModel) toAPI(ctx context.Context, api *KMSKeyAPIModel, diagnostics *diag.Diagnostics) {
	api.Name = data.Name.ValueString()
	api.Path = data.Path.ValueString()
	api.KeystoreName = data.KeystoreName.ValueString()
	api.Spec = data.Spec.ValueString()

	if !data.Attributes.IsNull() && !data.Attributes.IsUnknown() {
		attrs := map[string]string{}
		d := data.Attributes.ElementsAs(ctx, &attrs, false)
		diagnostics.Append(d...)
		api.Attributes = attrs
	}

	if !data.PublicIdentifierTypes.IsNull() {
		var piTypes []string
		d := data.PublicIdentifierTypes.ElementsAs(ctx, &piTypes, false)
		diagnostics.Append(d...)
		api.PublicIdentifierTypes = piTypes
	}
}

func (api *KMSKeyAPIModel) toData(ctx context.Context, data *KMSKeyResourceModel, diagnostics *diag.Diagnostics) {
	data.ID = types.StringValue(api.ID)
	data.Path = types.StringValue(api.Path)
	data.URI = types.StringValue(api.URI)
	data.Address = types.StringValue(api.Address)

	if api.Attributes != nil {
		tfMap, d := types.MapValueFrom(ctx, types.StringType, api.Attributes)
		diagnostics.Append(d...)
		data.Attributes = tfMap
	} else {
		data.Attributes = types.MapNull(types.StringType)
	}

	if api.PublicIdentifierTypes != nil {
		tfList, d := types.ListValueFrom(ctx, types.StringType, api.PublicIdentifierTypes)
		diagnostics.Append(d...)
		data.PublicIdentifierTypes = tfList
	} else {
		data.PublicIdentifierTypes = types.ListNull(types.StringType)
	}

	if len(api.PublicIdentifiers) > 0 {
		piMap := make(map[string]string, len(api.PublicIdentifiers))
		for _, pi := range api.PublicIdentifiers {
			piMap[pi.Type] = pi.Value
		}
		tfMap, d := types.MapValueFrom(ctx, types.StringType, piMap)
		diagnostics.Append(d...)
		data.PublicIdentifiers = tfMap
	} else {
		data.PublicIdentifiers = types.MapNull(types.StringType)
	}
}

// deriveV2Fields normalises a v2 response onto the v1-shaped fields toData
// reads: v2 calls the derivation path keyHandle instead of path, and has no
// address field at all — address only exists as a publicIdentifiers entry.
func (api *KMSKeyAPIModel) deriveV2Fields() {
	if api.KeyHandle != "" {
		api.Path = api.KeyHandle
	}
	for _, pi := range api.PublicIdentifiers {
		if pi.Type == "address_ethereum" {
			api.Address = pi.Value
			break
		}
	}
}

// apiPath resolves the wallet ID to its name (required by the KMS API) and returns
// both the full key API path and the wallet name. The wallet name is needed on Create
// to build the folder URI.
func (r *kms_keyResource) apiPath(ctx context.Context, data *KMSKeyResourceModel, diagnostics *diag.Diagnostics) (string, string, bool) {
	var wallet KMSWalletAPIModel
	walletPath := fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/wallets/%s", data.Environment.ValueString(), data.Service.ValueString(), data.Wallet.ValueString())
	ok, _ := r.apiRequest(ctx, http.MethodGet, walletPath, nil, &wallet, diagnostics)
	if !ok {
		return "", "", false
	}
	p := fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/wallets/%s/keys", data.Environment.ValueString(), data.Service.ValueString(), wallet.Name)
	if data.ID.ValueString() != "" {
		p = p + "/" + data.ID.ValueString()
	}
	return p, wallet.Name, true
}

func (data *KMSKeyResourceModel) hasFolderPath() bool {
	return !data.FolderPath.IsNull() && data.FolderPath.ValueString() != ""
}

// keyByIDPathV2 is the v2 global key-by-ID route. It's folder-agnostic (works
// the same for folder-placed keys), so Read/Update/Delete use it exclusively —
// no wallet-scoped v1 route or folder special-casing needed, and it works
// regardless of which API created the key.
func (r *kms_keyResource) keyByIDPathV2(data *KMSKeyResourceModel) string {
	return fmt.Sprintf("/endpoint/%s/%s/rest/api/v2/keys/%s",
		data.Environment.ValueString(), data.Service.ValueString(), data.ID.ValueString())
}

func (r *kms_keyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var data KMSKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	// Preserve planned publicIdentifierTypes, as API does not return them on GET
	plannedPublicIdentifierTypes := data.PublicIdentifierTypes
	// Preserve planned attributes: the server merges the wallet's
	// default_key_attributes into the response, so writing the API value straight
	// into state would break plan-consistency (config set fewer entries than the
	// server returns). When the user set attributes explicitly, keep exactly what
	// they asked for; when they didn't, take whatever the server returned.
	plannedAttributes := data.Attributes

	// Setting keystore_name or spec opts this key into the v2 create API, which
	// supports every entry in public_identifier_types. Otherwise Create uses
	// v1 PUT, which only creates an address_ethereum identifier.
	useV2 := (!data.KeystoreName.IsNull() && data.KeystoreName.ValueString() != "") ||
		(!data.Spec.IsNull() && data.Spec.ValueString() != "")

	var api KMSKeyAPIModel
	data.toAPI(ctx, &api, &resp.Diagnostics)
	apiPath, walletName, ok := r.apiPath(ctx, &data, &resp.Diagnostics)
	if !ok {
		return
	}
	// If the user specified a folder_path, build the URI so the API auto-creates
	// the folder hierarchy and places the key within it.
	if data.hasFolderPath() {
		folderPath := strings.TrimPrefix(data.FolderPath.ValueString(), "/")
		api.URI = fmt.Sprintf("kld:///keystore/%s/key/%s/%s", walletName, folderPath, data.Name.ValueString())
	}

	if useV2 {
		// v2 silently ignores a top-level path; pin one via attributes["bip44_path"]
		// instead (HD wallets only).
		if api.Path != "" {
			resp.Diagnostics.AddError(
				"path is not supported with keystore_name/spec",
				"v2 ignores a top-level path silently. Set attributes[\"bip44_path\"] instead to pin a derivation path.",
			)
			return
		}

		if api.KeystoreName == "" {
			api.KeystoreName = walletName
		}
		if api.Spec == "" {
			api.Spec = "secp256k1"
		}
		// Returns publicIdentifiers inline on the create response, so address
		// doesn't need a second round-trip to fetch.
		api.ReturnPublicIdentifiers = true

		createPath := fmt.Sprintf("/endpoint/%s/%s/rest/api/v2/keys", data.Environment.ValueString(), data.Service.ValueString())
		if ok, _ = r.apiRequest(ctx, http.MethodPost, createPath, api, &api, &resp.Diagnostics); !ok {
			return
		}
		api.deriveV2Fields()
	} else {
		if ok, _ = r.apiRequest(ctx, http.MethodPut /* upsert-by-name, unlike kms_wallet's plain POST create */, apiPath, api, &api, &resp.Diagnostics); !ok {
			return
		}
	}

	api.toData(ctx, &data, &resp.Diagnostics)
	// Restore planned values that the API does not echo back
	if !plannedPublicIdentifierTypes.IsNull() && !plannedPublicIdentifierTypes.IsUnknown() {
		data.PublicIdentifierTypes = plannedPublicIdentifierTypes
	}
	if !plannedAttributes.IsNull() && !plannedAttributes.IsUnknown() {
		data.Attributes = plannedAttributes
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)

}

func (r *kms_keyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

	var data KMSKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)

	// Preserve planned publicIdentifierTypes, as API does not return them on GET
	plannedPublicIdentifierTypes := data.PublicIdentifierTypes
	// Preserve planned attributes for the same reason as in Create.
	plannedAttributes := data.Attributes

	keyPath := r.keyByIDPathV2(&data)

	// Read the full current object first: v2 PATCH's response only carries
	// name/uri, so address/path/public_identifiers below come from this GET,
	// not the PATCH response.
	var api KMSKeyAPIModel
	if ok, _ := r.apiRequest(ctx, http.MethodGet, keyPath+"?fetchDetail=true", nil, &api, &resp.Diagnostics); !ok {
		return
	}

	// Update from plan. PATCH only accepts name/labels on v2.
	var patched KMSKeyAPIModel
	patch := KMSKeyAPIModel{Name: data.Name.ValueString()}
	if ok, _ := r.apiRequest(ctx, http.MethodPatch, keyPath, patch, &patched, &resp.Diagnostics); !ok {
		return
	}
	api.Name = patched.Name
	api.URI = patched.URI

	api.deriveV2Fields()
	api.toData(ctx, &data, &resp.Diagnostics)
	// Restore planned values that the API does not echo back
	if !plannedPublicIdentifierTypes.IsNull() && !plannedPublicIdentifierTypes.IsUnknown() {
		data.PublicIdentifierTypes = plannedPublicIdentifierTypes
	}
	if !plannedAttributes.IsNull() && !plannedAttributes.IsUnknown() {
		data.Attributes = plannedAttributes
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *kms_keyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data KMSKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	// Preserve fields that the API does not return on GET
	currentFolderPath := data.FolderPath
	currentPublicIdentifierTypes := data.PublicIdentifierTypes
	// Preserve state.Attributes if it's non-null: the API merges wallet defaults
	// into the read-back value, and we don't want to overwrite the user's config
	// value with the wider merged set.
	currentAttributes := data.Attributes

	var api KMSKeyAPIModel
	getPath := r.keyByIDPathV2(&data) + "?fetchDetail=true"
	ok, status := r.apiRequest(ctx, http.MethodGet, getPath, nil, &api, &resp.Diagnostics, Allow404())
	if !ok {
		return
	}
	if status == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	api.deriveV2Fields()
	api.toData(ctx, &data, &resp.Diagnostics)
	if !currentFolderPath.IsNull() {
		data.FolderPath = currentFolderPath
	}
	if !currentPublicIdentifierTypes.IsNull() && !currentPublicIdentifierTypes.IsUnknown() {
		data.PublicIdentifierTypes = currentPublicIdentifierTypes
	}
	if !currentAttributes.IsNull() {
		data.Attributes = currentAttributes
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *kms_keyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data KMSKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	deletePath := r.keyByIDPathV2(&data)
	if ok, _ := r.apiRequest(ctx, http.MethodDelete, deletePath, nil, nil, &resp.Diagnostics, Allow404()); !ok {
		return
	}

	r.waitForRemoval(ctx, deletePath, &resp.Diagnostics)
}
