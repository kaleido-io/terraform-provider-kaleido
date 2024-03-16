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

var kms_keyStep1 = `
resource "kaleido_platform_kms_key" "kms_key1" {
    environment = "env1"
	service = "service1"
	wallet = "wallet1_id"
    name = "kms_key1"
}
`

var kms_keyStep2 = `
resource "kaleido_platform_kms_key" "kms_key1" {
    environment = "env1"
	service = "service1"
	wallet = "wallet1_id"
    name = "kms_key1"
	path = "some/path"
}
`

func TestKMSKey1(t *testing.T) {

	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"GET /endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}",
			"PUT /endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}/keys",
			"GET /endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}",
			"GET /endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}/keys/{key}",
			"GET /endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}",
			"GET /endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}/keys/{key}",
			"GET /endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}",
			"GET /endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}/keys/{key}",
			"PATCH /endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}/keys/{key}",
			"GET /endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}",
			"GET /endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}/keys/{key}",
			"GET /endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}",
			"DELETE /endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}/keys/{key}",
			"GET /endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}/keys/{key}",
		})
		mp.server.Close()
	}()

	// KMS requires an external ID->Name resolution before making API key calls
	mp.kmsWallets["env1/service1/wallet1_id"] = &KMSWalletAPIModel{Name: "wallet1"}

	kms_key1Resource := "kaleido_platform_kms_key.kms_key1"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + kms_keyStep1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(kms_key1Resource, "id"),
					resource.TestCheckResourceAttr(kms_key1Resource, "name", `kms_key1`),
					resource.TestCheckResourceAttr(kms_key1Resource, "uri", `uri/for/kms_key1`),
					resource.TestCheckResourceAttrSet(kms_key1Resource, "address"),
				),
			},
			{
				Config: providerConfig + kms_keyStep2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(kms_key1Resource, "id"),
					resource.TestCheckResourceAttr(kms_key1Resource, "name", `kms_key1`),
					resource.TestCheckResourceAttr(kms_key1Resource, "path", `some/path`),
					resource.TestCheckResourceAttr(kms_key1Resource, "uri", `uri/for/kms_key1`),
					resource.TestCheckResourceAttrSet(kms_key1Resource, "address"),
					func(s *terraform.State) error {
						// Compare the final result on the mock-server side
						id := s.RootModule().Resources[kms_key1Resource].Primary.Attributes["id"]
						obj := mp.kmsKeys[fmt.Sprintf("env1/service1/wallet1/%s", id)]
						testJSONEqual(t, obj, fmt.Sprintf(`
						{
							"id": "%[1]s",
							"created": "%[2]s",
							"updated": "%[3]s",
							"name": "kms_key1",
							"path": "some/path",
							"address": "%[4]s",
							"uri": "uri/for/kms_key1"
						}
						`,
							// generated fields that vary per test run
							id,
							obj.Created.UTC().Format(time.RFC3339Nano),
							obj.Updated.UTC().Format(time.RFC3339Nano),
							obj.Address,
						))
						return nil
					},
				),
			},
		},
	})
}

func (mp *mockPlatform) getKMSKey(res http.ResponseWriter, req *http.Request) {
	obj := mp.kmsKeys[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["wallet"]+"/"+mux.Vars(req)["key"]]
	if obj == nil {
		mp.respond(res, nil, 404)
	} else {
		mp.respond(res, obj, 200)
	}
}

func (mp *mockPlatform) putKMSKey(res http.ResponseWriter, req *http.Request) {
	var obj KMSKeyAPIModel
	mp.getBody(req, &obj)
	obj.ID = nanoid.New()
	now := time.Now().UTC()
	obj.Created = &now
	obj.Updated = &now
	obj.Address = nanoid.New()
	obj.URI = "uri/for/" + obj.Name
	mp.kmsKeys[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["wallet"]+"/"+obj.ID] = &obj
	mp.respond(res, &obj, 201)
}

func (mp *mockPlatform) patchKMSKey(res http.ResponseWriter, req *http.Request) {
	obj := mp.kmsKeys[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["wallet"]+"/"+mux.Vars(req)["key"]] // expected behavior of provider is PUT only on exists
	assert.NotNil(mp.t, obj)
	var newObj KMSKeyAPIModel
	mp.getBody(req, &newObj)
	assert.Equal(mp.t, obj.ID, newObj.ID)            // expected behavior of provider
	assert.Equal(mp.t, obj.ID, mux.Vars(req)["key"]) // expected behavior of provider
	now := time.Now().UTC()
	newObj.Created = obj.Created
	newObj.Updated = &now
	newObj.URI = "uri/for/" + newObj.Name
	mp.kmsKeys[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["wallet"]+"/"+mux.Vars(req)["key"]] = &newObj
	mp.respond(res, &newObj, 200)
}

func (mp *mockPlatform) deleteKMSKey(res http.ResponseWriter, req *http.Request) {
	obj := mp.kmsKeys[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["wallet"]+"/"+mux.Vars(req)["key"]]
	assert.NotNil(mp.t, obj)
	delete(mp.kmsKeys, mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["wallet"]+"/"+mux.Vars(req)["key"])
	mp.respond(res, nil, 204)
}
