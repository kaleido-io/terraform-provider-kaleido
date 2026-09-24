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

var networkStep1 = `
resource "kaleido_platform_network" "network1" {
    environment = "env1"
	type = "BesuNetwork"
    name = "network1"
    config_json = jsonencode({
        "setting1": "value1"
    })
}
`

var networkStep2 = `
resource "kaleido_platform_network" "network1" {
    environment = "env1"
	type = "BesuNetwork"
    name = "network1"
    config_json = jsonencode({
        "setting1": "value1",
        "setting2": "value2",
    })
}
`

func TestNetwork1(t *testing.T) {

	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"POST /api/v1/environments/{env}/networks",
			"GET /api/v1/environments/{env}/networks/{network}",
			"GET /api/v1/environments/{env}/networks/{network}",
			"GET /api/v1/environments/{env}/networks/{network}",
			"GET /api/v1/environments/{env}/networks/{network}",
			"GET /api/v1/environments/{env}/networks/{network}",
			"GET /api/v1/environments/{env}/networks/{network}", // re-read after Create's readiness check completes
			"PUT /api/v1/environments/{env}/networks/{network}",
			"GET /api/v1/environments/{env}/networks/{network}",
			"GET /api/v1/environments/{env}/networks/{network}",
			"GET /api/v1/environments/{env}/networks/{network}",
			"GET /api/v1/environments/{env}/networks/{network}", // re-read after Update's readiness check completes
			"DELETE /api/v1/environments/{env}/networks/{network}",
			"GET /api/v1/environments/{env}/networks/{network}",
		})
		mp.server.Close()
	}()

	network1Resource := "kaleido_platform_network.network1"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + networkStep1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(network1Resource, "id"),
					resource.TestCheckResourceAttr(network1Resource, "name", `network1`),
					resource.TestCheckResourceAttr(network1Resource, "type", `BesuNetwork`),
					resource.TestCheckResourceAttr(network1Resource, "config_json", `{"setting1":"value1"}`),
					// genesis.json is only generated once the network is ready, and a
					// fresh value each time (see getNetwork) — this must match the
					// mock's current truth, proving Create re-read after
					// waitForReadyStatus instead of keeping the stale, pre-readiness
					// POST response (which would be missing it entirely).
					checkStatusInitFilesMatchesMock(network1Resource, "env1", mp),
				),
			},
			{
				Config: providerConfig + networkStep2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(network1Resource, "id"),
					resource.TestCheckResourceAttr(network1Resource, "name", `network1`),
					resource.TestCheckResourceAttr(network1Resource, "type", `BesuNetwork`),
					resource.TestCheckResourceAttr(network1Resource, "config_json", `{"setting1":"value1","setting2":"value2"}`),
					resource.TestCheckResourceAttr(network1Resource, "info.setting1", `value1`),
					resource.TestCheckResourceAttr(network1Resource, "info.setting2", `value2`),
					// Same as above, but proving Update's re-read: genesis.json is
					// regenerated to a new value on every ready-flip, so a provider
					// reusing the stale PUT response would show step1's old value
					// here instead of this step's new one.
					checkStatusInitFilesMatchesMock(network1Resource, "env1", mp),
					func(s *terraform.State) error {
						// Compare the final result on the mock-server side
						id := s.RootModule().Resources[network1Resource].Primary.Attributes["id"]
						rt := mp.networks[fmt.Sprintf("env1/%s", id)]
						testJSONEqual(t, rt, fmt.Sprintf(`
						{
							"id": "%[1]s",
							"created": "%[2]s",
							"updated": "%[3]s",
							"type": "BesuNetwork",
							"name": "network1",
							"config": {
								"setting1": "value1",
								"setting2": "value2"
							},
							"environmentMemberId": "%[4]s",
							"status": "ready",
							"statusDetails": {
								"initFiles": {
									"genesis.json": "%[5]s"
								}
							}
						}
						`,
							// generated fields that vary per test run
							id,
							rt.Created.UTC().Format(time.RFC3339Nano),
							rt.Updated.UTC().Format(time.RFC3339Nano),
							rt.EnvironmentMemberID,
							rt.StatusDetails["initFiles"].(map[string]interface{})["genesis.json"],
						))
						return nil
					},
				),
			},
		},
	})
}

// checkStatusInitFilesMatchesMock asserts Terraform's state for
// status_init_files.genesis.json equals the mock's current stored value —
// not just that it's set. genesis.json is regenerated on every ready-flip
// (see getNetwork), so this only passes if the provider actually re-read
// after the readiness check, rather than reusing a stale API response.
func checkStatusInitFilesMatchesMock(resourceName, envPrefix string, mp *mockPlatform) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id := s.RootModule().Resources[resourceName].Primary.Attributes["id"]
		rt := mp.networks[fmt.Sprintf("%s/%s", envPrefix, id)]
		if rt == nil || rt.StatusDetails == nil {
			return fmt.Errorf("no mock network found for %s/%s", envPrefix, id)
		}
		initFiles, _ := rt.StatusDetails["initFiles"].(map[string]interface{})
		want, _ := initFiles["genesis.json"].(string)
		return resource.TestCheckResourceAttr(resourceName, "status_init_files.genesis.json", want)(s)
	}
}

func (mp *mockPlatform) getNetwork(res http.ResponseWriter, req *http.Request) {
	rt := mp.networks[mux.Vars(req)["env"]+"/"+mux.Vars(req)["network"]]
	if rt == nil {
		mp.respond(res, nil, 404)
		return
	}
	mp.respond(res, rt, 200)
	// Next time will return ready, with a freshly generated genesis file —
	// mirrors the real API, where initFiles only exist once the network
	// finishes initializing, not at creation/update time. Regenerated (not
	// just set once) so Update's flip produces a different value than
	// Create's: a provider reusing stale, pre-readiness data would show the
	// old value instead of this new one, not just a missing one.
	if rt.Status != "ready" {
		rt.Status = "ready"
		rt.StatusDetails = NetworkStatusDetails{
			"initFiles": map[string]interface{}{
				"genesis.json": fmt.Sprintf("genesis-%s-for-%s", nanoid.New(), rt.ID),
			},
		}
	}
}

func (mp *mockPlatform) postNetwork(res http.ResponseWriter, req *http.Request) {
	var rt NetworkAPIModel
	mp.getBody(req, &rt)
	rt.ID = nanoid.New()
	now := time.Now().UTC()
	rt.Created = &now
	rt.Updated = &now
	rt.EnvironmentMemberID = nanoid.New()
	rt.Status = "pending"
	mp.networks[mux.Vars(req)["env"]+"/"+rt.ID] = &rt
	mp.respond(res, &rt, 201)
}

func (mp *mockPlatform) putNetwork(res http.ResponseWriter, req *http.Request) {
	rt := mp.networks[mux.Vars(req)["env"]+"/"+mux.Vars(req)["network"]] // expected behavior of provider is PUT only on exists
	assert.NotNil(mp.t, rt)
	var newRT NetworkAPIModel
	mp.getBody(req, &newRT)
	assert.Equal(mp.t, rt.ID, newRT.ID)                 // expected behavior of provider
	assert.Equal(mp.t, rt.ID, mux.Vars(req)["network"]) // expected behavior of provider
	now := time.Now().UTC()
	newRT.Created = rt.Created
	newRT.Updated = &now
	newRT.Status = "pending"
	mp.networks[mux.Vars(req)["env"]+"/"+mux.Vars(req)["network"]] = &newRT
	mp.respond(res, &newRT, 200)
}

func (mp *mockPlatform) deleteNetwork(res http.ResponseWriter, req *http.Request) {
	rt := mp.networks[mux.Vars(req)["env"]+"/"+mux.Vars(req)["network"]]
	assert.NotNil(mp.t, rt)
	delete(mp.networks, mux.Vars(req)["env"]+"/"+mux.Vars(req)["network"])
	mp.respond(res, nil, 204)
}
