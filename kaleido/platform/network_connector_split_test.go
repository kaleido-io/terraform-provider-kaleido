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
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	_ "embed"
)

var networkConnectivityPeerDataSourceStep = `
data "kaleido_platform_network_connectivity_peer" "peer1" {
  identity = "496f2bfe5cac576cb33f98778eb5617e3d3fe2e9ffeda8e7d0bde22f5e15d2dd4750f59a268ece9197aa10f4e709012564b514782ea86529c11d02a3c604ee7b"
  endpoints = [
    {
      host     = "10.244.3.64"
      port     = 30303
      protocol = "TCP"
      nat      = "None"
    },
    {
      host     = "86.13.78.205"
      port     = 30303
      protocol = "TCP"
      nat      = "Source"
    }
  ]
}
`

var permittedNetworkConnectorStep = `
data "kaleido_platform_network_connectivity_peer" "peer1" {
  identity = "496f2bfe5cac576cb33f98778eb5617e3d3fe2e9ffeda8e7d0bde22f5e15d2dd4750f59a268ece9197aa10f4e709012564b514782ea86529c11d02a3c604ee7b"
  endpoints = [
    {
      host     = "10.244.3.64"
      port     = 30303
      protocol = "TCP"
      nat      = "None"
    },
    {
      host     = "86.13.78.205"
      port     = 30303
      protocol = "TCP"
      nat      = "Source"
    }
  ]
}

resource "kaleido_platform_permitted_network_connector" "connector1" {
  environment = "env1"
  network     = "net1"
  name        = "perm_conn1"
  zone        = "zone1"
  peers_json  = jsonencode([jsondecode(data.kaleido_platform_network_connectivity_peer.peer1.json)])
}
`

var platformConnectorsSplitStep = `
resource "kaleido_platform_account_network_connector_requestor" "connector_requestor" {
  environment           = "env1"
  network              = "net1"
  name                 = "conn_req"
  zone                 = "zone1"
  target_account_id     = "shared_acct"
  target_environment_id = "env2"
  target_network_id     = "net2"
}

resource "kaleido_platform_account_network_connector_acceptor" "connector_acceptor" {
  environment           = "env2"
  network              = "net2"
  name                 = "conn_accpt"
  zone                 = "zone2"
  target_account_id     = "shared_acct"
  target_environment_id = "env1"
  target_network_id     = "net1"
  target_connector_id   = kaleido_platform_account_network_connector_requestor.connector_requestor.id
}
`

func TestNetworkConnectivityPeerDataSource(t *testing.T) {

	peerDataSource := "data.kaleido_platform_network_connectivity_peer.peer1"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: networkConnectivityPeerDataSourceStep,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(peerDataSource, "json"),
					func(s *terraform.State) error {
						// Check that the JSON output contains the expected structure
						jsonOutput := s.RootModule().Resources[peerDataSource].Primary.Attributes["json"]

						// Basic validation that JSON contains expected fields
						if len(jsonOutput) == 0 {
							return fmt.Errorf("JSON output should not be empty")
						}

						// We could add more detailed JSON parsing validation here
						// For now, just check that we have some JSON output
						return nil
					},
				),
			},
		},
	})
}

func TestPermittedNetworkConnectorSplit(t *testing.T) {

	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"POST /api/v1/environments/{env}/networks/{net}/connectors",
			"GET /api/v1/environments/{env}/networks/{net}/connectors/{connector}",
			"GET /api/v1/environments/{env}/networks/{net}/connectors/{connector}",
			"DELETE /api/v1/environments/{env}/networks/{net}/connectors/{connector}",
			"GET /api/v1/environments/{env}/networks/{net}/connectors/{connector}",
		})
		mp.server.Close()
	}()

	connector1Resource := "kaleido_platform_permitted_network_connector.connector1"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + permittedNetworkConnectorStep,
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
							"name": "perm_conn1",
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

func TestPlatformConnectorsSplit(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"POST /api/v1/environments/{env}/networks/{net}/connectors",               // create requestor
			"POST /api/v1/environments/{env}/networks/{net}/connectors",               // create acceptor
			"GET /api/v1/environments/{env}/networks/{net}/connectors/{connector}",    // check acceptor
			"GET /api/v1/environments/{env}/networks/{net}/connectors/{connector}",    // confirm acceptor is ready
			"GET /api/v1/environments/{env}/networks/{net}/connectors/{connector}",    // check requestor
			"GET /api/v1/environments/{env}/networks/{net}/connectors/{connector}",    // final check
			"DELETE /api/v1/environments/{env}/networks/{net}/connectors/{connector}", // delete acceptor
			"GET /api/v1/environments/{env}/networks/{net}/connectors/{connector}",
			"DELETE /api/v1/environments/{env}/networks/{net}/connectors/{connector}", // delete requestor
			"GET /api/v1/environments/{env}/networks/{net}/connectors/{connector}",
		})
		mp.server.Close()
	}()

	connectorRequestorResource := "kaleido_platform_account_network_connector_requestor.connector_requestor"
	connectorAcceptorResource := "kaleido_platform_account_network_connector_acceptor.connector_acceptor"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + platformConnectorsSplitStep,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(connectorRequestorResource, "id"),
					resource.TestCheckResourceAttrSet(connectorAcceptorResource, "id"),
					resource.TestCheckResourceAttr(connectorRequestorResource, "target_account_id", "shared_acct"),
					resource.TestCheckResourceAttr(connectorRequestorResource, "target_environment_id", "env2"),
					resource.TestCheckResourceAttr(connectorRequestorResource, "target_network_id", "net2"),
					resource.TestCheckResourceAttr(connectorAcceptorResource, "target_account_id", "shared_acct"),
					resource.TestCheckResourceAttr(connectorAcceptorResource, "target_environment_id", "env1"),
					resource.TestCheckResourceAttr(connectorAcceptorResource, "target_network_id", "net1"),
				),
			},
		},
	})
}
