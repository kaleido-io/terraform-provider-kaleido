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
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ConnectorResourceModel struct {
	ID                types.String                     `tfsdk:"id"`
	Name              types.String                     `tfsdk:"name"`
	Type              types.String                     `tfsdk:"type"`
	Environment       types.String                     `tfsdk:"environment"`
	Network           types.String                     `tfsdk:"network"`
	Zone              types.String                     `tfsdk:"zone"`
	PermittedJSON     types.String                     `tfsdk:"permitted_json"`
	PlatformRequestor *RequestorPlatformConnectorModel `tfsdk:"platform_requestor"`
	PlatformAcceptor  *AcceptorPlatformConnectorModel  `tfsdk:"platform_acceptor"`
}

type RequestorPlatformConnectorModel struct {
	TargetAccountID     types.String `tfsdk:"target_account_id"`
	TargetEnvironmentID types.String `tfsdk:"target_environment_id"`
	TargetNetworkID     types.String `tfsdk:"target_network_id"`
	TargetConnectorID   types.String `tfsdk:"target_connector_id"` // computed once accepted
}

type AcceptorPlatformConnectorModel struct {
	TargetAccountID     types.String `tfsdk:"target_account_id"`
	TargetEnvironmentID types.String `tfsdk:"target_environment_id"`
	TargetNetworkID     types.String `tfsdk:"target_network_id"`
	TargetConnectorID   types.String `tfsdk:"target_connector_id"`
}

type ConnectorAPIModel struct {
	ID        string                 `json:"id,omitempty"`
	Created   *time.Time             `json:"created,omitempty"`
	Updated   *time.Time             `json:"updated,omitempty"`
	Type      string                 `json:"type"`
	Name      string                 `json:"name"`
	NetworkID string                 `json:"networkId,omitempty"`
	Zone      string                 `json:"zone,omitempty"`
	Permitted map[string]interface{} `json:"permitted,omitempty"`
	Platform  map[string]interface{} `json:"platform,omitempty"`
	Deleted   bool                   `json:"deleted,omitempty"`
	Status    string                 `json:"status,omitempty"`
}

func ConnectorResourceFactory() resource.Resource {
	return &connectorResource{}
}

type connectorResource struct {
	commonResource
}

func (r *connectorResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_network_connector"
}

func (r *connectorResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
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
			"permitted_json": &schema.StringAttribute{
				Optional: true,
			},
			"platform_requestor": &schema.SingleNestedAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.Object{objectplanmodifier.RequiresReplace()},
				Attributes: map[string]schema.Attribute{
					"target_account_id": &schema.StringAttribute{
						Required: true,
					},
					"target_environment_id": &schema.StringAttribute{
						Required: true,
					},
					"target_network_id": &schema.StringAttribute{
						Required: true,
					},
					"target_connector_id": &schema.StringAttribute{
						Computed: true,
					},
				},
			},
			"platform_acceptor": &schema.SingleNestedAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.Object{objectplanmodifier.RequiresReplace()},
				Attributes: map[string]schema.Attribute{
					"target_account_id": &schema.StringAttribute{
						Required: true,
					},
					"target_environment_id": &schema.StringAttribute{
						Required: true,
					},
					"target_network_id": &schema.StringAttribute{
						Required: true,
					},
					"target_connector_id": &schema.StringAttribute{
						Required: true,
					},
				},
			},
		},
	}
}

func (data *ConnectorResourceModel) toAPI(ctx context.Context, api *ConnectorAPIModel, diagnostics *diag.Diagnostics) {
	// required fields
	api.Type = data.Type.ValueString()
	api.Name = data.Name.ValueString()
	api.NetworkID = data.Network.ValueString()

	api.Zone = data.Zone.ValueString()

	// optional fields
	if !data.PermittedJSON.IsNull() && strings.ToLower(api.Type) != "permitted" {
		diagnostics.AddError("Invalid Permitted JSON", "Permitted JSON is only valid for Permitted connectors")
		return
	}

	if strings.ToLower(api.Type) == "permitted" {
		api.Permitted = map[string]interface{}{}
		if !data.PermittedJSON.IsNull() && data.PermittedJSON.String() != "{}" {
			_ = json.Unmarshal([]byte(data.PermittedJSON.ValueString()), &api.Permitted)
		}
	}

	if data.PlatformRequestor != nil && strings.ToLower(api.Type) != "platform" {
		diagnostics.AddError("Invalid PlatformRequestor", "PlatformRequestor is only valid for Platform connectors")
		return
	} else if data.PlatformAcceptor != nil && strings.ToLower(api.Type) != "platform" {
		diagnostics.AddError("Invalid PlatformAcceptor", "PlatformAcceptor is only valid for Platform connectors")
		return
	} else if data.PlatformRequestor == nil && data.PlatformAcceptor == nil && strings.ToLower(api.Type) == "platform" {
		diagnostics.AddError("Invalid Platform", "PlatformRequestor or PlatformAcceptor is required for Platform connectors")
		return
	} else if data.PlatformRequestor != nil && data.PlatformAcceptor != nil {
		diagnostics.AddError("Invalid Platform", "PlatformRequestor and PlatformAcceptor are mutually exclusive")
		return
	}

	if data.PlatformRequestor != nil {
		api.Platform = map[string]interface{}{
			"targetAccountId":     data.PlatformRequestor.TargetAccountID.ValueString(),
			"targetEnvironmentId": data.PlatformRequestor.TargetEnvironmentID.ValueString(),
			"targetNetworkId":     data.PlatformRequestor.TargetNetworkID.ValueString(),
		}
	} else if data.PlatformAcceptor != nil {
		api.Platform = map[string]interface{}{
			"targetAccountId":     data.PlatformAcceptor.TargetAccountID.ValueString(),
			"targetEnvironmentId": data.PlatformAcceptor.TargetEnvironmentID.ValueString(),
			"targetNetworkId":     data.PlatformAcceptor.TargetNetworkID.ValueString(),
			"targetConnectorId":   data.PlatformAcceptor.TargetConnectorID.ValueString(),
		}
	}

}

func (api *ConnectorAPIModel) toData(data *ConnectorResourceModel, diagnostics *diag.Diagnostics) {
	data.ID = types.StringValue(api.ID)
	data.Name = types.StringValue(api.Name)

	data.Network = types.StringValue(api.NetworkID)
	data.Zone = types.StringValue(api.Zone)

	// TODO what does this do ?
	info := make(map[string]attr.Value)
	for k, v := range api.Permitted {
		v, isString := v.(string)
		if isString && v != "" {
			info[k] = types.StringValue(v)
		}
	}

	if data.PlatformRequestor != nil && api.Platform != nil {
		if targetConnectorID, ok := api.Platform["targetConnectorId"]; ok {
			data.PlatformRequestor.TargetConnectorID = types.StringValue(targetConnectorID.(string))
		} else {
			data.PlatformRequestor.TargetConnectorID = types.StringNull()
		}
	}
}

func (r *connectorResource) apiPath(data *ConnectorResourceModel) string {
	path := fmt.Sprintf("/api/v1/environments/%s/networks/%s/connectors", data.Environment.ValueString(), data.Network.ValueString())
	if data.ID.ValueString() != "" {
		path = path + "/" + data.ID.ValueString()
	}
	return path
}

func (r *connectorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var data ConnectorResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	var api ConnectorAPIModel
	data.toAPI(ctx, &api, &resp.Diagnostics)
	ok, _ := r.apiRequest(ctx, http.MethodPost, r.apiPath(&data), api, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	api.toData(&data, &resp.Diagnostics) // need the ID copied over
	if data.PlatformRequestor == nil {   // requestors will not go into ready w/o being accepted
		r.waitForReadyStatus(ctx, r.apiPath(&data), &resp.Diagnostics)
		api.toData(&data, &resp.Diagnostics) // need the latest status after the readiness check completes, to extract generated values
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *connectorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

	var data ConnectorResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)

	// Read full current object
	var api ConnectorAPIModel
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

func (r *connectorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ConnectorResourceModel
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

	api.toData(&data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *connectorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ConnectorResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data), nil, nil, &resp.Diagnostics, Allow404())

	r.waitForRemoval(ctx, r.apiPath(&data), &resp.Diagnostics)
}
