// Copyright © Kaleido, Inc. 2025

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
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type IdentityProviderResourceModel struct {
	ID                      types.String `tfsdk:"id"`
	Name                    types.String `tfsdk:"name"`
	ClientID                types.String `tfsdk:"client_id"`
	ClientSecret            types.String `tfsdk:"client_secret"`
	ClientType              types.String `tfsdk:"client_type"`
	Hostname                types.String `tfsdk:"hostname"`
	Issuer                  types.String `tfsdk:"issuer"`
	OIDCConfigURL           types.String `tfsdk:"oidc_config_url"`
	JWKS                    types.String `tfsdk:"jwks"`
	JWKSURL                 types.String `tfsdk:"jwks_url"`
	LoginURL                types.String `tfsdk:"login_url"`
	LogoutURL               types.String `tfsdk:"logout_url"`
	TokenURL                types.String `tfsdk:"token_url"`
	UserInfoURL             types.String `tfsdk:"user_info_url"`
	Scopes                  types.String `tfsdk:"scopes"`
	CACertificate           types.String `tfsdk:"ca_certificate"`
	ConfidentialPKCEEnabled types.Bool   `tfsdk:"confidential_pkce_enabled,omitempty"`
	IdTokenNonceEnabled     types.Bool   `tfsdk:"id_token_nonce_enabled,omitempty"`
}

type IdentityProviderAPIModel struct {
	ID                      string     `json:"id,omitempty"`
	Created                 *time.Time `json:"created,omitempty"`
	Updated                 *time.Time `json:"updated,omitempty"`
	Name                    string     `json:"name"`
	ClientID                string     `json:"clientId"`
	ClientSecret            string     `json:"clientSecret,omitempty"`
	ClientType              string     `json:"clientType"`
	Hostname                string     `json:"hostname"`
	Issuer                  string     `json:"issuer,omitempty"`
	OIDCConfigURL           string     `json:"oidcConfigURL,omitempty"`
	JWKS                    string     `json:"jwks,omitempty"`
	JWKSURL                 string     `json:"jwksURL,omitempty"`
	LoginURL                string     `json:"loginURL,omitempty"`
	LogoutURL               string     `json:"logoutURL,omitempty"`
	TokenURL                string     `json:"tokenURL,omitempty"`
	UserInfoURL             string     `json:"userInfoURL,omitempty"`
	Scopes                  string     `json:"scopes,omitempty"`
	CACertificate           string     `json:"caCertificate,omitempty"`
	ConfidentialPKCEEnabled *bool      `json:"confidentialPKCEEnabled,omitempty"`
	IdTokenNonceEnabled     *bool      `json:"idTokenNonceEnabled,omitempty"`
}

func IdentityProviderResourceFactory() resource.Resource {
	return &identityProviderResource{}
}

type identityProviderResource struct {
	commonResource
}

func (r *identityProviderResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_identity_provider"
}

func (r *identityProviderResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An identity provider (OIDC client) provides OAuth 2.0 / OpenID Connect authentication for platform accounts. It can be configured with various OIDC endpoints and settings.",
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": &schema.StringAttribute{
				Required:    true,
				Description: "The name of the identity provider",
			},
			"client_id": &schema.StringAttribute{
				Required:    true,
				Description: "OAuth client identifier",
			},
			"client_secret": &schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "OAuth client secret (required for confidential clients)",
			},
			"client_type": &schema.StringAttribute{
				Required:    true,
				Description: "The OAuth client type: 'public' or 'confidential'",
			},
			"hostname": &schema.StringAttribute{
				Required:    true,
				Description: "Hostname used to generate the redirect URL",
			},
			"issuer": &schema.StringAttribute{
				Optional:    true,
				Description: "Valid issuer for this identity provider (required if oidc_config_url not provided)",
			},
			"oidc_config_url": &schema.StringAttribute{
				Optional:    true,
				Description: "OpenID Connect provider configuration URL (often .well-known/openid-configuration)",
			},
			"jwks": &schema.StringAttribute{
				Optional:    true,
				Description: "In-line JWKS object (required if oidc_config_url and jwks_url not provided)",
			},
			"jwks_url": &schema.StringAttribute{
				Optional:    true,
				Description: "URL endpoint to download the JWKS package from",
			},
			"login_url": &schema.StringAttribute{
				Optional:    true,
				Description: "Authorization endpoint for OAuth server (required if oidc_config_url not provided)",
			},
			"logout_url": &schema.StringAttribute{
				Optional:    true,
				Description: "End session endpoint for OAuth server logout",
			},
			"token_url": &schema.StringAttribute{
				Optional:    true,
				Description: "Token endpoint for OAuth server (required if oidc_config_url not provided)",
			},
			"user_info_url": &schema.StringAttribute{
				Optional:    true,
				Description: "URL endpoint to query user information using an access token",
			},
			"scopes": &schema.StringAttribute{
				Optional:    true,
				Description: "Scopes to include in IDP requests (must include 'openid email profile')",
			},
			"ca_certificate": &schema.StringAttribute{
				Optional:    true,
				Description: "Custom CA certificate for IDP requests",
			},
			"confidential_pkce_enabled": &schema.BoolAttribute{
				Optional:    true,
				Description: "Enable PKCE for confidential clients (security best practice)",
			},
			"id_token_nonce_enabled": &schema.BoolAttribute{
				Optional:    true,
				Description: "Enable nonce parameter and claim validation for ID tokens",
			},
		},
	}
}

func (data *IdentityProviderResourceModel) toAPI(api *IdentityProviderAPIModel) {
	api.Name = data.Name.ValueString()
	api.ClientID = data.ClientID.ValueString()
	if !data.ClientSecret.IsNull() {
		api.ClientSecret = data.ClientSecret.ValueString()
	}
	api.ClientType = data.ClientType.ValueString()
	api.Hostname = data.Hostname.ValueString()
	if !data.Issuer.IsNull() {
		api.Issuer = data.Issuer.ValueString()
	}
	if !data.OIDCConfigURL.IsNull() {
		api.OIDCConfigURL = data.OIDCConfigURL.ValueString()
	}
	if !data.JWKS.IsNull() {
		api.JWKS = data.JWKS.ValueString()
	}
	if !data.JWKSURL.IsNull() {
		api.JWKSURL = data.JWKSURL.ValueString()
	}
	if !data.LoginURL.IsNull() {
		api.LoginURL = data.LoginURL.ValueString()
	}
	if !data.LogoutURL.IsNull() {
		api.LogoutURL = data.LogoutURL.ValueString()
	}
	if !data.TokenURL.IsNull() {
		api.TokenURL = data.TokenURL.ValueString()
	}
	if !data.UserInfoURL.IsNull() {
		api.UserInfoURL = data.UserInfoURL.ValueString()
	}
	if !data.Scopes.IsNull() {
		api.Scopes = data.Scopes.ValueString()
	}
	if !data.CACertificate.IsNull() {
		api.CACertificate = data.CACertificate.ValueString()
	}
	if !data.ConfidentialPKCEEnabled.IsNull() {
		api.ConfidentialPKCEEnabled = data.ConfidentialPKCEEnabled.ValueBoolPointer()
	}
	if !data.IdTokenNonceEnabled.IsNull() {
		api.IdTokenNonceEnabled = data.IdTokenNonceEnabled.ValueBoolPointer()
	}
}

func (api *IdentityProviderAPIModel) toData(data *IdentityProviderResourceModel) {
	data.ID = types.StringValue(api.ID)
	data.Name = types.StringValue(api.Name)
	data.ClientID = types.StringValue(api.ClientID)
	if api.ClientSecret != "" {
		data.ClientSecret = types.StringValue(api.ClientSecret)
	}
	data.ClientType = types.StringValue(api.ClientType)
	data.Hostname = types.StringValue(api.Hostname)
	if api.Issuer != "" {
		data.Issuer = types.StringValue(api.Issuer)
	}
	if api.OIDCConfigURL != "" {
		data.OIDCConfigURL = types.StringValue(api.OIDCConfigURL)
	}
	if api.JWKS != "" {
		data.JWKS = types.StringValue(api.JWKS)
	}
	if api.JWKSURL != "" {
		data.JWKSURL = types.StringValue(api.JWKSURL)
	}
	if api.LoginURL != "" {
		data.LoginURL = types.StringValue(api.LoginURL)
	}
	if api.LogoutURL != "" {
		data.LogoutURL = types.StringValue(api.LogoutURL)
	}
	if api.TokenURL != "" {
		data.TokenURL = types.StringValue(api.TokenURL)
	}
	if api.UserInfoURL != "" {
		data.UserInfoURL = types.StringValue(api.UserInfoURL)
	}
	if api.Scopes != "" {
		data.Scopes = types.StringValue(api.Scopes)
	}
	if api.CACertificate != "" {
		data.CACertificate = types.StringValue(api.CACertificate)
	}
	if api.ConfidentialPKCEEnabled != nil {
		data.ConfidentialPKCEEnabled = types.BoolPointerValue(api.ConfidentialPKCEEnabled)
	}
	if api.IdTokenNonceEnabled != nil {
		data.IdTokenNonceEnabled = types.BoolPointerValue(api.IdTokenNonceEnabled)
	}
}

func (r *identityProviderResource) apiPath(data *IdentityProviderResourceModel) string {
	path := "/api/v1/oidc-clients"
	if data.ID.ValueString() != "" {
		path = path + "/" + data.ID.ValueString()
	}
	return path
}

func (r *identityProviderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data IdentityProviderResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	var api IdentityProviderAPIModel
	data.toAPI(&api)
	ok, _ := r.apiRequest(ctx, http.MethodPost, r.apiPath(&data), api, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	api.toData(&data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *identityProviderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data IdentityProviderResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)

	// Read full current object
	var api IdentityProviderAPIModel
	if ok, _ := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics); !ok {
		return
	}

	// Update from plan
	data.toAPI(&api)
	if ok, _ := r.apiRequest(ctx, http.MethodPatch, r.apiPath(&data), api, &api, &resp.Diagnostics); !ok {
		return
	}

	api.toData(&data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *identityProviderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data IdentityProviderResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	var api IdentityProviderAPIModel
	api.ID = data.ID.ValueString()
	ok, status := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics, Allow404())
	if !ok {
		return
	}
	if status == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	api.toData(&data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *identityProviderResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data IdentityProviderResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data), nil, nil, &resp.Diagnostics, Allow404())

	r.waitForRemoval(ctx, r.apiPath(&data), &resp.Diagnostics)
}
