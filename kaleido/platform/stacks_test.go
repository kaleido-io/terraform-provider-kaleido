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

var stacksStep1 = `
resource "kaleido_platform_stack" "stack1" {
	name = "stack1"
	type = "chain_infrastructure"
	environment = "env1"
	network_id = "network123"
}
`

var stacksStep2 = `
resource "kaleido_platform_stack" "stack1" {
	name = "stack1_renamed"
	type = "chain_infrastructure"
	environment = "env1"
	network_id = "network123"
}
`

func TestStacks1(t *testing.T) {

	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"POST /api/v1/environments/{env}/stacks",
			"GET /api/v1/environments/{env}/stacks/{stack}",
			"GET /api/v1/environments/{env}/stacks/{stack}",
			"GET /api/v1/environments/{env}/stacks/{stack}",
			"PUT /api/v1/environments/{env}/stacks/{stack}",
			"GET /api/v1/environments/{env}/stacks/{stack}",
			"DELETE /api/v1/environments/{env}/stacks/{stack}",
			"GET /api/v1/environments/{env}/stacks/{stack}",
		})
		mp.server.Close()
	}()

	Stack1Resource := "kaleido_platform_stack.stack1"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + stacksStep1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(Stack1Resource, "id"),
					resource.TestCheckResourceAttr(Stack1Resource, "name", `stack1`),
					resource.TestCheckResourceAttr(Stack1Resource, "network_id", `network123`),
					resource.TestCheckResourceAttr(Stack1Resource, "type", `chain_infrastructure`),
				),
			},
			{
				Config: providerConfig + stacksStep2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(Stack1Resource, "id"),
					resource.TestCheckResourceAttr(Stack1Resource, "name", `stack1_renamed`),
					resource.TestCheckResourceAttr(Stack1Resource, "network_id", `network123`),
					resource.TestCheckResourceAttr(Stack1Resource, "type", `chain_infrastructure`),
					func(s *terraform.State) error {
						// Compare the final result on the mock-server side
						id := s.RootModule().Resources[Stack1Resource].Primary.Attributes["id"]
						rt := mp.stacks[fmt.Sprintf("env1/%s", id)]
						testJSONEqual(t, rt, fmt.Sprintf(`
						{
							"id": "%[1]s",
							"created": "%[2]s",
							"updated": "%[3]s",
							"type": "chain_infrastructure",
							"name": "stack1_renamed",
							"networkId": "network123",
							"environmentMemberId": "%[4]s"
						}
						`,
							// generated fields that vary per test run
							id,
							rt.Created.UTC().Format(time.RFC3339Nano),
							rt.Updated.UTC().Format(time.RFC3339Nano),
							rt.EnvironmentMemberID,
						))
						return nil
					},
				),
			},
		},
	})
}

func (mp *mockPlatform) getStacks(res http.ResponseWriter, req *http.Request) {
	rt := mp.stacks[mux.Vars(req)["env"]+"/"+mux.Vars(req)["stack"]]
	if rt == nil {
		mp.respond(res, nil, 404)
	} else {
		mp.respond(res, rt, 200)
	}
}

func (mp *mockPlatform) postStacks(res http.ResponseWriter, req *http.Request) {
	var rt StacksAPIModel
	mp.getBody(req, &rt)
	rt.ID = nanoid.New()
	now := time.Now().UTC()
	rt.Created = &now
	rt.Updated = &now
	rt.EnvironmentMemberID = nanoid.New()
	mp.stacks[mux.Vars(req)["env"]+"/"+rt.ID] = &rt
	mp.respond(res, &rt, 201)
}

func (mp *mockPlatform) putStacks(res http.ResponseWriter, req *http.Request) {
	rt := mp.stacks[mux.Vars(req)["env"]+"/"+mux.Vars(req)["stack"]] // expected behavior of provider is PUT only on exists
	assert.NotNil(mp.t, rt)
	var newRT StacksAPIModel
	mp.getBody(req, &newRT)
	assert.Equal(mp.t, rt.ID, newRT.ID)               // expected behavior of provider
	assert.Equal(mp.t, rt.ID, mux.Vars(req)["stack"]) // expected behavior of provider
	now := time.Now().UTC()
	newRT.Created = rt.Created
	newRT.Updated = &now
	mp.stacks[mux.Vars(req)["env"]+"/"+mux.Vars(req)["stack"]] = &newRT
	mp.respond(res, &newRT, 200)
}

func (mp *mockPlatform) deleteStacks(res http.ResponseWriter, req *http.Request) {
	rt := mp.stacks[mux.Vars(req)["env"]+"/"+mux.Vars(req)["stack"]]
	assert.NotNil(mp.t, rt)
	delete(mp.stacks, mux.Vars(req)["env"]+"/"+mux.Vars(req)["stack"])
	mp.respond(res, nil, 204)
}
