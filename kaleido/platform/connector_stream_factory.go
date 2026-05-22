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

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ConnectorStreamFactoryResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Environment types.String `tfsdk:"environment"`
	Service     types.String `tfsdk:"service"`
	Name        types.String `tfsdk:"name"`
	ConfigType  types.String `tfsdk:"config_type"`
}

type ConnectorStreamFactoryAPIModel struct {
	ID         string `json:"id,omitempty"`
	Name       string `json:"name,omitempty"`
	ConfigType string `json:"configType,omitempty"`
}

func ConnectorStreamFactoryResourceFactory() resource.Resource {
	return &connectorStreamFactoryResource{}
}

type connectorStreamFactoryResource struct {
	commonResource
}

func (r *connectorStreamFactoryResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_connector_stream_factory"
}

func (r *connectorStreamFactoryResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Deploys a connector stream factory from a connector service's embedded definitions. Stream factories are referenced by standard streams and by user-managed streams that share the same event source.",
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
				Description:   "Stream factory template name (e.g. blockEvents, transactionEvents)",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"config_type": &schema.StringAttribute{
				Computed:    true,
				Description: "The config type this factory expects on its streams, as reported by the connector.",
			},
		},
	}
}

func (r *connectorStreamFactoryResource) metadataPath(data *ConnectorStreamFactoryResourceModel, suffix string) string {
	return fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/metadata/connector-stream-factories/%s%s",
		data.Environment.ValueString(), data.Service.ValueString(), data.Name.ValueString(), suffix)
}

func (r *connectorStreamFactoryResource) instancePath(data *ConnectorStreamFactoryResourceModel) string {
	return fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/stream-factories/%s",
		data.Environment.ValueString(), data.Service.ValueString(), data.Name.ValueString())
}

func (r *connectorStreamFactoryResource) toData(api *ConnectorStreamFactoryAPIModel, data *ConnectorStreamFactoryResourceModel) {
	if api.ID != "" {
		data.ID = types.StringValue(api.ID)
	} else if data.ID.IsNull() || data.ID.IsUnknown() {
		data.ID = types.StringValue(api.Name)
	}
	data.ConfigType = types.StringValue(api.ConfigType)
}

func (r *connectorStreamFactoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ConnectorStreamFactoryResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var api ConnectorStreamFactoryAPIModel
	ok, _ := r.apiRequest(ctx, http.MethodPost, r.metadataPath(&data, "/deploy"), map[string]any{}, &api, &resp.Diagnostics)
	if !ok {
		return
	}
	r.toData(&api, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *connectorStreamFactoryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ConnectorStreamFactoryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var api ConnectorStreamFactoryAPIModel
	ok, status := r.apiRequest(ctx, http.MethodGet, r.instancePath(&data), nil, &api, &resp.Diagnostics, Allow404())
	if !ok {
		return
	}
	if status == 404 {
		resp.State.RemoveResource(ctx)
		return
	}
	r.toData(&api, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *connectorStreamFactoryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ConnectorStreamFactoryResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var api ConnectorStreamFactoryAPIModel
	ok, _ := r.apiRequest(ctx, http.MethodPost, r.metadataPath(&data, "/upgrade"), map[string]any{}, &api, &resp.Diagnostics)
	if !ok {
		return
	}
	r.toData(&api, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *connectorStreamFactoryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ConnectorStreamFactoryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.apiRequest(ctx, http.MethodDelete, r.instancePath(&data), nil, nil, &resp.Diagnostics, Allow404())
}
