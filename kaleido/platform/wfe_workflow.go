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

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"gopkg.in/yaml.v3"
)

type WFEWorkflowResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Name                types.String `tfsdk:"name"`
	Description         types.String `tfsdk:"description"`
	Environment         types.String `tfsdk:"environment"`
	Service             types.String `tfsdk:"service"`
	FlowYAML            types.String `tfsdk:"flow_yaml"` // yaml string containing workflow definition
	AppliedVersion      types.String `tfsdk:"applied_version"`
	Created             types.String `tfsdk:"created"`
	Updated             types.String `tfsdk:"updated"`
	HandlerBindingsJSON types.String `tfsdk:"handler_bindings_json"`
}

type WFEWorkflowAPIModel struct {
	ID              string                 `json:"id,omitempty"`
	Name            string                 `json:"name,omitempty"`
	Description     string                 `json:"description,omitempty"`
	Created         *time.Time             `json:"created,omitempty"`
	Updated         *time.Time             `json:"updated,omitempty"`
	CurrentVersion  string                 `json:"currentVersion,omitempty"`
	HandlerBindings map[string]interface{} `json:"handlerBindings,omitempty"`
}

type Definition map[string]interface{}

type WFEWorkflowVersionAPIModel struct {
	Definition
	ID          string     `json:"id,omitempty"`
	Name        string     `json:"name,omitempty"`
	WorkflowID  string     `json:"workflowId,omitempty"`
	Description string     `json:"description,omitempty"`
	Hash        string     `json:"hash,omitempty"`
	Created     *time.Time `json:"created,omitempty"`
	Updated     *time.Time `json:"updated,omitempty"`
}

func WFEWorkflowResourceFactory() resource.Resource {
	return &wfe_workflowResource{}
}

type wfe_workflowResource struct {
	commonResource
}

func (r *wfe_workflowResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_wfe_workflow"
}

func (r *wfe_workflowResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The Workflow Engine Workflow resource allows you to manage workflows in the Workflow Engine.",
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				Description:   "Unique ID of the workflow",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"environment": &schema.StringAttribute{
				Required:      true,
				Description:   "The environment ID",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"service": &schema.StringAttribute{
				Required:      true,
				Description:   "The service ID",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": &schema.StringAttribute{
				Required:      true,
				Description:   "The name of the workflow",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"description": &schema.StringAttribute{
				Optional:    true,
				Description: "A description of the workflow",
			},
			"flow_yaml": &schema.StringAttribute{
				Required:    true,
				Description: "The workflow definition as YAML.  This includes stages, events, operations and subflows but does not include handler bindings or subflow bindings",
			},
			"applied_version": &schema.StringAttribute{
				Computed:    true,
				Description: "The currently applied version of the workflow",
			},
			"created": &schema.StringAttribute{
				Computed:    true,
				Description: "Creation timestamp",
			},
			"updated": &schema.StringAttribute{
				Computed:    true,
				Description: "Last update timestamp",
			},
			"handler_bindings_json": &schema.StringAttribute{
				Optional:    true,
				Description: "The workflow handler bindings as JSON",
			},
		},
	}
}

func (r *wfe_workflowResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.commonResource.Configure(ctx, req, resp)
}

func (r *wfe_workflowResource) apiGetPath(data *WFEWorkflowResourceModel, idOrName string) string {
	return r.apiPath(data, idOrName) + "?withActive=true"
}

func (r *wfe_workflowResource) apiPath(data *WFEWorkflowResourceModel, idOrName string) string {
	env := data.Environment.ValueString()
	service := data.Service.ValueString()
	return fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/workflows/%s", env, service, idOrName)
}

func (r *wfe_workflowResource) apiWorkflowVersionPath(data *WFEWorkflowResourceModel, idOrName string) string {
	env := data.Environment.ValueString()
	service := data.Service.ValueString()
	return fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/workflows/%s/versions", env, service, idOrName)
}

func (r *wfe_workflowResource) toAPI(data *WFEWorkflowResourceModel, api *WFEWorkflowAPIModel, diagnostics *diag.Diagnostics) {
	api.Name = data.Name.ValueString()
	api.Description = data.Description.ValueString()
	handlerBindings := make(map[string]interface{})
	if err := json.Unmarshal([]byte(data.HandlerBindingsJSON.ValueString()), &handlerBindings); err != nil {
		diagnostics.AddError("Invalid JSON", fmt.Sprintf("Failed to parse workflow handler bindings JSON: %v", err))
		return
	}
	api.HandlerBindings = handlerBindings
}

func (r *wfe_workflowResource) toData(api *WFEWorkflowAPIModel, data *WFEWorkflowResourceModel, diagnostics *diag.Diagnostics) {
	data.ID = types.StringValue(api.ID)
	data.Name = types.StringValue(api.Name)
	data.AppliedVersion = types.StringValue(api.CurrentVersion)

	// Note: environment and service are not returned by the API, they remain as set in the resource

	if api.Description == "" {
		data.Description = types.StringNull()
	} else {
		data.Description = types.StringValue(api.Description)
	}

	if api.Created == nil {
		data.Created = types.StringNull()
	} else {
		data.Created = types.StringValue(api.Created.Format(time.RFC3339))
	}
	if api.Updated == nil {
		data.Updated = types.StringNull()
	} else {
		data.Updated = types.StringValue(api.Updated.Format(time.RFC3339))
	}
}

func (r *wfe_workflowResource) toVersionAPI(data *WFEWorkflowResourceModel, versionAPI *WFEWorkflowVersionAPIModel, diagnostics *diag.Diagnostics) bool {
	// Parse the definition YAML string
	var definition map[string]interface{}
	if err := yaml.Unmarshal([]byte(data.FlowYAML.ValueString()), &definition); err != nil {
		diagnostics.AddError("Invalid YAML", fmt.Sprintf("Failed to parse workflow definition YAML: %v", err))
		return false
	}

	versionAPI.Definition = definition
	versionAPI.Description = data.Description.ValueString()

	return true
}

func (r *wfe_workflowResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var data WFEWorkflowResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var api WFEWorkflowAPIModel
	r.toAPI(&data, &api, &resp.Diagnostics)

	// Create the workflow
	ok, _ := r.apiRequest(ctx, http.MethodPut, r.apiPath(&data, api.Name), &api, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	// Create the initial version with definition
	var versionAPI WFEWorkflowVersionAPIModel
	ok = r.toVersionAPI(&data, &versionAPI, &resp.Diagnostics)
	if !ok {
		return
	}

	ok, _ = r.apiRequest(ctx, http.MethodPost, r.apiWorkflowVersionPath(&data, api.Name), data.FlowYAML.ValueString(), nil, &resp.Diagnostics, YAMLBody())
	if !ok {
		return
	}

	// Fetch the updated workflow with the current version
	var updatedAPI WFEWorkflowAPIModel
	getPath := r.apiGetPath(&data, api.Name)
	ok, _ = r.apiRequest(ctx, http.MethodGet, getPath, nil, &updatedAPI, &resp.Diagnostics)
	if !ok {
		return
	}

	r.toData(&updatedAPI, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *wfe_workflowResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data WFEWorkflowResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var api WFEWorkflowAPIModel
	ok, status := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data, data.ID.ValueString()), nil, &api, &resp.Diagnostics, Allow404())
	if !ok {
		return
	}
	if status == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	r.toData(&api, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *wfe_workflowResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data WFEWorkflowResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var api WFEWorkflowAPIModel
	r.toAPI(&data, &api, &resp.Diagnostics)
	workflowID := data.ID.ValueString()

	// Update the workflow metadata
	ok, _ := r.apiRequest(ctx, http.MethodPatch, r.apiPath(&data, workflowID), &api, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	// Create a new version with updated definition
	var versionAPI WFEWorkflowVersionAPIModel
	ok = r.toVersionAPI(&data, &versionAPI, &resp.Diagnostics)
	if !ok {
		return
	}

	ok, _ = r.apiRequest(ctx, http.MethodPost, r.apiWorkflowVersionPath(&data, api.Name), data.FlowYAML.ValueString(), nil, &resp.Diagnostics, YAMLBody())
	if !ok {
		return
	}

	// Fetch the updated workflow with the current version
	var updatedAPI WFEWorkflowAPIModel
	getPath := r.apiGetPath(&data, workflowID)
	ok, _ = r.apiRequest(ctx, http.MethodGet, getPath, nil, &updatedAPI, &resp.Diagnostics)
	if !ok {
		return
	}

	r.toData(&updatedAPI, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *wfe_workflowResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data WFEWorkflowResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data, data.ID.ValueString()), nil, nil, &resp.Diagnostics, Allow404())
	r.waitForRemoval(ctx, r.apiPath(&data, data.ID.ValueString()), &resp.Diagnostics)
}
