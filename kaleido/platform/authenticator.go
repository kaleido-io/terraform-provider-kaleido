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

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type AuthenticatorResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Type          types.String `tfsdk:"type"`
	Environment   types.String `tfsdk:"environment"`
	Network       types.String `tfsdk:"network"`
	Zone          types.String `tfsdk:"zone"`
	Conn          types.String `tfsdk:"conn"`
	PermittedJSON types.String `tfsdk:"permitted_json"`
}

type AuthenticatorAPIModel struct {
	ID         string                 `json:"id,omitempty"`
	Created    *time.Time             `json:"created,omitempty"`
	Updated    *time.Time             `json:"updated,omitempty"`
	Type       string                 `json:"type"`
	Name       string                 `json:"name"`
	NetworkID  string                 `json:"networkId,omitempty"`
	Zone       string                 `json:"zone,omitempty"`
	Connection string                 `json:"connection,omitempty"`
	Permitted  map[string]interface{} `json:"permitted,omitempty"`
	Deleted    bool                   `json:"deleted,omitempty"`
	Status     string                 `json:"status,omitempty"`
}

func AuthenticatorResourceFactory() resource.Resource {
	return &authenticatorResource{}
}

type authenticatorResource struct {
	commonResource
}

func (r *authenticatorResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_authenticator"
}

func (r *authenticatorResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "BETA resource.",
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"type": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": &schema.StringAttribute{
				Required: true,
			},
			"network": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"environment": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"zone": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"conn": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"permitted_json": &schema.StringAttribute{
				Optional: true,
			},
		},
	}
}

func (data *AuthenticatorResourceModel) toAPI(ctx context.Context, api *AuthenticatorAPIModel, diagnostics *diag.Diagnostics) {
	// required fields
	api.Type = data.Type.ValueString()
	api.Name = data.Name.ValueString()
	api.NetworkID = data.Network.ValueString()

	api.Zone = data.Zone.ValueString()
	api.Connection = data.Conn.ValueString()

	// optional fields
	api.Permitted = map[string]interface{}{}
	if !data.PermittedJSON.IsNull() && data.PermittedJSON.String() != "{}" {
		_ = json.Unmarshal([]byte(data.PermittedJSON.ValueString()), &api.Permitted)
	}

}

func (api *AuthenticatorAPIModel) toData(data *AuthenticatorResourceModel, diagnostics *diag.Diagnostics) {
	data.ID = types.StringValue(api.ID)
	data.Name = types.StringValue(api.Name)

	data.Network = types.StringValue(api.NetworkID)
	data.Zone = types.StringValue(api.Zone)
	data.Conn = types.StringValue(api.Connection)

	info := make(map[string]attr.Value)
	for k, v := range api.Permitted {
		v, isString := v.(string)
		if isString && v != "" {
			info[k] = types.StringValue(v)
		}
	}
}

func (r *authenticatorResource) apiPath(data *AuthenticatorResourceModel) string {
	path := fmt.Sprintf("/api/v1/environments/%s/networks/%s/authenticators", data.Environment.ValueString(), data.Network.ValueString())
	if data.ID.ValueString() != "" {
		path = path + "/" + data.ID.ValueString()
	}
	return path
}

func (r *authenticatorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var data AuthenticatorResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	var api AuthenticatorAPIModel
	data.toAPI(ctx, &api, &resp.Diagnostics)
	ok, _ := r.apiRequest(ctx, http.MethodPost, r.apiPath(&data), api, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	api.toData(&data, &resp.Diagnostics) // need the ID copied over
	r.waitForReadyStatus(ctx, r.apiPath(&data), &resp.Diagnostics)
	api.toData(&data, &resp.Diagnostics) // need the latest status after the readiness check completes, to extract generated values
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)

}

func (r *authenticatorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

	var data AuthenticatorResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)

	// Read full current object
	var api AuthenticatorAPIModel
	if ok, _ := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics); !ok {
		return
	}

	// Update from plan
	data.toAPI(ctx, &api, &resp.Diagnostics)
	if ok, _ := r.apiRequest(ctx, http.MethodPut, r.apiPath(&data), api, &api, &resp.Diagnostics); !ok {
		return
	}

	api.toData(&data, &resp.Diagnostics) // need the ID copied over
	r.waitForReadyStatus(ctx, r.apiPath(&data), &resp.Diagnostics)
	api.toData(&data, &resp.Diagnostics) // need the latest status after the readiness check completes, to extract generated values
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *authenticatorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AuthenticatorResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	var api AuthenticatorAPIModel
	api.ID = data.ID.ValueString()
	ok, status := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics, Allow404())
	if !ok {
		return
	}
	if status == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	api.toData(&data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *authenticatorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AuthenticatorResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data), nil, nil, &resp.Diagnostics, Allow404())

	r.waitForRemoval(ctx, r.apiPath(&data), &resp.Diagnostics)
}
