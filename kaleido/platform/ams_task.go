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
	"gopkg.in/yaml.v3"
)

type AMSTaskResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	Environment    types.String `tfsdk:"environment"`
	Service        types.String `tfsdk:"service"`
	TaskYAML       types.String `tfsdk:"task_yaml"` // this is propagated to a task version
	AppliedVersion types.String `tfsdk:"applied_version"`
}

type AMSTaskAPIModel struct {
	ID             string `json:"id,omitempty"`
	Name           string `json:"name,omitempty"`
	Description    string `json:"description,omitempty"`
	Created        string `json:"created,omitempty"`
	Updated        string `json:"updated,omitempty"`
	CurrentVersion string `json:"currentVersion,omitempty"`
}

func AMSTaskResourceFactory() resource.Resource {
	return &ams_taskResource{}
}

type ams_taskResource struct {
	commonResource
}

func (r *ams_taskResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_ams_task"
}

func (r *ams_taskResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"name": &schema.StringAttribute{
				Required: true,
			},
			"description": &schema.StringAttribute{
				Optional: true,
			},
			"service": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"task_yaml": &schema.StringAttribute{
				Required:    true,
				Description: "This is the definition of the task - which will be put into a new version each time the task is updated. Name must be omitted from this YAML",
			},
			"applied_version": &schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (api *AMSTaskAPIModel) toData(data *AMSTaskResourceModel) {
	data.ID = types.StringValue(api.ID)
	data.AppliedVersion = types.StringValue(api.CurrentVersion)
}

func getYAMLString(yamlObj map[string]interface{}, key string) string {
	v := yamlObj[key]
	s, ok := v.(string)
	if ok {
		return s
	}
	return ""
}

func (data *AMSTaskResourceModel) toAPI(api *AMSTaskAPIModel, diagnostics *diag.Diagnostics) bool {
	var parsedYAML map[string]interface{}
	err := yaml.Unmarshal([]byte(data.TaskYAML.ValueString()), &parsedYAML)
	if err != nil {
		diagnostics.AddError("invalid task YAML", err.Error())
		return false
	}
	taskVersionName := getYAMLString(parsedYAML, "name")
	if taskVersionName != "" {
		diagnostics.AddError("task YAML must not include a name", "the task YAML will be used to create a version, with an auto-generated version ID each time it is updated")
		return false
	}
	api.Name = data.Name.ValueString()
	api.Description = data.Description.ValueString()
	return true
}

func (r *ams_taskResource) apiPath(data *AMSTaskResourceModel, idOrName string) string {
	return fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/tasks/%s", data.Environment.ValueString(), data.Service.ValueString(), idOrName)
}

func (r *ams_taskResource) apiTaskVersionPath(data *AMSTaskResourceModel, idOrName string) string {
	return fmt.Sprintf("%s/versions", r.apiPath(data, idOrName))
}

func (r *ams_taskResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var data AMSTaskResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	var api AMSTaskAPIModel
	ok := data.toAPI(&api, &resp.Diagnostics)
	if ok {
		// Task PUT
		ok, _ = r.apiRequest(ctx, http.MethodPut, r.apiPath(&data, api.Name), &api, &api, &resp.Diagnostics)
	}
	if ok {
		// Task version POST
		ok, _ = r.apiRequest(ctx, http.MethodPost, r.apiTaskVersionPath(&data, api.Name), data.TaskYAML.ValueString(), nil, &resp.Diagnostics, YAMLBody())
	}
	if !ok {
		return
	}

	api.toData(&data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)

}

func (r *ams_taskResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

	var data AMSTaskResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)

	var api AMSTaskAPIModel
	ok := data.toAPI(&api, &resp.Diagnostics)
	taskID := data.ID.ValueString()
	if ok {
		// Task PATCH
		ok, _ = r.apiRequest(ctx, http.MethodPatch, r.apiPath(&data, taskID), &api, &api, &resp.Diagnostics)
	}
	if ok {
		// Task version POST
		ok, _ = r.apiRequest(ctx, http.MethodPost, r.apiTaskVersionPath(&data, api.Name), data.TaskYAML.ValueString(), nil, &resp.Diagnostics, YAMLBody())
	}
	if !ok {
		return
	}

	api.toData(&data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *ams_taskResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AMSTaskResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	var api AMSTaskAPIModel
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

func (r *ams_taskResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AMSTaskResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data, data.ID.ValueString()), nil, nil, &resp.Diagnostics, Allow404())

	r.waitForRemoval(ctx, r.apiPath(&data, data.ID.ValueString()), &resp.Diagnostics)
}
