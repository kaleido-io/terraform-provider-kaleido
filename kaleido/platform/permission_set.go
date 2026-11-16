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
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// PermissionSetResourceModel is a permission set: a named list of permissions, each actions and
// a resource selector, that grants give to groups and applications. Its permissions and params
// are JSON in the API's own shape, checked by Access Manager against each service's RBAC schema.
type PermissionSetResourceModel struct {
	ID              types.String    `tfsdk:"id"`
	Name            types.String    `tfsdk:"name"`
	Description     types.String    `tfsdk:"description"`
	PermissionsJSON jsonStringValue `tfsdk:"permissions_json"`
	ParamsJSON      jsonStringValue `tfsdk:"params_json"`
	Labels          types.Map       `tfsdk:"labels"`
}

type PermissionSetAPIModel struct {
	ID          string            `json:"id,omitempty"`
	Created     *time.Time        `json:"created,omitempty"`
	Updated     *time.Time        `json:"updated,omitempty"`
	Name        *string           `json:"name,omitempty"`
	Description *string           `json:"description,omitempty"`
	Builtin     *bool             `json:"builtin,omitempty"`
	Inline      *bool             `json:"inline,omitempty"`
	Params      json.RawMessage   `json:"params,omitempty"`
	Permissions json.RawMessage   `json:"permissions,omitempty"`
	Labels      map[string]string `json:"labels"` // always sent: an empty map clears them, as PATCH merges
}

func PermissionSetResourceFactory() resource.Resource {
	return &permissionSetResource{}
}

type permissionSetResource struct {
	commonResource
}

func (r *permissionSetResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_permission_set"
}

func (r *permissionSetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A permission set: permissions (actions and a resource selector) that grants give to user groups and applications. Its types, fields and actions come from each service's RBAC schema and are checked by Access Manager. Needs access grants on (the access_grants toggle).",
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": &schema.StringAttribute{
				Required:    true,
				Description: "Name of the permission set, unique in the account",
			},
			"description": &schema.StringAttribute{
				Optional:    true,
				Description: "Description of the permission set",
			},
			"permissions_json": &schema.StringAttribute{
				CustomType:  jsonStringType{},
				Required:    true,
				Description: "The permissions (use jsonencode): a list of {actions, resourceSelector}, each condition of a selector {field, op, values} or {field, op, param}",
			},
			"params_json": &schema.StringAttribute{
				CustomType:  jsonStringType{},
				Optional:    true,
				Description: "The params the set declares (use jsonencode): a list of {name, description}, each filled by the grants of the set",
			},
			"labels": &schema.MapAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Labels on the permission set",
			},
		},
	}
}

// toAPI is the full body of the set. PATCH merges what it's sent, so every field is sent, with an
// empty value to clear one.
func (data *PermissionSetResourceModel) toAPI(ctx context.Context, api *PermissionSetAPIModel, diagnostics *diag.Diagnostics) {
	name := data.Name.ValueString()
	api.Name = &name
	description := data.Description.ValueString()
	api.Description = &description
	api.Permissions = jsonToAPI(data.PermissionsJSON)
	api.Params = jsonToAPI(data.ParamsJSON)
	if api.Params == nil {
		api.Params = json.RawMessage(`[]`)
	}
	api.Labels = labelsToAPI(ctx, data.Labels, diagnostics)
}

// toData sets the attributes from the API. What's configured is set only on a refresh: after a
// create or an update it keeps the configured value, which the API has accepted, and a refresh
// compares the API's (the JSON by meaning).
func (api *PermissionSetAPIModel) toData(ctx context.Context, data *PermissionSetResourceModel, refresh bool, diagnostics *diag.Diagnostics) {
	data.ID = types.StringValue(api.ID)
	if !refresh {
		return
	}
	if api.Name != nil {
		data.Name = types.StringValue(*api.Name)
	}
	if api.Description != nil && *api.Description != "" {
		data.Description = types.StringValue(*api.Description)
	} else {
		data.Description = types.StringNull()
	}
	data.PermissionsJSON = jsonFromAPI(jsonEncoded(api.Permissions))
	data.ParamsJSON = jsonObjectFromAPI(jsonEncoded(api.Params))
	data.Labels = labelsFromAPI(ctx, api.Labels, diagnostics)
}

func (r *permissionSetResource) apiPath(data *PermissionSetResourceModel) string {
	path := "/api/v1/permissionsets"
	if data.ID.ValueString() != "" {
		path = path + "/" + url.PathEscape(data.ID.ValueString())
	}
	return path
}

func (r *permissionSetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PermissionSetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var api PermissionSetAPIModel
	data.toAPI(ctx, &api, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if ok, _ := r.apiRequest(ctx, http.MethodPost, r.apiPath(&data), api, &api, &resp.Diagnostics); !ok {
		return
	}

	api.toData(ctx, &data, false, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *permissionSetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data PermissionSetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var api PermissionSetAPIModel
	data.toAPI(ctx, &api, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if ok, _ := r.apiRequest(ctx, http.MethodPatch, r.apiPath(&data), api, &api, &resp.Diagnostics); !ok {
		return
	}

	api.toData(ctx, &data, false, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *permissionSetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PermissionSetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var api PermissionSetAPIModel
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

func (r *permissionSetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PermissionSetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data), nil, nil, &resp.Diagnostics, Allow404())

	r.waitForRemoval(ctx, r.apiPath(&data), &resp.Diagnostics)
}

// jsonEncoded is JSON from the API as jsonencode writes it: compact, keys sorted. A refresh keeps
// the configured text when it means the same, but an import has none to keep, and would otherwise
// plan an update for keys the API returns in another order.
func jsonEncoded(raw json.RawMessage) json.RawMessage {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value interface{}
	if err := decoder.Decode(&value); err != nil {
		return raw
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return raw
	}
	return encoded
}

// labelsToAPI is a labels attribute as the API takes it: an empty map when it's unset, so a PATCH
// clears labels removed from the configuration.
func labelsToAPI(ctx context.Context, labels types.Map, diagnostics *diag.Diagnostics) map[string]string {
	out := map[string]string{}
	if !labels.IsNull() && !labels.IsUnknown() {
		diagnostics.Append(labels.ElementsAs(ctx, &out, false)...)
	}
	return out
}

// labelsFromAPI is the API's labels as the attribute: unset when there are none.
func labelsFromAPI(ctx context.Context, labels map[string]string, diagnostics *diag.Diagnostics) types.Map {
	if len(labels) == 0 {
		return types.MapNull(types.StringType)
	}
	m, d := types.MapValueFrom(ctx, types.StringType, labels)
	diagnostics.Append(d...)
	return m
}
