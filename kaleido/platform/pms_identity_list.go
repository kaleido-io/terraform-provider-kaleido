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
	"sort"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type PMSIdentityListResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	Environment    types.String `tfsdk:"environment"`
	Service        types.String `tfsdk:"service"`
	Identities     types.List   `tfsdk:"identities"` // Array of identity IDs/DIDs
	AppliedVersion types.String `tfsdk:"applied_version"`
	Created        types.String `tfsdk:"created"`
	Updated        types.String `tfsdk:"updated"`
}

type PMSIdentityListAPIModel struct {
	ID             string     `json:"id,omitempty"`
	Name           string     `json:"name,omitempty"`
	Description    string     `json:"description,omitempty"`
	Created        *time.Time `json:"created,omitempty"`
	Updated        *time.Time `json:"updated,omitempty"`
	CurrentVersion string     `json:"currentVersion,omitempty"`
	Identities     []string   `json:"identities,omitempty"`
}

type PMSIdentityListVersionAPIModel struct {
	ID             string     `json:"id,omitempty"`
	Name           string     `json:"name,omitempty"`
	IdentityListID string     `json:"identityListId,omitempty"`
	Identities     []string   `json:"identities,omitempty"`
	Description    string     `json:"description,omitempty"`
	Hash           string     `json:"hash,omitempty"`
	Created        *time.Time `json:"created,omitempty"`
	Updated        *time.Time `json:"updated,omitempty"`
}

func PMSIdentityListResourceFactory() resource.Resource {
	return &pms_identity_listResource{}
}

type pms_identity_listResource struct {
	commonResource
}

func (r *pms_identity_listResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_pms_identity_list"
}

func (r *pms_identity_listResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The Policy Manager Identity List resource allows you to manage identity lists in the Policy Manager.",
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				Description:   "Unique ID of the identity list",
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
				Required:    true,
				Description: "The name of the identity list",
			},
			"description": &schema.StringAttribute{
				Optional:    true,
				Description: "A description of the identity list",
			},
			"identities": &schema.ListAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "Array of identity IDs/DIDs to include in this identity list",
			},
			"applied_version": &schema.StringAttribute{
				Computed:    true,
				Description: "The currently applied version of the identity list",
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

func (r *pms_identity_listResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.commonResource.Configure(ctx, req, resp)
}

func (r *pms_identity_listResource) apiGetPath(data *PMSIdentityListResourceModel, idOrName string) string {
	return r.apiPath(data, idOrName) + "?withActive=true"
}

func (r *pms_identity_listResource) apiPath(data *PMSIdentityListResourceModel, idOrName string) string {
	env := data.Environment.ValueString()
	service := data.Service.ValueString()
	return fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/identity-lists/%s", env, service, idOrName)
}

func (r *pms_identity_listResource) apiIdentityListVersionPath(data *PMSIdentityListResourceModel, idOrName string) string {
	env := data.Environment.ValueString()
	service := data.Service.ValueString()
	return fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/identity-lists/%s/versions", env, service, idOrName)
}

func (r *pms_identity_listResource) toAPI(data *PMSIdentityListResourceModel, api *PMSIdentityListAPIModel) {
	api.Name = data.Name.ValueString()
	api.Description = data.Description.ValueString()
}

func (r *pms_identity_listResource) toData(api *PMSIdentityListAPIModel, data *PMSIdentityListResourceModel, diagnostics *diag.Diagnostics) {
	data.ID = types.StringValue(api.ID)
	data.Name = types.StringValue(api.Name)
	data.AppliedVersion = types.StringValue(api.CurrentVersion)

	// Note: environment and service are not returned by the API, they remain as set in the resource

	data.Description = types.StringValue(api.Description)
	data.Created = types.StringValue(api.Created.Format(time.RFC3339))
	data.Updated = types.StringValue(api.Updated.Format(time.RFC3339))

	if !r.identitiesEqual(data.Identities, api.Identities) {
		diagnostics.AddError("Identities mismatch", "The identities in the API do not match the plan")
		return
	} //else leave the identities in the data as they are which matches the current state of the plan

}

func (r *pms_identity_listResource) toVersionAPI(data *PMSIdentityListResourceModel, versionAPI *PMSIdentityListVersionAPIModel, diagnostics *diag.Diagnostics) bool {
	// Convert identities list to string slice
	identities := make([]string, 0, len(data.Identities.Elements()))
	for _, identity := range data.Identities.Elements() {
		if identity.IsNull() || identity.IsUnknown() {
			continue
		}
		identities = append(identities, identity.(types.String).ValueString())
	}

	versionAPI.Identities = identities
	versionAPI.Description = data.Description.ValueString()

	return true
}

func (r *pms_identity_listResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PMSIdentityListResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var api PMSIdentityListAPIModel
	r.toAPI(&data, &api)

	// Create the identity list
	ok, _ := r.apiRequest(ctx, http.MethodPut, r.apiPath(&data, api.Name), &api, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	// Create the initial version with identities
	var versionAPI PMSIdentityListVersionAPIModel
	ok = r.toVersionAPI(&data, &versionAPI, &resp.Diagnostics)
	if !ok {
		return
	}

	ok, _ = r.apiRequest(ctx, http.MethodPost, r.apiIdentityListVersionPath(&data, api.Name), &versionAPI, &versionAPI, &resp.Diagnostics)
	if !ok {
		return
	}

	// Fetch the updated identity list with the current version
	var updatedAPI PMSIdentityListAPIModel
	getPath := r.apiGetPath(&data, api.Name)
	ok, _ = r.apiRequest(ctx, http.MethodGet, getPath, nil, &updatedAPI, &resp.Diagnostics)
	if !ok {
		return
	}

	r.toData(&updatedAPI, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *pms_identity_listResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PMSIdentityListResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var api PMSIdentityListAPIModel
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

func (r *pms_identity_listResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data PMSIdentityListResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var api PMSIdentityListAPIModel
	r.toAPI(&data, &api)
	identityListID := data.ID.ValueString()

	// Update the identity list metadata
	ok, _ := r.apiRequest(ctx, http.MethodPatch, r.apiPath(&data, identityListID), &api, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	// Create a new version with updated identities
	var versionAPI PMSIdentityListVersionAPIModel
	ok = r.toVersionAPI(&data, &versionAPI, &resp.Diagnostics)
	if !ok {
		return
	}

	ok, _ = r.apiRequest(ctx, http.MethodPost, r.apiIdentityListVersionPath(&data, identityListID), &versionAPI, &versionAPI, &resp.Diagnostics)
	if !ok {
		return
	}

	// Fetch the updated identity list with the current version
	var updatedAPI PMSIdentityListAPIModel
	getPath := r.apiGetPath(&data, identityListID)
	ok, _ = r.apiRequest(ctx, http.MethodGet, getPath, nil, &updatedAPI, &resp.Diagnostics)
	if !ok {
		return
	}

	r.toData(&updatedAPI, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *pms_identity_listResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PMSIdentityListResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data, data.ID.ValueString()), nil, nil, &resp.Diagnostics, Allow404())
	r.waitForRemoval(ctx, r.apiPath(&data, data.ID.ValueString()), &resp.Diagnostics)
}

func (r *pms_identity_listResource) identitiesEqual(planIdentities types.List, updatedIdentitiesFromAPI []string) bool {
	if len(planIdentities.Elements()) != len(updatedIdentitiesFromAPI) {
		fmt.Println("Identities mismatch: length of planIdentities does not match length of updatedIdentitiesFromAPI")
		return false
	}
	sortedPlanIdentities := planIdentities.Elements()
	sortedUpdatedIdentitiesFromAPI := updatedIdentitiesFromAPI
	sort.Slice(sortedPlanIdentities, func(i, j int) bool {
		return sortedPlanIdentities[i].String() < sortedPlanIdentities[j].String()
	})
	sort.Slice(sortedUpdatedIdentitiesFromAPI, func(i, j int) bool {
		return sortedUpdatedIdentitiesFromAPI[i] < sortedUpdatedIdentitiesFromAPI[j]
	})
	for i, identity := range sortedPlanIdentities {
		if identity.(types.String).ValueString() != sortedUpdatedIdentitiesFromAPI[i] {
			fmt.Println("Identities mismatch: identity", identity.(types.String).ValueString(), "does not match updatedIdentitiesFromAPI[", i, "]", sortedUpdatedIdentitiesFromAPI[i])
			return false
		}
	}
	return true
}
