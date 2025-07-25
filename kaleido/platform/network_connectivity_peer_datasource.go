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

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type NetworkConnectivityPeerDataSourceModel struct {
	Identity  types.String `tfsdk:"identity"`
	Nat       types.String `tfsdk:"nat"`
	Endpoints types.List   `tfsdk:"endpoints"`
	JSON      types.String `tfsdk:"json"`
}

type EndpointModel struct {
	Host     types.String `tfsdk:"host"`
	Port     types.Int64  `tfsdk:"port"`
	Protocol types.String `tfsdk:"protocol"`
	Nat      types.String `tfsdk:"nat"`
}

type PeerData struct {
	Identity  string         `json:"identity"`
	Endpoints []EndpointData `json:"endpoints"`
}

type EndpointData struct {
	Host     string `json:"host"`
	Port     int64  `json:"port"`
	Protocol string `json:"protocol"`
	Nat      string `json:"nat"`
}

func NetworkConnectivityPeerDataSourceFactory() datasource.DataSource {
	return &networkConnectivityPeerDataSource{}
}

type networkConnectivityPeerDataSource struct{}

func (d *networkConnectivityPeerDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_platform_network_connectivity_peer"
}

func (d *networkConnectivityPeerDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Datasource for network connectivity peer information",
		Attributes: map[string]schema.Attribute{
			"identity": schema.StringAttribute{
				MarkdownDescription: "The identity of the peer",
				Required:            true,
			},
			"nat": schema.StringAttribute{
				MarkdownDescription: "The NAT configuration for the peer",
				Optional:            true,
			},
			"endpoints": schema.ListNestedAttribute{
				MarkdownDescription: "List of endpoints for the peer",
				Required:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"host": schema.StringAttribute{
							MarkdownDescription: "The host address",
							Required:            true,
						},
						"port": schema.Int64Attribute{
							MarkdownDescription: "The port number",
							Required:            true,
						},
						"protocol": schema.StringAttribute{
							MarkdownDescription: "The protocol (e.g., TCP)",
							Required:            true,
						},
						"nat": schema.StringAttribute{
							MarkdownDescription: "The NAT setting for this endpoint",
							Required:            true,
						},
					},
				},
			},
			"json": schema.StringAttribute{
				MarkdownDescription: "JSON representation of the peer data",
				Computed:            true,
			},
		},
	}
}

func (d *networkConnectivityPeerDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data NetworkConnectivityPeerDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	// Convert endpoints from terraform list to EndpointData slice
	var endpoints []EndpointModel
	resp.Diagnostics.Append(data.Endpoints.ElementsAs(ctx, &endpoints, false)...)

	var endpointData []EndpointData
	for _, ep := range endpoints {
		endpointData = append(endpointData, EndpointData{
			Host:     ep.Host.ValueString(),
			Port:     ep.Port.ValueInt64(),
			Protocol: ep.Protocol.ValueString(),
			Nat:      ep.Nat.ValueString(),
		})
	}

	// Create the peer data structure
	peerData := PeerData{
		Identity:  data.Identity.ValueString(),
		Endpoints: endpointData,
	}

	// Convert to JSON
	jsonBytes, err := json.Marshal(peerData)
	if err != nil {
		resp.Diagnostics.AddError("JSON Marshalling Error", "Failed to marshal peer data to JSON: "+err.Error())
		return
	}

	data.JSON = types.StringValue(string(jsonBytes))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
