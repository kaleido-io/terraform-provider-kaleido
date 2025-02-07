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

var applicationStep1 = `
resource "kaleido_platform_application" "application1" {
    name = "application1"
}
`

var applicationStep2 = `
resource "kaleido_platform_application" "application1" {
    name = "application1_renamed"
}
`

func TestApplication1(t *testing.T) {

	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"POST /api/v1/applications",
			"GET /api/v1/applications/{application}",
			"GET /api/v1/applications/{application}",
			"GET /api/v1/applications/{application}",
			"PATCH /api/v1/applications/{application}",
			"GET /api/v1/applications/{application}",
			"DELETE /api/v1/applications/{application}",
			"GET /api/v1/applications/{application}",
		})
		mp.server.Close()
	}()

	application1Resource := "kaleido_platform_application.application1"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + applicationStep1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(application1Resource, "id"),
					resource.TestCheckResourceAttr(application1Resource, "name", `application1`),
				),
			},
			{
				Config: providerConfig + applicationStep2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(application1Resource, "id"),
					resource.TestCheckResourceAttr(application1Resource, "name", `application1_renamed`),
					func(s *terraform.State) error {
						// Compare the final result on the mock-server side
						id := s.RootModule().Resources[application1Resource].Primary.Attributes["id"]
						rt := mp.applications[id]
						testJSONEqual(t, rt, fmt.Sprintf(`
						{
							"id": "%[1]s",
							"created": "%[2]s",
							"updated": "%[3]s",
							"name": "application1_renamed",
							"isAdmin": true,
							"enableOAuth": true,
							"oauth": {
								"oidcConfigURL": ""
							}
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

func (mp *mockPlatform) getApplication(res http.ResponseWriter, req *http.Request) {
	rt := mp.applications[mux.Vars(req)["application"]]
	if rt == nil {
		mp.respond(res, nil, 404)
	} else {
		mp.respond(res, rt, 200)
	}
}

func (mp *mockPlatform) postApplication(res http.ResponseWriter, req *http.Request) {
	var rt ApplicationAPIModel
	mp.getBody(req, &rt)
	rt.ID = nanoid.New()
	now := time.Now().UTC()
	rt.Created = &now
	rt.Updated = &now
	mp.applications[rt.ID] = &rt
	mp.respond(res, &rt, 201)
}

func (mp *mockPlatform) patchApplication(res http.ResponseWriter, req *http.Request) {
	rt := mp.applications[mux.Vars(req)["application"]] // expected behavior of provider is PATCH only on exists
	assert.NotNil(mp.t, rt)
	var newRT ApplicationAPIModel
	mp.getBody(req, &newRT)
	assert.Equal(mp.t, rt.ID, newRT.ID)                     // expected behavior of provider
	assert.Equal(mp.t, rt.ID, mux.Vars(req)["application"]) // expected behavior of provider
	now := time.Now().UTC()
	newRT.Created = rt.Created
	newRT.Updated = &now
	mp.applications[mux.Vars(req)["application"]] = &newRT
	mp.respond(res, &newRT, 200)
}

func (mp *mockPlatform) deleteApplication(res http.ResponseWriter, req *http.Request) {
	rt := mp.applications[mux.Vars(req)["application"]]
	assert.NotNil(mp.t, rt)
	delete(mp.applications, mux.Vars(req)["application"])
	mp.respond(res, nil, 204)
}
