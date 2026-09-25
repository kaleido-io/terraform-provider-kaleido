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
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type CatalogWeb3EcosystemDefaultsDatasourceModel struct {
	Ecosystem               types.String `tfsdk:"ecosystem"`
	Network                 types.String `tfsdk:"network"`
	ConfigProfiles          types.Map    `tfsdk:"config_profiles"`
	EcosystemConfigProfiles types.Map    `tfsdk:"ecosystem_config_profiles"`
	NetworkConfigProfiles   types.Map    `tfsdk:"network_config_profiles"`
}

type catalogEcosystemConfigAPIModel struct {
	ConfigProfiles map[string]interface{} `json:"configProfiles"`
}

type catalogEcosystemNetworkAPIModel struct {
	Name           string                 `json:"name"`
	ConfigProfiles map[string]interface{} `json:"configProfiles"`
}

type catalogEcosystemNetworksAPIModel struct {
	Items []catalogEcosystemNetworkAPIModel `json:"items"`
}

func CatalogWeb3EcosystemDefaultsDatasourceModelFactory() datasource.DataSource {
	return &catalogWeb3EcosystemDefaultsDatasource{}
}

type catalogWeb3EcosystemDefaultsDatasource struct {
	commonDataSource
}

func (s *catalogWeb3EcosystemDefaultsDatasource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_catalog_web3ecosystem_defaults"
}

func (s *catalogWeb3EcosystemDefaultsDatasource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	profiles := func(description string) schema.MapAttribute {
		return schema.MapAttribute{Computed: true, ElementType: types.StringType, Description: description}
	}
	resp.Schema = schema.Schema{
		Description: "The platform catalog's default config profile values for a web3 ecosystem, and optionally one of its networks - " +
			"the values the platform applies when a connector is set up for that chain (for example the number of confirmations to wait for). " +
			"Use them as the value_json of the connector's config profiles.",
		Attributes: map[string]schema.Attribute{
			"ecosystem": &schema.StringAttribute{
				Required:    true,
				Description: "Catalog name of the web3 ecosystem, e.g. ethereum, polygon, besu - the connector service's ecosystem name.",
			},
			"network": &schema.StringAttribute{
				Optional:    true,
				Description: "Catalog name of one of the ecosystem's networks, e.g. ethereum-sepolia-testnet - the connector service's network name. Some networks (typically testnets) override the ecosystem's values.",
			},
			"config_profiles": profiles("The defaults to use, keyed by config type (e.g. evm.confirmations), each a JSON-encoded profile value. " +
				"The network's values if it defines any, otherwise the ecosystem's: a network's values replace its ecosystem's as a whole, rather than being merged into them."),
			"ecosystem_config_profiles": profiles("The ecosystem's own defaults, keyed by config type, each a JSON-encoded profile value."),
			"network_config_profiles": profiles("The network's own values, keyed by config type, each a JSON-encoded profile value. " +
				"Null when no network is given, or the network defines none."),
		},
	}
}

func (s *catalogWeb3EcosystemDefaultsDatasource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data CatalogWeb3EcosystemDefaultsDatasourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// The catalog is platform-wide: service-manager's environment-scoped variant of these routes
	// returns exactly the same data, so it is not used - it would only make the read wait for an
	// environment ID, which is unknown on the first plan of a new environment.
	base := fmt.Sprintf("/api/v1/catalog/web3ecosystems/%s", data.Ecosystem.ValueString())

	var ecosystem catalogEcosystemConfigAPIModel
	if ok, _ := s.apiRequest(ctx, http.MethodGet, base+"/defaults", nil, &ecosystem, &resp.Diagnostics); !ok {
		return
	}
	data.EcosystemConfigProfiles = profileValuesMap(ecosystem.ConfigProfiles, &resp.Diagnostics)
	data.ConfigProfiles = data.EcosystemConfigProfiles
	data.NetworkConfigProfiles = types.MapNull(types.StringType)

	// The precedence below is the one the platform UI applies when it sets up a connector
	// (InitializeConnectorForm and ApplyConnectorUpdates): the network's profiles if it has any,
	// replacing the ecosystem's wholesale, otherwise the ecosystem's. The catalog API returns the two
	// separately and does not resolve them, so the rule is mirrored here to give every configuration
	// the same answer the UI gives.
	if !data.Network.IsNull() && !data.Network.IsUnknown() {
		var networks catalogEcosystemNetworksAPIModel
		if ok, _ := s.apiRequest(ctx, http.MethodGet, base+"/networks", nil, &networks, &resp.Diagnostics); !ok {
			return
		}
		found := false
		for _, n := range networks.Items {
			if n.Name != data.Network.ValueString() {
				continue
			}
			found = true
			if len(n.ConfigProfiles) > 0 {
				data.NetworkConfigProfiles = profileValuesMap(n.ConfigProfiles, &resp.Diagnostics)
				data.ConfigProfiles = data.NetworkConfigProfiles
			}
		}
		if !found {
			resp.Diagnostics.AddAttributeWarning(path.Root("network"), "Network not in the catalog",
				fmt.Sprintf("Network %q is not listed for ecosystem %q, so the ecosystem's defaults are used.", data.Network.ValueString(), data.Ecosystem.ValueString()))
		}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

// profileValuesMap renders each profile value as JSON, keyed by config type. Map keys are
// marshalled in sorted order, so the same catalog value always yields the same string.
func profileValuesMap(profiles map[string]interface{}, diagnostics *diag.Diagnostics) types.Map {
	values := make(map[string]attr.Value, len(profiles))
	for configType, v := range profiles {
		b, err := json.Marshal(v)
		if err != nil {
			diagnostics.AddError("Invalid catalog profile value", fmt.Sprintf("config profile value for %q could not be encoded: %s", configType, err))
			continue
		}
		values[configType] = types.StringValue(string(b))
	}
	m, d := types.MapValue(types.StringType, values)
	diagnostics.Append(d...)
	return m
}
