// Copyright © Kaleido, Inc. 2024-2025

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

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var SupportedArtifactFamilies = []string{
	"provider",
}

type ARSNamespaceResourceModel struct {
	ID              types.String `tfsdk:"id"`
	Environment     types.String `tfsdk:"environment"`
	Service         types.String `tfsdk:"service"`
	Name            types.String `tfsdk:"name"`
	AutoCreateRepos types.Bool   `tfsdk:"auto_create_repos"`
	ArtifactFamily  types.String `tfsdk:"artifact_family"`
	Description     types.String `tfsdk:"description"`
}

type ARSNamespaceAPIModel struct {
	ID              string `json:"id,omitempty"`
	Name            string `json:"name"`
	AutoCreateRepos bool   `json:"autoCreateRepos"`
	ArtifactFamily  string `json:"artifactFamily"`
	Description     string `json:"description,omitempty"`
}

func ARSNamespaceResourceFactory() resource.Resource {
	return &arsNamespaceResource{}
}

type arsNamespaceResource struct {
	commonResource
}

func (r *arsNamespaceResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_ars_namespace"
}

func (r *arsNamespaceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A namespace in the Kaleido Artifact Registry. Namespaces group repositories and define allowed manifest types.",
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"environment": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Environment ID",
			},
			"service": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Artifact Registry service ID",
			},
			"name": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Namespace name",
			},
			"auto_create_repos": &schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Default:       booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
				Description:   "Whether to automatically create repositories when pushing unknown names.",
			},
			"artifact_family": &schema.StringAttribute{
				Optional:      false,
				Required:      true,
				Validators:    []validator.String{stringvalidator.OneOf(SupportedArtifactFamilies...)},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "List of allowed artifact families (e.g. provider).",
			},
			"description": &schema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Optional description for the namespace.",
			},
		},
	}
}

func (data *ARSNamespaceResourceModel) toAPI(api *ARSNamespaceAPIModel) {
	api.Name = data.Name.ValueString()
	api.AutoCreateRepos = data.AutoCreateRepos.ValueBool()
	api.Description = data.Description.ValueString()
	api.ArtifactFamily = data.ArtifactFamily.ValueString()
}

func (api *ARSNamespaceAPIModel) toData(ctx context.Context, data *ARSNamespaceResourceModel) {
	data.ID = types.StringValue(api.ID)
	data.Name = types.StringValue(api.Name)
	data.AutoCreateRepos = types.BoolValue(api.AutoCreateRepos)
	data.Description = types.StringValue(api.Description)
	data.ArtifactFamily = types.StringValue(api.ArtifactFamily)
}

func (r *arsNamespaceResource) apiPath(data *ARSNamespaceResourceModel) string {
	path := fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/namespaces", data.Environment.ValueString(), data.Service.ValueString())
	if data.ID.ValueString() != "" {
		path = path + "/" + data.ID.ValueString()
	}
	return path
}

func (r *arsNamespaceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ARSNamespaceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var api ARSNamespaceAPIModel
	data.toAPI(&api)
	ok, _ := r.apiRequest(ctx, http.MethodPost, r.apiPath(&data), api, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	api.toData(ctx, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *arsNamespaceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ARSNamespaceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var api ARSNamespaceAPIModel
	ok, status := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics, Allow404())
	if !ok {
		return
	}
	if status == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	api.toData(ctx, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *arsNamespaceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Artifact Registry namespace API has no PATCH/PUT; all mutable attributes use RequiresReplace
	// so Terraform will replace. If we get here due to a no-op change, just read and set state.
	var data ARSNamespaceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var api ARSNamespaceAPIModel
	ok, status := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics, Allow404())
	if !ok || status == 404 {
		return
	}
	api.toData(ctx, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *arsNamespaceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ARSNamespaceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data), nil, nil, &resp.Diagnostics, Allow404())
	r.waitForRemoval(ctx, r.apiPath(&data), &resp.Diagnostics)
}
