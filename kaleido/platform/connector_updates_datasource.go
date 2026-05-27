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

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ConnectorUpdatesDatasourceModel struct {
	Environment types.String `tfsdk:"environment"`
	Service     types.String `tfsdk:"service"`
	Updates     types.List   `tfsdk:"updates"`
}

type ConnectorUpdateAPIModel struct {
	Type           string `json:"type,omitempty"`
	Name           string `json:"name,omitempty"`
	ID             string `json:"id,omitempty"`
	ReferenceName  string `json:"referenceName,omitempty"`
	CurrentVersion string `json:"currentVersion,omitempty"`
	LatestVersion  string `json:"latestVersion,omitempty"`
	Applicable     bool   `json:"applicable,omitempty"`
	Reason         string `json:"reason,omitempty"`
	Message        string `json:"message,omitempty"`
}

func ConnectorUpdatesDatasourceFactory() datasource.DataSource {
	return &ConnectorUpdatesDatasource{}
}

type ConnectorUpdatesDatasource struct {
	commonDataSource
}

func (s *ConnectorUpdatesDatasource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_connector_updates"
}

func (s *ConnectorUpdatesDatasource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Returns the list of pending template upgrades on a connector service (config types, connector flows, standard APIs whose templates have a newer version than what's deployed). Useful for surfacing drift in CI.",
		Attributes: map[string]schema.Attribute{
			"environment": &schema.StringAttribute{
				Required:    true,
				Description: "Environment ID",
			},
			"service": &schema.StringAttribute{
				Required:    true,
				Description: "Connector service ID",
			},
			"updates": &schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of pending updates as reported by the connector manager.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type":            &schema.StringAttribute{Computed: true, Description: "Resource type: config_type, connector_flow, standard_api"},
						"name":            &schema.StringAttribute{Computed: true},
						"id":              &schema.StringAttribute{Computed: true},
						"reference_name":  &schema.StringAttribute{Computed: true},
						"current_version": &schema.StringAttribute{Computed: true},
						"latest_version":  &schema.StringAttribute{Computed: true},
						"applicable":      &schema.BoolAttribute{Computed: true, Description: "True if the upgrade can be applied automatically. False means manual intervention is required."},
						"reason":          &schema.StringAttribute{Computed: true},
						"message":         &schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (s *ConnectorUpdatesDatasource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ConnectorUpdatesDatasourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	path := fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/metadata/updates",
		data.Environment.ValueString(), data.Service.ValueString())

	var updates []ConnectorUpdateAPIModel
	ok, _ := s.apiRequest(ctx, http.MethodGet, path, nil, &updates, &resp.Diagnostics)
	if !ok {
		return
	}

	objectType := types.ObjectType{AttrTypes: connectorUpdateObjectAttrTypes()}
	items := make([]attr.Value, 0, len(updates))
	for _, u := range updates {
		obj, diags := types.ObjectValue(objectType.AttrTypes, map[string]attr.Value{
			"type":            types.StringValue(u.Type),
			"name":            types.StringValue(u.Name),
			"id":              types.StringValue(u.ID),
			"reference_name":  types.StringValue(u.ReferenceName),
			"current_version": types.StringValue(u.CurrentVersion),
			"latest_version":  types.StringValue(u.LatestVersion),
			"applicable":      types.BoolValue(u.Applicable),
			"reason":          types.StringValue(u.Reason),
			"message":         types.StringValue(u.Message),
		})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		items = append(items, obj)
	}
	listValue, diags := types.ListValue(objectType, items)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Updates = listValue
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func connectorUpdateObjectAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"type":            types.StringType,
		"name":            types.StringType,
		"id":              types.StringType,
		"reference_name":  types.StringType,
		"current_version": types.StringType,
		"latest_version":  types.StringType,
		"applicable":      types.BoolType,
		"reason":          types.StringType,
		"message":         types.StringType,
	}
}
