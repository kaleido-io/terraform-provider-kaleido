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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type GroupMembershipResourceModel struct {
	ID        types.String `tfsdk:"id"`
	GroupID   types.String `tfsdk:"group_id"`
	UserID    types.String `tfsdk:"user_id"`
	GroupName types.String `tfsdk:"group_name"`
	UserName  types.String `tfsdk:"user_name"`
	AccountID types.String `tfsdk:"account_id"`
}

type GroupMembershipAPIModel struct {
	ID        string     `json:"id,omitempty"`
	Created   *time.Time `json:"created,omitempty"`
	Updated   *time.Time `json:"updated,omitempty"`
	GroupID   string     `json:"groupId,omitempty"`
	UserID    string     `json:"userId,omitempty"`
	GroupName string     `json:"groupName,omitempty"`
	UserName  string     `json:"userName,omitempty"`
	Account   string     `json:"account,omitempty"`
}

type GroupMembershipCreateAPIModel struct {
	UserID string `json:"userId"`
}

func GroupMembershipResourceFactory() resource.Resource {
	return &groupMembershipResource{}
}

type groupMembershipResource struct {
	commonResource
}

func (r *groupMembershipResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_group_membership"
}

func (r *groupMembershipResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A group membership represents the association between a user and a group. This resource manages the assignment of users to groups for access control purposes.",
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"group_id": &schema.StringAttribute{
				Required:    true,
				Description: "ID of the group",
			},
			"user_id": &schema.StringAttribute{
				Required:    true,
				Description: "ID of the user to add to the group",
			},
			"group_name": &schema.StringAttribute{
				Computed:    true,
				Description: "Name of the group",
			},
			"user_name": &schema.StringAttribute{
				Computed:    true,
				Description: "Name of the user",
			},
			"account_id": &schema.StringAttribute{
				Computed:    true,
				Description: "ID of the account this membership belongs to",
			},
		},
	}
}

func (data *GroupMembershipResourceModel) toAPI(api *GroupMembershipCreateAPIModel) {
	api.UserID = data.UserID.ValueString()
}

func (api *GroupMembershipAPIModel) toData(data *GroupMembershipResourceModel) {
	data.ID = types.StringValue(api.ID)
	data.GroupID = types.StringValue(api.GroupID)
	data.UserID = types.StringValue(api.UserID)
	data.GroupName = types.StringValue(api.GroupName)
	data.UserName = types.StringValue(api.UserName)
	data.AccountID = types.StringValue(api.Account)
}

func (r *groupMembershipResource) apiPath(data *GroupMembershipResourceModel) string {
	return fmt.Sprintf("/groups/%s/members", data.GroupID.ValueString())
}

func (r *groupMembershipResource) memberPath(data *GroupMembershipResourceModel) string {
	if data.ID.ValueString() != "" {
		return fmt.Sprintf("/groups/%s/members/%s", data.GroupID.ValueString(), data.ID.ValueString())
	}
	return r.apiPath(data)
}

func (r *groupMembershipResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data GroupMembershipResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	var createRequest GroupMembershipCreateAPIModel
	data.toAPI(&createRequest)

	var api GroupMembershipAPIModel
	ok, _ := r.apiRequest(ctx, http.MethodPost, r.apiPath(&data), createRequest, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	api.toData(&data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *groupMembershipResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data GroupMembershipResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)

	// Group memberships are typically not updated, but we'll return the current state
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *groupMembershipResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data GroupMembershipResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	// Read the specific membership
	var api GroupMembershipAPIModel
	ok, status := r.apiRequest(ctx, http.MethodGet, r.memberPath(&data), nil, &api, &resp.Diagnostics, Allow404())
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

func (r *groupMembershipResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data GroupMembershipResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.memberPath(&data), nil, nil, &resp.Diagnostics, Allow404())

	r.waitForRemoval(ctx, r.memberPath(&data), &resp.Diagnostics)
}
