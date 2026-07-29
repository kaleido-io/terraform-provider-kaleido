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

var connector_flow_unpinned = `
resource "kaleido_platform_connector_flow" "submission" {
  environment = "test-env"
  service     = "test-service"
  name        = "submission"
  config_type_bindings = {
    "evm.gasPricing" = "evm.gasPricing"
  }
}
`

var connector_flow_pinned_current = `
resource "kaleido_platform_connector_flow" "submission" {
  environment = "test-env"
  service     = "test-service"
  name        = "submission"
  version     = "2026.03.0"
  config_type_bindings = {
    "evm.gasPricing" = "evm.gasPricing"
  }
}
`

var connector_flow_pinned_next = `
resource "kaleido_platform_connector_flow" "submission" {
  environment = "test-env"
  service     = "test-service"
  name        = "submission"
  version     = "2026.04.0"
  config_type_bindings = {
    "evm.gasPricing" = "evm.gasPricing"
  }
}
`

func TestConnectorFlowUnpinned(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"POST /endpoint/{env}/{service}/rest/api/v1/metadata/connector-flows/{flow}/deploy",
			"GET /endpoint/{env}/{service}/rest/api/v1/connector-flows/{flow}",
			"DELETE /endpoint/{env}/{service}/rest/api/v1/connector-flows/{flow}",
		})
		mp.server.Close()
	}()

	res := "kaleido_platform_connector_flow.submission"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + connector_flow_unpinned,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(res, "id"),
					resource.TestCheckResourceAttr(res, "flow_type", "submission"),
					resource.TestCheckResourceAttr(res, "version", "2026.03.0"),
					resource.TestCheckResourceAttr(res, "current_version", "2026.03.0"),
				),
			},
		},
	})
}

func TestConnectorFlowPinnedUpgrade(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"POST /endpoint/{env}/{service}/rest/api/v1/metadata/connector-flows/{flow}/deploy",
			"GET /endpoint/{env}/{service}/rest/api/v1/connector-flows/{flow}",
			"GET /endpoint/{env}/{service}/rest/api/v1/connector-flows/{flow}",
			"POST /endpoint/{env}/{service}/rest/api/v1/metadata/connector-flows/{flow}/upgrade",
			"GET /endpoint/{env}/{service}/rest/api/v1/connector-flows/{flow}",
			"DELETE /endpoint/{env}/{service}/rest/api/v1/connector-flows/{flow}",
		})
		mp.server.Close()
	}()

	res := "kaleido_platform_connector_flow.submission"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + connector_flow_pinned_current,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(res, "version", "2026.03.0"),
					resource.TestCheckResourceAttr(res, "current_version", "2026.03.0"),
				),
			},
			{
				// The connector service image has been upgraded and now embeds a newer
				// template; bumping the pin drives the /upgrade.
				PreConfig: func() { mp.connectorFlowTemplateVersion = "2026.04.0" },
				Config:    providerConfig + connector_flow_pinned_next,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(res, "version", "2026.04.0"),
					resource.TestCheckResourceAttr(res, "current_version", "2026.04.0"),
				),
			},
		},
	})
}

func TestConnectorFlowPinMismatch(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"POST /endpoint/{env}/{service}/rest/api/v1/metadata/connector-flows/{flow}/deploy",
			"DELETE /endpoint/{env}/{service}/rest/api/v1/connector-flows/{flow}",
		})
		mp.server.Close()
	}()

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				// The mock service only embeds 2026.03.0, so pinning 2026.04.0 fails.
				Config:      providerConfig + connector_flow_pinned_next,
				ExpectError: regexp.MustCompile(`Connector flow version mismatch`),
			},
		},
	})
}

// Mock server handlers

func (mp *mockPlatform) connectorFlowKey(vars map[string]string) string {
	return vars["env"] + "/" + vars["service"] + "/" + vars["flow"]
}

func (mp *mockPlatform) deployConnectorFlow(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	var body ConnectorFlowDeployAPIModel
	mp.getBody(req, &body)
	flow := &ConnectorFlowAPIModel{
		ID:             "wfl:" + vars["flow"],
		Name:           vars["flow"],
		FlowType:       vars["flow"],
		CurrentVersion: mp.connectorFlowTemplateVersion,
		Description:    body.Description,
	}
	mp.connectorFlows[mp.connectorFlowKey(vars)] = flow
	mp.respond(res, flow, http.StatusOK)
}

func (mp *mockPlatform) upgradeConnectorFlow(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	flow := mp.connectorFlows[mp.connectorFlowKey(vars)]
	if flow == nil {
		mp.respond(res, nil, http.StatusNotFound)
		return
	}
	flow.CurrentVersion = mp.connectorFlowTemplateVersion
	mp.respond(res, flow, http.StatusOK)
}

func (mp *mockPlatform) getConnectorFlow(res http.ResponseWriter, req *http.Request) {
	flow := mp.connectorFlows[mp.connectorFlowKey(mux.Vars(req))]
	if flow == nil {
		mp.respond(res, nil, http.StatusNotFound)
		return
	}
	mp.respond(res, flow, http.StatusOK)
}

func (mp *mockPlatform) deleteConnectorFlow(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	key := mp.connectorFlowKey(vars)
	if mp.connectorFlows[key] == nil {
		mp.respond(res, nil, http.StatusNotFound)
		return
	}
	delete(mp.connectorFlows, key)
	mp.respond(res, nil, http.StatusNoContent)
}
