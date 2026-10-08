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
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ARSRepositoryResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Environment types.String `tfsdk:"environment"`
	Service     types.String `tfsdk:"service"`
	Namespace   types.String `tfsdk:"namespace"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

type ARSRepositoryAPIModel struct {
	ID            string `json:"id,omitempty"`
	NamespaceName string `json:"namespaceName,omitempty"`
	Name          string `json:"name"`
	Description   string `json:"description,omitempty"`
}

func ARSRepositoryResourceFactory() resource.Resource {
	return &arsRepositoryResource{}
}

type arsRepositoryResource struct {
	commonResource
}

var _ resource.ResourceWithImportState = &arsRepositoryResource{}

func (r *arsRepositoryResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_ars_repository"
}

func (r *arsRepositoryResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A repository in a Kaleido Artifact Registry namespace" +
			"Required for artifacts in namespaces with auto_create_repos = false. Destroying a repository that still holds artifact versions will fail and leave it in the registry namespace with a warning.",
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
			"namespace": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Name of the namespace to create the repository in",
			},
			"name": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Repository name, e.g. 'path/to/myfilename.ext'. Validated by the Artifact Registry against the namespace's artifact family.",
			},
			"description": &schema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Optional description for the repository.",
			},
		},
	}
}

func (api *ARSRepositoryAPIModel) toData(data *ARSRepositoryResourceModel) {
	data.ID = types.StringValue(api.ID)
	data.Name = types.StringValue(api.Name)
	if api.Description != "" || !data.Description.IsNull() {
		data.Description = types.StringValue(api.Description)
	}
}

func (r *arsRepositoryResource) namespacePath(data *ARSRepositoryResourceModel) string {
	return fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/namespaces/%s",
		data.Environment.ValueString(), data.Service.ValueString(), data.Namespace.ValueString())
}

func (r *arsRepositoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ARSRepositoryResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	api := ARSRepositoryAPIModel{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
	}
	var createDiags diag.Diagnostics
	if ok, _ := r.apiRequest(ctx, http.MethodPost, r.namespacePath(&data)+"/repositories", api, &api, &createDiags); !ok {
		for _, d := range createDiags.Errors() {
			if strings.Contains(d.Detail(), "already exists") {
				resp.Diagnostics.AddError("Repository already exists",
					fmt.Sprintf("Repository '%s' already exists in namespace '%s', for example created by an artifact push to a namespace with auto_create_repos = true. "+
						"Import it with: terraform import <address> %s/%s/%s/%s\n\n%s",
						data.Name.ValueString(), data.Namespace.ValueString(),
						data.Environment.ValueString(), data.Service.ValueString(), data.Namespace.ValueString(), data.Name.ValueString(), d.Detail()))
			} else {
				resp.Diagnostics.Append(d)
			}
		}
		return
	}

	api.toData(&data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

// getRepository returns false with no diagnostics when the repository (or its namespace) no longer exists
func (r *arsRepositoryResource) getRepository(ctx context.Context, data *ARSRepositoryResourceModel, diagnostics *diag.Diagnostics) bool {
	var api ARSRepositoryAPIModel
	var getDiags diag.Diagnostics
	ok, status := r.apiRequest(ctx, http.MethodGet, r.namespacePath(data)+"/repositories/"+data.Name.ValueString(), nil, &api, &getDiags, Allow404())
	if !ok {
		for _, d := range getDiags.Errors() {
			// The registry reports a missing repository or namespace as a plain (non-404) error
			if !arsAlreadyDeleted(d.Detail()) {
				diagnostics.Append(d)
			}
		}
		return false
	}
	if status == 404 {
		return false
	}
	api.toData(data)
	return true
}

func (r *arsRepositoryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ARSRepositoryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !r.getRepository(ctx, &data, &resp.Diagnostics) {
		if !resp.Diagnostics.HasError() {
			resp.State.RemoveResource(ctx)
		}
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *arsRepositoryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ARSRepositoryResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if r.getRepository(ctx, &data, &resp.Diagnostics) {
		resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
	}
}

func (r *arsRepositoryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ARSRepositoryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var deleteDiags diag.Diagnostics
	if ok, _ := r.apiRequest(ctx, http.MethodDelete, r.namespacePath(&data)+"/files/"+data.Name.ValueString(), nil, nil, &deleteDiags, Allow404()); !ok {
		for _, d := range deleteDiags.Errors() {
			switch {
			case arsAlreadyDeleted(d.Detail()):
			case arsHasTaggedVersions(d.Detail()):
				resp.Diagnostics.AddWarning(
					fmt.Sprintf("Repository '%s' still contains versions", data.Name.ValueString()),
					"The repository was removed from Terraform state but left in the Artifact Registry, because it holds versions "+
						"not tracked by any artifact resource (for example versions retained by remove_old_versions = false). "+
						"It blocks destroying the namespace unless the namespace sets force_destroy = true. "+d.Detail())
			default:
				resp.Diagnostics.Append(d)
			}
		}
	}
}

func arsHasTaggedVersions(errorDetail string) bool {
	return strings.Contains(errorDetail, "tagged version(s)")
}

func (r *arsRepositoryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: environment/service/namespace/name - the name may contain slashes
	parts := strings.SplitN(req.ID, "/", 4)
	if len(parts) != 4 || parts[0] == "" || parts[1] == "" || parts[2] == "" || parts[3] == "" {
		resp.Diagnostics.AddError("Invalid import ID", "Import ID must be in format: environment/service/namespace/name")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("service"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("namespace"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), parts[3])...)
}
