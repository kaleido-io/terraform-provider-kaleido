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

var apiKeyStep1 = `
resource "kaleido_platform_api_key" "apiKey1" {
    name = "apiKey1"
	application_id = "ap:1234"
	no_expiry = true
}
`

var apiKeyStep2 = `
resource "kaleido_platform_api_key" "apiKey1" {
    name = "apiKey1_renamed"
	application_id = "ap:1234"
	no_expiry = true
}
`

func TestApiKey1(t *testing.T) {

	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"POST /api/v1/applications/{application}/api-keys",
			"GET /api/v1/applications/{application}/api-keys/{api-key}",
			"GET /api/v1/applications/{application}/api-keys/{api-key}",
			"DELETE /api/v1/applications/{application}/api-keys/{api-key}",
			"GET /api/v1/applications/{application}/api-keys/{api-key}",
			"POST /api/v1/applications/{application}/api-keys",
			"GET /api/v1/applications/{application}/api-keys/{api-key}",
			"DELETE /api/v1/applications/{application}/api-keys/{api-key}",
			"GET /api/v1/applications/{application}/api-keys/{api-key}",
		})
		mp.server.Close()
	}()

	apiKey1Resource := "kaleido_platform_api_key.apiKey1"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + apiKeyStep1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(apiKey1Resource, "id"),
					resource.TestCheckResourceAttr(apiKey1Resource, "name", `apiKey1`),
				),
			},
			{
				Config: providerConfig + apiKeyStep2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(apiKey1Resource, "id"),
					resource.TestCheckResourceAttr(apiKey1Resource, "name", `apiKey1_renamed`),
					func(s *terraform.State) error {
						// Compare the final result on the mock-server side
						id := s.RootModule().Resources[apiKey1Resource].Primary.Attributes["id"]
						rt := mp.apiKeys[fmt.Sprintf("ap:1234/%s", id)]
						testJSONEqual(t, rt, fmt.Sprintf(`
						{
							"id": "%[1]s",
							"created": "%[2]s",
							"name": "apiKey1_renamed",
							"application": "ap:1234",
							"no_expiry": true
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

func (mp *mockPlatform) getApiKey(res http.ResponseWriter, req *http.Request) {
	rt := mp.apiKeys[mux.Vars(req)["application"]+"/"+mux.Vars(req)["api-key"]]
	if rt == nil {
		mp.respond(res, nil, 404)
	} else {
		mp.respond(res, rt, 200)
	}
}

func (mp *mockPlatform) postApiKey(res http.ResponseWriter, req *http.Request) {
	var rt APIKeyAPIModel
	mp.getBody(req, &rt)
	rt.ID = nanoid.New()
	now := time.Now().UTC()
	rt.Created = &now
	mp.apiKeys[mux.Vars(req)["application"]+"/"+rt.ID] = &rt
	mp.respond(res, &rt, 201)
}

func (mp *mockPlatform) deleteApiKey(res http.ResponseWriter, req *http.Request) {
	rt := mp.apiKeys[mux.Vars(req)["application"]+"/"+mux.Vars(req)["api-key"]]
	assert.NotNil(mp.t, rt)
	delete(mp.apiKeys, mux.Vars(req)["application"]+"/"+mux.Vars(req)["api-key"])
	mp.respond(res, nil, 204)
}
