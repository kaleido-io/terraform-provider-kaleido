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

var authenticatorStep1 = `
resource "kaleido_platform_authenticator" "authenticator1" {
    environment = "env1"
	network = "net1"
    type = "permitted"
    name = "auth1"
	zone = "zone1"
	conn = "conn1"
    permitted_json = jsonencode({"peers":[{"endpoints":[{"host":"10.244.3.64","nat":"None","port":30303,"protocol":"TCP"},{"host":"86.13.78.205","nat":"Source","port":30303,"protocol":"TCP"}],"identity":"496f2bfe5cac576cb33f98778eb5617e3d3fe2e9ffeda8e7d0bde22f5e15d2dd4750f59a268ece9197aa10f4e709012564b514782ea86529c11d02a3c604ee7b"}]})
}
`

var authenticatorStep2 = `
resource "kaleido_platform_authenticator" "authenticator1" {
    environment = "env1"
	network = "net1"
    type = "permitted"
    name = "auth1"
	zone = "zone1"
	conn = "conn1"
    permitted_json = jsonencode({"peers":[{"endpoints":[{"host":"10.244.3.64","nat":"None","port":30303,"protocol":"TCP"},{"host":"86.13.78.205","nat":"Source","port":30303,"protocol":"TCP"}],"identity":"496f2bfe5cac576cb33f98778eb5617e3d3fe2e9ffeda8e7d0bde22f5e15d2dd4750f59a268ece9197aa10f4e709012564b514782ea86529c11d02a3c604ee7b"}]})
}
`

func TestAuthenticator(t *testing.T) {

	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"POST /api/v1/environments/{env}/networks/{net}/authenticators",
			"GET /api/v1/environments/{env}/networks/{net}/authenticators/{authenticator}",
			"GET /api/v1/environments/{env}/networks/{net}/authenticators/{authenticator}",
			"GET /api/v1/environments/{env}/networks/{net}/authenticators/{authenticator}",
			"GET /api/v1/environments/{env}/networks/{net}/authenticators/{authenticator}",
			"GET /api/v1/environments/{env}/networks/{net}/authenticators/{authenticator}",
			"DELETE /api/v1/environments/{env}/networks/{net}/authenticators/{authenticator}",
			"GET /api/v1/environments/{env}/networks/{net}/authenticators/{authenticator}",
		})
		mp.server.Close()
	}()

	authenticator1Resource := "kaleido_platform_authenticator.authenticator1"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + authenticatorStep1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(authenticator1Resource, "id"),
				),
			},
			{
				Config: providerConfig + authenticatorStep2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(authenticator1Resource, "id"),
					func(s *terraform.State) error {
						// Compare the final result on the mock-server side
						id := s.RootModule().Resources[authenticator1Resource].Primary.Attributes["id"]
						auth := mp.authenticators[fmt.Sprintf("env1/net1/%s", id)]
						testJSONEqual(t, auth, fmt.Sprintf(`
						{
							"id": "%[1]s",
							"created": "%[2]s",
							"updated": "%[3]s",
							"type": "permitted",
							"name": "auth1",
							"networkId": "net1",
							"permitted": {
							  "peers": [
					            {
							      "endpoints": [
								     {
								       "host": "10.244.3.64",
								       "nat": "None",
								       "port": 30303,
								       "protocol": "TCP"
							         },
									 {
								       "host": "86.13.78.205",
								       "nat": "Source",
								       "port": 30303,
								       "protocol": "TCP"
							         }
								  ],
								  "identity": "496f2bfe5cac576cb33f98778eb5617e3d3fe2e9ffeda8e7d0bde22f5e15d2dd4750f59a268ece9197aa10f4e709012564b514782ea86529c11d02a3c604ee7b"
					            }
							  ]
							},
							"zone": "zone1",
							"connection": "conn1",
							"status": "ready"
						}
						`,
							// generated fields that vary per test run
							id,
							auth.Created.UTC().Format(time.RFC3339Nano),
							auth.Updated.UTC().Format(time.RFC3339Nano),
						))
						return nil
					},
				),
			},
		},
	})
}

func (mp *mockPlatform) getAuthenticator(res http.ResponseWriter, req *http.Request) {
	auth := mp.authenticators[mux.Vars(req)["env"]+"/"+mux.Vars(req)["net"]+"/"+mux.Vars(req)["authenticator"]]
	if auth == nil {
		mp.respond(res, nil, 404)
	} else {
		mp.respond(res, auth, 200)
		// Next time will return ready
		auth.Status = "ready"
	}
}

func (mp *mockPlatform) postAuthenticator(res http.ResponseWriter, req *http.Request) {
	var auth AuthenticatorAPIModel
	mp.getBody(req, &auth)
	auth.ID = nanoid.New()
	now := time.Now().UTC()
	auth.Created = &now
	auth.Updated = &now
	auth.Status = "pending"
	mp.authenticators[mux.Vars(req)["env"]+"/"+mux.Vars(req)["net"]+"/"+auth.ID] = &auth
	mp.respond(res, &auth, 201)
}

func (mp *mockPlatform) putAuthenticator(res http.ResponseWriter, req *http.Request) {
	auth := mp.authenticators[mux.Vars(req)["env"]+"/"+mux.Vars(req)["net"]+"/"+mux.Vars(req)["authenticator"]]
	assert.NotNil(mp.t, auth)
	var newAuth AuthenticatorAPIModel
	mp.getBody(req, &newAuth)
	assert.Equal(mp.t, auth.ID, newAuth.ID)                     // expected behavior of provider
	assert.Equal(mp.t, auth.ID, mux.Vars(req)["authenticator"]) // expected behavior of provider
	now := time.Now().UTC()
	newAuth.Created = auth.Created
	newAuth.Updated = &now
	newAuth.Status = "pending"
	mp.authenticators[mux.Vars(req)["env"]+"/"+mux.Vars(req)["net"]+"/"+mux.Vars(req)["authenticator"]] = &newAuth
	mp.respond(res, &newAuth, 200)
}

func (mp *mockPlatform) deleteAuthenticator(res http.ResponseWriter, req *http.Request) {
	rt := mp.authenticators[mux.Vars(req)["env"]+"/"+mux.Vars(req)["net"]+"/"+mux.Vars(req)["authenticator"]]
	assert.NotNil(mp.t, rt)
	delete(mp.authenticators, mux.Vars(req)["env"]+"/"+mux.Vars(req)["net"]+"/"+mux.Vars(req)["authenticator"])
	mp.respond(res, nil, 204)
}
