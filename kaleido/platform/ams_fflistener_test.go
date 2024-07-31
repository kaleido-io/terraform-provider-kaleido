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

var ams_fflistenerStep1 = `
resource "kaleido_platform_ams_fflistener" "ams_fflistener1" {
    environment = "env1"
	service = "service1"
	name = "listener1"
    config_json = jsonencode({
		firstEvent = "0",
		namespace = "ns1",
		taskName = "task1",
		blockchainEvents = {
			abiEvents = [
				{
					name = "event1"
				}
			],
			locations = [
				{
					address = "0x123456"
				}
			]
		}
    })
}
`

var ams_fflistenerStep2 = `
resource "kaleido_platform_ams_fflistener" "ams_fflistener1" {
    environment = "env1"
	service = "service1"
	name = "listener1"
	disabled = true
    config_json = jsonencode({
		firstEvent = "0",
		namespace = "ns1",
		taskName = "task1",
		taskVersion = "2024.01.02",
		blockchainEvents = {
			abiEvents = [
				{
					name = "event1"
				}
			],
			locations = [
				{
					address = "0x123456"
				}
			]
		}
    })

}
`

func TestAMSFFListener1(t *testing.T) {

	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"PUT /endpoint/{env}/{service}/rest/api/v1/listeners/firefly/{listener}", // by name initially
			"GET /endpoint/{env}/{service}/rest/api/v1/listeners/firefly/{listener}",
			"GET /endpoint/{env}/{service}/rest/api/v1/listeners/firefly/{listener}",
			"PUT /endpoint/{env}/{service}/rest/api/v1/listeners/firefly/{listener}", // then by ID
			"GET /endpoint/{env}/{service}/rest/api/v1/listeners/firefly/{listener}",
			"DELETE /endpoint/{env}/{service}/rest/api/v1/listeners/firefly/{listener}",
			"GET /endpoint/{env}/{service}/rest/api/v1/listeners/firefly/{listener}",
		})
		mp.server.Close()
	}()

	ams_fflistener1Resource := "kaleido_platform_ams_fflistener.ams_fflistener1"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + ams_fflistenerStep1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(ams_fflistener1Resource, "id"),
					resource.TestCheckResourceAttr(ams_fflistener1Resource, "disabled", "false"),
				),
			},
			{
				Config: providerConfig + ams_fflistenerStep2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(ams_fflistener1Resource, "id"),
					resource.TestCheckResourceAttr(ams_fflistener1Resource, "disabled", "true"),
					func(s *terraform.State) error {
						// Compare the final result on the mock-server side
						id := s.RootModule().Resources[ams_fflistener1Resource].Primary.Attributes["id"]
						obj := mp.amsFFListeners[fmt.Sprintf("env1/service1/%s", id)]
						testYAMLEqual(t, obj, fmt.Sprintf(`{
								"id": "%[1]s",
								"created": "%[2]s",
								"updated": "%[3]s",
								"name": "listener1",
								"disabled": true,
								"config": {
									"blockchainEvents": {
										"abiEvents": [
											{
												"name": "event1"
											}
										],
										"locations": [
											{
												"address": "0x123456"
											}
										]
									},
									"firstEvent": "0",
									"namespace": "ns1",
									"taskName": "task1",
									"taskVersion": "2024.01.02"
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

func (mp *mockPlatform) getAMSFFListener(res http.ResponseWriter, req *http.Request) {
	obj := mp.amsFFListeners[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["listener"]]
	if obj == nil {
		mp.respond(res, nil, 404)
	} else {
		mp.respond(res, obj, 200)
	}
}

func (mp *mockPlatform) putAMSFFListener(res http.ResponseWriter, req *http.Request) {
	now := time.Now().UTC()
	obj := mp.amsFFListeners[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["listener"]] // expected behavior of provider is PUT only on exists
	var newObj AMSFFListenerAPIModel
	mp.getBody(req, &newObj)
	if obj == nil {
		assert.Equal(mp.t, newObj.Name, mux.Vars(req)["listener"])
		newObj.ID = nanoid.New()
		newObj.Created = now.Format(time.RFC3339Nano)
	} else {
		assert.Equal(mp.t, obj.ID, mux.Vars(req)["listener"])
		newObj.ID = mux.Vars(req)["listener"]
		newObj.Created = obj.Created
	}
	newObj.Updated = now.Format(time.RFC3339Nano)
	mp.amsFFListeners[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+newObj.ID] = &newObj
	mp.respond(res, &newObj, 200)
}

func (mp *mockPlatform) deleteAMSFFListener(res http.ResponseWriter, req *http.Request) {
	obj := mp.amsFFListeners[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["listener"]]
	assert.NotNil(mp.t, obj)
	delete(mp.amsFFListeners, mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["listener"])
	mp.respond(res, nil, 204)
}
