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
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type AMSDMListenerResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Environment types.String `tfsdk:"environment"`
	Service     types.String `tfsdk:"service"`
	Name        types.String `tfsdk:"name"`
	TaskID      types.String `tfsdk:"task_id"`
	TaskName    types.String `tfsdk:"task_name"`
	TaskVersion types.String `tfsdk:"task_version"`
	TopicFilter types.String `tfsdk:"topic_filter"`
}

type AMSDMListenerAPIModel struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	Created     string `json:"created,omitempty"`
	Updated     string `json:"updated,omitempty"`
	TaskID      string `json:"taskId,omitempty"`
	TaskName    string `json:"taskName,omitempty"`
	TaskVersion string `json:"taskVersion,omitempty"`
	TopicFilter string `tfsdk:"topicFilter"`
}

func AMSDMListenerResourceFactory() resource.Resource {
	return &ams_dmlistenerResource{}
}

type ams_dmlistenerResource struct {
	commonResource
}

func (r *ams_dmlistenerResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_ams_dmlistener"
}

func (r *ams_dmlistenerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"environment": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"service": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": &schema.StringAttribute{
				Required: true,
			},
			"task_id": &schema.StringAttribute{
				Optional: true,
			},
			"task_name": &schema.StringAttribute{
				Optional: true,
			},
			"task_version": &schema.StringAttribute{
				Optional: true,
			},
			"topic_filter": &schema.StringAttribute{
				Optional: true,
			},
		},
	}
}

func (data *AMSDMListenerResourceModel) toAPI(api *AMSDMListenerAPIModel, diagnostics *diag.Diagnostics) bool {
	api.Name = data.Name.ValueString()
	if !data.TaskID.IsNull() {
		api.TaskID = data.TaskID.ValueString()
	}
	if !data.TaskName.IsNull() {
		api.TaskName = data.TaskName.ValueString()
	}
	if !data.TaskVersion.IsNull() {
		api.TaskVersion = data.TaskVersion.ValueString()
	}
	if !data.TopicFilter.IsNull() {
		api.TopicFilter = data.TopicFilter.ValueString()
	}
	return true
}

func (api *AMSDMListenerAPIModel) toData(data *AMSDMListenerResourceModel) {
	data.ID = types.StringValue(api.ID)
	data.TaskID = types.StringValue(api.TaskID)
	data.TaskName = types.StringValue(api.TaskName)
	data.TaskVersion = types.StringValue(api.TaskVersion)
	data.TopicFilter = types.StringValue(api.TopicFilter)
}

func (r *ams_dmlistenerResource) apiPath(data *AMSDMListenerResourceModel, idOrName string) string {
	return fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/listeners/datamodel/%s", data.Environment.ValueString(), data.Service.ValueString(), idOrName)
}

func (r *ams_dmlistenerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var data AMSDMListenerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	var api AMSDMListenerAPIModel
	ok := data.toAPI(&api, &resp.Diagnostics)
	if ok {
		ok, _ = r.apiRequest(ctx, http.MethodPut, r.apiPath(&data, data.Name.ValueString()), &api, &api, &resp.Diagnostics)
	}
	if !ok {
		return
	}

	api.toData(&data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)

}

func (r *ams_dmlistenerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

	var data AMSDMListenerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)

	var api AMSDMListenerAPIModel
	ok := data.toAPI(&api, &resp.Diagnostics)
	if ok {
		ok, _ = r.apiRequest(ctx, http.MethodPut, r.apiPath(&data, data.ID.ValueString()), &api, &api, &resp.Diagnostics)
	}
	if !ok {
		return
	}

	api.toData(&data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *ams_dmlistenerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AMSDMListenerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	var api AMSDMListenerAPIModel
	api.ID = data.ID.ValueString()
	ok, status := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data, data.ID.ValueString()), nil, &api, &resp.Diagnostics, Allow404())
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

func (r *ams_dmlistenerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AMSDMListenerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data, data.ID.ValueString()), nil, nil, &resp.Diagnostics, Allow404())

	r.waitForRemoval(ctx, r.apiPath(&data, data.ID.ValueString()), &resp.Diagnostics)
}
