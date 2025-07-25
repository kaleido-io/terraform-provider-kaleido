// Copyright © Kaleido, Inc. 2025

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

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type PermittedNetworkConnectorResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Environment types.String `tfsdk:"environment"`
	Network     types.String `tfsdk:"network"`
	Zone        types.String `tfsdk:"zone"`
	PeersJSON   types.String `tfsdk:"peers_json"`
}

func PermittedNetworkConnectorResourceFactory() resource.Resource {
	return &permittedNetworkConnectorResource{}
}

type permittedNetworkConnectorResource struct {
	commonResource
}

func (r *permittedNetworkConnectorResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_permitted_network_connector"
}

func (r *permittedNetworkConnectorResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Permitted network connector resource",
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": &schema.StringAttribute{
				Required: true,
			},
			"environment": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"network": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"zone": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"peers_json": &schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "JSON array of peers for the permitted connector",
			},
		},
	}
}

func (data *PermittedNetworkConnectorResourceModel) toAPI(ctx context.Context, api *ConnectorAPIModel, diagnostics *diag.Diagnostics) {
	api.Type = "permitted"
	api.Name = data.Name.ValueString()
	api.NetworkID = data.Network.ValueString()
	api.Zone = data.Zone.ValueString()

	// Parse peers JSON
	api.Permitted = map[string]interface{}{}
	if !data.PeersJSON.IsNull() && data.PeersJSON.ValueString() != "" {
		var peers []interface{}
		if err := json.Unmarshal([]byte(data.PeersJSON.ValueString()), &peers); err != nil {
			diagnostics.AddError("Invalid Peers JSON", "Failed to parse peers_json: "+err.Error())
			return
		}
		api.Permitted["peers"] = peers
	}
}

func (api *ConnectorAPIModel) toPermittedData(data *PermittedNetworkConnectorResourceModel, diagnostics *diag.Diagnostics) {
	data.ID = types.StringValue(api.ID)
	data.Name = types.StringValue(api.Name)
	data.Network = types.StringValue(api.NetworkID)
	data.Zone = types.StringValue(api.Zone)

	// Convert permitted peers back to JSON
	if api.Permitted != nil {
		if peersData, exists := api.Permitted["peers"]; exists {
			peersJSON, err := json.Marshal(peersData)
			if err != nil {
				diagnostics.AddError("JSON Marshalling Error", "Failed to marshal peers to JSON: "+err.Error())
				return
			}
			data.PeersJSON = types.StringValue(string(peersJSON))
		}
	}
}

func (r *permittedNetworkConnectorResource) apiPath(data *PermittedNetworkConnectorResourceModel) string {
	path := fmt.Sprintf("/api/v1/environments/%s/networks/%s/connectors", data.Environment.ValueString(), data.Network.ValueString())
	if data.ID.ValueString() != "" {
		path = path + "/" + data.ID.ValueString()
	}
	return path
}

func (r *permittedNetworkConnectorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PermittedNetworkConnectorResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	var api ConnectorAPIModel
	data.toAPI(ctx, &api, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	ok, _ := r.apiRequest(ctx, http.MethodPost, r.apiPath(&data), api, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	api.toPermittedData(&data, &resp.Diagnostics)
	r.waitForReadyStatus(ctx, r.apiPath(&data), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *permittedNetworkConnectorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data PermittedNetworkConnectorResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)

	// Read full current object
	var api ConnectorAPIModel
	if ok, _ := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics); !ok {
		return
	}

	// Update from plan
	data.toAPI(ctx, &api, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if ok, _ := r.apiRequest(ctx, http.MethodPut, r.apiPath(&data), api, &api, &resp.Diagnostics); !ok {
		return
	}

	api.toPermittedData(&data, &resp.Diagnostics)
	r.waitForReadyStatus(ctx, r.apiPath(&data), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *permittedNetworkConnectorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PermittedNetworkConnectorResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	var api ConnectorAPIModel
	api.ID = data.ID.ValueString()
	ok, status := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics, Allow404())
	if !ok {
		return
	}
	if status == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	api.toPermittedData(&data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *permittedNetworkConnectorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PermittedNetworkConnectorResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data), nil, nil, &resp.Diagnostics, Allow404())
	r.waitForRemoval(ctx, r.apiPath(&data), &resp.Diagnostics)
}
