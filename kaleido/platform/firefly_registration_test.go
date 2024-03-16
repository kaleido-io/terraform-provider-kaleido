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

	"github.com/aidarkhanov/nanoid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/assert"

	_ "embed"
)

var firefly_registrationStep1 = `
resource "kaleido_platform_firefly_registration" "firefly_registration1" {
    environment = "env1"
	service = "service1"
}
`

func TestFireFlyRegistration1(t *testing.T) {

	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"GET /endpoint/{env}/{service}/rest/api/v1/status",
			"POST /endpoint/{env}/{service}/rest/api/v1/network/organizations/self",
			"GET /endpoint/{env}/{service}/rest/api/v1/status",
			"POST /endpoint/{env}/{service}/rest/api/v1/network/nodes/self",
			"GET /endpoint/{env}/{service}/rest/api/v1/status",
			"GET /endpoint/{env}/{service}/rest/api/v1/status",
		})
		mp.server.Close()
	}()

	firefly_registration1Resource := "kaleido_platform_firefly_registration.firefly_registration1"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + firefly_registrationStep1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(firefly_registration1Resource, "org_did"),
					resource.TestCheckResourceAttr(firefly_registration1Resource, "org_verifiers.#", "1"),
					func(s *terraform.State) error {
						assert.Equal(t, mp.ffsNode.ID, s.RootModule().Resources[firefly_registration1Resource].Primary.Attributes["node_id"])
						assert.Equal(t, mp.ffsOrg.ID, s.RootModule().Resources[firefly_registration1Resource].Primary.Attributes["org_id"])
						assert.Equal(t, mp.ffsOrg.DID, s.RootModule().Resources[firefly_registration1Resource].Primary.Attributes["org_did"])
						assert.Equal(t, mp.ffsOrg.Verifiers[0].Value, s.RootModule().Resources[firefly_registration1Resource].Primary.Attributes["org_verifiers.0"])
						return nil
					},
				),
			},
		},
	})
}

func (mp *mockPlatform) postFireFlyRegistrationNode(res http.ResponseWriter, _ *http.Request) {
	mp.ffsNode = &FireFlyStatusNodeAPIModel{
		Registered: true,
		Name:       "node1",
		ID:         nanoid.New(),
	}
	mp.respond(res, nil, 204)
}

func (mp *mockPlatform) postFireFlyRegistrationOrg(res http.ResponseWriter, _ *http.Request) {
	mp.ffsOrg = &FireFlyStatusOrgAPIModel{
		Registered: true,
		Name:       "org1",
		ID:         nanoid.New(),
		DID:        "did:firefly:org/org1",
		Verifiers: []FireFlyStatusVerifierAPIModel{
			{Type: "ethereum_address", Value: nanoid.New()},
		},
	}
	mp.respond(res, nil, 204)
}

func (mp *mockPlatform) getFireFlyStatus(res http.ResponseWriter, _ *http.Request) {
	var status FireFlyStatusAPIModel
	if mp.ffsNode != nil {
		status.Node = *mp.ffsNode
	}
	if mp.ffsOrg != nil {
		status.Org = *mp.ffsOrg
	}
	mp.respond(res, &status, 200)
}
