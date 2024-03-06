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
	"net/http"
	"testing"
	"time"

	"github.com/aidarkhanov/nanoid"
	"github.com/gorilla/mux"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"

	_ "embed"
)

var runtimeStep1 = `
resource "kaleido_platform_runtime" "runtime1" {
    environment = "env1"
    type = "besu"
    name = "runtime1"
    config_json = jsonencode({
        "setting1": "value1"
    })
}
`

var runtimeStep2 = `
resource "kaleido_platform_runtime" "runtime1" {
    environment = "env1"
    type = "besu"
    name = "runtime1"
    config_json = jsonencode({
        "setting1": "value1",
        "setting2": "value2",
    })
    log_level = "trace"
    size = "large"
    stopped = true
}
`

func TestRuntime1(t *testing.T) {

	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"POST /api/v1/environments/{env}/runtimes",
			"GET /api/v1/environments/{env}/runtimes/{runtime}",
			"GET /api/v1/environments/{env}/runtimes/{runtime}",
			"GET /api/v1/environments/{env}/runtimes/{runtime}",
			"GET /api/v1/environments/{env}/runtimes/{runtime}",
			"GET /api/v1/environments/{env}/runtimes/{runtime}",
			"PUT /api/v1/environments/{env}/runtimes/{runtime}",
			"GET /api/v1/environments/{env}/runtimes/{runtime}",
			"GET /api/v1/environments/{env}/runtimes/{runtime}",
			"GET /api/v1/environments/{env}/runtimes/{runtime}",
			"DELETE /api/v1/environments/{env}/runtimes/{runtime}",
		})
		mp.server.Close()
	}()

	runtime1Resource := "kaleido_platform_runtime.runtime1"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + runtimeStep1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(runtime1Resource, "id"),
					resource.TestCheckResourceAttr(runtime1Resource, "name", `runtime1`),
					resource.TestCheckResourceAttr(runtime1Resource, "type", `besu`),
					resource.TestCheckResourceAttr(runtime1Resource, "config_json", `{"setting1":"value1"}`),
					resource.TestCheckResourceAttr(runtime1Resource, "log_level", `info`),
					resource.TestCheckResourceAttr(runtime1Resource, "size", `small`),
					resource.TestCheckResourceAttr(runtime1Resource, "stopped", `false`),
				),
			},
			{
				Config: providerConfig + runtimeStep2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(runtime1Resource, "id"),
					resource.TestCheckResourceAttr(runtime1Resource, "name", `runtime1`),
					resource.TestCheckResourceAttr(runtime1Resource, "type", `besu`),
					resource.TestCheckResourceAttr(runtime1Resource, "config_json", `{"setting1":"value1","setting2":"value2"}`),
					resource.TestCheckResourceAttr(runtime1Resource, "log_level", `trace`),
					resource.TestCheckResourceAttr(runtime1Resource, "size", `large`),
					resource.TestCheckResourceAttr(runtime1Resource, "stopped", `true`),
				),
			},
		},
	})
}

func (mp *mockPlatform) getRuntime(res http.ResponseWriter, req *http.Request) {
	rt := mp.runtimes[mux.Vars(req)["env"]+"/"+mux.Vars(req)["runtime"]]
	if rt == nil {
		mp.respond(res, nil, 404)
	} else {
		mp.respond(res, rt, 200)
		// Next time will return ready
		rt.Status = "ready"
	}
}

func (mp *mockPlatform) postRuntime(res http.ResponseWriter, req *http.Request) {
	var rt RuntimeAPIModel
	mp.getBody(req, &rt)
	rt.ID = nanoid.New()
	now := time.Now()
	rt.Created = &now
	rt.Updated = &now
	rt.EnvironmentMemberID = nanoid.New()
	if rt.LogLevel == "" {
		rt.LogLevel = "info"
	}
	if rt.Size == "" {
		rt.Size = "small"
	}
	rt.Status = "pending"
	mp.runtimes[mux.Vars(req)["env"]+"/"+rt.ID] = &rt
	mp.respond(res, &rt, 201)
}

func (mp *mockPlatform) putRuntime(res http.ResponseWriter, req *http.Request) {
	rt := mp.runtimes[mux.Vars(req)["env"]+"/"+mux.Vars(req)["runtime"]] // expected behavior of provider is PUT only on exists
	assert.NotNil(mp.t, rt)
	var newRT RuntimeAPIModel
	mp.getBody(req, &newRT)
	assert.Equal(mp.t, rt.ID, newRT.ID)                 // expected behavior of provider
	assert.Equal(mp.t, rt.ID, mux.Vars(req)["runtime"]) // expected behavior of provider
	now := time.Now()
	newRT.Created = rt.Created
	newRT.Updated = &now
	newRT.Status = "pending"
	mp.runtimes[mux.Vars(req)["env"]+"/"+mux.Vars(req)["runtime"]] = &newRT
	mp.respond(res, &newRT, 200)
}

func (mp *mockPlatform) deleteRuntime(res http.ResponseWriter, req *http.Request) {
	rt := mp.runtimes[mux.Vars(req)["env"]+"/"+mux.Vars(req)["runtime"]]
	assert.NotNil(mp.t, rt)
	delete(mp.runtimes, mux.Vars(req)["env"]+"/"+mux.Vars(req)["runtime"])
	mp.respond(res, nil, 204)
}
