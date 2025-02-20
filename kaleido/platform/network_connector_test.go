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

var connectorStep1 = `
resource "kaleido_network_connector" "connector1" {
    environment = "env1"
	network = "net1"
    type = "permitted"
    name = "nconn1"
	zone = "zone1"
    permitted_json = jsonencode({"peers":[{"endpoints":[{"host":"10.244.3.64","nat":"None","port":30303,"protocol":"TCP"},{"host":"86.13.78.205","nat":"Source","port":30303,"protocol":"TCP"}],"identity":"496f2bfe5cac576cb33f98778eb5617e3d3fe2e9ffeda8e7d0bde22f5e15d2dd4750f59a268ece9197aa10f4e709012564b514782ea86529c11d02a3c604ee7b"}]})
}
`

var connectorStep2 = `
resource "kaleido_network_connector" "connector1" {
    environment = "env1"
	network = "net1"
    type = "permitted"
    name = "nconn1"
	zone = "zone1"
    permitted_json = jsonencode({"peers":[{"endpoints":[{"host":"10.244.3.64","nat":"None","port":30303,"protocol":"TCP"},{"host":"86.13.78.205","nat":"Source","port":30303,"protocol":"TCP"}],"identity":"496f2bfe5cac576cb33f98778eb5617e3d3fe2e9ffeda8e7d0bde22f5e15d2dd4750f59a268ece9197aa10f4e709012564b514782ea86529c11d02a3c604ee7b"}]})
}
`

func TestConnector(t *testing.T) {

	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"POST /api/v1/environments/{env}/networks/{net}/connectors",
			"GET /api/v1/environments/{env}/networks/{net}/connectors/{connector}",
			"GET /api/v1/environments/{env}/networks/{net}/connectors/{connector}",
			"GET /api/v1/environments/{env}/networks/{net}/connectors/{connector}",
			"GET /api/v1/environments/{env}/networks/{net}/connectors/{connector}",
			"GET /api/v1/environments/{env}/networks/{net}/connectors/{connector}",
			"DELETE /api/v1/environments/{env}/networks/{net}/connectors/{connector}",
			"GET /api/v1/environments/{env}/networks/{net}/connectors/{connector}",
		})
		mp.server.Close()
	}()

	connector1Resource := "kaleido_network_connector.connector1"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + connectorStep1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(connector1Resource, "id"),
				),
			},
			{
				Config: providerConfig + connectorStep2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(connector1Resource, "id"),
					func(s *terraform.State) error {
						// Compare the final result on the mock-server side
						id := s.RootModule().Resources[connector1Resource].Primary.Attributes["id"]
						auth := mp.connectors[fmt.Sprintf("env1/net1/%s", id)]
						testJSONEqual(t, auth, fmt.Sprintf(`
						{
							"id": "%[1]s",
							"created": "%[2]s",
							"updated": "%[3]s",
							"type": "permitted",
							"name": "nconn1",
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

func (mp *mockPlatform) getConnector(res http.ResponseWriter, req *http.Request) {
	auth := mp.connectors[mux.Vars(req)["env"]+"/"+mux.Vars(req)["net"]+"/"+mux.Vars(req)["connector"]]
	if auth == nil {
		mp.respond(res, nil, 404)
	} else {
		mp.respond(res, auth, 200)
		// Next time will return ready
		auth.Status = "ready"
	}
}

func (mp *mockPlatform) postConnector(res http.ResponseWriter, req *http.Request) {
	var conn ConnectorAPIModel
	mp.getBody(req, &conn)
	conn.ID = nanoid.New()
	now := time.Now().UTC()
	conn.Created = &now
	conn.Updated = &now
	conn.Status = "pending"
	mp.connectors[mux.Vars(req)["env"]+"/"+mux.Vars(req)["net"]+"/"+conn.ID] = &conn
	mp.respond(res, &conn, 201)
}

func (mp *mockPlatform) putConnector(res http.ResponseWriter, req *http.Request) {
	auth := mp.connectors[mux.Vars(req)["env"]+"/"+mux.Vars(req)["net"]+"/"+mux.Vars(req)["connector"]]
	assert.NotNil(mp.t, auth)
	var newConn ConnectorAPIModel
	mp.getBody(req, &newConn)
	assert.Equal(mp.t, auth.ID, newConn.ID)                 // expected behavior of provider
	assert.Equal(mp.t, auth.ID, mux.Vars(req)["connector"]) // expected behavior of provider
	now := time.Now().UTC()
	newConn.Created = auth.Created
	newConn.Updated = &now
	newConn.Status = "pending"
	mp.connectors[mux.Vars(req)["env"]+"/"+mux.Vars(req)["net"]+"/"+mux.Vars(req)["connector"]] = &newConn
	mp.respond(res, &newConn, 200)
}

func (mp *mockPlatform) deleteConnector(res http.ResponseWriter, req *http.Request) {
	rt := mp.connectors[mux.Vars(req)["env"]+"/"+mux.Vars(req)["net"]+"/"+mux.Vars(req)["connector"]]
	assert.NotNil(mp.t, rt)
	delete(mp.connectors, mux.Vars(req)["env"]+"/"+mux.Vars(req)["net"]+"/"+mux.Vars(req)["connector"])
	mp.respond(res, nil, 204)
}
