// Copyright © Kaleido, Inc. 2025

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

var stackAccessStep1 = `
resource "kaleido_platform_stack_access" "stackAccess1" {
	application_id = "ap:1234"
	stack_id = "st:1234"
}
`

var stackAccessStep2 = `
resource "kaleido_platform_stack_access" "stackAccess1" {
	group_id = "g:1234"
	stack_id = "st:1234"
}
`

func TestStackAccess1(t *testing.T) {

	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"POST /api/v1/stack-access/{stack}/permissions",
			"GET /api/v1/stack-access/{stack}/permissions/{permission}",
			"GET /api/v1/stack-access/{stack}/permissions/{permission}",
			"DELETE /api/v1/stack-access/{stack}/permissions/{permission}",
			"GET /api/v1/stack-access/{stack}/permissions/{permission}",
			"POST /api/v1/stack-access/{stack}/permissions",
			"GET /api/v1/stack-access/{stack}/permissions/{permission}",
			"DELETE /api/v1/stack-access/{stack}/permissions/{permission}",
			"GET /api/v1/stack-access/{stack}/permissions/{permission}",
		})
		mp.server.Close()
	}()

	stackAccess1Resource := "kaleido_platform_stack_access.stackAccess1"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + stackAccessStep1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(stackAccess1Resource, "id"),
				),
			},
			{
				Config: providerConfig + stackAccessStep2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(stackAccess1Resource, "id"),
					func(s *terraform.State) error {
						// Compare the final result on the mock-server side
						id := s.RootModule().Resources[stackAccess1Resource].Primary.Attributes["id"]
						rt := mp.stackAccess[fmt.Sprintf("st:1234/%s", id)]
						testJSONEqual(t, rt, fmt.Sprintf(`
						{
							"id": "%[1]s",
							"created": "%[2]s",
							"stackId": "st:1234",
							"groupId" : "g:1234"
						}
						`,
							// generated fields that vary per test run
							id,
							rt.Created.UTC().Format(time.RFC3339Nano),
						))
						return nil
					},
				),
			},
		},
	})
}

func (mp *mockPlatform) getStackAccessPermission(res http.ResponseWriter, req *http.Request) {
	rt := mp.stackAccess[mux.Vars(req)["stack"]+"/"+mux.Vars(req)["permission"]]
	if rt == nil {
		mp.respond(res, nil, 404)
	} else {
		mp.respond(res, rt, 200)
	}
}

func (mp *mockPlatform) postStackAccessPermission(res http.ResponseWriter, req *http.Request) {
	var rt StackAccessAPIModel
	mp.getBody(req, &rt)
	rt.ID = nanoid.New()
	now := time.Now().UTC()
	rt.Created = &now
	mp.stackAccess[mux.Vars(req)["stack"]+"/"+rt.ID] = &rt
	mp.respond(res, &rt, 201)
}

func (mp *mockPlatform) deleteStackAccessPermission(res http.ResponseWriter, req *http.Request) {
	rt := mp.stackAccess[mux.Vars(req)["stack"]+"/"+mux.Vars(req)["permission"]]
	assert.NotNil(mp.t, rt)
	delete(mp.stackAccess, mux.Vars(req)["stack"]+"/"+mux.Vars(req)["permission"])
	mp.respond(res, nil, 204)
}
