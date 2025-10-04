// Copyright © Kaleido, Inc. 2024

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
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/aidarkhanov/nanoid"
	"github.com/gorilla/mux"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/assert"

	_ "embed"
)

var wms_assetStep1 = `
resource "kaleido_platform_wms_asset" "assets" {
  environment             = "env1"
  service                 = "service1"
  name                    = "asset1"
  description             = "test asset"
  symbol                  = "MYCOIN"
  color                   = "#0000FF"
  account_identifier_type = "eth_address"
  protocol_id             = "0x92c580144897F7613f25d80C844e3DfCF8E78524"
  config_json = jsonencode({
    decimals = 6
    units = [
      {
        name   = "c"
        factor = 4
        prefix = false
      },
      {
        name   = "m"
        factor = 2
        prefix = false
      }
    ]
    transfers = {
      backend   = "asset-manager1"
      backendId = "erc_0x92c580144897F7613f25d80C844e3DfCF8E78524"
    }
    operations = {
      mint: {
        schema         = "mint schema"
        fieldMapping = "mint field mapping"
        flow = "erc20-workflow"
        backendMapping = "mint backend mapping"
      },
      transfer: {
        schema         = "transfer schema"
        fieldMapping = "transfer field mapping"
        flow = "erc20-workflow",
        backendMapping = "transfer backend mapping"
      }
    }
  })
}
`
var wms_assetStep2 = `
resource "kaleido_platform_wms_asset" "assets" {
  environment             = "env1"	
  service                 = "service1"
  name                    = "asset1"
  description             = "test asset"
  symbol                  = "MYCOIN"
  color                   = "#FF0000"
  account_identifier_type = "eth_address"
  protocol_id             = "0x92c580144897F7613f25d80C844e3DfCF8E78524"
  config_json = jsonencode({
    decimals = 10
    units = [
      {
        name   = "c"
        factor = 6
        prefix = false
      },
      {
        name   = "m"
        factor = 8
        prefix = false
      }
    ]
    transfers = {
      backend   = "asset-manager1"
      backendId = "erc_0x92c580144897F7613f25d80C844e3DfCF8E78524"
    }
    operations = {
      mint: {
        schema         = "altered mint schema"
        fieldMapping = "altered mint field mapping"
        flow = "erc20-workflow"
        backendMapping = "altered mint backend mapping"
      },
      transfer: {
        schema         = "altered transfer schema"
        fieldMapping = "altered transfer field mapping"
        flow = "erc20-workflow",
        backendMapping = "altered transfer backend mapping"
      }
    }
  })
}
`

func TestWMSAsset1(t *testing.T) {

	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"POST /endpoint/{env}/{service}/rest/api/v1/assets",
			"GET /endpoint/{env}/{service}/rest/api/v1/assets/{asset}",
			"GET /endpoint/{env}/{service}/rest/api/v1/assets/{asset}",
			"PATCH /endpoint/{env}/{service}/rest/api/v1/assets/{asset}",
			"GET /endpoint/{env}/{service}/rest/api/v1/assets/{asset}",
			"DELETE /endpoint/{env}/{service}/rest/api/v1/assets/{asset}",
			"GET /endpoint/{env}/{service}/rest/api/v1/assets/{asset}",
		})
		mp.server.Close()
	}()

	wms_asset1Resource := "kaleido_platform_wms_asset.assets"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + wms_assetStep1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(wms_asset1Resource, "id"),
					resource.TestCheckResourceAttr(wms_asset1Resource, "color", "#0000FF"),
				),
			},
			{
				Config: providerConfig + wms_assetStep2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(wms_asset1Resource, "id"),
					resource.TestCheckResourceAttr(wms_asset1Resource, "color", "#FF0000"),
					func(s *terraform.State) error {
						// Compare the final result on the mock-server side
						id := s.RootModule().Resources[wms_asset1Resource].Primary.Attributes["id"]
						obj := mp.wmsAssets[fmt.Sprintf("env1/service1/%s", id)]
						testJSONEqual(t, obj, fmt.Sprintf(`{
								"id": "%[1]s",
								"created": "%[2]s",
								"updated": "%[3]s",
								"name": "asset1",
								"symbol": "MYCOIN",
								"protocolId": "0x92c580144897F7613f25d80C844e3DfCF8E78524",
								"accountIdentifierType": "eth_address",
								"description": "test asset",
								"color": "#FF0000",
								"config": {
									"decimals": 10,
									"units": [
										{
										"name": "c",
										"factor": 6,
										"prefix": false
										},
										{
										"name": "m",
										"factor": 8,
										"prefix": false
										}
									],
									"transfers": {
										"backend": "asset-manager1",
										"backendId": "erc_0x92c580144897F7613f25d80C844e3DfCF8E78524"
									},
									"operations": {
										"mint": {
											"schema": "altered mint schema",
											"fieldMapping": "altered mint field mapping",
											"flow": "erc20-workflow",
											"backendMapping": "altered mint backend mapping"
										},
										"transfer": {
											"schema": "altered transfer schema",
											"fieldMapping": "altered transfer field mapping",
											"flow": "erc20-workflow",
											"backendMapping": "altered transfer backend mapping"
										}
									}
								}
							}`,
							// generated fields that vary per test run
							id,
							obj.Created,
							obj.Updated,
						))
						return nil
					},
				),
			},
		},
	})
}

func (mp *mockPlatform) getWMSAsset(res http.ResponseWriter, req *http.Request) {
	obj := mp.wmsAssets[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["asset"]]
	if obj == nil {
		mp.respond(res, nil, 404)
	} else {
		mp.respond(res, obj, 200)
	}

}

func (mp *mockPlatform) putWMSAsset(res http.ResponseWriter, req *http.Request) {
	obj := mp.wmsAssets[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["asset"]] // expected behavior of provider is PUT only on exists
	assert.NotNil(mp.t, obj)
	var newObj WMSAssetAPIModel
	mp.getBody(req, &newObj)
	assert.Equal(mp.t, obj.ID, newObj.ID)               // expected behavior of provider
	assert.Equal(mp.t, obj.ID, mux.Vars(req)["wallet"]) // expected behavior of provider
	now := time.Now().UTC().Format(time.RFC3339Nano)
	newObj.Created = obj.Created
	newObj.Updated = now
	delete(mp.wmsAssets, mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+obj.Name)
	delete(mp.wmsAssets, mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+newObj.ID)
	mp.wmsAssets[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+newObj.Name] = &newObj
	mp.wmsAssets[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+newObj.ID] = &newObj
	mp.respond(res, &newObj, 200)

}

func (mp *mockPlatform) deleteWMSAsset(res http.ResponseWriter, req *http.Request) {
	obj := mp.wmsAssets[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["asset"]]
	assert.NotNil(mp.t, obj)
	delete(mp.wmsAssets, mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+obj.Name)
	delete(mp.wmsAssets, mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+obj.ID)
	mp.respond(res, nil, 204)
}

func (mp *mockPlatform) postWMSAsset(res http.ResponseWriter, req *http.Request) {
	var obj WMSAssetAPIModel
	mp.getBody(req, &obj)
	obj.ID = nanoid.New()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	obj.Created = now
	obj.Updated = now

	// Store with both ID and name as keys for different lookup patterns
	env := mux.Vars(req)["env"]
	service := mux.Vars(req)["service"]
	mp.wmsAssets[env+"/"+service+"/"+obj.ID] = &obj
	mp.wmsAssets[env+"/"+service+"/"+obj.Name] = &obj

	mp.respond(res, &obj, 201)
}

func (mp *mockPlatform) patchWMSAsset(res http.ResponseWriter, req *http.Request) {
	obj := mp.wmsAssets[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["asset"]]
	assert.NotNil(mp.t, obj)
	var newObj WMSAssetAPIModel
	mp.getBody(req, &newObj)
	assert.Equal(mp.t, obj.ID, newObj.ID)
	now := time.Now().UTC().Format(time.RFC3339Nano)
	newObj.Created = obj.Created
	newObj.Updated = now
	mp.wmsAssets[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["asset"]] = &newObj
	mp.respond(res, &newObj, 200)
}
