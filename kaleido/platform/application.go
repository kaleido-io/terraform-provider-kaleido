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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ApplicationResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	OIDCConfigURL types.String `tfsdk:"oidc_config_url"`
	OAuthEnabled  types.Bool   `tfsdk:"oauth_enabled"`
	AdminEnabled  types.Bool   `tfsdk:"admin_enabled"`
}

type ApplicationAPIModel struct {
	ID          string            `json:"id,omitempty"`
	Created     *time.Time        `json:"created,omitempty"`
	Updated     *time.Time        `json:"updated,omitempty"`
	Name        string            `json:"name"`
	OAuth       map[string]string `json:"oauth,omitempty"`
	IsAdmin     *bool             `json:"isAdmin,omitempty"`
	EnableOAuth *bool             `json:"enableOAuth,omitempty"`
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
		Description: "An application provides authentication and access control to external applications & integrations connecting to the Kaleido platform using APIs. Applications are granted access separately to users and groups. There are two mechanisms for authenticating applications, using API keys or by identity provider.",
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
			"oidc_config_url": &schema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},
	}
}

func (data *ApplicationResourceModel) toAPI(api *ApplicationAPIModel) {
	api.Name = data.Name.ValueString()
	api.IsAdmin = data.AdminEnabled.ValueBoolPointer()
	api.EnableOAuth = data.OAuthEnabled.ValueBoolPointer()
	api.OAuth = make(map[string]string)
	if !data.OIDCConfigURL.IsNull() {
		api.OAuth["oidcConfigURL"] = data.OIDCConfigURL.ValueString()
	}
}

func (api *ApplicationAPIModel) toData(data *ApplicationResourceModel) {
	data.ID = types.StringValue(api.ID)
	data.AdminEnabled = types.BoolPointerValue(api.IsAdmin)
	if api.EnableOAuth != nil {
		data.OAuthEnabled = types.BoolPointerValue(api.EnableOAuth)
	}
	if api.OAuth != nil && api.OAuth["oidcConfigURL"] != "" {
		data.OIDCConfigURL = types.StringValue(api.OAuth["oidcConfigURL"])
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
