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

var pms_identity_step1 = `
resource "kaleido_platform_pms_identity" "test_identity" {
  environment = "test-env"
  service = "test-service"
  name = "test-identity"
  description = "Test identity for policy management"
  owner = "user123"
  preferred_assertion_method = "local"
  assertion_method = [
    {
      name = "key-1"
      type = "ethereum-address"
      verification_material = "0x1234567890123456789012345678901234567890"
      signing_method = "local"
    }
  ]
  notification_method = [
    {
      name = "test-notification-method"
      type = "workflow"
      value = "{\"workflow\": \"test-workflow\", \"operation\": \"test-operation\"}"
    }
  ]
}
`

func TestPMSIdentity1(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"POST /endpoint/{env}/{service}/rest/api/v1/identities",
			"GET /endpoint/{env}/{service}/rest/api/v1/identities/{identity}",
			"DELETE /endpoint/{env}/{service}/rest/api/v1/identities/{identity}",
		})
		mp.server.Close()
	}()

	pms_identity_resource := "kaleido_platform_pms_identity.test_identity"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + pms_identity_step1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(pms_identity_resource, "id"),
					resource.TestCheckResourceAttr(pms_identity_resource, "environment", "test-env"),
					resource.TestCheckResourceAttr(pms_identity_resource, "service", "test-service"),
					resource.TestCheckResourceAttr(pms_identity_resource, "name", "test-identity"),
					resource.TestCheckResourceAttr(pms_identity_resource, "description", "Test identity for policy management"),
					resource.TestCheckResourceAttr(pms_identity_resource, "owner", "user123"),
					resource.TestCheckResourceAttr(pms_identity_resource, "preferred_assertion_method", "local"),
					resource.TestCheckResourceAttr(pms_identity_resource, "assertion_method.0.name", "key-1"),
					resource.TestCheckResourceAttr(pms_identity_resource, "assertion_method.0.type", "ethereum-address"),
					resource.TestCheckResourceAttr(pms_identity_resource, "assertion_method.0.verification_material", "0x1234567890123456789012345678901234567890"),
					resource.TestCheckResourceAttr(pms_identity_resource, "assertion_method.0.signing_method", "local"),
					resource.TestCheckResourceAttr(pms_identity_resource, "notification_method.0.name", "test-notification-method"),
					resource.TestCheckResourceAttr(pms_identity_resource, "notification_method.0.type", "workflow"),
					resource.TestCheckResourceAttr(pms_identity_resource, "notification_method.0.value", "{\"workflow\": \"test-workflow\", \"operation\": \"test-operation\"}"),
				),
			},
		},
	})
}

func (mp *mockPlatform) postPolicyIdentity(res http.ResponseWriter, req *http.Request) {
	var obj PolicyIdentityAPIModel
	mp.getBody(req, &obj)
	obj.ID = nanoid.New()
	now := time.Now().UTC()
	obj.Created = &now
	obj.Updated = &now
	mp.policyIdentities[obj.ID] = &obj
	mp.respond(res, &obj, 201)
}

func (mp *mockPlatform) getPolicyIdentity(res http.ResponseWriter, req *http.Request) {
	obj := mp.policyIdentities[mux.Vars(req)["identity"]]
	if obj == nil {
		mp.respond(res, nil, 404)
	} else {
		mp.respond(res, obj, 200)
	}
}

func (mp *mockPlatform) putPolicyIdentity(res http.ResponseWriter, req *http.Request) {
	obj := mp.policyIdentities[mux.Vars(req)["identity"]]
	assert.NotNil(mp.t, obj)
	var newObj PolicyIdentityAPIModel
	mp.getBody(req, &newObj)
	assert.Equal(mp.t, obj.ID, newObj.ID)
	now := time.Now().UTC()
	newObj.Created = obj.Created
	newObj.Updated = &now
	mp.policyIdentities[mux.Vars(req)["identity"]] = &newObj
	mp.respond(res, &newObj, 200)
}

func (mp *mockPlatform) deletePolicyIdentity(res http.ResponseWriter, req *http.Request) {
	obj := mp.policyIdentities[mux.Vars(req)["identity"]]
	assert.NotNil(mp.t, obj)
	delete(mp.policyIdentities, mux.Vars(req)["identity"])
	mp.respond(res, nil, 204)
}
