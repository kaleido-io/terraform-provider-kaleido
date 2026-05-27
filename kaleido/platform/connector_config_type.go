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

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ConnectorConfigTypeResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Environment    types.String `tfsdk:"environment"`
	Service        types.String `tfsdk:"service"`
	Name           types.String `tfsdk:"name"`
	CurrentVersion types.String `tfsdk:"current_version"`
}

type ConnectorConfigTypeAPIModel struct {
	ID      string         `json:"id,omitempty"`
	Name    string         `json:"name,omitempty"`
	Version string         `json:"version,omitempty"`
	Schema  map[string]any `json:"schema,omitempty"`
}

func ConnectorConfigTypeResourceFactory() resource.Resource {
	return &connectorConfigTypeResource{}
}

type connectorConfigTypeResource struct {
	commonResource
}

func (r *connectorConfigTypeResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_connector_config_type"
}

func (r *connectorConfigTypeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Ensures a connector config type template is deployed on the target connector service. Config types are bundled with the connector image; this resource pins them to the latest available version and re-deploys/upgrades on apply.",
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"environment": &schema.StringAttribute{
				Required:      true,
				Description:   "Environment ID",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"service": &schema.StringAttribute{
				Required:      true,
				Description:   "Connector service ID",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": &schema.StringAttribute{
				Required:      true,
				Description:   "Name of the config type (e.g. evm.confirmations, btc.feeRate)",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"current_version": &schema.StringAttribute{
				Computed:    true,
				Description: "The version of the deployed config type template as reported by the connector manager.",
			},
		},
	}
}

func (r *connectorConfigTypeResource) apiPath(data *ConnectorConfigTypeResourceModel, suffix string) string {
	return fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/metadata/config-types/%s%s",
		data.Environment.ValueString(), data.Service.ValueString(), data.Name.ValueString(), suffix)
}

func (r *connectorConfigTypeResource) deploy(ctx context.Context, data *ConnectorConfigTypeResourceModel, diagnostics *diag.Diagnostics) {
	var api ConnectorConfigTypeAPIModel
	ok, _ := r.apiRequest(ctx, http.MethodPost, r.apiPath(data, "/deploy"), map[string]any{}, &api, diagnostics)
	if !ok {
		return
	}
	data.ID = types.StringValue(api.ID)
	if api.ID == "" {
		data.ID = types.StringValue(api.Name)
	}
	data.CurrentVersion = types.StringValue(api.Version)
}

func (r *connectorConfigTypeResource) upgrade(ctx context.Context, data *ConnectorConfigTypeResourceModel, diagnostics *diag.Diagnostics) {
	var api ConnectorConfigTypeAPIModel
	ok, _ := r.apiRequest(ctx, http.MethodPost, r.apiPath(data, "/upgrade"), map[string]any{}, &api, diagnostics)
	if !ok {
		return
	}
	if api.ID != "" {
		data.ID = types.StringValue(api.ID)
	}
	data.CurrentVersion = types.StringValue(api.Version)
}

func (r *connectorConfigTypeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ConnectorConfigTypeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.deploy(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *connectorConfigTypeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ConnectorConfigTypeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var api ConnectorConfigTypeAPIModel
	ok, status := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data, ""), nil, &api, &resp.Diagnostics, Allow404())
	if !ok {
		return
	}
	if status == 404 {
		resp.State.RemoveResource(ctx)
		return
	}
	data.CurrentVersion = types.StringValue(api.Version)
	if api.ID != "" {
		data.ID = types.StringValue(api.ID)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *connectorConfigTypeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ConnectorConfigTypeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Upgrade is idempotent on the same version, so always call it on Update.
	r.upgrade(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *connectorConfigTypeResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// Config types are template-level resources bundled with the connector image.
	// The connector-manager does not currently expose a delete endpoint; removing this
	// resource from state is a no-op. See .ai/plan.md for the open verification item.
}
