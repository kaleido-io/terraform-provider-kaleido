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

type ConnectorStandardStreamResourceModel struct {
	ID                    types.String `tfsdk:"id"`
	Environment           types.String `tfsdk:"environment"`
	Service               types.String `tfsdk:"service"`
	Name                  types.String `tfsdk:"name"`
	Description           types.String `tfsdk:"description"`
	ConfigProfileNameOrID types.String `tfsdk:"config_profile_name_or_id"`
}

type ConnectorStandardStreamDeployAPIModel struct {
	Name                  string `json:"name,omitempty"`
	Description           string `json:"description,omitempty"`
	ConfigProfileNameOrID string `json:"configProfileNameOrId,omitempty"`
}

type ConnectorStandardStreamAPIModel struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

func ConnectorStandardStreamResourceFactory() resource.Resource {
	return &connectorStandardStreamResource{}
}

type connectorStandardStreamResource struct {
	commonResource
}

func (r *connectorStandardStreamResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_connector_standard_stream"
}

func (r *connectorStandardStreamResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Deploys a standard stream from a connector service's embedded definitions, optionally binding it to a config profile.",
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
				Description:   "Standard stream template name (e.g. newBlocks)",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"description": &schema.StringAttribute{
				Optional:    true,
				Description: "Optional description override.",
			},
			"config_profile_name_or_id": &schema.StringAttribute{
				Optional:    true,
				Description: "Name or ID of the config profile to bind to the stream (must be of the config type expected by the stream's factory).",
			},
		},
	}
}

func (r *connectorStandardStreamResource) metadataPath(data *ConnectorStandardStreamResourceModel, suffix string) string {
	return fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/metadata/standard-streams/%s%s",
		data.Environment.ValueString(), data.Service.ValueString(), data.Name.ValueString(), suffix)
}

func (r *connectorStandardStreamResource) instancePath(data *ConnectorStandardStreamResourceModel) string {
	return fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/streams/%s",
		data.Environment.ValueString(), data.Service.ValueString(), data.Name.ValueString())
}

func (r *connectorStandardStreamResource) deployBody(data *ConnectorStandardStreamResourceModel) *ConnectorStandardStreamDeployAPIModel {
	body := &ConnectorStandardStreamDeployAPIModel{Name: data.Name.ValueString()}
	if !data.Description.IsNull() {
		body.Description = data.Description.ValueString()
	}
	if !data.ConfigProfileNameOrID.IsNull() {
		body.ConfigProfileNameOrID = data.ConfigProfileNameOrID.ValueString()
	}
	return body
}

func (r *connectorStandardStreamResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ConnectorStandardStreamResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var api ConnectorStandardStreamAPIModel
	ok, _ := r.apiRequest(ctx, http.MethodPost, r.metadataPath(&data, "/deploy"), r.deployBody(&data), &api, &resp.Diagnostics)
	if !ok {
		return
	}
	if api.ID != "" {
		data.ID = types.StringValue(api.ID)
	} else {
		data.ID = types.StringValue(api.Name)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *connectorStandardStreamResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ConnectorStandardStreamResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var api ConnectorStandardStreamAPIModel
	ok, status := r.apiRequest(ctx, http.MethodGet, r.instancePath(&data), nil, &api, &resp.Diagnostics, Allow404())
	if !ok {
		return
	}
	if status == 404 {
		resp.State.RemoveResource(ctx)
		return
	}
	if api.ID != "" {
		data.ID = types.StringValue(api.ID)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *connectorStandardStreamResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ConnectorStandardStreamResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// /deploy is idempotent; re-deploy with updated binding/description.
	var api ConnectorStandardStreamAPIModel
	ok, _ := r.apiRequest(ctx, http.MethodPost, r.metadataPath(&data, "/deploy"), r.deployBody(&data), &api, &resp.Diagnostics)
	if !ok {
		return
	}
	if api.ID != "" {
		data.ID = types.StringValue(api.ID)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *connectorStandardStreamResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ConnectorStandardStreamResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.apiRequest(ctx, http.MethodDelete, r.instancePath(&data), nil, nil, &resp.Diagnostics, Allow404())
}
