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
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	_ "embed"
)

var cms_action_creatapiStep1 = `
resource "kaleido_platform_cms_action_creatapi" "cms_action_creatapi1" {
    environment = "env1"
	service = "service1"
	build = "build1"
	name = "api1"
	api_name = "api1"
    firefly_namespace = "ns1"
	contract_address = "0xaabbccdd"	
}
`

var cms_action_creatapiStep2 = `
resource "kaleido_platform_cms_action_creatapi" "cms_action_creatapi1" {
    environment = "env1"
	service = "service1"
	build = "build1"
	name = "api1"
	api_name = "api1"
    firefly_namespace = "ns1"
	contract_address = "0xaabbccdd"	
	description = "create an API for a thing"
}
`

func TestCMSActionCreateAPI1(t *testing.T) {

	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"POST /endpoint/{env}/{service}/rest/api/v1/actions",
			"GET /endpoint/{env}/{service}/rest/api/v1/actions/{action}",
			"GET /endpoint/{env}/{service}/rest/api/v1/actions/{action}",
			"GET /endpoint/{env}/{service}/rest/api/v1/actions/{action}",
			"GET /endpoint/{env}/{service}/rest/api/v1/actions/{action}",
			"PATCH /endpoint/{env}/{service}/rest/api/v1/actions/{action}",
			"GET /endpoint/{env}/{service}/rest/api/v1/actions/{action}",
			"DELETE /endpoint/{env}/{service}/rest/api/v1/actions/{action}",
			"GET /endpoint/{env}/{service}/rest/api/v1/actions/{action}",
		})
		mp.server.Close()
	}()

	cms_action_creatapi1Resource := "kaleido_platform_cms_action_creatapi.cms_action_creatapi1"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + cms_action_creatapiStep1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(cms_action_creatapi1Resource, "id"),
					resource.TestCheckResourceAttr(cms_action_creatapi1Resource, "name", `api1`),
					resource.TestCheckResourceAttr(cms_action_creatapi1Resource, "firefly_namespace", `ns1`),
				),
			},
			{
				Config: providerConfig + cms_action_creatapiStep2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(cms_action_creatapi1Resource, "id"),
					resource.TestCheckResourceAttr(cms_action_creatapi1Resource, "name", `api1`),
					resource.TestCheckResourceAttr(cms_action_creatapi1Resource, "firefly_namespace", `ns1`),
					resource.TestCheckResourceAttr(cms_action_creatapi1Resource, "description", `create an API for a thing`),
					func(s *terraform.State) error {
						// Compare the final result on the mock-server side
						id := s.RootModule().Resources[cms_action_creatapi1Resource].Primary.Attributes["id"]
						obj := mp.cmsActions[fmt.Sprintf("env1/service1/%s", id)].((*CMSActionCreateAPIAPIModel))
						testJSONEqual(t, obj, fmt.Sprintf(`
						{
							"id": "%[1]s",
							"created": "%[2]s",
							"updated": "%[3]s",
							"name": "api1",
							"type": "createapi",
							"input": {
								"namespace": "ns1",
								"build": {
									"id": "build1"
								},
								"location": {
									"address": "0xaabbccdd"
								},
								"apiName": "api1"
							},
							"output": {
								"status": "pending",
								"apiId": "%[4]s"
							}
						}
						`,
							// generated fields that vary per test run
							id,
							obj.Created.UTC().Format(time.RFC3339Nano),
							obj.Updated.UTC().Format(time.RFC3339Nano),
							obj.Output.APIID,
						))
						return nil
					},
				),
			},
		},
	})
}
