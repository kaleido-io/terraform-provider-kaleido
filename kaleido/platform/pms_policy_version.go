// Copyright © Kaleido, Inc. 2026

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
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"gopkg.in/yaml.v3"
)

type PMSPolicyVersionResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Environment    types.String `tfsdk:"environment"`
	Service        types.String `tfsdk:"service"`
	Policy         types.String `tfsdk:"policy"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	DefinitionYAML types.String `tfsdk:"definition_yaml"`
	Hash           types.String `tfsdk:"hash"`
	Created        types.String `tfsdk:"created"`
	Updated        types.String `tfsdk:"updated"`
}

// PMSPolicyVersionPatchAPIModel is the PATCH body - a version's definition is immutable,
// only its description can be changed.
type PMSPolicyVersionPatchAPIModel struct {
	Description string `json:"description,omitempty"`
}

func PMSPolicyVersionResourceFactory() resource.Resource {
	return &pms_policyVersionResource{}
}

type pms_policyVersionResource struct {
	commonResource
}

func (r *pms_policyVersionResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_pms_policy_version"
}

func (r *pms_policyVersionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a version of a Policy Manager policy, and activates it as the policy's current version. Use this with a kaleido_platform_pms_policy that has no definition_yaml, when the bindings a definition depends on are declared as their own resources: terraform then orders the bindings ahead of the version. For the simpler case, set definition_yaml on the policy itself instead.",
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				Description:   "The version ID assigned by the server",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"environment": &schema.StringAttribute{
				Required:      true,
				Description:   "Environment ID",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"service": &schema.StringAttribute{
				Required:      true,
				Description:   "Policy Manager service ID",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"policy": &schema.StringAttribute{
				Required:      true,
				Description:   "Name or ID of the policy this version belongs to",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": &schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Name of the version. If omitted the server assigns one based on the previous version.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
			},
			"description": &schema.StringAttribute{
				Optional: true,
				Computed: true,
				// A definition_yaml carrying its own top level 'description' sets the
				// version's description, so the server may return one that was never set
				// as an attribute here
				Description:   "Description of this version. Defaults to the 'description' field of definition_yaml, if it has one.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"definition_yaml": &schema.StringAttribute{
				Required:      true,
				Description:   "The policy definition as YAML, containing components, constants, evidence, decision, output, parameters, parameterValues and summaryTemplate. Versions are immutable, so changing this creates a new version.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"hash": &schema.StringAttribute{
				Computed:    true,
				Description: "Hash of the version, for irrefutable post hoc comparison",
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

func (r *pms_policyVersionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.commonResource.Configure(ctx, req, resp)
}

func (r *pms_policyVersionResource) listPath(data *PMSPolicyVersionResourceModel) string {
	return fmt.Sprintf("/endpoint/%s/%s/rest/api/v2/policies/%s/versions",
		data.Environment.ValueString(), data.Service.ValueString(), url.PathEscape(data.Policy.ValueString()))
}

func (r *pms_policyVersionResource) instancePath(data *PMSPolicyVersionResourceModel) string {
	return fmt.Sprintf("%s/%s", r.listPath(data), url.PathEscape(data.ID.ValueString()))
}

// toAPI parses definition_yaml into the version body. The definition fields sit at the
// top level of the request alongside the version metadata, which is what the API
// expects. It is sent as JSON so the request does not depend on the server accepting a
// YAML content type.
func (r *pms_policyVersionResource) toAPI(data *PMSPolicyVersionResourceModel, diagnostics *diag.Diagnostics) map[string]interface{} {
	body := map[string]interface{}{}
	if err := yaml.Unmarshal([]byte(data.DefinitionYAML.ValueString()), &body); err != nil {
		diagnostics.AddError("Invalid YAML", fmt.Sprintf("Failed to parse policy definition YAML: %v", err))
		return nil
	}
	if len(body) == 0 {
		diagnostics.AddError("Invalid policy definition", "definition_yaml must not be empty - a policy version requires a definition")
		return nil
	}
	if name := data.Name.ValueString(); name != "" {
		body["name"] = name
	}
	if description := data.Description.ValueString(); description != "" {
		body["description"] = description
	}
	return body
}

func (r *pms_policyVersionResource) toData(api *PMSPolicyVersionAPIModel, data *PMSPolicyVersionResourceModel) {
	data.ID = types.StringValue(api.ID)
	data.Name = types.StringValue(api.Name)
	data.Hash = optionalString(api.Hash)
	data.Description = optionalString(api.Description)
	data.Created = timeAttr(api.Created)
	data.Updated = timeAttr(api.Updated)
}

func (r *pms_policyVersionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PMSPolicyVersionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := r.toAPI(&data, &resp.Diagnostics)
	if body == nil {
		return
	}

	// Creating a version activates it as the policy's current version
	var api PMSPolicyVersionAPIModel
	ok, _ := r.apiRequest(ctx, http.MethodPost, r.listPath(&data), body, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	r.toData(&api, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *pms_policyVersionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PMSPolicyVersionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var api PMSPolicyVersionAPIModel
	ok, status := r.apiRequest(ctx, http.MethodGet, r.instancePath(&data), nil, &api, &resp.Diagnostics, Allow404())
	if !ok {
		return
	}
	if status == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	r.toData(&api, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *pms_policyVersionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data PMSPolicyVersionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Everything except the description forces replacement, because a version is immutable
	patch := PMSPolicyVersionPatchAPIModel{Description: data.Description.ValueString()}
	var api PMSPolicyVersionAPIModel
	ok, _ := r.apiRequest(ctx, http.MethodPatch, r.instancePath(&data), &patch, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	r.toData(&api, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *pms_policyVersionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PMSPolicyVersionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The server refuses to delete the version a policy is currently on. That is the
	// normal state for the most recently applied version, so it is reported as a warning
	// and the version is left in place rather than failing the destroy.
	_, status := r.apiRequest(ctx, http.MethodDelete, r.instancePath(&data), nil, nil, &resp.Diagnostics, Allow404(), AllowStatus(http.StatusConflict))
	if status == http.StatusConflict {
		resp.Diagnostics.AddWarning("Policy version not deleted",
			fmt.Sprintf("Version %s is the current version of policy %s and cannot be deleted. It has been removed from terraform state but remains on the policy.",
				data.Name.ValueString(), data.Policy.ValueString()))
	}
}
