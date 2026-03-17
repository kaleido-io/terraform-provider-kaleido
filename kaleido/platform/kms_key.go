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
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type KMSKeyResourceModel struct {
	ID                    types.String `tfsdk:"id"`
	Environment           types.String `tfsdk:"environment"`
	Service               types.String `tfsdk:"service"`
	Wallet                types.String `tfsdk:"wallet"`
	Name                  types.String `tfsdk:"name"`
	Path                  types.String `tfsdk:"path"`
	URI                   types.String `tfsdk:"uri"`
	Address               types.String `tfsdk:"address"`
	Attributes            types.Map    `tfsdk:"attributes"`
	PublicIdentifierTypes types.List   `tfsdk:"public_identifier_types"`
}

type KMSKeyAPIModel struct {
	ID                    string            `json:"id,omitempty"`
	Created               *time.Time        `json:"created,omitempty"`
	Updated               *time.Time        `json:"updated,omitempty"`
	Name                  string            `json:"name"`
	Path                  string            `json:"path,omitempty"`
	URI                   string            `json:"uri,omitempty"`
	Address               string            `json:"address,omitempty"`
	Attributes            map[string]string `json:"attributes,omitempty"`
	PublicIdentifierTypes []string          `json:"publicIdentifierTypes,omitempty"`
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
	resp.Schema = schema.Schema{
		Description: "A reference to a signing key (also known as a key mapping) that is directly/indirectly derived from a piece of key material, and can be used for signing.",
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"environment": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Environment ID",
			},
			"service": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Key Manager Service ID",
			},
			"wallet": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Wallet ID",
			},
			"name": &schema.StringAttribute{
				Required:    true, // technically optional in Kaleido service, but it is an anti-pattern we do not support in the terraform provider
				Description: "Key Display Name",
			},
			"path": &schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "A unique identifier for a piece of key material that is understood by the associated signing technology for a wallet. Each key that exists must have a path to associate the key with the key material that is used for signing.",
			},
			"uri": &schema.StringAttribute{
				Computed: true,
			},
			"address": &schema.StringAttribute{
				Computed: true,
			},
			"attributes": &schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Optional attributes of the key for key creation.",
			},
			"public_identifier_types": &schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Optional public identifier types to create for the key.",
			},
		},
	}
}

func (data *KMSKeyResourceModel) toAPI(ctx context.Context, api *KMSKeyAPIModel, diagnostics *diag.Diagnostics) {
	api.Name = data.Name.ValueString()
	api.Path = data.Path.ValueString()

	if !data.Attributes.IsNull() {
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
}

func (r *kms_keyResource) apiPath(ctx context.Context, data *KMSKeyResourceModel, diagnostics *diag.Diagnostics) (string, bool) {
	// KMS requires that key operations are performed using the NAME of the wallet, not the ID.
	// Whereas we strongly encourage in the terraform provider using the ID of the wallet.
	var wallet KMSWalletAPIModel
	walletPath := fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/wallets/%s", data.Environment.ValueString(), data.Service.ValueString(), data.Wallet.ValueString())
	ok, _ := r.apiRequest(ctx, http.MethodGet, walletPath, nil, &wallet, diagnostics)
	if !ok {
		return "", false
	}
	path := fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/wallets/%s/keys", data.Environment.ValueString(), data.Service.ValueString(), wallet.Name)
	if data.ID.ValueString() != "" {
		path = path + "/" + data.ID.ValueString()
	}
	return path, true
}

func (r *kms_keyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var data KMSKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	// Preserve planned publicIdentifierTypes, as API does not return them on GET
	plannedPublicIdentifierTypes := data.PublicIdentifierTypes

	var api KMSKeyAPIModel
	data.toAPI(ctx, &api, &resp.Diagnostics)
	apiPath, ok := r.apiPath(ctx, &data, &resp.Diagnostics)
	if ok {
		ok, _ = r.apiRequest(ctx, http.MethodPut /* note different to wallets */, apiPath, api, &api, &resp.Diagnostics)
	}
	if !ok {
		return
	}

	api.toData(ctx, &data, &resp.Diagnostics)
	// Restore planned value into state
	if !plannedPublicIdentifierTypes.IsNull() && !plannedPublicIdentifierTypes.IsUnknown() {
		data.PublicIdentifierTypes = plannedPublicIdentifierTypes
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)

}

func (r *kms_keyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

	var data KMSKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)

	// Preserve planned publicIdentifierTypes, as API does not return them on GET
	plannedPublicIdentifierTypes := data.PublicIdentifierTypes

	// Read full current object
	var api KMSKeyAPIModel
	apiPath, ok := r.apiPath(ctx, &data, &resp.Diagnostics)
	if ok {
		ok, _ = r.apiRequest(ctx, http.MethodGet, apiPath, nil, &api, &resp.Diagnostics)
	}
	if !ok {
		return
	}

	// Update from plan
	data.toAPI(ctx, &api, &resp.Diagnostics)
	if ok, _ = r.apiRequest(ctx, http.MethodPatch /* note there is no put-by-ID */, apiPath, api, &api, &resp.Diagnostics); !ok {
		return
	}

	api.toData(ctx, &data, &resp.Diagnostics)
	// Restore planned value into state
	if !plannedPublicIdentifierTypes.IsNull() && !plannedPublicIdentifierTypes.IsUnknown() {
		data.PublicIdentifierTypes = plannedPublicIdentifierTypes
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *kms_keyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data KMSKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	// Preserve current state's publicIdentifierTypes, as API does not return them on GET
	currentPublicIdentifierTypes := data.PublicIdentifierTypes

	var api KMSKeyAPIModel
	api.ID = data.ID.ValueString()
	apiPath, ok := r.apiPath(ctx, &data, &resp.Diagnostics)
	if !ok {
		return
	}
	ok, status := r.apiRequest(ctx, http.MethodGet, apiPath, nil, &api, &resp.Diagnostics, Allow404())
	if !ok {
		return
	}
	if status == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	api.toData(ctx, &data, &resp.Diagnostics)
	// Restore current state's value into state
	if !currentPublicIdentifierTypes.IsNull() && !currentPublicIdentifierTypes.IsUnknown() {
		data.PublicIdentifierTypes = currentPublicIdentifierTypes
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *kms_keyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data KMSKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	apiPath, ok := r.apiPath(ctx, &data, &resp.Diagnostics)
	if !ok {
		return
	}
	_, _ = r.apiRequest(ctx, http.MethodDelete, apiPath, nil, nil, &resp.Diagnostics, Allow404())

	r.waitForRemoval(ctx, apiPath, &resp.Diagnostics)
}
