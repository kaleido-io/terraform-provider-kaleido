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
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type UserResourceModel struct {
	ID      types.String `tfsdk:"id"`
	Name    types.String `tfsdk:"name"`
	Account types.String `tfsdk:"account"`
	Email   types.String `tfsdk:"email"`
	Sub     types.String `tfsdk:"sub"`
	IsAdmin types.Bool   `tfsdk:"is_admin"`
}

type UserAPIModel struct {
	ID      string     `json:"id,omitempty"`
	Created *time.Time `json:"created,omitempty"`
	Updated *time.Time `json:"updated,omitempty"`
	Account string     `json:"account,omitempty"`
	Name    string     `json:"name,omitempty"`
	Email   string     `json:"email,omitempty"`
	Sub     string     `json:"sub,omitempty"`
	IsAdmin *bool      `json:"isAdmin,omitempty"`
}

// User create/update request body
type UserCreateUpdateRequestModel struct {
	Name    string `json:"name,omitempty"`
	Email   string `json:"email,omitempty"`
	Sub     string `json:"sub,omitempty"`
	IsAdmin *bool  `json:"isAdmin,omitempty"`
}

func UserResourceFactory() resource.Resource {
	return &userResource{}
}

type userResource struct {
	commonResource
}

func (r *userResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_user"
}

func (r *userResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A platform user represents an individual who can authenticate and access resources within an account. Users are scoped to a specific account and can be assigned to groups.",
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"account": &schema.StringAttribute{
				Computed:    true,
				Description: "ID of the account this user belongs to",
			},
			"name": &schema.StringAttribute{
				Required:    true,
				Description: "The username",
			},
			"email": &schema.StringAttribute{
				Optional:    true,
				Description: "Email address of the user",
			},
			"sub": &schema.StringAttribute{
				Optional:    true,
				Description: "OAuth subject identifier of the user",
			},
			"is_admin": &schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether the user is a platform administrator",
			},
		},
	}
}

func (data *UserResourceModel) toCreateUpdateRequest(req *UserCreateUpdateRequestModel) {
	req.Name = data.Name.ValueString()
	if !data.Email.IsNull() {
		req.Email = data.Email.ValueString()
	}
	if !data.Sub.IsNull() {
		req.Sub = data.Sub.ValueString()
	}
	if !data.IsAdmin.IsNull() {
		req.IsAdmin = data.IsAdmin.ValueBoolPointer()
	}
}

func (api *UserAPIModel) toData(data *UserResourceModel) {
	data.ID = types.StringValue(api.ID)
	data.Account = types.StringValue(api.Account)
	data.Name = types.StringValue(api.Name)
	if api.Email != "" {
		data.Email = types.StringValue(api.Email)
	}
	if api.Sub != "" {
		data.Sub = types.StringValue(api.Sub)
	}
	if api.IsAdmin != nil {
		data.IsAdmin = types.BoolPointerValue(api.IsAdmin)
	}
}

func (r *userResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data UserResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	// Create user using POST /users
	var userReq UserCreateUpdateRequestModel
	data.toCreateUpdateRequest(&userReq)

	var api UserAPIModel
	ok, _ := r.apiRequest(ctx, http.MethodPost, "/api/v1/users", userReq, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	// Update the data model with the response
	api.toData(&data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *userResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data UserResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)

	// Update user using PATCH /users/{userId}
	var userReq UserCreateUpdateRequestModel
	data.toCreateUpdateRequest(&userReq)

	var api UserAPIModel
	ok, _ := r.apiRequest(ctx, http.MethodPatch, fmt.Sprintf("/api/v1/users/%s", data.ID.ValueString()), userReq, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	// Update the data model with the response
	api.toData(&data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *userResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data UserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	// Read user using GET /users/{userNameOrId}
	var api UserAPIModel
	ok, status := r.apiRequest(ctx, http.MethodGet, fmt.Sprintf("/api/v1/users/%s", data.ID.ValueString()), nil, &api, &resp.Diagnostics, Allow404())
	if !ok {
		return
	}
	if status == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	// Update the data model with the response
	api.toData(&data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *userResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data UserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	// Delete user using DELETE /users/{userNameOrId}
	ok, _ := r.apiRequest(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/users/%s", data.ID.ValueString()), nil, nil, &resp.Diagnostics, Allow404())
	if !ok {
		return
	}
}
