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
	"encoding/json"
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

type RuntimeResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Environment         types.String `tfsdk:"environment"`
	Type                types.String `tfsdk:"type"`
	Name                types.String `tfsdk:"name"`
	ConfigJSON          types.String `tfsdk:"config_json"`
	LogLevel            types.String `tfsdk:"log_level"`
	Size                types.String `tfsdk:"size"`
	EnvironmentMemberID types.String `tfsdk:"environment_member_id"`
	Stopped             types.Bool   `tfsdk:"stopped"`
}

type RuntimeAPIModel struct {
	ID                  string                 `json:"id,omitempty"`
	Created             *time.Time             `json:"created,omitempty"`
	Updated             *time.Time             `json:"updated,omitempty"`
	Type                string                 `json:"type"`
	Name                string                 `json:"name"`
	Config              map[string]interface{} `json:"config"`
	LogLevel            string                 `json:"loglevel,omitempty"`
	Size                string                 `json:"size,omitempty"`
	EnvironmentMemberID string                 `json:"environmentMemberId,omitempty"`
	Status              string                 `json:"status,omitempty"`
	Deleted             bool                   `json:"deleted,omitempty"`
	Stopped             bool                   `json:"stopped,omitempty"`
}

func RuntimeResourceFactory() resource.Resource {
	return &runtimeResource{}
}

type runtimeResource struct {
	commonResource
}

func (r *runtimeResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_runtime"
}

func (r *runtimeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"type": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": &schema.StringAttribute{
				Required: true,
			},
			"environment": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"environment_member_id": &schema.StringAttribute{
				Computed: true,
			},
			"config_json": &schema.StringAttribute{
				Required: true,
			},
			"log_level": &schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"size": &schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"stopped": &schema.BoolAttribute{
				Optional: true,
				Computed: true,
			},
		},
	}
}

func (data *RuntimeResourceModel) toAPI(api *RuntimeAPIModel) {
	// required fields
	api.Type = data.Type.ValueString()
	api.Name = data.Name.ValueString()
	// optional fields
	api.Config = map[string]interface{}{}
	if !data.ConfigJSON.IsNull() {
		_ = json.Unmarshal([]byte(data.ConfigJSON.ValueString()), &api.Config)
	}
	if !data.LogLevel.IsNull() {
		api.LogLevel = data.LogLevel.ValueString()
	}
	if !data.Size.IsNull() {
		api.Size = data.Size.ValueString()
	}
	if !data.Stopped.IsNull() {
		api.Stopped = data.Stopped.ValueBool()
	}
}

func (api *RuntimeAPIModel) toData(data *RuntimeResourceModel) {
	data.ID = types.StringValue(api.ID)
	data.EnvironmentMemberID = types.StringValue(api.EnvironmentMemberID)
	data.LogLevel = types.StringValue(api.LogLevel)
	data.Size = types.StringValue(api.Size)
	data.Stopped = types.BoolValue(api.Stopped)
}

func (r *runtimeResource) apiPath(data *RuntimeResourceModel) string {
	path := fmt.Sprintf("/api/v1/environments/%s/runtimes", data.Environment.ValueString())
	if data.ID.ValueString() != "" {
		path = path + "/" + data.ID.ValueString()
	}
	return path
}

func (r *runtimeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var data RuntimeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	var api RuntimeAPIModel
	data.toAPI(&api)
	ok, _ := r.apiRequest(ctx, http.MethodPost, r.apiPath(&data), api, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	api.toData(&data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)

}

func (r *runtimeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

	var data RuntimeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)

	// Read full current object
	var api RuntimeAPIModel
	if ok, _ := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics); !ok {
		return
	}

	// Update from plan
	data.toAPI(&api)
	if ok, _ := r.apiRequest(ctx, http.MethodPut, r.apiPath(&data), api, &api, &resp.Diagnostics); !ok {
		return
	}

	api.toData(&data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *runtimeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RuntimeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	var api RuntimeAPIModel
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

func (r *runtimeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data RuntimeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data), nil, nil, &resp.Diagnostics, Allow404())

	r.waitForRemoval(ctx, r.apiPath(&data), &resp.Diagnostics)
}