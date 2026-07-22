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
	"testing"

	"github.com/aidarkhanov/nanoid"
	"github.com/gorilla/mux"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var connector_flow_config_binding_step1 = `
resource "kaleido_platform_connector_flow_config_binding" "test_binding" {
  environment = "test-env"
  service     = "test-service"
  flow        = "submission"
  config_type = "evm.gasPricing"
  dynamic_mapping = {
    name_prefix = "s:test-service/"
    jsonata     = "$exists(state.input.options.gasPricing.configProfileName) ? state.input.options.gasPricing.configProfileName : \"evm.gasPricing\""
  }
}
`

var connector_flow_config_binding_step2 = `
resource "kaleido_platform_connector_flow_config_binding" "test_binding" {
  environment = "test-env"
  service     = "test-service"
  flow        = "submission"
  config_type = "evm.gasPricing"
  dynamic_mapping = {
    name_prefix = "s:test-service/"
    jsonata     = "$exists(state.input.options.gasPricing.configProfileName) ? state.input.options.gasPricing.configProfileName : \"evm.gasPricing_medium\""
  }
}
`

var connector_flow_config_binding_step_static = `
resource "kaleido_platform_connector_flow_config_binding" "test_binding" {
  environment       = "test-env"
  service           = "test-service"
  flow              = "submission"
  config_type       = "evm.gasPricing"
  config_profile_id = "fcp:test123"
}
`

var connector_flow_config_binding_step_static2 = `
resource "kaleido_platform_connector_flow_config_binding" "test_binding" {
  environment       = "test-env"
  service           = "test-service"
  flow              = "submission"
  config_type       = "evm.gasPricing"
  config_profile_id = "fcp:test456"
}
`

func TestConnectorFlowConfigBinding1(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"GET /endpoint/{env}/{service}/rest/api/v1/connector-flows/{flow}/config-profile-bindings",
			"PATCH /endpoint/{env}/{service}/rest/api/v1/connector-flows/{flow}/config-profile-bindings/{binding}",
			"GET /endpoint/{env}/{service}/rest/api/v1/connector-flows/{flow}/config-profile-bindings/{binding}",
			"PATCH /endpoint/{env}/{service}/rest/api/v1/connector-flows/{flow}/config-profile-bindings/{binding}",
		})
		mp.server.Close()
	}()

	res := "kaleido_platform_connector_flow_config_binding.test_binding"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + connector_flow_config_binding_step1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(res, "id"),
					resource.TestCheckResourceAttr(res, "environment", "test-env"),
					resource.TestCheckResourceAttr(res, "service", "test-service"),
					resource.TestCheckResourceAttr(res, "flow", "submission"),
					resource.TestCheckResourceAttr(res, "config_type", "evm.gasPricing"),
					resource.TestCheckResourceAttr(res, "dynamic_mapping.name_prefix", "s:test-service/"),
				),
			},
		},
	})
}

func TestConnectorFlowConfigBinding2(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"GET /endpoint/{env}/{service}/rest/api/v1/connector-flows/{flow}/config-profile-bindings",
			"PATCH /endpoint/{env}/{service}/rest/api/v1/connector-flows/{flow}/config-profile-bindings/{binding}",
			"GET /endpoint/{env}/{service}/rest/api/v1/connector-flows/{flow}/config-profile-bindings/{binding}",
			"GET /endpoint/{env}/{service}/rest/api/v1/connector-flows/{flow}/config-profile-bindings/{binding}",
			"PATCH /endpoint/{env}/{service}/rest/api/v1/connector-flows/{flow}/config-profile-bindings/{binding}",
			"GET /endpoint/{env}/{service}/rest/api/v1/connector-flows/{flow}/config-profile-bindings/{binding}",
			"PATCH /endpoint/{env}/{service}/rest/api/v1/connector-flows/{flow}/config-profile-bindings/{binding}",
		})
		mp.server.Close()
	}()

	res := "kaleido_platform_connector_flow_config_binding.test_binding"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + connector_flow_config_binding_step1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(res, "id"),
					resource.TestCheckResourceAttr(res, "dynamic_mapping.name_prefix", "s:test-service/"),
					resource.TestCheckResourceAttr(res, "dynamic_mapping.jsonata", `$exists(state.input.options.gasPricing.configProfileName) ? state.input.options.gasPricing.configProfileName : "evm.gasPricing"`),
				),
			},
			{
				Config: providerConfig + connector_flow_config_binding_step2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(res, "id"),
					resource.TestCheckResourceAttr(res, "dynamic_mapping.jsonata", `$exists(state.input.options.gasPricing.configProfileName) ? state.input.options.gasPricing.configProfileName : "evm.gasPricing_medium"`),
				),
			},
		},
	})
}

func TestConnectorFlowConfigBinding3(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"GET /endpoint/{env}/{service}/rest/api/v1/connector-flows/{flow}/config-profile-bindings",
			"PATCH /endpoint/{env}/{service}/rest/api/v1/connector-flows/{flow}/config-profile-bindings/{binding}",
			"GET /endpoint/{env}/{service}/rest/api/v1/connector-flows/{flow}/config-profile-bindings/{binding}",
			"GET /endpoint/{env}/{service}/rest/api/v1/connector-flows/{flow}/config-profile-bindings/{binding}",
			"PATCH /endpoint/{env}/{service}/rest/api/v1/connector-flows/{flow}/config-profile-bindings/{binding}",
			"GET /endpoint/{env}/{service}/rest/api/v1/connector-flows/{flow}/config-profile-bindings/{binding}",
			"GET /endpoint/{env}/{service}/rest/api/v1/connector-flows/{flow}/config-profile-bindings/{binding}",
			"PATCH /endpoint/{env}/{service}/rest/api/v1/connector-flows/{flow}/config-profile-bindings/{binding}",
			"GET /endpoint/{env}/{service}/rest/api/v1/connector-flows/{flow}/config-profile-bindings/{binding}",
			"PATCH /endpoint/{env}/{service}/rest/api/v1/connector-flows/{flow}/config-profile-bindings/{binding}",
		})
		mp.server.Close()
	}()

	res := "kaleido_platform_connector_flow_config_binding.test_binding"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + connector_flow_config_binding_step_static,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(res, "id"),
					resource.TestCheckResourceAttr(res, "config_profile_id", "fcp:test123"),
					resource.TestCheckNoResourceAttr(res, "dynamic_mapping"),
				),
			},
			{
				Config: providerConfig + connector_flow_config_binding_step_static2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(res, "config_profile_id", "fcp:test456"),
				),
			},
			{
				Config: providerConfig + connector_flow_config_binding_step1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr(res, "config_profile_id"),
					resource.TestCheckResourceAttr(res, "dynamic_mapping.name_prefix", "s:test-service/"),
				),
			},
		},
	})
}

// Mock server handlers

func (mp *mockPlatform) listConnectorFlowConfigBindings(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	flowKey := vars["env"] + "/" + vars["service"] + "/" + vars["flow"]
	// Seed a binding for the requested config type if one doesn't exist yet.
	configType := req.URL.Query().Get("workflowconfigprofile")
	if configType == "" {
		configType = "evm.gasPricing"
	}
	bindingKey := flowKey + "/" + configType
	if mp.connectorFlowConfigBindings == nil {
		mp.connectorFlowConfigBindings = make(map[string]*ConnectorFlowConfigBindingAPIModel)
	}
	if _, exists := mp.connectorFlowConfigBindings[bindingKey]; !exists {
		mp.connectorFlowConfigBindings[bindingKey] = &ConnectorFlowConfigBindingAPIModel{
			ID:                    "fcb:" + nanoid.New(),
			WorkflowConfigProfile: configType,
		}
	}
	binding := mp.connectorFlowConfigBindings[bindingKey]
	mp.respond(res, struct {
		Items []ConnectorFlowConfigBindingAPIModel `json:"items"`
	}{Items: []ConnectorFlowConfigBindingAPIModel{*binding}}, http.StatusOK)
}

func (mp *mockPlatform) getConnectorFlowConfigBinding(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	bindingID := vars["binding"]
	for _, b := range mp.connectorFlowConfigBindings {
		if b.ID == bindingID {
			mp.respond(res, b, http.StatusOK)
			return
		}
	}
	mp.respond(res, nil, http.StatusNotFound)
}

func (mp *mockPlatform) patchConnectorFlowConfigBinding(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	bindingID := vars["binding"]
	for _, b := range mp.connectorFlowConfigBindings {
		if b.ID == bindingID {
			var patch ConnectorFlowConfigBindingPatchAPIModel
			mp.getBody(req, &patch)
			b.DynamicMapping = patch.DynamicMapping
			if patch.ConfigProfileID != nil {
				b.ConfigProfileID = *patch.ConfigProfileID
			} else {
				b.ConfigProfileID = ""
			}
			mp.respond(res, b, http.StatusOK)
			return
		}
	}
	mp.respond(res, nil, http.StatusNotFound)
}
