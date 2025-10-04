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

var wms_managed_wallet_Step1 = `
resource "kaleido_platform_wms_wallet" "wms_wallet1" {
    environment = "env1"
	service = "service1"
    name = "wms_wallet1"
	color = "#AFAFAF"
    config = {
		type = "kms"    
		kms = {
			key_id = "kld:///keystore/treasury-wallets/key/redemptions"
		}
    }
}
`

var wms_managed_wallet_Step2 = `
resource "kaleido_platform_wms_wallet" "wms_wallet1" {
    environment = "env1"
	service = "service1"
    name = "wms_wallet1"
	color = "#000000"
    config = {
        type = "kms"
        kms = {
            key_id = "kld:///keystore/treasury-wallets/key/redemptions_renamed"
        }
    }
}
`

var wms_readonly_wallet_Step1 = `
resource "kaleido_platform_wms_wallet" "wms_wallet1" {
    environment = "env1"
	service = "service1"
    name = "wms_wallet1"
    config = {
        type = "readonly"
        readonly = {
            identifier_map = {
                eth_address = "0xabababababababababababababababababababab"
            }
        }
    }
}
`

var wms_readonly_wallet_Step2 = `
resource "kaleido_platform_wms_wallet" "wms_wallet1" {
    environment = "env1"
	service = "service1"
    name = "wms_wallet1"
    config = {
        type = "readonly"
        readonly = {
            identifier_map = {
                eth_address = "0xcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcd"
            }
        }
    }
}
`

func TestWMSManagedWallet(t *testing.T) {

	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"POST /endpoint/{env}/{service}/rest/api/v1/wallets",
			"GET /endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}",
			"GET /endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}",
			"PATCH /endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}",
			"GET /endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}",
			"DELETE /endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}",
			"GET /endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}",
		})
		mp.server.Close()
	}()

	wms_wallet1Resource := "kaleido_platform_wms_wallet.wms_wallet1"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + wms_managed_wallet_Step1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(wms_wallet1Resource, "id"),
					resource.TestCheckResourceAttr(wms_wallet1Resource, "name", `wms_wallet1`),
					resource.TestCheckResourceAttr(wms_wallet1Resource, "config.type", `kms`),
					resource.TestCheckResourceAttr(wms_wallet1Resource, "color", `#AFAFAF`),
					resource.TestCheckResourceAttr(wms_wallet1Resource, "config.kms.key_id", `kld:///keystore/treasury-wallets/key/redemptions`),
				),
			},
			{
				Config: providerConfig + wms_managed_wallet_Step2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(wms_wallet1Resource, "id"),
					resource.TestCheckResourceAttr(wms_wallet1Resource, "name", `wms_wallet1`),
					resource.TestCheckResourceAttr(wms_wallet1Resource, "config.type", `kms`),
					resource.TestCheckResourceAttr(wms_wallet1Resource, "color", `#000000`),
					resource.TestCheckResourceAttr(wms_wallet1Resource, "config.kms.key_id", `kld:///keystore/treasury-wallets/key/redemptions_renamed`),

					func(s *terraform.State) error {
						// Compare the final result on the mock-server side
						id := s.RootModule().Resources[wms_wallet1Resource].Primary.Attributes["id"]
						obj := mp.wmsWallets[fmt.Sprintf("env1/service1/%s", id)]
						testJSONEqual(t, obj, fmt.Sprintf(`
						{
							"id": "%[1]s",
							"created": "%[2]s",
							"updated": "%[3]s",
							"color": "#000000",
							"name": "wms_wallet1",
							"config": {
								"type": "kms",
								"kms": {
									"keyId": "kld:///keystore/treasury-wallets/key/redemptions_renamed"
								}
							}
						}
						`,
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

func TestWMSReadOnlyWallet(t *testing.T) {

	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"POST /endpoint/{env}/{service}/rest/api/v1/wallets",
			"GET /endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}",
			"GET /endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}",
			"PATCH /endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}",
			"GET /endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}",
			"DELETE /endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}",
			"GET /endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}",
		})
		mp.server.Close()
	}()

	wms_wallet1Resource := "kaleido_platform_wms_wallet.wms_wallet1"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + wms_readonly_wallet_Step1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(wms_wallet1Resource, "id"),
					resource.TestCheckResourceAttr(wms_wallet1Resource, "name", `wms_wallet1`),
					resource.TestCheckResourceAttr(wms_wallet1Resource, "config.type", `readonly`),
					resource.TestCheckResourceAttr(wms_wallet1Resource, "config.readonly.identifier_map.eth_address", `0xabababababababababababababababababababab`),
				),
			},
			{
				Config: providerConfig + wms_readonly_wallet_Step2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(wms_wallet1Resource, "id"),
					resource.TestCheckResourceAttr(wms_wallet1Resource, "name", `wms_wallet1`),
					resource.TestCheckResourceAttr(wms_wallet1Resource, "config.type", `readonly`),
					resource.TestCheckResourceAttr(wms_wallet1Resource, "config.readonly.identifier_map.eth_address", `0xcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcd`),

					func(s *terraform.State) error {
						// Compare the final result on the mock-server side
						id := s.RootModule().Resources[wms_wallet1Resource].Primary.Attributes["id"]
						obj := mp.wmsWallets[fmt.Sprintf("env1/service1/%s", id)]
						testJSONEqual(t, obj, fmt.Sprintf(`
						{
							"id": "%[1]s",
							"created": "%[2]s",
							"updated": "%[3]s",
							"name": "wms_wallet1",
							"config": {
								"type": "readonly",
								"readonly": {
									"identifierMap": {
										"eth_address": "0xcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcd"
									}
								}
							}
						}
						`,
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

func (mp *mockPlatform) getWMSWallet(res http.ResponseWriter, req *http.Request) {
	obj := mp.wmsWallets[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["wallet"]]
	if obj == nil {
		mp.respond(res, nil, 404)
	} else {
		mp.respond(res, obj, 200)
	}
}

func (mp *mockPlatform) postWMSWallet(res http.ResponseWriter, req *http.Request) {
	var obj WMSWalletAPIModel
	mp.getBody(req, &obj)
	obj.ID = nanoid.New()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	obj.Created = now
	obj.Updated = now
	mp.wmsWallets[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+obj.ID] = &obj
	mp.respond(res, &obj, 201)
}

func (mp *mockPlatform) putWMSWallet(res http.ResponseWriter, req *http.Request) {
	obj := mp.wmsWallets[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["wallet"]] // expected behavior of provider is PUT only on exists
	assert.NotNil(mp.t, obj)
	var newObj WMSWalletAPIModel
	mp.getBody(req, &newObj)
	assert.Equal(mp.t, obj.ID, newObj.ID)               // expected behavior of provider
	assert.Equal(mp.t, obj.ID, mux.Vars(req)["wallet"]) // expected behavior of provider
	now := time.Now().UTC().Format(time.RFC3339Nano)
	newObj.Created = obj.Created
	newObj.Updated = now
	mp.wmsWallets[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["wallet"]] = &newObj
	mp.respond(res, &newObj, 200)
}

func (mp *mockPlatform) deleteWMSWallet(res http.ResponseWriter, req *http.Request) {
	obj := mp.wmsWallets[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["wallet"]]
	assert.NotNil(mp.t, obj)
	delete(mp.wmsWallets, mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["wallet"])
	mp.respond(res, nil, 204)
}

func (mp *mockPlatform) patchWMSWallet(res http.ResponseWriter, req *http.Request) {
	obj := mp.wmsWallets[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["wallet"]]
	assert.NotNil(mp.t, obj)
	var newObj WMSWalletAPIModel
	mp.getBody(req, &newObj)
	assert.Equal(mp.t, obj.ID, newObj.ID)
	now := time.Now().UTC().Format(time.RFC3339Nano)
	newObj.Created = obj.Created
	newObj.Updated = now
	mp.wmsWallets[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["wallet"]] = &newObj
	mp.respond(res, &newObj, 200)
}
