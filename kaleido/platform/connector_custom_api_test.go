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
	"encoding/json"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"
)

var customAPIStep1 = `
resource "kaleido_platform_connector_custom_api" "api1" {
    environment = "env1"
    service_id = "service1"
    name = "test-api"
    abi = jsonencode([{"type": "function", "name": "test"}])
    bytecode = "0x6080604052"
}
`

var customAPIStep2 = `
resource "kaleido_platform_connector_custom_api" "api1" {
    environment = "env1"
    service_id = "service1"
    name = "test-api"
    abi = jsonencode([{"type": "function", "name": "test"}])
    bytecode = "0x6080604052"
    devdoc = jsonencode({"methods": {}})
    flow_type_bindings = {
        "deploy" = "deploy-flow"
        "invoke" = "invoke-flow"
    }
}
`

func TestConnectorCustomAPI(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		// Terraform makes multiple calls during plan/apply/refresh, so we just verify the service was called
		assert.Contains(t, mp.calls, "GET /api/v1/environments/{env}/services/{service}")
		mp.calls = []string{}
	}()

	// Setup mock service with REST endpoint
	service1 := &ServiceAPIModel{
		ID:     "service1",
		Type:   "EVMConnectorService",
		Name:   "evm1",
		Status: "ready",
		Endpoints: map[string]ServiceAPIEndpoint{
			"rest": {
				Type: "rest",
				URLS: []string{"http://connector-service.example.com"},
			},
		},
	}
	mp.services["env1/service1"] = service1

	// Setup mock connector service endpoints
	mp.router.HandleFunc("/endpoint/{env}/{service}/rest/api/v1/metadata/setup-info", func(res http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodGet {
			setupInfo := struct {
				ConnectorFlows []ConnectorFlowInfo `json:"connectorFlows"`
			}{
				ConnectorFlows: []ConnectorFlowInfo{
					{Name: "deploy-flow", FlowType: "deploy"},
					{Name: "invoke-flow", FlowType: "invoke"},
				},
			}
			json.NewEncoder(res).Encode(setupInfo)
			return
		}
		res.WriteHeader(http.StatusMethodNotAllowed)
	}).Methods(http.MethodGet)

	mp.router.HandleFunc("/endpoint/{env}/{service}/rest/api/v1/connector-flows", func(res http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodGet {
			flowsResult := struct {
				Items []struct {
					Labels map[string]string `json:"labels"`
				} `json:"items"`
			}{
				Items: []struct {
					Labels map[string]string `json:"labels"`
				}{
					{Labels: map[string]string{"connector_flow": "deploy-flow", "connector_flow_type": "deploy"}},
					{Labels: map[string]string{"connector_flow": "invoke-flow", "connector_flow_type": "invoke"}},
				},
			}
			json.NewEncoder(res).Encode(flowsResult)
			return
		}
		res.WriteHeader(http.StatusMethodNotAllowed)
	}).Methods(http.MethodGet)

	// Mock custom API endpoints
	mp.router.HandleFunc("/endpoint/{env}/{service}/rest/api/v1/apis/{name}", func(res http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodDelete {
			res.WriteHeader(http.StatusOK)
			return
		}
		res.WriteHeader(http.StatusMethodNotAllowed)
	}).Methods(http.MethodDelete)

	mp.router.HandleFunc("/endpoint/{env}/{service}/rest/api/v1/metadata/custom-api", func(res http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodPost {
			var deployBody CustomAPIDeploy
			json.NewDecoder(req.Body).Decode(&deployBody)
			// Return success
			res.WriteHeader(http.StatusOK)
			json.NewEncoder(res).Encode(map[string]interface{}{
				"name": deployBody.Name,
			})
			return
		}
		res.WriteHeader(http.StatusMethodNotAllowed)
	}).Methods(http.MethodPost)

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + customAPIStep1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("kaleido_platform_connector_custom_api.api1", "id"),
					resource.TestCheckResourceAttr("kaleido_platform_connector_custom_api.api1", "service_id", "service1"),
					resource.TestCheckResourceAttr("kaleido_platform_connector_custom_api.api1", "environment", "env1"),
					resource.TestCheckResourceAttr("kaleido_platform_connector_custom_api.api1", "name", "test-api"),
				),
			},
			{
				Config: providerConfig + customAPIStep2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("kaleido_platform_connector_custom_api.api1", "id"),
					resource.TestCheckResourceAttr("kaleido_platform_connector_custom_api.api1", "service_id", "service1"),
					resource.TestCheckResourceAttr("kaleido_platform_connector_custom_api.api1", "environment", "env1"),
					resource.TestCheckResourceAttr("kaleido_platform_connector_custom_api.api1", "name", "test-api"),
					resource.TestCheckResourceAttr("kaleido_platform_connector_custom_api.api1", "flow_type_bindings.%", "2"),
				),
			},
		},
	})

	// Verify service was retrieved (already checked in defer)
}
