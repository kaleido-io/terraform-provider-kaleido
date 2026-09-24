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
	"net/http"
	"regexp"
	"testing"

	"github.com/gorilla/mux"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// mockCatalog mirrors the shape of the real catalog: ethereum's ecosystem defaults, a testnet that
// overrides them, and a mainnet that does not. The ecosystem also carries a second config type so a
// test can show a network's values replace the ecosystem's wholesale rather than merge into them.
var mockCatalog = map[string]struct {
	defaults map[string]interface{}
	networks []catalogEcosystemNetworkAPIModel
}{
	"ethereum": {
		defaults: map[string]interface{}{
			"evm.confirmations": map[string]interface{}{"count": 12, "resubmission": map[string]interface{}{"enabled": true}},
			"evm.gasEstimation": map[string]interface{}{"scaleFactor": 1.5},
		},
		networks: []catalogEcosystemNetworkAPIModel{
			{Name: "ethereum-mainnet"},
			{Name: "ethereum-sepolia-testnet", ConfigProfiles: map[string]interface{}{
				"evm.confirmations": map[string]interface{}{"count": 6, "resubmission": map[string]interface{}{"enabled": true}},
			}},
		},
	},
	"besu": {defaults: map[string]interface{}{"evm.confirmations": map[string]interface{}{"count": 0}}},
}

func catalogDefaultsConfig(ecosystem, network string) string {
	n := ""
	if network != "" {
		n = `  network   = "` + network + `"` + "\n"
	}
	return `
data "kaleido_platform_catalog_web3ecosystem_defaults" "this" {
  ecosystem = "` + ecosystem + `"
` + n + `}
`
}

const catalogDS = "data.kaleido_platform_catalog_web3ecosystem_defaults.this"

func TestCatalogWeb3EcosystemDefaults(t *testing.T) {
	for _, tc := range []struct {
		name, ecosystem, network string
		check                    resource.TestCheckFunc
	}{
		{
			name: "ecosystem only", ecosystem: "ethereum",
			check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(catalogDS, "config_profiles.evm.confirmations", `{"count":12,"resubmission":{"enabled":true}}`),
				resource.TestCheckResourceAttr(catalogDS, "config_profiles.evm.gasEstimation", `{"scaleFactor":1.5}`),
				resource.TestCheckNoResourceAttr(catalogDS, "network_config_profiles.%"),
			),
		},
		{
			// The network's values replace the ecosystem's wholesale: gasEstimation, which only the
			// ecosystem sets, is absent - exactly as the platform UI resolves it.
			name: "network with overrides replaces the ecosystem's", ecosystem: "ethereum", network: "ethereum-sepolia-testnet",
			check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(catalogDS, "config_profiles.evm.confirmations", `{"count":6,"resubmission":{"enabled":true}}`),
				resource.TestCheckNoResourceAttr(catalogDS, "config_profiles.evm.gasEstimation"),
				resource.TestCheckResourceAttr(catalogDS, "ecosystem_config_profiles.evm.confirmations", `{"count":12,"resubmission":{"enabled":true}}`),
				resource.TestCheckResourceAttr(catalogDS, "network_config_profiles.evm.confirmations", `{"count":6,"resubmission":{"enabled":true}}`),
			),
		},
		{
			name: "network without overrides uses the ecosystem's", ecosystem: "ethereum", network: "ethereum-mainnet",
			check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(catalogDS, "config_profiles.evm.confirmations", `{"count":12,"resubmission":{"enabled":true}}`),
				resource.TestCheckNoResourceAttr(catalogDS, "network_config_profiles.%"),
			),
		},
		{
			name: "unknown network falls back to the ecosystem's", ecosystem: "ethereum", network: "ethereum-typo",
			check: resource.TestCheckResourceAttr(catalogDS, "config_profiles.evm.confirmations", `{"count":12,"resubmission":{"enabled":true}}`),
		},
		{
			name: "ecosystem with no networks", ecosystem: "besu",
			check: resource.TestCheckResourceAttr(catalogDS, "config_profiles.evm.confirmations", `{"count":0}`),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mp, providerConfig := testSetup(t)
			defer mp.server.Close()
			resource.Test(t, resource.TestCase{
				IsUnitTest:               true,
				ProtoV6ProviderFactories: testAccProviders,
				Steps:                    []resource.TestStep{{Config: providerConfig + catalogDefaultsConfig(tc.ecosystem, tc.network), Check: tc.check}},
			})
		})
	}
}

func TestCatalogWeb3EcosystemDefaultsUnknownEcosystem(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer mp.server.Close()
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{{
			Config:      providerConfig + catalogDefaultsConfig("etherium", ""),
			ExpectError: regexp.MustCompile(`404`),
		}},
	})
}

func (mp *mockPlatform) getCatalogWeb3EcosystemDefaults(res http.ResponseWriter, req *http.Request) {
	eco, ok := mockCatalog[mux.Vars(req)["ecosystem"]]
	if !ok {
		mp.respond(res, map[string]string{"error": "KA010000: ecosystem not found"}, http.StatusNotFound)
		return
	}
	mp.respond(res, catalogEcosystemConfigAPIModel{ConfigProfiles: eco.defaults}, http.StatusOK)
}

func (mp *mockPlatform) getCatalogWeb3EcosystemNetworks(res http.ResponseWriter, req *http.Request) {
	eco, ok := mockCatalog[mux.Vars(req)["ecosystem"]]
	if !ok {
		mp.respond(res, map[string]string{"error": "KA010000: ecosystem not found"}, http.StatusNotFound)
		return
	}
	networks := eco.networks
	if networks == nil {
		networks = []catalogEcosystemNetworkAPIModel{}
	}
	mp.respond(res, map[string]interface{}{"count": len(networks), "items": networks}, http.StatusOK)
}
