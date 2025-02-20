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

type StackAccessResourceModel struct {
	ID            types.String `tfsdk:"id"`
	GroupID       types.String `tfsdk:"group_id"`
	StackID       types.String `tfsdk:"stack_id"`
	ApplicationID types.String `tfsdk:"application_id"`
}

type StackAccessAPIModel struct {
	ID            string     `json:"id,omitempty"`
	GroupID       string     `json:"groupId,omitempty"`
	Created       *time.Time `json:"created,omitempty"`
	Updated       *time.Time `json:"updated,omitempty"`
	StackID       string     `json:"stackId,omitempty"`
	ApplicationID string     `json:"applicationId,omitempty"`
}

func StackAccessResourceFactory() resource.Resource {
	return &stackAccessResource{}
}

type stackAccessResource struct {
	commonResource
}

func (r *stackAccessResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_stack_access"
}

func (r *stackAccessResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"group_id": &schema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"service_id": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"application_id": &schema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},
	}
}

func (data *StackAccessResourceModel) toAPI(api *StackAccessAPIModel) {
	if !data.GroupID.IsNull() {
		api.GroupID = data.GroupID.ValueString()
	}
	if !data.ApplicationID.IsNull() {
		api.ApplicationID = data.ApplicationID.ValueString()
	}
	api.StackID = data.StackID.ValueString()
}

func (api *StackAccessAPIModel) toData(data *StackAccessResourceModel) {
	data.ID = types.StringValue(api.ID)
	if api.GroupID != "" {
		data.GroupID = types.StringValue(api.GroupID)
	}
	if api.ApplicationID != "" {
		data.ApplicationID = types.StringValue(api.ApplicationID)
	}
	data.StackID = types.StringValue(api.StackID)
}

func (r *stackAccessResource) apiPath(data *StackAccessResourceModel) string {
	path := fmt.Sprintf("/api/v1/stack-access/%s/permissions", data.StackID.ValueString())
	if data.ID.ValueString() != "" {
		path = path + "/" + data.ID.ValueString()
	}
	return path
}

func (r *stackAccessResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var data StackAccessResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	var api StackAccessAPIModel
	data.toAPI(&api)
	ok, _ := r.apiRequest(ctx, http.MethodPost, r.apiPath(&data), api, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	api.toData(&data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)

}

func (r *stackAccessResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data StackAccessResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)

	// Read full current object
	var api StackAccessAPIModel
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

func (r *stackAccessResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data StackAccessResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	var api StackAccessAPIModel
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

func (r *stackAccessResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data StackAccessResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data), nil, nil, &resp.Diagnostics, Allow404())

	r.waitForRemoval(ctx, r.apiPath(&data), &resp.Diagnostics)
}
