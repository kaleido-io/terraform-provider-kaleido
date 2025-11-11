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
)

type PMSPolicyDeploymentResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	Environment    types.String `tfsdk:"environment"`
	Service        types.String `tfsdk:"service"`
	Policy         types.String `tfsdk:"policy"`
	PolicyVersion  types.String `tfsdk:"policy_version"`
	Config         types.String `tfsdk:"config"`
	CurrentVersion types.String `tfsdk:"current_version"`
	Created        types.String `tfsdk:"created"`
	Updated        types.String `tfsdk:"updated"`
}

type PMSPolicyDeploymentAPIModel struct {
	ID             string         `json:"id,omitempty"`
	Name           string         `json:"name,omitempty"`
	Description    string         `json:"description,omitempty"`
	Policy         string         `json:"policy,omitempty"`
	PolicyVersion  string         `json:"policyVersion,omitempty"`
	Config         map[string]any `json:"config,omitempty"`
	CurrentVersion string         `json:"currentVersion,omitempty"`
	Created        *time.Time     `json:"created,omitempty"`
	Updated        *time.Time     `json:"updated,omitempty"`
}

type PMSPolicyDeploymentVersionAPIModel struct {
	ID                 string         `json:"id,omitempty"`
	Name               string         `json:"name,omitempty"`
	Description        string         `json:"description,omitempty"`
	PolicyDeploymentID string         `json:"policyDeploymentId,omitempty"`
	Policy             string         `json:"policy,omitempty"`
	PolicyVersion      string         `json:"policyVersion,omitempty"`
	Config             map[string]any `json:"config,omitempty"`
	Hash               string         `json:"hash,omitempty"`
	Created            *time.Time     `json:"created,omitempty"`
	Updated            *time.Time     `json:"updated,omitempty"`
}

func PMSPolicyDeploymentResourceFactory() resource.Resource {
	return &pms_policy_deploymentResource{}
}

type pms_policy_deploymentResource struct {
	commonResource
}

func (r *pms_policy_deploymentResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_pms_policy_deployment"
}

func (r *pms_policy_deploymentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The Policy Manager Policy Deployment resource allows you to manage policy deployments in the Policy Manager.",
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				Description:   "Unique ID of the policy deployment",
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
				Description:   "The name of the policy deployment",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"description": &schema.StringAttribute{
				Optional:      true,
				Description:   "A description of the policy deployment",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"policy": &schema.StringAttribute{
				Required:      true,
				Description:   "The name of the policy to deploy",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"policy_version": &schema.StringAttribute{
				Optional:      true,
				Description:   "The version of the policy to deploy. If not provided, the latest version will be used",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"config": &schema.StringAttribute{
				Required:    true,
				Description: "The policy-specific configuration as JSON",
			},
			"current_version": &schema.StringAttribute{
				Computed:    true,
				Description: "The currently applied version of the policy deployment",
			},
			"created": &schema.StringAttribute{
				Computed:    true,
				Description: "Creation timestamp",
			},
			"updated": &schema.StringAttribute{
				Computed:    true,
				Description: "Last update timestamp",
			},
		},
	}
}

func (r *pms_policy_deploymentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.commonResource.Configure(ctx, req, resp)
}

func (r *pms_policy_deploymentResource) apiPath(data *PMSPolicyDeploymentResourceModel, idOrName string) string {
	env := data.Environment.ValueString()
	service := data.Service.ValueString()
	return fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/policy-deployments/%s", env, service, idOrName)
}

func (r *pms_policy_deploymentResource) apiGetPath(data *PMSPolicyDeploymentResourceModel, idOrName string) string {
	return r.apiPath(data, idOrName) + "?withActive=true"
}

func (r *pms_policy_deploymentResource) apiPolicyDeploymentVersionPath(data *PMSPolicyDeploymentResourceModel, idOrName string) string {
	env := data.Environment.ValueString()
	service := data.Service.ValueString()
	return fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/policy-deployments/%s/versions", env, service, idOrName)
}

func (r *pms_policy_deploymentResource) toAPI(data *PMSPolicyDeploymentResourceModel, api *PMSPolicyDeploymentAPIModel, diagnostics *diag.Diagnostics) {
	api.Name = data.Name.ValueString()
	api.Description = data.Description.ValueString()
	api.Policy = data.Policy.ValueString()
	api.PolicyVersion = data.PolicyVersion.ValueString()

	configJson := data.Config.ValueString()
	var config map[string]any
	err := json.Unmarshal([]byte(configJson), &config)
	if err != nil {
		diagnostics.AddError("Error unmarshalling config", err.Error())
		return
	}
	api.Config = config
}

func (r *pms_policy_deploymentResource) toData(api *PMSPolicyDeploymentAPIModel, data *PMSPolicyDeploymentResourceModel, diagnostics *diag.Diagnostics) {
	data.ID = types.StringValue(api.ID)
	data.Name = types.StringValue(api.Name)
	data.Policy = types.StringValue(api.Policy)

	//Optional fields may be empty in the API but TF treats empty different from null
	if api.PolicyVersion == "" {
		data.PolicyVersion = types.StringNull()
	} else {
		data.PolicyVersion = types.StringValue(api.PolicyVersion)
	}
	if api.Description == "" {
		data.Description = types.StringNull()
	} else {
		data.Description = types.StringValue(api.Description)
	}

	configJson, err := json.Marshal(api.Config)
	if err != nil {
		diagnostics.AddError("Error marshalling config", err.Error())
		return
	}
	data.Config = types.StringValue(string(configJson))
	data.CurrentVersion = types.StringValue(api.CurrentVersion)

	// Note: environment and service are not returned by the API, they remain as set in the resource

	data.Created = types.StringValue(api.Created.Format(time.RFC3339))
	data.Updated = types.StringValue(api.Updated.Format(time.RFC3339))

}

func (r *pms_policy_deploymentResource) toVersionAPI(data *PMSPolicyDeploymentResourceModel, versionAPI *PMSPolicyDeploymentVersionAPIModel, diagnostics *diag.Diagnostics) bool {
	versionAPI.Policy = data.Policy.ValueString()
	versionAPI.PolicyVersion = data.PolicyVersion.ValueString()
	configJson := data.Config.ValueString()
	var config map[string]any
	err := json.Unmarshal([]byte(configJson), &config)
	if err != nil {
		diagnostics.AddError("Error unmarshalling config", err.Error())
		return false
	}
	versionAPI.Config = config
	versionAPI.Description = data.Description.ValueString()

	return true
}

func (r *pms_policy_deploymentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PMSPolicyDeploymentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var api PMSPolicyDeploymentAPIModel
	r.toAPI(&data, &api, &resp.Diagnostics)

	// Create the policy deployment
	ok, _ := r.apiRequest(ctx, http.MethodPut, r.apiPath(&data, api.Name), &api, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	// Create the initial version with policy and config
	var versionAPI PMSPolicyDeploymentVersionAPIModel
	ok = r.toVersionAPI(&data, &versionAPI, &resp.Diagnostics)
	if !ok {
		return
	}

	ok, _ = r.apiRequest(ctx, http.MethodPost, r.apiPolicyDeploymentVersionPath(&data, api.Name), &versionAPI, &versionAPI, &resp.Diagnostics)
	if !ok {
		return
	}

	// Fetch the updated policy deployment with the current version
	var updatedAPI PMSPolicyDeploymentAPIModel
	getPath := r.apiGetPath(&data, api.Name)
	ok, _ = r.apiRequest(ctx, http.MethodGet, getPath, nil, &updatedAPI, &resp.Diagnostics)
	if !ok {
		return
	}

	r.toData(&updatedAPI, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *pms_policy_deploymentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PMSPolicyDeploymentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var api PMSPolicyDeploymentAPIModel
	ok, status := r.apiRequest(ctx, http.MethodGet, r.apiGetPath(&data, data.ID.ValueString()), nil, &api, &resp.Diagnostics, Allow404())
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

func (r *pms_policy_deploymentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data PMSPolicyDeploymentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var api PMSPolicyDeploymentAPIModel
	r.toAPI(&data, &api, &resp.Diagnostics)
	policyDeploymentID := data.ID.ValueString()

	// Update the policy deployment metadata
	ok, _ := r.apiRequest(ctx, http.MethodPatch, r.apiPath(&data, policyDeploymentID), &api, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	// Create a new version with updated policy and config
	var versionAPI PMSPolicyDeploymentVersionAPIModel
	ok = r.toVersionAPI(&data, &versionAPI, &resp.Diagnostics)
	if !ok {
		return
	}

	ok, _ = r.apiRequest(ctx, http.MethodPost, r.apiPolicyDeploymentVersionPath(&data, policyDeploymentID), &versionAPI, &versionAPI, &resp.Diagnostics)
	if !ok {
		return
	}

	// Fetch the updated policy deployment with the current version
	var updatedAPI PMSPolicyDeploymentAPIModel
	getPath := r.apiGetPath(&data, policyDeploymentID)
	ok, _ = r.apiRequest(ctx, http.MethodGet, getPath, nil, &updatedAPI, &resp.Diagnostics)
	if !ok {
		return
	}

	r.toData(&updatedAPI, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *pms_policy_deploymentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PMSPolicyDeploymentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data, data.ID.ValueString())+"?force=true", nil, nil, &resp.Diagnostics, Allow404())
	r.waitForRemoval(ctx, r.apiPath(&data, data.ID.ValueString()), &resp.Diagnostics)
}
