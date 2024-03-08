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

var cms_action_deployStep1 = `
resource "kaleido_platform_cms_action_deploy" "cms_action_deploy1" {
    environment = "env1"
	service = "service1"
	build = "build1"
	name = "deploy1"
    firefly_namespace = "ns1"
	signing_key = "0xaabbccdd"	
}
`

var cms_action_deployStep2 = `
resource "kaleido_platform_cms_action_deploy" "cms_action_deploy1" {
    environment = "env1"
	service = "service1"
	build = "build1"
	name = "deploy1"
    firefly_namespace = "ns1"
	signing_key = "0xaabbccdd"
	description = "deploy a thing"
}
`

func TestCMSActionDeploy1(t *testing.T) {

	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"POST /endpoint/{env}/{service}/rest/api/v1/actions",
			"GET /endpoint/{env}/{service}/rest/api/v1/actions/{build}",
			"GET /endpoint/{env}/{service}/rest/api/v1/actions/{build}",
			"GET /endpoint/{env}/{service}/rest/api/v1/actions/{build}",
			"GET /endpoint/{env}/{service}/rest/api/v1/actions/{build}",
			"PATCH /endpoint/{env}/{service}/rest/api/v1/actions/{build}",
			"GET /endpoint/{env}/{service}/rest/api/v1/actions/{build}",
			"DELETE /endpoint/{env}/{service}/rest/api/v1/actions/{build}",
			"GET /endpoint/{env}/{service}/rest/api/v1/actions/{build}",
		})
		mp.server.Close()
	}()

	cms_action_deploy1Resource := "kaleido_platform_cms_action_deploy.cms_action_deploy1"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + cms_action_deployStep1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(cms_action_deploy1Resource, "id"),
					resource.TestCheckResourceAttr(cms_action_deploy1Resource, "name", `deploy1`),
					resource.TestCheckResourceAttr(cms_action_deploy1Resource, "firefly_namespace", `ns1`),
					resource.TestCheckResourceAttrSet(cms_action_deploy1Resource, "transaction_id"),
					resource.TestCheckResourceAttrSet(cms_action_deploy1Resource, "idempotency_key"),
					resource.TestCheckResourceAttrSet(cms_action_deploy1Resource, "operation_id"),
					resource.TestCheckResourceAttrSet(cms_action_deploy1Resource, "contract_address"),
					resource.TestCheckResourceAttrSet(cms_action_deploy1Resource, "block_number"),
				),
			},
			{
				Config: providerConfig + cms_action_deployStep2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(cms_action_deploy1Resource, "id"),
					resource.TestCheckResourceAttr(cms_action_deploy1Resource, "name", `deploy1`),
					resource.TestCheckResourceAttr(cms_action_deploy1Resource, "firefly_namespace", `ns1`),
					resource.TestCheckResourceAttr(cms_action_deploy1Resource, "description", `deploy a thing`),
					resource.TestCheckResourceAttrSet(cms_action_deploy1Resource, "transaction_id"),
					resource.TestCheckResourceAttrSet(cms_action_deploy1Resource, "idempotency_key"),
					resource.TestCheckResourceAttrSet(cms_action_deploy1Resource, "operation_id"),
					resource.TestCheckResourceAttrSet(cms_action_deploy1Resource, "contract_address"),
					resource.TestCheckResourceAttrSet(cms_action_deploy1Resource, "block_number"),
					func(s *terraform.State) error {
						// Compare the final result on the mock-server side
						id := s.RootModule().Resources[cms_action_deploy1Resource].Primary.Attributes["id"]
						obj := mp.cmsActions[fmt.Sprintf("env1/service1/%s", id)].((*CMSActionDeployAPIModel))
						testJSONEqual(t, obj, fmt.Sprintf(`
						{
							"id": "%[1]s",
							"created": "%[2]s",
							"updated": "%[3]s",
							"name": "deploy1",
							"type": "deploy",
							"input": {
								"namespace": "ns1",
								"build": {
									"id": "build1"
								},
								"signingKey": "0xaabbccdd"
							},
							"output": {
								"status": "pending",
								"transactionId": "%[4]s",
								"idempotencyKey": "%[5]s",
								"operationId": "%[6]s",
								"location": {
									"address": "%[7]s"
								},
								"blockNumber": "12345"
							}
						}
						`,
							// generated fields that vary per test run
							id,
							obj.Created.UTC().Format(time.RFC3339Nano),
							obj.Updated.UTC().Format(time.RFC3339Nano),
							obj.Output.TransactionID,
							obj.Output.IdempotencyKey,
							obj.Output.OperationID,
							obj.Output.Location.Address,
							obj.Output.BlockNumber,
						))
						return nil
					},
				),
			},
		},
	})
}
