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

type AccountNetworkConnectorAcceptorResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Name                types.String `tfsdk:"name"`
	Environment         types.String `tfsdk:"environment"`
	Network             types.String `tfsdk:"network"`
	Zone                types.String `tfsdk:"zone"`
	TargetAccountID     types.String `tfsdk:"target_account_id"`
	TargetEnvironmentID types.String `tfsdk:"target_environment_id"`
	TargetNetworkID     types.String `tfsdk:"target_network_id"`
	TargetConnectorID   types.String `tfsdk:"target_connector_id"`
}

func AccountNetworkConnectorAcceptorResourceFactory() resource.Resource {
	return &accountNetworkConnectorAcceptorResource{}
}

type accountNetworkConnectorAcceptorResource struct {
	commonResource
}

func (r *accountNetworkConnectorAcceptorResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_account_network_connector_acceptor"
}

func (r *accountNetworkConnectorAcceptorResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Platform acceptor network connector resource",
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
			"target_account_id": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"target_environment_id": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"target_network_id": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"target_connector_id": &schema.StringAttribute{
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				MarkdownDescription: "ID of the requestor connector to accept",
			},
		},
	}
}

func (data *AccountNetworkConnectorAcceptorResourceModel) toAPI(ctx context.Context, api *ConnectorAPIModel, diagnostics *diag.Diagnostics) {
	api.Type = "platform"
	api.Name = data.Name.ValueString()
	api.NetworkID = data.Network.ValueString()
	api.Zone = data.Zone.ValueString()

	api.Platform = map[string]interface{}{
		"targetAccountId":     data.TargetAccountID.ValueString(),
		"targetEnvironmentId": data.TargetEnvironmentID.ValueString(),
		"targetNetworkId":     data.TargetNetworkID.ValueString(),
		"targetConnectorId":   data.TargetConnectorID.ValueString(),
	}
}

func (api *ConnectorAPIModel) toAcceptorData(data *AccountNetworkConnectorAcceptorResourceModel, diagnostics *diag.Diagnostics) {
	data.ID = types.StringValue(api.ID)
	data.Name = types.StringValue(api.Name)
	data.Network = types.StringValue(api.NetworkID)
	data.Zone = types.StringValue(api.Zone)

	if api.Platform != nil {
		if targetAccountID, exists := api.Platform["targetAccountId"]; exists {
			data.TargetAccountID = types.StringValue(targetAccountID.(string))
		}
		if targetEnvironmentID, exists := api.Platform["targetEnvironmentId"]; exists {
			data.TargetEnvironmentID = types.StringValue(targetEnvironmentID.(string))
		}
		if targetNetworkID, exists := api.Platform["targetNetworkId"]; exists {
			data.TargetNetworkID = types.StringValue(targetNetworkID.(string))
		}
		if targetConnectorID, exists := api.Platform["targetConnectorId"]; exists {
			data.TargetConnectorID = types.StringValue(targetConnectorID.(string))
		}
	}
}

func (r *accountNetworkConnectorAcceptorResource) apiPath(data *AccountNetworkConnectorAcceptorResourceModel) string {
	path := fmt.Sprintf("/api/v1/environments/%s/networks/%s/connectors", data.Environment.ValueString(), data.Network.ValueString())
	if data.ID.ValueString() != "" {
		path = path + "/" + data.ID.ValueString()
	}
	return path
}

func (r *accountNetworkConnectorAcceptorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AccountNetworkConnectorAcceptorResourceModel
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

	api.toAcceptorData(&data, &resp.Diagnostics)
	r.waitForReadyStatus(ctx, r.apiPath(&data), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *accountNetworkConnectorAcceptorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data AccountNetworkConnectorAcceptorResourceModel
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

	api.toAcceptorData(&data, &resp.Diagnostics)
	r.waitForReadyStatus(ctx, r.apiPath(&data), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *accountNetworkConnectorAcceptorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AccountNetworkConnectorAcceptorResourceModel
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

	api.toAcceptorData(&data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *accountNetworkConnectorAcceptorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AccountNetworkConnectorAcceptorResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data), nil, nil, &resp.Diagnostics, Allow404())
	r.waitForRemoval(ctx, r.apiPath(&data), &resp.Diagnostics)
}
