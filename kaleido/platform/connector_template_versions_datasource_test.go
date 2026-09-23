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
	"sort"
	"testing"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const templateVersionsDataSource = `
data "kaleido_platform_connector_template_versions" "submission" {
  environment = "test-env"
  service     = "test-service"
  kind        = "connector_flow"
  name        = "submission"
}
`

// latest is the newest stored version, and supported reflects the protocol's minimum supported
// version - a version below it is listed (it can still be read) but marked unsupported.
func TestConnectorTemplateVersions(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer mp.server.Close()
	mp.connectorFlowTemplateVersion = "2026.04.0"
	mp.connectorFlowMinimumSupportedVersion = "2026.04.0"

	ds := "data.kaleido_platform_connector_template_versions.submission"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + templateVersionsDataSource,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(ds, "latest", "2026.04.0"),
					resource.TestCheckResourceAttr(ds, "versions.#", "2"),
					resource.TestCheckResourceAttr(ds, "versions.0.version", "2026.04.0"),
					resource.TestCheckResourceAttr(ds, "versions.0.latest", "true"),
					resource.TestCheckResourceAttr(ds, "versions.0.supported", "true"),
					resource.TestCheckResourceAttr(ds, "versions.1.version", "2026.03.0"),
					resource.TestCheckResourceAttr(ds, "versions.1.latest", "false"),
					resource.TestCheckResourceAttr(ds, "versions.1.supported", "false"),
				),
			},
		},
	})
}

// Tracking latest: a flow whose version comes from the data source is deployed at the newest
// version, and when the connector offers a newer one the next plan shows the upgrade and the apply
// performs it - visible and reviewable, never silent.
func TestConnectorFlowTracksLatest(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer mp.server.Close()

	config := providerConfig + templateVersionsDataSource + `
resource "kaleido_platform_connector_flow" "submission" {
  environment          = "test-env"
  service              = "test-service"
  name                 = "submission"
  version              = data.kaleido_platform_connector_template_versions.submission.latest
  config_type_bindings = { "evm.gasPricing" = "gas-standard" }
}
`
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{Config: config, Check: resource.TestCheckResourceAttr(flowResource, "version", "2026.03.0")},
			{
				PreConfig:          func() { mp.connectorFlowTemplateVersion = "2026.04.0" },
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			{Config: config, Check: resource.TestCheckResourceAttr(flowResource, "version", "2026.04.0")},
		},
	})
}

func TestConnectorTemplateVersionsRejectsUnknownKind(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer mp.server.Close()

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
data "kaleido_platform_connector_template_versions" "bad" {
  environment = "test-env"
  service     = "test-service"
  kind        = "connector_flows"
  name        = "submission"
}
`,
				ExpectError: regexp.MustCompile(`(?s)kind.*value must be one of`),
			},
		},
	})
}

func (mp *mockPlatform) listConnectorFlowVersions(res http.ResponseWriter, req *http.Request) {
	var stored []string
	for v := range connectorFlowCatalogue {
		if _, ok := mp.storedFlowTemplate(v); ok {
			stored = append(stored, v)
		}
	}
	sort.Slice(stored, func(i, j int) bool {
		return version.Must(version.NewVersion(stored[i])).GreaterThan(version.Must(version.NewVersion(stored[j])))
	})
	result := make([]ConnectorTemplateVersionAPIModel, 0, len(stored))
	for _, v := range stored {
		supported := mp.connectorFlowMinimumSupportedVersion == "" ||
			!version.Must(version.NewVersion(v)).LessThan(version.Must(version.NewVersion(mp.connectorFlowMinimumSupportedVersion)))
		result = append(result, ConnectorTemplateVersionAPIModel{Version: v, Latest: v == mp.connectorFlowTemplateVersion, Supported: supported})
	}
	mp.respond(res, result, http.StatusOK)
}
