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
	"crypto/sha256"
	"encoding/hex"
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

var ams_policyStep1 = `
resource "kaleido_platform_ams_policy" "ams_policy1" {
    environment = "env1"
	service = "service1"
	name = "ams_policy1"
    document = "document 1"
}
`

var ams_policyStep2 = `
resource "kaleido_platform_ams_policy" "ams_policy1" {
    environment = "env1"
	service = "service1"
	name = "ams_policy1"
	description = "shiny policy that does stuff and more stuff"
    document = "document 2"
	example_input = "input 2"
}
`

func TestAMSPolicy1(t *testing.T) {

	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"PUT /endpoint/{env}/{service}/rest/api/v1/policies/{policy}", // by name initially
			"POST /endpoint/{env}/{service}/rest/api/v1/policies/{policy}/versions",
			"GET /endpoint/{env}/{service}/rest/api/v1/policies/{policy}",
			"GET /endpoint/{env}/{service}/rest/api/v1/policies/{policy}",
			"PATCH /endpoint/{env}/{service}/rest/api/v1/policies/{policy}", // then by ID
			"POST /endpoint/{env}/{service}/rest/api/v1/policies/{policy}/versions",
			"GET /endpoint/{env}/{service}/rest/api/v1/policies/{policy}",
			"DELETE /endpoint/{env}/{service}/rest/api/v1/policies/{policy}",
			"GET /endpoint/{env}/{service}/rest/api/v1/policies/{policy}",
		})
		mp.server.Close()
	}()

	ams_policy1Resource := "kaleido_platform_ams_policy.ams_policy1"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + ams_policyStep1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(ams_policy1Resource, "id"),
					resource.TestCheckResourceAttrSet(ams_policy1Resource, "hash"),
					resource.TestCheckResourceAttrSet(ams_policy1Resource, "applied_version"),
				),
			},
			{
				Config: providerConfig + ams_policyStep2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(ams_policy1Resource, "id"),
					resource.TestCheckResourceAttrSet(ams_policy1Resource, "hash"),
					resource.TestCheckResourceAttrSet(ams_policy1Resource, "applied_version"),
					func(s *terraform.State) error {
						// Compare the final result on the mock-server side
						id := s.RootModule().Resources[ams_policy1Resource].Primary.Attributes["id"]
						obj := mp.amsPolicies[fmt.Sprintf("env1/service1/%s", id)]
						testJSONEqual(t, obj, fmt.Sprintf(`{
								"id": "%[1]s",
								"name": "ams_policy1",
								"description": "shiny policy that does stuff and more stuff",
								"created": "%[2]s",
								"updated": "%[3]s",
								"currentVersion": "%[4]s"
							}`,
							// generated fields that vary per test run
							id,
							obj.Created,
							obj.Updated,
							obj.CurrentVersion,
						))
						return nil
					},
				),
			},
		},
	})
}

func (mp *mockPlatform) getAMSPolicy(res http.ResponseWriter, req *http.Request) {
	obj := mp.amsPolicies[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["policy"]]
	if obj == nil {
		mp.respond(res, nil, 404)
	} else {
		mp.respond(res, obj, 200)
	}
}

func (mp *mockPlatform) putAMSPolicy(res http.ResponseWriter, req *http.Request) {
	now := time.Now().UTC()
	obj := mp.amsPolicies[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["policy"]] // expected behavior of provider is PUT only on exists
	var newObj AMSPolicyAPIModel
	mp.getBody(req, &newObj)
	assert.Nil(mp.t, obj)
	assert.Equal(mp.t, newObj.Name, mux.Vars(req)["policy"])
	newObj.ID = nanoid.New()
	newObj.Created = now.Format(time.RFC3339Nano)
	newObj.Updated = now.Format(time.RFC3339Nano)
	newObj.CurrentVersion = nanoid.New()
	mp.amsPolicies[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+newObj.ID] = &newObj
	mp.respond(res, &newObj, 200)
}

func (mp *mockPlatform) patchAMSPolicy(res http.ResponseWriter, req *http.Request) {
	now := time.Now().UTC()
	obj := mp.amsPolicies[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["policy"]] // expected behavior of provider is PUT only on exists
	var newObj AMSPolicyAPIModel
	mp.getBody(req, &newObj)
	assert.NotNil(mp.t, obj)
	assert.Equal(mp.t, obj.ID, mux.Vars(req)["policy"])
	newObj.ID = mux.Vars(req)["policy"]
	newObj.Created = obj.Created
	newObj.Updated = now.Format(time.RFC3339Nano)
	newObj.CurrentVersion = nanoid.New()
	mp.amsPolicies[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+newObj.ID] = &newObj
	mp.respond(res, &newObj, 200)
}

func (mp *mockPlatform) postAMSPolicyVersion(res http.ResponseWriter, req *http.Request) {
	var newObj AMSPolicyVersionAPIModel
	mp.getBody(req, &newObj)
	mp.amsPolicyVersions[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["policy"]] = &newObj
	hash := sha256.New()
	hash.Write([]byte(newObj.Document))
	newObj.Hash = hex.EncodeToString(hash.Sum(nil))
	mp.respond(res, &newObj, 200)
}

func (mp *mockPlatform) deleteAMSPolicy(res http.ResponseWriter, req *http.Request) {
	obj := mp.amsPolicies[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["policy"]]
	assert.NotNil(mp.t, obj)
	delete(mp.amsPolicies, mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["policy"])
	mp.respond(res, nil, 204)
}
