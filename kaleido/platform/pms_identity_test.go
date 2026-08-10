// Copyright © Kaleido, Inc. 2026

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
	"encoding/json"
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

var pms_identity_notification_value = `
resource "kaleido_platform_pms_identity" "notify_identity" {
  environment = "test-env"
  service = "test-service"
  name = "notify-identity"
  notification_method = [
    {
      name = "notify-workflow"
      type = "workflow"
      value = jsonencode({
        workflow  = "test-workflow"
        operation = "test-operation"
      })
    }
  ]
}
`

func TestPMSIdentityNotificationMethodValue(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"POST /endpoint/{env}/{service}/rest/api/v1/identities",
			"GET /endpoint/{env}/{service}/rest/api/v1/identities/{identity}",
			"GET /endpoint/{env}/{service}/rest/api/v1/identities/{identity}",
			"DELETE /endpoint/{env}/{service}/rest/api/v1/identities/{identity}",
		})
		mp.server.Close()
	}()

	pms_identity_resource := "kaleido_platform_pms_identity.notify_identity"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + pms_identity_notification_value,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(pms_identity_resource, "notification_method.0.name", "notify-workflow"),
					resource.TestCheckResourceAttr(pms_identity_resource, "notification_method.0.type", "workflow"),
					func(s *terraform.State) error {
						// The value must have been stored server-side as a structured object,
						// not as a string containing JSON
						id := s.RootModule().Resources[pms_identity_resource].Primary.Attributes["id"]
						obj := mp.policyIdentities[id]
						assert.Len(t, obj.NotificationMethod, 1)
						assert.Equal(t, map[string]interface{}{
							"workflow":  "test-workflow",
							"operation": "test-operation",
						}, obj.NotificationMethod[0].Value)
						return nil
					},
				),
			},
			{
				// Re-planning against the value the API returns must produce no diff
				Config:             providerConfig + pms_identity_notification_value,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// assertNotificationMethodValuesAreObjects fails the test if a notification method value
// reaches the API as a JSON string rather than as a JSON object
func (mp *mockPlatform) assertNotificationMethodValuesAreObjects(rawBody []byte) {
	var wire struct {
		NotificationMethod []struct {
			Value json.RawMessage `json:"value"`
		} `json:"notificationMethod"`
	}
	err := json.Unmarshal(rawBody, &wire)
	assert.NoError(mp.t, err)
	for _, nm := range wire.NotificationMethod {
		if len(nm.Value) == 0 {
			continue
		}
		assert.Equal(mp.t, uint8('{'), nm.Value[0],
			"notificationMethod value must be sent as a JSON object, got: %s", nm.Value)
	}
}

func (mp *mockPlatform) postPolicyIdentity(res http.ResponseWriter, req *http.Request) {
	var obj PolicyIdentityAPIModel
	rawBody := mp.peekBody(req, &obj)
	mp.assertNotificationMethodValuesAreObjects(rawBody)
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
