// Copyright © Kaleido, Inc. 2024

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

type AccountAccessPolicyResourceModel struct {
	ID            types.String `tfsdk:"id"`
	GroupID       types.String `tfsdk:"group_id"`
	ApplicationID types.String `tfsdk:"application_id"`
	Policy        types.String `tfsdk:"policy"`
}

type AccountAccessPolicyAPIModel struct {
	ID            string     `json:"id,omitempty"`
	GroupID       string     `json:"groupId,omitempty"`
	Created       *time.Time `json:"created,omitempty"`
	Updated       *time.Time `json:"updated,omitempty"`
	ApplicationID string     `json:"applicationId,omitempty"`
	Policy        string     `json:"policy,omitempty"`
}

func AccountAccessPolicyResourceFactory() resource.Resource {
	return &accountAccessPolicyResource{}
}

type accountAccessPolicyResource struct {
	commonResource
}

func (r *accountAccessPolicyResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_account_access_policy"
}

func (r *accountAccessPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Grant a User Group or Application access to account resources.",
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"group_id": &schema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "User Group ID. Specify either group_id or application_id",
			},
			"application_id": &schema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Application ID. Specify either group_id or application_id",
			},
			"policy": &schema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Policy. rego policy document",
			},
		},
	}
}

func (data *AccountAccessPolicyResourceModel) toAPI(api *AccountAccessPolicyAPIModel) {
	if !data.GroupID.IsNull() {
		api.GroupID = data.GroupID.ValueString()
	}
	if !data.ApplicationID.IsNull() {
		api.ApplicationID = data.ApplicationID.ValueString()
	}
	api.Policy = data.Policy.ValueString()
}

func (api *AccountAccessPolicyAPIModel) toData(data *AccountAccessPolicyResourceModel) {
	data.ID = types.StringValue(api.ID)
	if api.GroupID != "" {
		data.GroupID = types.StringValue(api.GroupID)
	}
	if api.ApplicationID != "" {
		data.ApplicationID = types.StringValue(api.ApplicationID)
	}
	data.Policy = types.StringValue(api.Policy)
}

func (r *accountAccessPolicyResource) apiPath(data *AccountAccessPolicyResourceModel) string {
	path := "/api/v1/account-access/policies"
	if data.ID.ValueString() != "" {
		path = path + "/" + data.ID.ValueString()
	}
	return path
}

func (r *accountAccessPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var data AccountAccessPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	var api AccountAccessPolicyAPIModel
	data.toAPI(&api)
	ok, _ := r.apiRequest(ctx, http.MethodPost, r.apiPath(&data), api, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	api.toData(&data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)

}

func (r *accountAccessPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data AccountAccessPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)

	// Read full current object
	var api AccountAccessPolicyAPIModel
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

func (r *accountAccessPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AccountAccessPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	var api AccountAccessPolicyAPIModel
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

func (r *accountAccessPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AccountAccessPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data), nil, nil, &resp.Diagnostics, Allow404())

	r.waitForRemoval(ctx, r.apiPath(&data), &resp.Diagnostics)
}
