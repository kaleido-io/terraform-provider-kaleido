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
	"time"

	"github.com/aidarkhanov/nanoid"
	"github.com/gorilla/mux"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const wfe_stream_factory_step1 = `
resource "kaleido_platform_wfe_stream_factory" "test_factory" {
  environment = "test-env"
  service = "test-service"
  name = "test-factory"
  description = "Test stream factory"
  config_type = "config-type-1"
  uniqueness_prefix = "evmtx."
  event_source_type = "handler"
  event_source_json = jsonencode({
      "name" = "evmTransactions"
      "provider" = "evm-provider"
      "configMapping" = {
        "jsonata" = "$merge([$, {'fromBlock': $params.fromBlock}])"
      }
  })
  constants_json = jsonencode({
      "network" = "mainnet"
  })
  parameters_schema_json = jsonencode({
      "type" = "object"
      "properties" = {
        "fromBlock" = { "type" = "string" }
      }
  })
}
`

const wfe_stream_factory_step2 = `
resource "kaleido_platform_wfe_stream_factory" "test_factory" {
  environment = "test-env"
  service = "test-service"
  name = "test-factory"
  description = "Test stream factory - updated"
  config_type = "config-type-1"
  uniqueness_prefix = "evmtx.v2."
  event_source_type = "handler"
  event_source_json = jsonencode({
      "name" = "evmTransactions"
      "provider" = "evm-provider"
      "configMapping" = {
        "jsonata" = "$merge([$, {'fromBlock': $params.fromBlock, 'batchSize': $params.batchSize}])"
      }
  })
  constants_json = jsonencode({
      "network" = "mainnet"
      "version" = "v2"
  })
  parameters_schema_json = jsonencode({
      "type" = "object"
      "properties" = {
        "fromBlock" = { "type" = "string" }
        "batchSize" = { "type" = "number" }
      }
  })
}
`

const wfe_stream_factory_resource = "kaleido_platform_wfe_stream_factory.test_factory"

func TestWFEStreamFactory1(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"PUT /endpoint/{env}/{service}/rest/api/v1/stream-factories/{streamFactory}",
			"GET /endpoint/{env}/{service}/rest/api/v1/stream-factories/{streamFactory}",
			"DELETE /endpoint/{env}/{service}/rest/api/v1/stream-factories/{streamFactory}",
			"GET /endpoint/{env}/{service}/rest/api/v1/stream-factories/{streamFactory}",
		})
		mp.server.Close()
	}()

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + wfe_stream_factory_step1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(wfe_stream_factory_resource, "environment", "test-env"),
					resource.TestCheckResourceAttr(wfe_stream_factory_resource, "service", "test-service"),
					resource.TestCheckResourceAttr(wfe_stream_factory_resource, "name", "test-factory"),
					resource.TestCheckResourceAttr(wfe_stream_factory_resource, "description", "Test stream factory"),
					resource.TestCheckResourceAttr(wfe_stream_factory_resource, "config_type", "config-type-1"),
					resource.TestCheckResourceAttr(wfe_stream_factory_resource, "uniqueness_prefix", "evmtx."),
					resource.TestCheckResourceAttr(wfe_stream_factory_resource, "event_source_type", "handler"),
					resource.TestCheckResourceAttrSet(wfe_stream_factory_resource, "id"),
					resource.TestCheckResourceAttrSet(wfe_stream_factory_resource, "created"),
					resource.TestCheckResourceAttrSet(wfe_stream_factory_resource, "updated"),
				),
			},
		},
	})
}

func TestWFEStreamFactory2(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"PUT /endpoint/{env}/{service}/rest/api/v1/stream-factories/{streamFactory}",
			"GET /endpoint/{env}/{service}/rest/api/v1/stream-factories/{streamFactory}",
			"GET /endpoint/{env}/{service}/rest/api/v1/stream-factories/{streamFactory}",
			"PUT /endpoint/{env}/{service}/rest/api/v1/stream-factories/{streamFactory}",
			"GET /endpoint/{env}/{service}/rest/api/v1/stream-factories/{streamFactory}",
			"DELETE /endpoint/{env}/{service}/rest/api/v1/stream-factories/{streamFactory}",
			"GET /endpoint/{env}/{service}/rest/api/v1/stream-factories/{streamFactory}",
		})
		mp.server.Close()
	}()

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + wfe_stream_factory_step1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(wfe_stream_factory_resource, "environment", "test-env"),
					resource.TestCheckResourceAttr(wfe_stream_factory_resource, "service", "test-service"),
					resource.TestCheckResourceAttr(wfe_stream_factory_resource, "name", "test-factory"),
					resource.TestCheckResourceAttr(wfe_stream_factory_resource, "description", "Test stream factory"),
					resource.TestCheckResourceAttr(wfe_stream_factory_resource, "config_type", "config-type-1"),
					resource.TestCheckResourceAttr(wfe_stream_factory_resource, "uniqueness_prefix", "evmtx."),
					resource.TestCheckResourceAttr(wfe_stream_factory_resource, "event_source_type", "handler"),
					resource.TestCheckResourceAttrSet(wfe_stream_factory_resource, "id"),
					resource.TestCheckResourceAttrSet(wfe_stream_factory_resource, "created"),
					resource.TestCheckResourceAttrSet(wfe_stream_factory_resource, "updated"),
				),
			},
			{
				Config: providerConfig + wfe_stream_factory_step2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(wfe_stream_factory_resource, "environment", "test-env"),
					resource.TestCheckResourceAttr(wfe_stream_factory_resource, "service", "test-service"),
					resource.TestCheckResourceAttr(wfe_stream_factory_resource, "name", "test-factory"),
					resource.TestCheckResourceAttr(wfe_stream_factory_resource, "description", "Test stream factory - updated"),
					resource.TestCheckResourceAttr(wfe_stream_factory_resource, "config_type", "config-type-1"),
					resource.TestCheckResourceAttr(wfe_stream_factory_resource, "uniqueness_prefix", "evmtx.v2."),
					resource.TestCheckResourceAttr(wfe_stream_factory_resource, "event_source_type", "handler"),
					resource.TestCheckResourceAttrSet(wfe_stream_factory_resource, "id"),
					resource.TestCheckResourceAttrSet(wfe_stream_factory_resource, "created"),
					resource.TestCheckResourceAttrSet(wfe_stream_factory_resource, "updated"),
				),
			},
		},
	})
}

// Mock handlers for stream factories

func (mp *mockPlatform) putWFEStreamFactory(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	factoryNameOrID := vars["streamFactory"]
	existing, exists := mp.wfeStreamFactories[factoryNameOrID]
	newFactory := new(WFEStreamFactoryAPIModel)
	mp.getBody(req, newFactory)
	now := time.Now().UTC()
	if !exists {
		newFactory.ID = nanoid.New()
		newFactory.Created = &now
	} else {
		newFactory.ID = existing.ID
		newFactory.Created = existing.Created
	}
	newFactory.Updated = &now
	mp.wfeStreamFactories[newFactory.Name] = newFactory
	mp.wfeStreamFactories[newFactory.ID] = newFactory
	mp.respond(res, newFactory, http.StatusOK)
}

func (mp *mockPlatform) getWFEStreamFactory(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	factoryNameOrID := vars["streamFactory"]
	factory, exists := mp.wfeStreamFactories[factoryNameOrID]
	if !exists {
		for _, f := range mp.wfeStreamFactories {
			if f.Name == factoryNameOrID {
				factory = f
				exists = true
				break
			}
		}
	}
	if !exists {
		mp.respond(res, nil, 404)
		return
	}
	mp.respond(res, factory, http.StatusOK)
}

func (mp *mockPlatform) deleteWFEStreamFactory(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	factoryID := vars["streamFactory"]
	factory, exists := mp.wfeStreamFactories[factoryID]
	if !exists {
		mp.respond(res, nil, 404)
		return
	}
	delete(mp.wfeStreamFactories, factoryID)
	delete(mp.wfeStreamFactories, factory.Name)
	mp.respond(res, nil, http.StatusNoContent)
}
