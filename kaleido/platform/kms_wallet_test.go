// Copyright © Kaleido, Inc. 2024-2025

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

var kms_walletStep1 = `
resource "kaleido_platform_kms_wallet" "kms_wallet1" {
    environment = "env1"
	service = "service1"
    type = "hdwallet"
    name = "kms_wallet1"
    config_json = jsonencode({
        "setting1": "value1"
    })
		creds_json = jsonencode({
        "cred1": "value1"
    })
}
`

var kms_walletStep2 = `
resource "kaleido_platform_kms_wallet" "kms_wallet1" {
    environment = "env1"
	service = "service1"
    type = "hdwallet"
    name = "kms_wallet1"
    config_json = jsonencode({
        "setting1": "value1"
        "setting2": "value2"
    })
		creds_json = jsonencode({
        "cred1": "value1"
				"cred2": "value2"
    })
}
`

func TestKMSWallet1(t *testing.T) {

	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"POST /endpoint/{env}/{service}/rest/api/v1/wallets",
			"GET /endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}",
			"GET /endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}",
			"GET /endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}",
			"PATCH /endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}",
			"GET /endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}",
			"DELETE /endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}",
			"GET /endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}",
		})
		mp.server.Close()
	}()

	kms_wallet1Resource := "kaleido_platform_kms_wallet.kms_wallet1"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + kms_walletStep1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(kms_wallet1Resource, "id"),
					resource.TestCheckResourceAttr(kms_wallet1Resource, "name", `kms_wallet1`),
					resource.TestCheckResourceAttr(kms_wallet1Resource, "type", `hdwallet`),
					resource.TestCheckResourceAttr(kms_wallet1Resource, "config_json", `{"setting1":"value1"}`),
					resource.TestCheckResourceAttr(kms_wallet1Resource, "creds_json", `{"cred1":"value1"}`),
				),
			},
			{
				Config: providerConfig + kms_walletStep2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(kms_wallet1Resource, "id"),
					resource.TestCheckResourceAttr(kms_wallet1Resource, "name", `kms_wallet1`),
					resource.TestCheckResourceAttr(kms_wallet1Resource, "type", `hdwallet`),
					resource.TestCheckResourceAttr(kms_wallet1Resource, "config_json", `{"setting1":"value1","setting2":"value2"}`),
					resource.TestCheckResourceAttr(kms_wallet1Resource, "creds_json", `{"cred1":"value1","cred2":"value2"}`),
					func(s *terraform.State) error {
						// Compare the final result on the mock-server side
						id := s.RootModule().Resources[kms_wallet1Resource].Primary.Attributes["id"]
						obj := mp.kmsWallets[fmt.Sprintf("env1/service1/%s", id)]
						testJSONEqual(t, obj, fmt.Sprintf(`
						{
							"id": "%[1]s",
							"created": "%[2]s",
							"updated": "%[3]s",
							"type": "hdwallet",
							"name": "kms_wallet1",
							"configuration": {
								"setting1": "value1",
								"setting2": "value2"
							},
							"credentials": {
								"cred1": "value1",
								"cred2": "value2"
							}
						}
						`,
							// generated fields that vary per test run
							id,
							obj.Created.UTC().Format(time.RFC3339Nano),
							obj.Updated.UTC().Format(time.RFC3339Nano),
						))
						return nil
					},
				),
			},
		},
	})
}

func (mp *mockPlatform) getKMSWallet(res http.ResponseWriter, req *http.Request) {
	obj := mp.kmsWallets[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["wallet"]]
	if obj == nil {
		mp.respond(res, nil, 404)
	} else {
		mp.respond(res, obj, 200)
	}
}

func (mp *mockPlatform) postKMSWallet(res http.ResponseWriter, req *http.Request) {
	var obj KMSWalletAPIModel
	mp.getBody(req, &obj)
	obj.ID = nanoid.New()
	now := time.Now().UTC()
	obj.Created = &now
	obj.Updated = &now
	mp.kmsWallets[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+obj.ID] = &obj
	mp.respond(res, &obj, 201)
}

func (mp *mockPlatform) patchKMSWallet(res http.ResponseWriter, req *http.Request) {
	obj := mp.kmsWallets[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["wallet"]] // expected behavior of provider is PATCH only on exists
	assert.NotNil(mp.t, obj)
	var newObj KMSWalletAPIModel
	mp.getBody(req, &newObj)
	assert.Equal(mp.t, obj.ID, newObj.ID)               // expected behavior of provider
	assert.Equal(mp.t, obj.ID, mux.Vars(req)["wallet"]) // expected behavior of provider
	now := time.Now().UTC()
	newObj.Created = obj.Created
	newObj.Updated = &now
	mp.kmsWallets[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["wallet"]] = &newObj
	mp.respond(res, &newObj, 200)
}

func (mp *mockPlatform) deleteKMSWallet(res http.ResponseWriter, req *http.Request) {
	obj := mp.kmsWallets[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["wallet"]]
	assert.NotNil(mp.t, obj)
	delete(mp.kmsWallets, mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["wallet"])
	mp.respond(res, nil, 204)
}

var kms_walletKeyDiscoveryConfigStep1 = `
resource "kaleido_platform_kms_wallet" "kms_wallet_keydiscovery" {
    environment = "env1"
	service = "service1"
    type = "azurekeyvault"
    name = "keystore1"
    creds_json = jsonencode({
        "baseURL": "https://vault.azure.net"
        "clientId": "id1234"
        "clientSecret": "secret1234"
        "keyVaultName": "keyvault1"
        "tenantId": "tenant1234"
    })
    key_discovery_config = {
        secp256k1 = ["address_ethereum"]
    }
}
`

var kms_walletKeyDiscoveryConfigStep2 = `
resource "kaleido_platform_kms_wallet" "kms_wallet_keydiscovery" {
    environment = "env1"
	service = "service1"
    type = "azurekeyvault"
    name = "keystore1"
    creds_json = jsonencode({
        "baseURL": "https://vault.azure.net"
        "clientId": "id1234"
        "clientSecret": "secret1234"
        "keyVaultName": "keyvault1"
        "tenantId": "tenant1234"
    })
    key_discovery_config = {
        secp256k1 = ["address_ethereum", "address_ethereum_checksum"]
    }
}
`

func TestKMSWalletKeyDiscoveryConfig(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"POST /endpoint/{env}/{service}/rest/api/v1/wallets",
			"GET /endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}",
			"GET /endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}",
			"GET /endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}",
			"PATCH /endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}",
			"GET /endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}",
			"DELETE /endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}",
			"GET /endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}",
		})
		mp.server.Close()
	}()

	kms_walletResource := "kaleido_platform_kms_wallet.kms_wallet_keydiscovery"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + kms_walletKeyDiscoveryConfigStep1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(kms_walletResource, "id"),
					resource.TestCheckResourceAttr(kms_walletResource, "name", `keystore1`),
					resource.TestCheckResourceAttr(kms_walletResource, "type", `azurekeyvault`),
					resource.TestCheckResourceAttr(kms_walletResource, "key_discovery_config.%", "1"),
					resource.TestCheckResourceAttr(kms_walletResource, "key_discovery_config.secp256k1.#", "1"),
					resource.TestCheckResourceAttr(kms_walletResource, "key_discovery_config.secp256k1.0", "address_ethereum"),
					func(s *terraform.State) error {
						// Compare the final result on the mock-server side
						id := s.RootModule().Resources[kms_walletResource].Primary.Attributes["id"]
						obj := mp.kmsWallets[fmt.Sprintf("env1/service1/%s", id)]
						assert.NotNil(t, obj)
						assert.Equal(t, "keystore1", obj.Name)
						assert.Equal(t, "azurekeyvault", obj.Type)
						assert.NotNil(t, obj.KeyDiscoveryConfig)
						assert.Equal(t, 1, len(obj.KeyDiscoveryConfig))
						assert.Equal(t, []string{"address_ethereum"}, obj.KeyDiscoveryConfig["secp256k1"])
						testJSONEqual(t, obj, fmt.Sprintf(`
						{
							"id": "%[1]s",
							"created": "%[2]s",
							"updated": "%[3]s",
							"type": "azurekeyvault",
							"name": "keystore1",
							"credentials": {
								"baseURL": "https://vault.azure.net",
								"clientId": "id1234",
								"clientSecret": "secret1234",
								"keyVaultName": "keyvault1",
								"tenantId": "tenant1234"
							},
							"keyDiscoveryConfig": {
								"secp256k1": ["address_ethereum"]
							}
						}
						`,
							// generated fields that vary per test run
							id,
							obj.Created.UTC().Format(time.RFC3339Nano),
							obj.Updated.UTC().Format(time.RFC3339Nano),
						))
						return nil
					},
				),
			},
			{
				Config: providerConfig + kms_walletKeyDiscoveryConfigStep2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(kms_walletResource, "id"),
					resource.TestCheckResourceAttr(kms_walletResource, "name", `keystore1`),
					resource.TestCheckResourceAttr(kms_walletResource, "type", `azurekeyvault`),
					resource.TestCheckResourceAttr(kms_walletResource, "key_discovery_config.%", "1"),
					resource.TestCheckResourceAttr(kms_walletResource, "key_discovery_config.secp256k1.#", "2"),
					resource.TestCheckResourceAttr(kms_walletResource, "key_discovery_config.secp256k1.0", "address_ethereum"),
					resource.TestCheckResourceAttr(kms_walletResource, "key_discovery_config.secp256k1.1", "address_ethereum_checksum"),
					func(s *terraform.State) error {
						// Compare the final result on the mock-server side
						id := s.RootModule().Resources[kms_walletResource].Primary.Attributes["id"]
						obj := mp.kmsWallets[fmt.Sprintf("env1/service1/%s", id)]
						assert.NotNil(t, obj)
						assert.Equal(t, "keystore1", obj.Name)
						assert.Equal(t, "azurekeyvault", obj.Type)
						assert.NotNil(t, obj.KeyDiscoveryConfig)
						assert.Equal(t, 1, len(obj.KeyDiscoveryConfig))
						assert.Equal(t, []string{"address_ethereum", "address_ethereum_checksum"}, obj.KeyDiscoveryConfig["secp256k1"])
						testJSONEqual(t, obj, fmt.Sprintf(`
						{
							"id": "%[1]s",
							"created": "%[2]s",
							"updated": "%[3]s",
							"type": "azurekeyvault",
							"name": "keystore1",
							"credentials": {
								"baseURL": "https://vault.azure.net",
								"clientId": "id1234",
								"clientSecret": "secret1234",
								"keyVaultName": "keyvault1",
								"tenantId": "tenant1234"
							},
							"keyDiscoveryConfig": {
								"secp256k1": ["address_ethereum", "address_ethereum_checksum"]
							}
						}
						`,
							// generated fields that vary per test run
							id,
							obj.Created.UTC().Format(time.RFC3339Nano),
							obj.Updated.UTC().Format(time.RFC3339Nano),
						))
						return nil
					},
				),
			},
		},
	})
}
