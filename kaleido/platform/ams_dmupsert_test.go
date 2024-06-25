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
	"net/http"
	"testing"

	"github.com/gorilla/mux"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	_ "embed"
)

var ams_dmupsertStep1 = `
resource "kaleido_platform_ams_dmupsert" "ams_dmupsert1" {
    environment = "env1"
	service = "service1"
    bulk_upsert_yaml = yamlencode(
		{
			"addresses": [
				{
					"updateType": "create_or_replace",
					"address": "0x93976AE88d24130979FE554bFdfF32008839b04B",
					"displayName": "bob"
				}
			]
		}
    )
}
`

var ams_dmupsertStep2 = `
resource "kaleido_platform_ams_dmupsert" "ams_dmupsert1" {
    environment = "env1"
	service = "service1"
    bulk_upsert_yaml = yamlencode(
		{
			"addresses": [
				{
					"updateType": "create_or_replace",
					"address": "0x93976AE88d24130979FE554bFdfF32008839b04B",
					"displayName": "sally"
				}
			]
		}
    )
}
`

func TestAMSDMUpsert1(t *testing.T) {

	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"PUT /endpoint/{env}/{service}/rest/api/v1/bulk/datamodel",
			"PUT /endpoint/{env}/{service}/rest/api/v1/bulk/datamodel",
		})
		mp.server.Close()
	}()

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + ams_dmupsertStep1,
			},
			{
				Config: providerConfig + ams_dmupsertStep2,
				Check: resource.ComposeAggregateTestCheckFunc(
					func(s *terraform.State) error {
						// Compare the final result on the mock-server side
						obj := mp.amsDMUpserts["env1/service1"]
						testYAMLEqual(t, obj, `{
							"addresses": [
								{
									"updateType": "create_or_replace",
									"address": "0x93976AE88d24130979FE554bFdfF32008839b04B",
									"displayName": "sally"
								}
							]
						}`)
						return nil
					},
				),
			},
		},
	})
}

func (mp *mockPlatform) putAMSDMUpsert(res http.ResponseWriter, req *http.Request) {
	var newObj map[string]interface{}
	mp.getBody(req, &newObj)
	mp.amsDMUpserts[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]] = newObj
	mp.respond(res, &newObj, 200)
}
