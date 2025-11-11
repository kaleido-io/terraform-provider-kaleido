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

var environmentStep1 = `
resource "kaleido_platform_environment" "environment1" {
    name = "environment1"
}
`

var environmentStep2 = `
resource "kaleido_platform_environment" "environment1" {
    name = "environment1_renamed"
	version = "1.0.0"
	update_strategy = "manual"
}
`

func TestEnvironment1(t *testing.T) {

	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"POST /api/v1/environments",
			"GET /api/v1/environments/{env}",
			"GET /api/v1/environments/{env}",
			"GET /api/v1/environments/{env}",
			"PUT /api/v1/environments/{env}",
			"GET /api/v1/environments/{env}",
			"DELETE /api/v1/environments/{env}",
			"GET /api/v1/environments/{env}",
		})
		mp.server.Close()
	}()

	environment1Resource := "kaleido_platform_environment.environment1"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + environmentStep1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(environment1Resource, "id"),
					resource.TestCheckResourceAttr(environment1Resource, "name", `environment1`),
				),
			},
			{
				Config: providerConfig + environmentStep2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(environment1Resource, "id"),
					resource.TestCheckResourceAttr(environment1Resource, "name", `environment1_renamed`),
					resource.TestCheckResourceAttr(environment1Resource, "version", `1.0.0`),
					resource.TestCheckResourceAttr(environment1Resource, "update_strategy", `manual`),
					func(s *terraform.State) error {
						// Compare the final result on the mock-server side
						id := s.RootModule().Resources[environment1Resource].Primary.Attributes["id"]
						rt := mp.environments[id]
						testJSONEqual(t, rt, fmt.Sprintf(`
						{
							"id": "%[1]s",
							"created": "%[2]s",
							"updated": "%[3]s",
							"name": "environment1_renamed",
							"version": "1.0.0",
							"updateStrategy": "manual"
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

func (mp *mockPlatform) getEnvironment(res http.ResponseWriter, req *http.Request) {
	rt := mp.environments[mux.Vars(req)["env"]]
	if rt == nil {
		mp.respond(res, nil, 404)
	} else {
		mp.respond(res, rt, 200)
	}
}

func (mp *mockPlatform) postEnvironment(res http.ResponseWriter, req *http.Request) {
	var rt EnvironmentAPIModel
	mp.getBody(req, &rt)
	rt.ID = nanoid.New()
	now := time.Now().UTC()
	rt.Created = &now
	rt.Updated = &now
	mp.environments[rt.ID] = &rt
	mp.respond(res, &rt, 201)
}

func (mp *mockPlatform) putEnvironment(res http.ResponseWriter, req *http.Request) {
	rt := mp.environments[mux.Vars(req)["env"]] // expected behavior of provider is PUT only on exists
	assert.NotNil(mp.t, rt)
	var newRT EnvironmentAPIModel
	mp.getBody(req, &newRT)
	assert.Equal(mp.t, rt.ID, newRT.ID)             // expected behavior of provider
	assert.Equal(mp.t, rt.ID, mux.Vars(req)["env"]) // expected behavior of provider
	now := time.Now().UTC()
	newRT.Created = rt.Created
	newRT.Updated = &now
	mp.environments[mux.Vars(req)["env"]] = &newRT
	mp.respond(res, &newRT, 200)
}

func (mp *mockPlatform) deleteEnvironment(res http.ResponseWriter, req *http.Request) {
	rt := mp.environments[mux.Vars(req)["env"]]
	assert.NotNil(mp.t, rt)
	delete(mp.environments, mux.Vars(req)["env"])
	mp.respond(res, nil, 204)
}
