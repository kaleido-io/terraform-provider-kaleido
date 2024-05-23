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

var groupStep1 = `
resource "kaleido_platform_group" "group1" {
    name = "group1"
}
`

var groupStep2 = `
resource "kaleido_platform_group" "group1" {
    name = "group1_renamed"
}
`

func TestGroup1(t *testing.T) {

	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"POST /api/v1/groups",
			"GET /api/v1/groups/{group}",
			"GET /api/v1/groups/{group}",
			"GET /api/v1/groups/{group}",
			"PATCH /api/v1/groups/{group}",
			"GET /api/v1/groups/{group}",
			"DELETE /api/v1/groups/{group}",
			"GET /api/v1/groups/{group}",
		})
		mp.server.Close()
	}()

	group1Resource := "kaleido_platform_group.group1"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + groupStep1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(group1Resource, "id"),
					resource.TestCheckResourceAttr(group1Resource, "name", `group1`),
				),
			},
			{
				Config: providerConfig + groupStep2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(group1Resource, "id"),
					resource.TestCheckResourceAttr(group1Resource, "name", `group1_renamed`),
					func(s *terraform.State) error {
						// Compare the final result on the mock-server side
						id := s.RootModule().Resources[group1Resource].Primary.Attributes["id"]
						rt := mp.groups[id]
						testJSONEqual(t, rt, fmt.Sprintf(`
						{
							"id": "%[1]s",
							"created": "%[2]s",
							"updated": "%[3]s",
							"name": "group1_renamed"
						}
						`,
							// generated fields that vary per test run
							id,
							rt.Created.UTC().Format(time.RFC3339Nano),
							rt.Updated.UTC().Format(time.RFC3339Nano),
						))
						return nil
					},
				),
			},
		},
	})
}

func (mp *mockPlatform) getGroup(res http.ResponseWriter, req *http.Request) {
	rt := mp.groups[mux.Vars(req)["group"]]
	if rt == nil {
		mp.respond(res, nil, 404)
	} else {
		mp.respond(res, rt, 200)
	}
}

func (mp *mockPlatform) postGroup(res http.ResponseWriter, req *http.Request) {
	var rt GroupAPIModel
	mp.getBody(req, &rt)
	rt.ID = nanoid.New()
	now := time.Now().UTC()
	rt.Created = &now
	rt.Updated = &now
	mp.groups[rt.ID] = &rt
	mp.respond(res, &rt, 201)
}

func (mp *mockPlatform) patchGroup(res http.ResponseWriter, req *http.Request) {
	rt := mp.groups[mux.Vars(req)["group"]] // expected behavior of provider is PATCH only on exists
	assert.NotNil(mp.t, rt)
	var newRT GroupAPIModel
	mp.getBody(req, &newRT)
	assert.Equal(mp.t, rt.ID, newRT.ID)               // expected behavior of provider
	assert.Equal(mp.t, rt.ID, mux.Vars(req)["group"]) // expected behavior of provider
	now := time.Now().UTC()
	newRT.Created = rt.Created
	newRT.Updated = &now
	mp.groups[mux.Vars(req)["group"]] = &newRT
	mp.respond(res, &newRT, 200)
}

func (mp *mockPlatform) deleteGroup(res http.ResponseWriter, req *http.Request) {
	rt := mp.groups[mux.Vars(req)["group"]]
	assert.NotNil(mp.t, rt)
	delete(mp.groups, mux.Vars(req)["group"])
	mp.respond(res, nil, 204)
}
