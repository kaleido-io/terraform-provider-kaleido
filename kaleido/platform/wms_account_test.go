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

var wms_accountStep1 = `
resource "kaleido_platform_wms_account" "account1" {
  environment             = "env1"
  service                 = "service1"
  asset                   = "asset1"
  wallet                  = "wallet1"  
}

resource "kaleido_platform_wms_account" "account2" {
  environment             = "env1"
  service                 = "service1"
  asset                   = "asset2"
  wallet                  = "wallet2"  
}
`
var wms_accountStep2 = `
resource "kaleido_platform_wms_account" "account1" {
  environment             = "env1"
  service                 = "service1"
  asset                   = "asset1"
  wallet                  = "wallet2"  
}

resource "kaleido_platform_wms_account" "account2" {
  environment             = "env1"
  service                 = "service1"
  asset                   = "asset2"
  wallet                  = "wallet1"  
}
`

func TestWMSAccount(t *testing.T) {

	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{

			"POST /endpoint/{env}/{service}/rest/api/v1/assets/{asset}/connect/{wallet}",
			"POST /endpoint/{env}/{service}/rest/api/v1/assets/{asset}/connect/{wallet}",
			"GET /endpoint/{env}/{service}/rest/api/v1/accounts/{account}",
			"GET /endpoint/{env}/{service}/rest/api/v1/accounts/{account}",
			"GET /endpoint/{env}/{service}/rest/api/v1/accounts/{account}",
			"GET /endpoint/{env}/{service}/rest/api/v1/accounts/{account}",
			"DELETE /endpoint/{env}/{service}/rest/api/v1/accounts/{account}",
			"DELETE /endpoint/{env}/{service}/rest/api/v1/accounts/{account}",
			"GET /endpoint/{env}/{service}/rest/api/v1/accounts/{account}",
			"GET /endpoint/{env}/{service}/rest/api/v1/accounts/{account}",
			"POST /endpoint/{env}/{service}/rest/api/v1/assets/{asset}/connect/{wallet}",
			"POST /endpoint/{env}/{service}/rest/api/v1/assets/{asset}/connect/{wallet}",
			"GET /endpoint/{env}/{service}/rest/api/v1/accounts/{account}",
			"GET /endpoint/{env}/{service}/rest/api/v1/accounts/{account}",
			"DELETE /endpoint/{env}/{service}/rest/api/v1/accounts/{account}",
			"DELETE /endpoint/{env}/{service}/rest/api/v1/accounts/{account}",
			"GET /endpoint/{env}/{service}/rest/api/v1/accounts/{account}",
			"GET /endpoint/{env}/{service}/rest/api/v1/accounts/{account}",
		})
		mp.server.Close()
	}()

	wms_account1Resource := "kaleido_platform_wms_account.account1"
	wms_account2Resource := "kaleido_platform_wms_account.account2"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + wms_accountStep1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(wms_account1Resource, "id"),
					resource.TestCheckResourceAttrSet(wms_account1Resource, "identifier"),
					resource.TestCheckResourceAttrSet(wms_account1Resource, "identifier_type"),
					resource.TestCheckResourceAttrSet(wms_account2Resource, "id"),
					resource.TestCheckResourceAttrSet(wms_account2Resource, "identifier"),
					resource.TestCheckResourceAttrSet(wms_account2Resource, "identifier_type"),
					resource.TestCheckResourceAttr(wms_account1Resource, "asset", "asset1"),
					resource.TestCheckResourceAttr(wms_account1Resource, "wallet", "wallet1"),
					resource.TestCheckResourceAttr(wms_account2Resource, "asset", "asset2"),
					resource.TestCheckResourceAttr(wms_account2Resource, "wallet", "wallet2"),
					func(s *terraform.State) error {
						//step1Account1ID = s.RootModule().Resources[wms_account1Resource].Primary.Attributes["id"]
						//step1Account2ID = s.RootModule().Resources[wms_account2Resource].Primary.Attributes["id"]
						return nil
					},
				),
			},
			{
				Config: providerConfig + wms_accountStep2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(wms_account1Resource, "id"),
					resource.TestCheckResourceAttrSet(wms_account2Resource, "id"),
					resource.TestCheckResourceAttr(wms_account1Resource, "asset", "asset1"),
					resource.TestCheckResourceAttr(wms_account1Resource, "wallet", "wallet2"),
					resource.TestCheckResourceAttr(wms_account2Resource, "asset", "asset2"),
					resource.TestCheckResourceAttr(wms_account2Resource, "wallet", "wallet1"),
					//TODO test that the account ids are different than they were in step 1
					func(s *terraform.State) error {
						// Compare the final result on the mock-server side
						id1 := s.RootModule().Resources[wms_account1Resource].Primary.Attributes["id"]
						obj1 := mp.wmsAccounts[fmt.Sprintf("env1/service1/%s", id1)]
						testJSONEqual(t, obj1, fmt.Sprintf(`{
								"id": "%[1]s",
								"created": "%[2]s",
								"updated": "%[3]s",
								"assetId": "asset1",
								"walletId": "wallet2",
								"identifier": "0x1234567890123456789012345678901234567890",
								"identifierType": "eth_address"
							}`,
							// generated fields that vary per test run
							id1,
							obj1.Created,
							obj1.Updated,
						))
						id2 := s.RootModule().Resources[wms_account2Resource].Primary.Attributes["id"]
						obj2 := mp.wmsAccounts[fmt.Sprintf("env1/service1/%s", id2)]
						testJSONEqual(t, obj2, fmt.Sprintf(`{
								"id": "%[1]s",
								"created": "%[2]s",
								"updated": "%[3]s",
								"assetId": "asset2",
								"walletId": "wallet1",
								"identifier": "0x1234567890123456789012345678901234567890",
								"identifierType": "eth_address"
							}`,
							// generated fields that vary per test run
							id2,
							obj2.Created,
							obj2.Updated,
						))
						return nil
					},
				),
			},
		},
	})
}

func (mp *mockPlatform) getWMSAccount(res http.ResponseWriter, req *http.Request) {
	obj := mp.wmsAccounts[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["account"]]
	if obj == nil {
		mp.respond(res, nil, 404)
	} else {
		mp.respond(res, obj, 200)
	}

}

func (mp *mockPlatform) connectWMSAccount(res http.ResponseWriter, req *http.Request) {

	var obj WMSAccountAPIModel
	obj.ID = nanoid.New()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	obj.Created = now
	obj.Updated = now
	obj.AssetID = mux.Vars(req)["asset"]
	obj.WalletID = mux.Vars(req)["wallet"]
	obj.Identifier = "0x1234567890123456789012345678901234567890"
	obj.IdentifierType = "eth_address"
	mp.wmsAccounts[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+obj.ID] = &obj
	mp.respond(res, &obj, 201)

}

func (mp *mockPlatform) deleteWMSAccount(res http.ResponseWriter, req *http.Request) {
	obj := mp.wmsAccounts[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["account"]]
	assert.NotNil(mp.t, obj)
	delete(mp.wmsAccounts, mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["account"])
	mp.respond(res, nil, 204)
}
