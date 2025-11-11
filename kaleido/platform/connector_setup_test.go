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

	"github.com/gorilla/mux"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"
)

var connectorSetupStep1 = `
resource "kaleido_platform_connector_setup" "setup1" {
    environment = "env1"
    service_id = "service1"
}
`

var connectorSetupStep2 = `
resource "kaleido_platform_connector_setup" "setup1" {
    environment = "env1"
    service_id = "service1"
    config_profiles = {
        "evm.confirmations" = jsonencode({
            "count": 1
        })
        "evm.gasEstimation" = jsonencode({
            "scaleFactor": 2.0
        })
    }
}
`

func TestConnectorSetup(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"GET /api/v1/environments/{env}/services/{service}",
			"GET /api/v1/environments/{env}/services/{service}",
		})
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
	mp.router.HandleFunc("/api/v1/metadata/setup-info", func(res http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodGet {
			setupInfo := SetupInfo{
				RequiredConfigTypes: []ConfigTypeInfo{
					{Name: "evm.confirmations"},
					{Name: "evm.gasEstimation"},
				},
				ConnectorFlows: []ConnectorFlowInfo{
					{Name: "flow1", FlowType: "deploy"},
				},
				ConnectorStreamFactories: []StreamFactoryInfo{
					{Name: "factory1"},
				},
				StandardAPIs: []StandardAPIInfo{
					{Name: "deploySmartContract"},
				},
				StandardStreams: []StandardStreamInfo{
					{Name: "stream1"},
				},
			}
			json.NewEncoder(res).Encode(setupInfo)
			return
		}
		res.WriteHeader(http.StatusMethodNotAllowed)
	}).Methods(http.MethodGet)

	// Mock config type endpoints
	mp.router.HandleFunc("/api/v1/metadata/config-types/{name}", func(res http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodPut {
			res.WriteHeader(http.StatusOK)
			return
		}
		res.WriteHeader(http.StatusMethodNotAllowed)
	}).Methods(http.MethodPut)

	// Mock config profile endpoints
	mp.router.HandleFunc("/api/v1/config-profiles/{name}", func(res http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodPut {
			var cp ConfigProfile
			json.NewDecoder(req.Body).Decode(&cp)
			cp.ID = "profile-" + mux.Vars(req)["name"]
			json.NewEncoder(res).Encode(cp)
			return
		}
		res.WriteHeader(http.StatusMethodNotAllowed)
	}).Methods(http.MethodPut)

	// Mock connector flow endpoints
	mp.router.HandleFunc("/api/v1/connector-flows/{name}", func(res http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodDelete {
			res.WriteHeader(http.StatusOK)
			return
		}
		if req.Method == http.MethodPost {
			var cf ConnectorFlow
			json.NewDecoder(req.Body).Decode(&cf)
			cf.Name = mux.Vars(req)["name"]
			json.NewEncoder(res).Encode(cf)
			return
		}
		res.WriteHeader(http.StatusMethodNotAllowed)
	}).Methods(http.MethodDelete, http.MethodPost)

	// Mock stream factory endpoints
	mp.router.HandleFunc("/api/v1/metadata/connector-stream-factories/{name}", func(res http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodPut {
			res.WriteHeader(http.StatusOK)
			return
		}
		res.WriteHeader(http.StatusMethodNotAllowed)
	}).Methods(http.MethodPut)

	// Mock standard API endpoints
	mp.router.HandleFunc("/api/v1/apis/{name}", func(res http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodDelete {
			res.WriteHeader(http.StatusOK)
			return
		}
		res.WriteHeader(http.StatusMethodNotAllowed)
	}).Methods(http.MethodDelete)

	mp.router.HandleFunc("/api/v1/metadata/standard-apis/{name}", func(res http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodPost {
			res.WriteHeader(http.StatusOK)
			return
		}
		res.WriteHeader(http.StatusMethodNotAllowed)
	}).Methods(http.MethodPost)

	// Mock standard stream endpoints
	mp.router.HandleFunc("/api/v1/streams/{name}", func(res http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodDelete {
			res.WriteHeader(http.StatusOK)
			return
		}
		res.WriteHeader(http.StatusMethodNotAllowed)
	}).Methods(http.MethodDelete)

	mp.router.HandleFunc("/api/v1/metadata/standard-streams/{name}", func(res http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodPost {
			res.WriteHeader(http.StatusOK)
			return
		}
		res.WriteHeader(http.StatusMethodNotAllowed)
	}).Methods(http.MethodPost)

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + connectorSetupStep1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("kaleido_platform_connector_setup.setup1", "id"),
					resource.TestCheckResourceAttr("kaleido_platform_connector_setup.setup1", "service_id", "service1"),
					resource.TestCheckResourceAttr("kaleido_platform_connector_setup.setup1", "environment", "env1"),
				),
			},
			{
				Config: providerConfig + connectorSetupStep2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("kaleido_platform_connector_setup.setup1", "id"),
					resource.TestCheckResourceAttr("kaleido_platform_connector_setup.setup1", "service_id", "service1"),
					resource.TestCheckResourceAttr("kaleido_platform_connector_setup.setup1", "environment", "env1"),
					resource.TestCheckResourceAttr("kaleido_platform_connector_setup.setup1", "config_profiles.%", "2"),
				),
			},
		},
	})

	// Verify service was retrieved
	assert.Contains(t, mp.calls, "GET /api/v1/environments/env1/services/service1")
}

