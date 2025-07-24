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

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type CMSGithubBuildResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Environment      types.String `tfsdk:"environment"`
	Service          types.String `tfsdk:"service"`
	Name             types.String `tfsdk:"name"`
	Path             types.String `tfsdk:"path"`
	Description      types.String `tfsdk:"description"`
	EVMVersion       types.String `tfsdk:"evm_version"`
	SolcVersion      types.String `tfsdk:"solc_version"`
	ContractURL      types.String `tfsdk:"contract_url"`
	ContractName     types.String `tfsdk:"contract_name"`
	AuthToken        types.String `tfsdk:"auth_token"`
	OptimizerEnabled types.Bool   `tfsdk:"optimizer_enabled"`
	OptimizerRuns    types.Int64  `tfsdk:"optimizer_runs"`
	OptimizerViaIR   types.Bool   `tfsdk:"optimizer_via_ir"`
	ABI              types.String `tfsdk:"abi"`
	Bytecode         types.String `tfsdk:"bytecode"`
	DevDocs          types.String `tfsdk:"dev_docs"`
	CommitHash       types.String `tfsdk:"commit_hash"`
}

func CMSGithubBuildResourceFactory() resource.Resource {
	return &cmsGithubBuildResource{}
}

type cmsGithubBuildResource struct {
	cmsBuildResourceBase
}

func (r *cmsGithubBuildResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_cms_github_build"
}

func (r *cmsGithubBuildResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"path": &schema.StringAttribute{
				Required: true,
			},
			"name": &schema.StringAttribute{
				Required: true,
			},
			"description": &schema.StringAttribute{
				Optional: true,
			},
			"evm_version": &schema.StringAttribute{
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Optional:      true,
			},
			"solc_version": &schema.StringAttribute{
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Optional:      true,
			},
			"contract_url": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"contract_name": &schema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"auth_token": &schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
			},
			"optimizer_enabled": &schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace(), boolplanmodifier.UseStateForUnknown()},
			},
			"optimizer_runs": &schema.Int64Attribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace(), int64planmodifier.UseStateForUnknown()},
			},
			"optimizer_via_ir": &schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace(), boolplanmodifier.UseStateForUnknown()},
			},
			"abi": &schema.StringAttribute{
				Computed: true,
			},
			"bytecode": &schema.StringAttribute{
				Computed: true,
			},
			"dev_docs": &schema.StringAttribute{
				Computed: true,
			},
			"commit_hash": &schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (data *CMSGithubBuildResourceModel) toAPI(api *CMSBuildAPIModel, isUpdate bool) {
	api.Name = data.Name.ValueString()
	api.Path = data.Path.ValueString()
	api.Description = data.Description.ValueString()

	if !isUpdate {
		api.EVMVersion = data.EVMVersion.ValueString()
		api.SolcVersion = data.SolcVersion.ValueString()

		// Set up GitHub configuration
		api.GitHub = &CMSBuildGithubAPIModel{
			ContractURL:  data.ContractURL.ValueString(),
			ContractName: data.ContractName.ValueString(),
			AuthToken:    data.AuthToken.ValueString(),
		}

		// Set up optimizer configuration
		api.Optimizer = &CMSBuildOptimizerAPIModel{}

		// Set enabled value - default to false if not specified
		if !data.OptimizerEnabled.IsNull() {
			enabled := data.OptimizerEnabled.ValueBool()
			api.Optimizer.Enabled = &enabled
		} else {
			enabled := false
			api.Optimizer.Enabled = &enabled
		}

		// Set viaIR value - default to false if not specified
		if !data.OptimizerViaIR.IsNull() {
			viaIR := data.OptimizerViaIR.ValueBool()
			api.Optimizer.ViaIR = &viaIR
		} else {
			viaIR := false
			api.Optimizer.ViaIR = &viaIR
		}

		// Set runs value - default to 200 if not specified
		if !data.OptimizerRuns.IsNull() {
			api.Optimizer.Runs = float64(data.OptimizerRuns.ValueInt64())
		} else {
			api.Optimizer.Runs = 200 // default
		}
	}
}

func (api *CMSBuildAPIModel) toGithubData(data *CMSGithubBuildResourceModel) {
	data.ID = types.StringValue(api.ID)
	data.ABI = types.StringValue(interfaceToJSONString(api.ABI))
	data.Bytecode = types.StringValue(api.Bytecode)
	data.DevDocs = types.StringValue(interfaceToJSONString(api.DevDocs))

	if api.GitHub != nil {
		data.CommitHash = types.StringValue(api.GitHub.CommitHash)
	} else {
		data.CommitHash = types.StringValue("")
	}

	if api.Optimizer != nil {
		if api.Optimizer.Enabled != nil {
			data.OptimizerEnabled = types.BoolValue(*api.Optimizer.Enabled)
		}
		if api.Optimizer.ViaIR != nil {
			data.OptimizerViaIR = types.BoolValue(*api.Optimizer.ViaIR)
		}
		data.OptimizerRuns = types.Int64Value(int64(api.Optimizer.Runs))
	}
}

func (r *cmsGithubBuildResource) apiPath(data *CMSGithubBuildResourceModel) string {
	return r.buildAPIPath(data.Environment.ValueString(), data.Service.ValueString(), data.ID.ValueString())
}

func (r *cmsGithubBuildResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data CMSGithubBuildResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	var api CMSBuildAPIModel
	data.toAPI(&api, false)
	ok, _ := r.apiRequest(ctx, http.MethodPost, r.apiPath(&data), api, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	api.toGithubData(&data) // need the ID copied over
	path := r.apiPath(&data)
	r.waitForBuildStatus(ctx, path, &api, &resp.Diagnostics)
	api.toGithubData(&data) // capture the build info
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *cmsGithubBuildResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data CMSGithubBuildResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)

	// Update from plan
	var api CMSBuildAPIModel
	data.toAPI(&api, true)
	if ok, _ := r.apiRequest(ctx, http.MethodPatch, r.apiPath(&data), api, &api, &resp.Diagnostics); !ok {
		return
	}

	api.toGithubData(&data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *cmsGithubBuildResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CMSGithubBuildResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	var api CMSBuildAPIModel
	api.ID = data.ID.ValueString()
	ok, status := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics, Allow404())
	if !ok {
		return
	}
	if status == 404 {
		resp.State.RemoveResource(ctx)
		return
	}
	api.toGithubData(&data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *cmsGithubBuildResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CMSGithubBuildResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data), nil, nil, &resp.Diagnostics, Allow404())

	r.waitForRemoval(ctx, r.apiPath(&data), &resp.Diagnostics)
}
