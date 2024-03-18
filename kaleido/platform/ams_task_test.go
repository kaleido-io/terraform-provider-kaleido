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

var ams_taskStep1 = `
resource "kaleido_platform_ams_task" "ams_task1" {
    environment = "env1"
	service = "service1"
    task_yaml = yamlencode({
		name = "ams_task1"
		steps = [{
          name = "step1"
		  things = "stuff"
		}]
    })
}
`

var ams_taskStep2 = `
resource "kaleido_platform_ams_task" "ams_task1" {
    environment = "env1"
	service = "service1"
    task_yaml = yamlencode({
		name = "ams_task1"
		description = "shiny task that does stuff and more stuff"
		steps = [{
          name = "step1"
		  things = "stuff"
		},
		{
		  name = "step2"
		  stuff = "other stuff"
		}]
    })
}
`

func TestAMSTask1(t *testing.T) {

	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"PUT /endpoint/{env}/{service}/rest/api/v1/tasks/{task}", // by name initially
			"GET /endpoint/{env}/{service}/rest/api/v1/tasks/{task}",
			"GET /endpoint/{env}/{service}/rest/api/v1/tasks/{task}",
			"PUT /endpoint/{env}/{service}/rest/api/v1/tasks/{task}", // then by ID
			"GET /endpoint/{env}/{service}/rest/api/v1/tasks/{task}",
			"DELETE /endpoint/{env}/{service}/rest/api/v1/tasks/{task}",
			"GET /endpoint/{env}/{service}/rest/api/v1/tasks/{task}",
		})
		mp.server.Close()
	}()

	ams_task1Resource := "kaleido_platform_ams_task.ams_task1"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + ams_taskStep1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(ams_task1Resource, "id"),
					resource.TestCheckResourceAttrSet(ams_task1Resource, "applied_version"),
				),
			},
			{
				Config: providerConfig + ams_taskStep2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(ams_task1Resource, "id"),
					resource.TestCheckResourceAttrSet(ams_task1Resource, "applied_version"),
					func(s *terraform.State) error {
						// Compare the final result on the mock-server side
						id := s.RootModule().Resources[ams_task1Resource].Primary.Attributes["id"]
						obj := mp.amsTasks[fmt.Sprintf("env1/service1/%s", id)]
						testYAMLEqual(t, obj, fmt.Sprintf(`{
								"id": "%[1]s",
								"name": "ams_task1",
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

func (mp *mockPlatform) getAMSTask(res http.ResponseWriter, req *http.Request) {
	obj := mp.amsTasks[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["task"]]
	if obj == nil {
		mp.respond(res, nil, 404)
	} else {
		mp.respond(res, obj, 200)
	}
}

func (mp *mockPlatform) putAMSTask(res http.ResponseWriter, req *http.Request) {
	now := time.Now().UTC()
	obj := mp.amsTasks[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["task"]] // expected behavior of provider is PUT only on exists
	var newObj AMSTaskAPIModel
	mp.getBody(req, &newObj)
	if obj == nil {
		assert.Equal(mp.t, newObj.Name, mux.Vars(req)["task"])
		newObj.ID = nanoid.New()
		newObj.Created = now.Format(time.RFC3339Nano)
	} else {
		assert.Equal(mp.t, obj.ID, mux.Vars(req)["task"])
		newObj.ID = mux.Vars(req)["task"]
		newObj.Created = obj.Created
	}
	newObj.Updated = now.Format(time.RFC3339Nano)
	newObj.CurrentVersion = nanoid.New()
	mp.amsTasks[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+newObj.ID] = &newObj
	mp.respond(res, &newObj, 200)
}

func (mp *mockPlatform) deleteAMSTask(res http.ResponseWriter, req *http.Request) {
	obj := mp.amsTasks[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["task"]]
	assert.NotNil(mp.t, obj)
	delete(mp.amsTasks, mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["task"])
	mp.respond(res, nil, 204)
}