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

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ApplicationResourceModel struct {
	ID           types.String                   `tfsdk:"id"`
	Name         types.String                   `tfsdk:"name"`
	OAuthEnabled types.Bool                     `tfsdk:"oauth_enabled"`
	AdminEnabled types.Bool                     `tfsdk:"admin_enabled"`
	OAuth        *ApplicationOAuthResourceModel `tfsdk:"oauth"`
}

type ApplicationAPIModel struct {
	ID          string                    `json:"id,omitempty"`
	Created     *time.Time                `json:"created,omitempty"`
	Updated     *time.Time                `json:"updated,omitempty"`
	Name        string                    `json:"name"`
	OAuth       *ApplicationOAuthAPIModel `json:"oauth,omitempty"`
	IsAdmin     *bool                     `json:"isAdmin,omitempty"`
	EnableOAuth *bool                     `json:"enableOAuth,omitempty"`
}

type ApplicationOAuthAPIModel struct {
	Issuer          string `json:"issuer,omitempty"`
	JWKSEndpoint    string `json:"jwksEndpoint,omitempty"`
	JWKS            string `json:"jwks,omitempty"`
	Audience        string `json:"aud,omitempty"`
	AuthorizedParty string `json:"azp,omitempty"`
	OIDCConfigURL   string `json:"oidcConfigURL,omitempty"`
	CACertificate   string `json:"caCertificate,omitempty"`
}

type ApplicationOAuthResourceModel struct {
	Issuer          types.String `tfsdk:"issuer"`
	JWKSEndpoint    types.String `tfsdk:"jwks_endpoint"`
	JWKS            types.String `tfsdk:"jwks"`
	Audience        types.String `tfsdk:"aud"`
	AuthorizedParty types.String `tfsdk:"azp"`
	OIDCConfigURL   types.String `tfsdk:"oidc_config_url"`
	CACertificate   types.String `tfsdk:"ca_certificate"`
}

func ApplicationResourceFactory() resource.Resource {
	return &applicationResource{}
}

type applicationResource struct {
	commonResource
}

func (r *applicationResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_application"
}

func (r *applicationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An application provides configurable access control, authentication, and authorization for external systems and integrations leveraging the Kaleido Platform APIs. Applications are granted access separately from users and groups via service, stack, and fine-grained policies. There are two mechanisms for authenticating applications: using an API key, or via an OIDC provider.",
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": &schema.StringAttribute{
				Required: true,
			},
			"admin_enabled": &schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Grant the application the ability to act as an administrator of the platform",
			},
			"oauth_enabled": &schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Default:       booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplaceIfConfigured()},
				Description:   "Default true. An Identity Provider can be bound to an application to allow it to federate its own OAuth 2.0 authentication realm into the APIs of the platform.",
			},
			"oauth": &schema.SingleNestedAttribute{
				Optional: true,
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"aud": &schema.StringAttribute{
						Optional:      true,
						PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
					},
					"azp": &schema.StringAttribute{
						Optional:      true,
						PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
					},
					"ca_certificate": &schema.StringAttribute{
						Optional:      true,
						PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
					},
					"issuer": &schema.StringAttribute{
						Optional:      true,
						Computed:      true,
						PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
					},
					"jwks": &schema.StringAttribute{
						Optional:      true,
						PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
					},
					"jwks_endpoint": &schema.StringAttribute{
						Optional:      true,
						Computed:      true,
						PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
					},
					"oidc_config_url": &schema.StringAttribute{
						Optional:      true,
						PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
					},
				},
				Default: objectdefault.StaticValue(
					types.ObjectNull(
						map[string]attr.Type{
							"aud":             types.StringType,
							"azp":             types.StringType,
							"ca_certificate":  types.StringType,
							"issuer":          types.StringType,
							"jwks":            types.StringType,
							"jwks_endpoint":   types.StringType,
							"oidc_config_url": types.StringType,
						},
					),
				),
			},
		},
	}
}

func (data *ApplicationResourceModel) toAPI(api *ApplicationAPIModel) {
	api.Name = data.Name.ValueString()
	api.IsAdmin = data.AdminEnabled.ValueBoolPointer()
	api.EnableOAuth = data.OAuthEnabled.ValueBoolPointer()
	if data.OAuth != nil {
		api.OAuth = &ApplicationOAuthAPIModel{}
		if !data.OAuth.Issuer.IsNull() {
			api.OAuth.Issuer = data.OAuth.Issuer.ValueString()
		}
		if !data.OAuth.JWKSEndpoint.IsNull() {
			api.OAuth.JWKSEndpoint = data.OAuth.JWKSEndpoint.ValueString()
		}
		if !data.OAuth.JWKS.IsNull() {
			api.OAuth.JWKS = data.OAuth.JWKS.ValueString()
		}
		if !data.OAuth.AuthorizedParty.IsNull() {
			api.OAuth.AuthorizedParty = data.OAuth.AuthorizedParty.ValueString()
		}
		if !data.OAuth.CACertificate.IsNull() {
			api.OAuth.CACertificate = data.OAuth.CACertificate.ValueString()
		}
		if !data.OAuth.Audience.IsNull() {
			api.OAuth.Audience = data.OAuth.Audience.ValueString()
		}
		if !data.OAuth.OIDCConfigURL.IsNull() {
			api.OAuth.OIDCConfigURL = data.OAuth.OIDCConfigURL.ValueString()
		}
	}

}

func (api *ApplicationAPIModel) toData(data *ApplicationResourceModel) {
	data.ID = types.StringValue(api.ID)
	data.Name = types.StringValue(api.Name)
	data.AdminEnabled = types.BoolPointerValue(api.IsAdmin)
	if api.EnableOAuth != nil {
		data.OAuthEnabled = types.BoolPointerValue(api.EnableOAuth)
	}
	if api.OAuth != nil {
		data.OAuth = &ApplicationOAuthResourceModel{}
		if api.OAuth.Issuer != "" {
			data.OAuth.Issuer = types.StringValue(api.OAuth.Issuer)
		}
		if api.OAuth.JWKSEndpoint != "" {
			data.OAuth.JWKSEndpoint = types.StringValue(api.OAuth.JWKSEndpoint)
		}
		if api.OAuth.JWKS != "" {
			data.OAuth.JWKS = types.StringValue(api.OAuth.JWKS)
		}
		if api.OAuth.AuthorizedParty != "" {
			data.OAuth.AuthorizedParty = types.StringValue(api.OAuth.AuthorizedParty)
		}
		if api.OAuth.CACertificate != "" {
			data.OAuth.CACertificate = types.StringValue(api.OAuth.CACertificate)
		}
		if api.OAuth.Audience != "" {
			data.OAuth.Audience = types.StringValue(api.OAuth.Audience)
		}
		if api.OAuth.OIDCConfigURL != "" {
			data.OAuth.OIDCConfigURL = types.StringValue(api.OAuth.OIDCConfigURL)
		}
	}
}

func (r *applicationResource) apiPath(data *ApplicationResourceModel) string {
	path := "/api/v1/applications"
	if data.ID.ValueString() != "" {
		path = path + "/" + data.ID.ValueString()
	}
	return path
}

func (r *applicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var data ApplicationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	var api ApplicationAPIModel
	data.toAPI(&api)
	ok, _ := r.apiRequest(ctx, http.MethodPost, r.apiPath(&data), api, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	api.toData(&data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)

}

func (r *applicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

	var data ApplicationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)

	// Read full current object
	var api ApplicationAPIModel
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

func (r *applicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ApplicationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	var api ApplicationAPIModel
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

func (r *applicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ApplicationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data), nil, nil, &resp.Diagnostics, Allow404())

	r.waitForRemoval(ctx, r.apiPath(&data), &resp.Diagnostics)
}
