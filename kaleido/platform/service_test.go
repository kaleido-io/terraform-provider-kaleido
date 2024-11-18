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
	"regexp"
	"testing"
	"time"

	"github.com/aidarkhanov/nanoid"
	"github.com/gorilla/mux"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/assert"

	_ "embed"
)

var serviceStep1 = `
resource "kaleido_platform_service" "service1" {
    environment = "env1"
	stack = "stack1"
	runtime = "runtime1"
    type = "besu"
    name = "service1"
    config_json = jsonencode({
        "setting1": "value1"
    })
}
`

var serviceStep2 = `
resource "kaleido_platform_service" "service1" {
    environment = "env1"
	stack = "stack1"
	runtime = "runtime1"
    type = "besu"
    name = "service1"
    config_json = jsonencode({
        "setting1": "value1",
        "setting2": "value2",
    })
	hostnames = {
		"host1": [ "api", "ws" ]
	}
	file_sets = {
		"fs1": {
			"files": {
				"hello.txt": {
					"type": "text/plain",
					"data": {
						"text": "world"
					}
				},
				"goodbye.txt": {
					"type": "text/plain",
					"data": {
						"base64": "Y3J1ZWwgd29ybGQK"
					}
				}
			}
		}
	}
	cred_sets = {
		"auth1": {
			"type": "basic_auth"
			"basic_auth": {
				"username": "user1"
				"password": "pass1"
			}
		}
		"key1": {
			"type": "key"
			"key": {
				"value": "abce12345"
			}
		}
	}
}
`

func TestService1(t *testing.T) {

	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"POST /api/v1/environments/{env}/services",
			"GET /api/v1/environments/{env}/services/{service}",
			"GET /api/v1/environments/{env}/services/{service}",
			"GET /api/v1/environments/{env}/services/{service}",
			"GET /api/v1/environments/{env}/services/{service}",
			"GET /api/v1/environments/{env}/services/{service}",
			"GET /api/v1/environments/{env}/services/{service}",
			"PUT /api/v1/environments/{env}/services/{service}",
			"GET /api/v1/environments/{env}/services/{service}",
			"GET /api/v1/environments/{env}/services/{service}",
			"GET /api/v1/environments/{env}/services/{service}",
			"GET /api/v1/environments/{env}/services/{service}",
			"DELETE /api/v1/environments/{env}/services/{service}",
			"GET /api/v1/environments/{env}/services/{service}",
		})
		mp.server.Close()
	}()

	service1Resource := "kaleido_platform_service.service1"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + serviceStep1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(service1Resource, "id"),
					resource.TestCheckResourceAttr(service1Resource, "name", `service1`),
					resource.TestCheckResourceAttr(service1Resource, "type", `besu`),
					resource.TestCheckResourceAttr(service1Resource, "config_json", `{"setting1":"value1"}`),
					resource.TestCheckResourceAttr(service1Resource, "endpoints.%", `1`),
					resource.TestCheckResourceAttr(service1Resource, "endpoints.api.type", `http`),
					resource.TestCheckResourceAttr(service1Resource, "endpoints.api.urls.#", `1`),
					resource.TestMatchResourceAttr(service1Resource, "endpoints.api.urls.0", regexp.MustCompile(`^https://example.com/api/v1/environments/env1/services/.*$`)),
				),
			},
			{
				Config: providerConfig + serviceStep2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(service1Resource, "id"),
					resource.TestCheckResourceAttr(service1Resource, "name", `service1`),
					resource.TestCheckResourceAttr(service1Resource, "type", `besu`),
					resource.TestCheckResourceAttr(service1Resource, "config_json", `{"setting1":"value1","setting2":"value2"}`),
					func(s *terraform.State) error {
						// Compare the final result on the mock-server side
						id := s.RootModule().Resources[service1Resource].Primary.Attributes["id"]
						svc := mp.services[fmt.Sprintf("env1/%s", id)]
						testJSONEqual(t, svc, fmt.Sprintf(`
						{
							"id": "%[1]s",
							"created": "%[2]s",
							"updated": "%[3]s",
							"type": "besu",
							"name": "service1",
							"runtime": {
								"id": "runtime1"
							},
							"environmentMemberId": "%[4]s",
							"stackId": "%[5]s",
							"status": "ready",
							"config": {
								"setting1": "value1",
								"setting2": "value2"
							},
							"endpoints": {
								"api": {
									"type": "http",
									"urls": [
										"https://example.com/api/v1/environments/env1/services/%[1]s"
									]
								},
								"ws": {
									"type": "ws",
									"urls": [
										"wss://example.com/api/v1/environments/env1/services/%[1]s"
									]
								}
							},
							"hostnames": {
								"host1": [
									"api",
									"ws"
								]
							},
							"fileSets": {
								"fs1": {
									"name": "fs1",
									"files": {
										"goodbye.txt": {
											"type": "text/plain",
											"data": {
												"base64": "Y3J1ZWwgd29ybGQK"
											}
										},
										"hello.txt": {
											"type": "text/plain",
											"data": {
												"text": "world"
											}
										}
									}
								}
							},
							"credSets": {
								"auth1": {
									"name": "auth1",
									"type": "basic_auth",
									"basicAuth": {
										"username": "user1",
										"password": "pass1"
									}
								},
								"key1": {
									"name": "key1",
									"type": "key",
									"key": {
										"value": "abce12345"
									}
								}
							},
							"statusDetails": {
								"connectivity": {
									"identity": "192ea525cecb7302efa31283a205142b989217afef2d555a0af8370417e233fe9fa47a11effed21f1dfcfd7887e7ba5d1b983b03980c88c0ef9543f1a2be80c7"
								}
							}
							
						}
						`,
							// generated fields that vary per test run
							id,
							svc.Created.UTC().Format(time.RFC3339Nano),
							svc.Updated.UTC().Format(time.RFC3339Nano),
							svc.EnvironmentMemberID,
							svc.StackId,
						))
						return nil
					},
				),
			},
		},
	})
}

func (mp *mockPlatform) getService(res http.ResponseWriter, req *http.Request) {
	svc := mp.services[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]]
	if svc == nil {
		mp.respond(res, nil, 404)
	} else {
		mp.respond(res, svc, 200)

		// Next +1 time will return ready and have details
		svc.Status = "ready"
		/*endpoints := []map[string]interface{}{
			{
				"host":     "10.244.3.35",
				"port":     30303,
				"protocol": "TCP",
			},
		}*/
		svc.StatusDetails = ServiceStatusDetails{
			Connectivity: &Connectivity{
				Identity: "192ea525cecb7302efa31283a205142b989217afef2d555a0af8370417e233fe9fa47a11effed21f1dfcfd7887e7ba5d1b983b03980c88c0ef9543f1a2be80c7",
			},
		}
	}
}

func (mp *mockPlatform) postService(res http.ResponseWriter, req *http.Request) {
	var svc ServiceAPIModel
	mp.getBody(req, &svc)
	svc.ID = nanoid.New()
	now := time.Now().UTC()
	svc.Created = &now
	svc.Updated = &now
	svc.EnvironmentMemberID = nanoid.New()
	svc.StackId = nanoid.New()
	svc.Status = "pending"
	svc.Endpoints = map[string]ServiceAPIEndpoint{
		"api": {
			Type: "http",
			URLS: []string{fmt.Sprintf("https://example.com/api/v1/environments/%s/services/%s", mux.Vars(req)["env"], svc.ID)},
		},
	}
	mp.services[mux.Vars(req)["env"]+"/"+svc.ID] = &svc
	mp.respond(res, &svc, 201)
}

func (mp *mockPlatform) putService(res http.ResponseWriter, req *http.Request) {
	svc := mp.services[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]] // expected behavior of provider is PUT only on exists
	assert.NotNil(mp.t, svc)
	var newSVC ServiceAPIModel
	mp.getBody(req, &newSVC)
	assert.Equal(mp.t, svc.ID, newSVC.ID)                // expected behavior of provider
	assert.Equal(mp.t, svc.ID, mux.Vars(req)["service"]) // expected behavior of provider
	now := time.Now().UTC()
	newSVC.Created = svc.Created
	newSVC.Updated = &now
	newSVC.Status = "pending"
	newSVC.Endpoints = map[string]ServiceAPIEndpoint{
		"api": {
			Type: "http",
			URLS: []string{fmt.Sprintf("https://example.com/api/v1/environments/%s/services/%s", mux.Vars(req)["env"], svc.ID)},
		},
		// Add in another one on PUT response
		"ws": {
			Type: "ws",
			URLS: []string{fmt.Sprintf("wss://example.com/api/v1/environments/%s/services/%s", mux.Vars(req)["env"], svc.ID)},
		},
	}
	newSVC.StatusDetails = svc.StatusDetails
	mp.services[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]] = &newSVC
	mp.respond(res, &newSVC, 200)
}

func (mp *mockPlatform) deleteService(res http.ResponseWriter, req *http.Request) {
	svc := mp.services[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]]
	assert.NotNil(mp.t, svc)
	delete(mp.services, mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"])
	mp.respond(res, nil, 204)
}
