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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type CMSPrecompiledBuildResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Environment types.String `tfsdk:"environment"`
	Service     types.String `tfsdk:"service"`
	Name        types.String `tfsdk:"name"`
	Path        types.String `tfsdk:"path"`
	Description types.String `tfsdk:"description"`
	EVMVersion  types.String `tfsdk:"evm_version"`
	SolcVersion types.String `tfsdk:"solc_version"`
	ABI         types.String `tfsdk:"abi"`
	Bytecode    types.String `tfsdk:"bytecode"`
	DevDocs     types.String `tfsdk:"dev_docs"`
}

func CMSPrecompiledBuildResourceFactory() resource.Resource {
	return &cmsPrecompiledBuildResource{}
}

type cmsPrecompiledBuildResource struct {
	cmsBuildResourceBase
}

func (r *cmsPrecompiledBuildResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_cms_precompiled_build"
}

func (r *cmsPrecompiledBuildResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"abi": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"bytecode": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"dev_docs": &schema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},
	}
}

func (data *CMSPrecompiledBuildResourceModel) toAPI(api *CMSBuildAPIModel, isUpdate bool) {
	api.Name = data.Name.ValueString()
	api.Path = data.Path.ValueString()
	api.Description = data.Description.ValueString()

	if !isUpdate {
		api.EVMVersion = data.EVMVersion.ValueString()
		api.SolcVersion = data.SolcVersion.ValueString()

		// Set up precompiled data
		jsonStringToInterface(data.ABI.ValueString(), &api.ABI)
		api.Bytecode = data.Bytecode.ValueString()
		if !data.DevDocs.IsNull() {
			jsonStringToInterface(data.DevDocs.ValueString(), &api.DevDocs)
		}

		// no optimizer for precompiled builds
	}
}

func (api *CMSBuildAPIModel) toPrecompiledData(data *CMSPrecompiledBuildResourceModel) {
	data.ID = types.StringValue(api.ID)
	data.ABI = types.StringValue(interfaceToJSONString(api.ABI))
	data.Bytecode = types.StringValue(api.Bytecode)

	if api.DevDocs != nil {
		data.DevDocs = types.StringValue(interfaceToJSONString(api.DevDocs))
	}
}

func (r *cmsPrecompiledBuildResource) apiPath(data *CMSPrecompiledBuildResourceModel) string {
	return r.buildAPIPath(data.Environment.ValueString(), data.Service.ValueString(), data.ID.ValueString())
}

func (r *cmsPrecompiledBuildResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data CMSPrecompiledBuildResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	var api CMSBuildAPIModel
	data.toAPI(&api, false)
	ok, _ := r.apiRequest(ctx, http.MethodPost, r.apiPath(&data), api, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	api.toPrecompiledData(&data) // need the ID copied over
	path := r.apiPath(&data)
	r.waitForBuildStatus(ctx, path, &api, &resp.Diagnostics)
	api.toPrecompiledData(&data) // capture the build info
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *cmsPrecompiledBuildResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data CMSPrecompiledBuildResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)

	// Update from plan
	var api CMSBuildAPIModel
	data.toAPI(&api, true)
	if ok, _ := r.apiRequest(ctx, http.MethodPatch, r.apiPath(&data), api, &api, &resp.Diagnostics); !ok {
		return
	}

	api.toPrecompiledData(&data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *cmsPrecompiledBuildResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CMSPrecompiledBuildResourceModel
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
	api.toPrecompiledData(&data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *cmsPrecompiledBuildResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CMSPrecompiledBuildResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data), nil, nil, &resp.Diagnostics, Allow404())

	r.waitForRemoval(ctx, r.apiPath(&data), &resp.Diagnostics)
}
