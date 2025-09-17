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
	"encoding/json"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type AccountResourceModel struct {
	ID                                   types.String `tfsdk:"id"`
	Name                                 types.String `tfsdk:"name"`
	OIDCClientID                         types.String `tfsdk:"oidc_client_id"`
	ValidationPolicy                     types.String `tfsdk:"validation_policy"`
	FirstUserEmail                       types.String `tfsdk:"first_user_email"`
	FirstUserSub                         types.String `tfsdk:"first_user_sub"`
	Hostnames                            types.Map    `tfsdk:"hostnames"`
	UserJITEnabled                       types.Bool   `tfsdk:"user_jit_enabled"`
	UserJITDefaultGroup                  types.String `tfsdk:"user_jit_default_group"`
	BootstrapApplicationName             types.String `tfsdk:"bootstrap_application_name"`
	BootstrapApplicationOAuthJSON        types.String `tfsdk:"bootstrap_application_oauth_json"`
	BootstrapApplicationValidationPolicy types.String `tfsdk:"bootstrap_application_validation_policy"`
}

type AccountAPIModel struct {
	ID                   string                               `json:"id,omitempty"`
	Created              *time.Time                           `json:"created,omitempty"`
	Updated              *time.Time                           `json:"updated,omitempty"`
	Name                 string                               `json:"name"`
	OIDCClientID         string                               `json:"oidcClientId,omitempty"`
	ValidationPolicy     string                               `json:"validationPolicy,omitempty"`
	FirstUserEmail       string                               `json:"firstUserEmail,omitempty"`
	FirstUserSub         string                               `json:"firstUserSub,omitempty"`
	Hostnames            map[string][]string                  `json:"hostnames,omitempty"`
	UserJITEnabled       *bool                                `json:"userJITEnabled,omitempty"`
	UserJITDefaultGroup  *string                              `json:"userJITDefaultGroup,omitempty"`
	BootstrapApplication *AccountBootstrapApplicationAPIModel `json:"bootstrapApplication,omitempty"`
}

type AccountBootstrapApplicationAPIModel struct {
	Name             string            `json:"name"`
	OAuth            map[string]string `json:"oauth,omitempty"`
	ValidationPolicy string            `json:"validationPolicy,omitempty"`
}

func AccountResourceFactory() resource.Resource {
	return &accountResource{}
}

type accountResource struct {
	commonResource
}

var (
	_ resource.Resource                = &accountResource{}
	_ resource.ResourceWithConfigure   = &accountResource{}
	_ resource.ResourceWithImportState = &accountResource{}
)

func (r *accountResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_account"
}

func (r *accountResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{

		Description: "A platform account represents a tenant or child account that can be managed independently. Accounts provide isolation and can have their own OIDC configuration, users, and groups.",
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": &schema.StringAttribute{
				Required:    true,
				Description: "Unique name of the account",
			},
			"oidc_client_id": &schema.StringAttribute{
				Optional:    true,
				Description: "ID of an existing OIDC Client to use for authentication",
			},
			"validation_policy": &schema.StringAttribute{
				Optional:    true,
				Description: "Optional policy for validating the account identity",
			},
			"first_user_email": &schema.StringAttribute{
				Optional:    true,
				Description: "Email address of the initial admin user for the new account",
			},
			"first_user_sub": &schema.StringAttribute{
				Optional:    true,
				Description: "OIDC subject identifier of the initial admin user for the new account",
			},
			"hostnames": &schema.MapAttribute{
				ElementType: types.ListType{ElemType: types.StringType},
				Optional:    true,
				Description: "Hostname binding map for the account",
			},
			"user_jit_enabled": &schema.BoolAttribute{
				Optional:    true,
				Description: "Enable Just-In-Time (JIT) user provisioning for the account, requires a validation policy to also be provided.",
			},
			"user_jit_default_group": &schema.StringAttribute{
				Optional:    true,
				Description: "Default group to add users to when JIT is enabled",
			},
			"bootstrap_application_name": &schema.StringAttribute{
				Optional:    true,
				Description: "Name of the bootstrap application for the account",
			},
			"bootstrap_application_oauth_json": &schema.StringAttribute{
				Optional:    true,
				Description: "OAuth configuration for the bootstrap application",
			},
			"bootstrap_application_validation_policy": &schema.StringAttribute{
				Optional:    true,
				Description: "Validation policy for the bootstrap application",
			},
		},
	}
}

func (data *AccountResourceModel) toAPI(api *AccountAPIModel) {
	api.Name = data.Name.ValueString()
	if !data.OIDCClientID.IsNull() {
		api.OIDCClientID = data.OIDCClientID.ValueString()
	}
	if !data.ValidationPolicy.IsNull() {
		api.ValidationPolicy = data.ValidationPolicy.ValueString()
	}
	if !data.FirstUserEmail.IsNull() {
		api.FirstUserEmail = data.FirstUserEmail.ValueString()
	}
	if !data.FirstUserSub.IsNull() {
		api.FirstUserSub = data.FirstUserSub.ValueString()
	}
	if !data.Hostnames.IsNull() {
		hostnamesMap := make(map[string][]string)
		data.Hostnames.ElementsAs(context.Background(), &hostnamesMap, false)
		api.Hostnames = hostnamesMap
	}
	if !data.UserJITEnabled.IsNull() {
		api.UserJITEnabled = data.UserJITEnabled.ValueBoolPointer()
	}
	if !data.UserJITDefaultGroup.IsNull() {
		api.UserJITDefaultGroup = data.UserJITDefaultGroup.ValueStringPointer()
	}
	if !data.BootstrapApplicationName.IsNull() && !data.BootstrapApplicationOAuthJSON.IsNull() {
		api.BootstrapApplication = &AccountBootstrapApplicationAPIModel{
			Name:             data.BootstrapApplicationName.ValueString(),
			OAuth:            make(map[string]string),
			ValidationPolicy: data.BootstrapApplicationValidationPolicy.ValueString(),
		}

		if err := json.Unmarshal([]byte(data.BootstrapApplicationOAuthJSON.ValueString()), &api.BootstrapApplication.OAuth); err != nil {
			api.BootstrapApplication.OAuth = make(map[string]string)
			// log warning ??
		}

		if !data.BootstrapApplicationValidationPolicy.IsNull() {
			api.BootstrapApplication.ValidationPolicy = data.BootstrapApplicationValidationPolicy.ValueString()
		} else {
			api.BootstrapApplication.ValidationPolicy = ""
		}
	}

}

func (api *AccountAPIModel) toData(data *AccountResourceModel) {
	data.ID = types.StringValue(api.ID)
	data.Name = types.StringValue(api.Name)
	if api.OIDCClientID != "" {
		data.OIDCClientID = types.StringValue(api.OIDCClientID)
	}
	if api.ValidationPolicy != "" {
		data.ValidationPolicy = types.StringValue(api.ValidationPolicy)
	}
	if api.FirstUserEmail != "" {
		data.FirstUserEmail = types.StringValue(api.FirstUserEmail)
	}
	if api.FirstUserSub != "" {
		data.FirstUserSub = types.StringValue(api.FirstUserSub)
	}
	if len(api.Hostnames) > 0 {
		hostnames, _ := types.MapValueFrom(context.Background(), types.ListType{ElemType: types.StringType}, api.Hostnames)
		data.Hostnames = hostnames
	}
	if api.UserJITEnabled != nil {
		data.UserJITEnabled = types.BoolValue(*api.UserJITEnabled)
	}
	if api.UserJITDefaultGroup != nil {
		data.UserJITDefaultGroup = types.StringValue(*api.UserJITDefaultGroup)
	}
	if api.BootstrapApplication != nil {
		data.BootstrapApplicationName = types.StringValue(api.BootstrapApplication.Name)
		data.BootstrapApplicationValidationPolicy = types.StringValue(api.BootstrapApplication.ValidationPolicy)
	}
}

func (r *accountResource) apiPath(data *AccountResourceModel) string {
	path := "/api/v1/accounts"
	if data.ID.ValueString() != "" {
		path = path + "/" + data.ID.ValueString()
	}
	return path
}

func (r *accountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AccountResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	var api AccountAPIModel
	data.toAPI(&api)
	ok, _ := r.apiRequest(ctx, http.MethodPost, r.apiPath(&data), api, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	api.toData(&data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *accountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data AccountResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)

	// Read full current object
	var api AccountAPIModel
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

func (r *accountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AccountResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	var api AccountAPIModel
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

func (r *accountResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AccountResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data), nil, nil, &resp.Diagnostics, Allow404())

	r.waitForRemoval(ctx, r.apiPath(&data), &resp.Diagnostics)
}
