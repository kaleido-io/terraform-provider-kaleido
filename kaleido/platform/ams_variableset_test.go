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
	_ "embed"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/aidarkhanov/nanoid"
	"github.com/gorilla/mux"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/assert"
)

var ams_variablesetStep1 = `
  resource "kaleido_platform_ams_variableset" "ams_variableset1" {
  environment = "env1"
	service = "service1"
	name = "variable_set_1"
	classification = "secret" 
  variables_json = "{\"foo\":\"bar\"}"
}
`

var ams_variablesetUpdateDescription = `
  resource "kaleido_platform_ams_variableset" "ams_variableset1" {
  environment = "env1"
  service = "service1"
  name = "variable_set_1"
  description = "this is my updated description"
  classification = "secret" 
  variables_json = "{\"foo\":\"bar\"}"
}
`

var ams_variablesetUpdateVariable = `
  resource "kaleido_platform_ams_variableset" "ams_variableset1" {
  environment = "env1"
  service = "service1"
  name = "variable_set_1"
  description = "this is my updated description"
  classification = "secret" 
  variables_json = "{\"baz\":\"quz\"}"
}
`

func TestAMSVariableSet1(t *testing.T) {

	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"PUT /endpoint/{env}/{service}/rest/api/v1/variable-sets/{variable-set}", // by name initially
			"GET /endpoint/{env}/{service}/rest/api/v1/variable-sets/{variable-set}",
			"GET /endpoint/{env}/{service}/rest/api/v1/variable-sets/{variable-set}",
			"PUT /endpoint/{env}/{service}/rest/api/v1/variable-sets/{variable-set}", // by name initially
			"GET /endpoint/{env}/{service}/rest/api/v1/variable-sets/{variable-set}",
			"GET /endpoint/{env}/{service}/rest/api/v1/variable-sets/{variable-set}",
			"PUT /endpoint/{env}/{service}/rest/api/v1/variable-sets/{variable-set}", // by name initially
			"GET /endpoint/{env}/{service}/rest/api/v1/variable-sets/{variable-set}",
			"DELETE /endpoint/{env}/{service}/rest/api/v1/variable-sets/{variable-set}",
			"GET /endpoint/{env}/{service}/rest/api/v1/variable-sets/{variable-set}",
		})
		mp.server.Close()
	}()

	ams_variableset1Resource := "kaleido_platform_ams_variableset.ams_variableset1"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + ams_variablesetStep1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(ams_variableset1Resource, "id"),
					resource.TestCheckResourceAttr(ams_variableset1Resource, "classification", "secret"),
					resource.TestCheckNoResourceAttr(ams_variableset1Resource, "description"),
				),
			},
			{
				Config: providerConfig + ams_variablesetUpdateDescription,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(ams_variableset1Resource, "id"),
					resource.TestCheckResourceAttr(ams_variableset1Resource, "classification", "secret"),
					resource.TestCheckResourceAttr(ams_variableset1Resource, "description", "this is my updated description"),
					resource.TestCheckResourceAttr(ams_variableset1Resource, "variables_json", "{\"foo\":\"bar\"}"),
				),
			},
			{
				Config: providerConfig + ams_variablesetUpdateVariable,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(ams_variableset1Resource, "id"),
					resource.TestCheckResourceAttr(ams_variableset1Resource, "classification", "secret"),
					resource.TestCheckResourceAttr(ams_variableset1Resource, "description", "this is my updated description"),
					resource.TestCheckResourceAttr(ams_variableset1Resource, "variables_json", "{\"baz\":\"quz\"}"),
					func(s *terraform.State) error {
						// Compare the final result on the mock-server side
						id := s.RootModule().Resources[ams_variableset1Resource].Primary.Attributes["id"]
						obj := mp.amsVariableSets[fmt.Sprintf("env1/service1/%s", id)]
						testJSONEqual(t, obj, fmt.Sprintf(`
						{
							"id": "%[1]s",
							"created": "%[2]s",
							"updated": "%[3]s",
							"name": "variable_set_1",
							"classification":"secret",
							"description": "this is my updated description",
							"variables":{
								"baz":"quz"
							}
						}
						`,
							// generated fields that vary per test run
							id,
							obj.Created.UTC().Format(time.RFC3339Nano),
							obj.Updated.UTC().Format(time.RFC3339Nano),
						))
						return nil
					},
				),
			},
		},
	})
}

func (mp *mockPlatform) getAMSVariableSet(res http.ResponseWriter, req *http.Request) {
	obj := mp.amsVariableSets[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["variable-set"]]
	if obj == nil {
		mp.respond(res, nil, 404)
	} else {
		mp.respond(res, obj, 200)
	}
}

func (mp *mockPlatform) putAMSVariableSet(res http.ResponseWriter, req *http.Request) {
	now := time.Now().UTC()
	obj := mp.amsVariableSets[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["variable-set"]]
	var newObj AMSVariableSetAPIModel
	mp.getBody(req, &newObj)
	if obj == nil {
		assert.Equal(mp.t, newObj.Name, mux.Vars(req)["variable-set"])
		newObj.ID = nanoid.New()
		newObj.Created = &now
	} else {
		assert.Equal(mp.t, obj.ID, mux.Vars(req)["variable-set"])
		newObj.ID = mux.Vars(req)["variable-set"]
		newObj.Created = obj.Created
	}
	newObj.Updated = &now
	mp.amsVariableSets[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+newObj.ID] = &newObj
	mp.respond(res, &newObj, 200)
}

func (mp *mockPlatform) deleteAMSVariableSet(res http.ResponseWriter, req *http.Request) {
	obj := mp.amsVariableSets[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["variable-set"]]
	assert.NotNil(mp.t, obj)
	delete(mp.amsVariableSets, mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["variable-set"])
	mp.respond(res, nil, 204)
}
