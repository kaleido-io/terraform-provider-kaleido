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
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// GrantResourceModel is a grant: one permission set given to one user group or application, with
// a value for each of the set's params.
type GrantResourceModel struct {
	ID                       types.String    `tfsdk:"id"`
	Name                     types.String    `tfsdk:"name"`
	GroupID                  types.String    `tfsdk:"group_id"`
	ApplicationID            types.String    `tfsdk:"application_id"`
	PermissionSetID          types.String    `tfsdk:"permission_set_id"`
	PermissionSetJSON        jsonStringValue `tfsdk:"permission_set_json"`
	ParamsJSON               jsonStringValue `tfsdk:"params_json"`
	Labels                   types.Map       `tfsdk:"labels"`
	EffectivePermissionsJSON jsonStringValue `tfsdk:"effective_permissions_json"`
	Warnings                 types.List      `tfsdk:"warnings"`
}

type GrantAPIModel struct {
	ID                   string            `json:"id,omitempty"`
	Created              *time.Time        `json:"created,omitempty"`
	Updated              *time.Time        `json:"updated,omitempty"`
	Name                 *string           `json:"name,omitempty"`
	GroupID              string            `json:"groupId,omitempty"`
	ApplicationID        string            `json:"applicationId,omitempty"`
	PermissionSetID      string            `json:"permissionSetId,omitempty"`
	PermissionSet        json.RawMessage   `json:"permissionSet,omitempty"`
	Params               json.RawMessage   `json:"params,omitempty"`
	Labels               map[string]string `json:"labels"` // always sent: an empty map clears them, as PATCH merges
	EffectivePermissions json.RawMessage   `json:"effectivePermissions,omitempty"`
	Warnings             []string          `json:"warnings,omitempty"`
}

// GrantPatchAPIModel is what can change on a grant: its subject and its set can't.
type GrantPatchAPIModel struct {
	Name   *string           `json:"name,omitempty"`
	Params json.RawMessage   `json:"params,omitempty"`
	Labels map[string]string `json:"labels"` // always sent: an empty map clears them, as PATCH merges
}

func GrantResourceFactory() resource.Resource {
	return &grantResource{}
}

type grantResource struct {
	commonResource
}

func (r *grantResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_grant"
}

func (r *grantResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A grant: a permission set given to a user group or an application, with a value for each of the set's params. Needs access grants on (the access_grants toggle).",
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": &schema.StringAttribute{
				Required:    true,
				Description: "Name of the grant, unique in the account",
			},
			"group_id": &schema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "User group the grant is for. Specify either group_id or application_id",
			},
			"application_id": &schema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Application the grant is for. Specify either group_id or application_id",
			},
			"permission_set_id": &schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplaceIfConfigured(),
				},
				Description: "Permission set the grant gives: a set's id, or a built-in set's id. Specify either permission_set_id or permission_set_json. With permission_set_json, the id of the grant's inline set",
			},
			"permission_set_json": &schema.StringAttribute{
				CustomType:    jsonStringType{},
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "An inline permission set of the grant's own (use jsonencode), in place of permission_set_id: {permissions, params}",
			},
			"params_json": &schema.StringAttribute{
				CustomType:  jsonStringType{},
				Optional:    true,
				Description: "A value for each param the set declares (use jsonencode): a string fills an eq condition, a list of strings an in condition",
			},
			"labels": &schema.MapAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Labels on the grant",
			},
			"effective_permissions_json": &schema.StringAttribute{
				CustomType:  jsonStringType{},
				Computed:    true,
				Description: "The set's permissions with this grant's params filled in",
			},
			"warnings": &schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "Where the grant's permissions give nothing yet, and why. The grant is saved all the same",
			},
		},
	}
}

func (r *grantResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.ExactlyOneOf(path.MatchRoot("group_id"), path.MatchRoot("application_id")),
		resourcevalidator.ExactlyOneOf(path.MatchRoot("permission_set_id"), path.MatchRoot("permission_set_json")),
	}
}

func (data *GrantResourceModel) toAPI(ctx context.Context, api *GrantAPIModel, diagnostics *diag.Diagnostics) {
	name := data.Name.ValueString()
	api.Name = &name
	api.GroupID = data.GroupID.ValueString()
	api.ApplicationID = data.ApplicationID.ValueString()
	if !data.PermissionSetJSON.IsNull() {
		api.PermissionSet = jsonToAPI(data.PermissionSetJSON)
	} else {
		api.PermissionSetID = data.PermissionSetID.ValueString()
	}
	api.Params = jsonToAPI(data.ParamsJSON)
	api.Labels = labelsToAPI(ctx, data.Labels, diagnostics)
}

func (data *GrantResourceModel) toPatchAPI(ctx context.Context, diagnostics *diag.Diagnostics) *GrantPatchAPIModel {
	name := data.Name.ValueString()
	params := jsonToAPI(data.ParamsJSON)
	if params == nil {
		params = json.RawMessage(`{}`)
	}
	return &GrantPatchAPIModel{
		Name:   &name,
		Params: params,
		Labels: labelsToAPI(ctx, data.Labels, diagnostics),
	}
}

// toData sets the attributes from the API. What's configured is set only on a refresh, as for a
// permission set; the computed attributes always. The inline set isn't returned by the API, so
// permission_set_json keeps what's configured.
func (api *GrantAPIModel) toData(ctx context.Context, data *GrantResourceModel, refresh bool, diagnostics *diag.Diagnostics) {
	data.ID = types.StringValue(api.ID)
	data.PermissionSetID = types.StringValue(api.PermissionSetID)
	data.EffectivePermissionsJSON = jsonFromAPI(api.EffectivePermissions)
	if refresh {
		if api.Name != nil {
			data.Name = types.StringValue(*api.Name)
		}
		if api.GroupID != "" {
			data.GroupID = types.StringValue(api.GroupID)
		}
		if api.ApplicationID != "" {
			data.ApplicationID = types.StringValue(api.ApplicationID)
		}
		data.ParamsJSON = jsonObjectFromAPI(jsonEncoded(api.Params))
		data.Labels = labelsFromAPI(ctx, api.Labels, diagnostics)
	}
	if api.Warnings != nil || data.Warnings.IsUnknown() || data.Warnings.IsNull() {
		warnings, d := types.ListValueFrom(ctx, types.StringType, api.Warnings)
		diagnostics.Append(d...)
		data.Warnings = warnings
	}
}

// warn shows a grant's warnings: its permissions that give nothing yet, and why.
func (api *GrantAPIModel) warn(diagnostics *diag.Diagnostics) {
	for _, w := range api.Warnings {
		diagnostics.AddWarning(fmt.Sprintf("Grant %q gives nothing yet", api.ID), w)
	}
}

func (r *grantResource) apiPath(data *GrantResourceModel) string {
	path := "/api/v1/grants"
	if data.ID.ValueString() != "" {
		path = path + "/" + url.PathEscape(data.ID.ValueString())
	}
	return path
}

func (r *grantResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data GrantResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var api GrantAPIModel
	data.toAPI(ctx, &api, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if ok, _ := r.apiRequest(ctx, http.MethodPost, r.apiPath(&data), api, &api, &resp.Diagnostics); !ok {
		return
	}

	api.warn(&resp.Diagnostics)
	api.toData(ctx, &data, false, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *grantResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data GrantResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	patch := data.toPatchAPI(ctx, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	var api GrantAPIModel
	if ok, _ := r.apiRequest(ctx, http.MethodPatch, r.apiPath(&data), patch, &api, &resp.Diagnostics); !ok {
		return
	}

	api.warn(&resp.Diagnostics)
	api.toData(ctx, &data, false, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *grantResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data GrantResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var api GrantAPIModel
	ok, status := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics, Allow404())
	if !ok {
		return
	}
	if status == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	api.toData(ctx, &data, true, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *grantResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data GrantResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data), nil, nil, &resp.Diagnostics, Allow404())

	r.waitForRemoval(ctx, r.apiPath(&data), &resp.Diagnostics)
}
