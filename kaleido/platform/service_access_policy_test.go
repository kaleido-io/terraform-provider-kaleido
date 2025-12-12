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

var serviceAccessPolicyStep1 = `
resource "kaleido_platform_service_access_policy" "serviceAccessPolicy1" {
	application_id = "ap:1234"
	service_id = "s:1234"
	policy = "package key_permission\nimport rego.v1\ndefault allow := false\nis_valid_action(action) := action in [\"resolve\"]\nallow if {\n    is_valid_action(input.action)\n}\n"
}
`

var serviceAccessPolicyStep2 = `
resource "kaleido_platform_service_access_policy" "serviceAccessPolicy1" {
	group_id = "g:1234"
	service_id = "s:1234"
	policy = "package key_permission\nimport rego.v1\ndefault allow := false\nis_valid_action(action) := action in [\"resolve\"]\nallow if {\n    is_valid_action(input.action)\n}\n"
}
`

func TestServiceAccessPolicy1(t *testing.T) {

	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"POST /api/v1/service-access/{service}/policies",
			"GET /api/v1/service-access/{service}/policies/{policy}",
			"GET /api/v1/service-access/{service}/policies/{policy}",
			"DELETE /api/v1/service-access/{service}/policies/{policy}",
			"GET /api/v1/service-access/{service}/policies/{policy}",
			"POST /api/v1/service-access/{service}/policies",
			"GET /api/v1/service-access/{service}/policies/{policy}",
			"DELETE /api/v1/service-access/{service}/policies/{policy}",
			"GET /api/v1/service-access/{service}/policies/{policy}",
		})
		mp.server.Close()
	}()

	serviceAccessPolicy1Resource := "kaleido_platform_service_access_policy.serviceAccessPolicy1"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + serviceAccessPolicyStep1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(serviceAccessPolicy1Resource, "id"),
				),
			},
			{
				Config: providerConfig + serviceAccessPolicyStep2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(serviceAccessPolicy1Resource, "id"),
					func(s *terraform.State) error {
						// Compare the final result on the mock-server side
						id := s.RootModule().Resources[serviceAccessPolicy1Resource].Primary.Attributes["id"]
						rt := mp.serviceAccessPolicies[fmt.Sprintf("s:1234/%s", id)]
						testJSONEqual(t, rt, fmt.Sprintf(`
						{
							"id": "%[1]s",
							"created": "%[2]s",
							"serviceId": "s:1234",
							"groupId": "g:1234",
							"policy": "package key_permission\nimport rego.v1\ndefault allow := false\nis_valid_action(action) := action in [\"resolve\"]\nallow if {\n    is_valid_action(input.action)\n}\n"
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

func (mp *mockPlatform) getServiceAccessPolicy(res http.ResponseWriter, req *http.Request) {
	rt := mp.serviceAccessPolicies[mux.Vars(req)["service"]+"/"+mux.Vars(req)["policy"]]
	if rt == nil {
		mp.respond(res, nil, 404)
	} else {
		mp.respond(res, rt, 200)
	}
}

func (mp *mockPlatform) postServiceAccessPolicy(res http.ResponseWriter, req *http.Request) {
	var rt ServiceAccessPolicyAPIModel
	mp.getBody(req, &rt)
	rt.ID = nanoid.New()
	now := time.Now().UTC()
	rt.Created = &now
	mp.serviceAccessPolicies[mux.Vars(req)["service"]+"/"+rt.ID] = &rt
	mp.respond(res, &rt, 201)
}

func (mp *mockPlatform) deleteServiceAccessPolicy(res http.ResponseWriter, req *http.Request) {
	rt := mp.serviceAccessPolicies[mux.Vars(req)["service"]+"/"+mux.Vars(req)["policy"]]
	assert.NotNil(mp.t, rt)
	delete(mp.serviceAccessPolicies, mux.Vars(req)["service"]+"/"+mux.Vars(req)["policy"])
	mp.respond(res, nil, 204)
}
