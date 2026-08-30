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
	"net/http"
	"testing"
	"time"

	"github.com/aidarkhanov/nanoid"
	"github.com/gorilla/mux"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	_ "embed"
)

var pms_identity_list_binding_step1 = `
resource "kaleido_platform_pms_policy_identity_list_binding" "treasury_ops" {
  environment = "test-env"
  service = "test-service"
  policy = "test-policy"
  attester_label = "treasuryOperations"
  identity_list_version_id = "pmilv:12345abcde"
}
`

var pms_identity_list_binding_step2 = `
resource "kaleido_platform_pms_policy_identity_list_binding" "treasury_ops" {
  environment = "test-env"
  service = "test-service"
  policy = "test-policy"
  attester_label = "treasuryOperations"
  identity_list_version_id = "pmilv:67890fghij"
}
`

func TestPMSIdentityListBinding1(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"POST /endpoint/{env}/{service}/rest/api/v2/policies/{policy}/identity-list-bindings",
			"GET /endpoint/{env}/{service}/rest/api/v2/policies/{policy}/identity-list-bindings/{binding}",
			"GET /endpoint/{env}/{service}/rest/api/v2/policies/{policy}/identity-list-bindings/{binding}",
			"PATCH /endpoint/{env}/{service}/rest/api/v2/policies/{policy}/identity-list-bindings/{binding}",
			"GET /endpoint/{env}/{service}/rest/api/v2/policies/{policy}/identity-list-bindings/{binding}",
			"DELETE /endpoint/{env}/{service}/rest/api/v2/policies/{policy}/identity-list-bindings/{binding}",
		})
		mp.server.Close()
	}()

	bindingResource := "kaleido_platform_pms_policy_identity_list_binding.treasury_ops"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + pms_identity_list_binding_step1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(bindingResource, "id"),
					resource.TestCheckResourceAttr(bindingResource, "policy", "test-policy"),
					resource.TestCheckResourceAttr(bindingResource, "attester_label", "treasuryOperations"),
					resource.TestCheckResourceAttr(bindingResource, "identity_list_version_id", "pmilv:12345abcde"),
				),
			},
			{
				// Re-pointing at a new identity list version patches the binding in place
				Config: providerConfig + pms_identity_list_binding_step2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(bindingResource, "attester_label", "treasuryOperations"),
					resource.TestCheckResourceAttr(bindingResource, "identity_list_version_id", "pmilv:67890fghij"),
				),
			},
		},
	})
}

func (mp *mockPlatform) postPMSIdentityListBinding(res http.ResponseWriter, req *http.Request) {
	var binding PMSIdentityListBindingAPIModel
	mp.getBody(req, &binding)
	now := time.Now().UTC()
	binding.ID = nanoid.New()
	binding.PolicyID = mux.Vars(req)["policy"]
	binding.Created = &now
	binding.Updated = &now
	mp.pmsIdentityListBindings[binding.ID] = &binding
	mp.respond(res, &binding, http.StatusCreated)
}

func (mp *mockPlatform) getPMSIdentityListBinding(res http.ResponseWriter, req *http.Request) {
	binding := mp.pmsIdentityListBindings[mux.Vars(req)["binding"]]
	if binding == nil {
		mp.respond(res, nil, 404)
		return
	}
	mp.respond(res, binding, http.StatusOK)
}

func (mp *mockPlatform) patchPMSIdentityListBinding(res http.ResponseWriter, req *http.Request) {
	binding := mp.pmsIdentityListBindings[mux.Vars(req)["binding"]]
	if binding == nil {
		mp.respond(res, nil, 404)
		return
	}
	var updates PMSIdentityListBindingPatchAPIModel
	mp.getBody(req, &updates)
	if updates.IdentityListVersionID != "" {
		binding.IdentityListVersionID = updates.IdentityListVersionID
	}
	now := time.Now().UTC()
	binding.Updated = &now
	mp.respond(res, binding, http.StatusOK)
}

func (mp *mockPlatform) deletePMSIdentityListBinding(res http.ResponseWriter, req *http.Request) {
	id := mux.Vars(req)["binding"]
	if mp.pmsIdentityListBindings[id] == nil {
		mp.respond(res, nil, 404)
		return
	}
	delete(mp.pmsIdentityListBindings, id)
	mp.respond(res, nil, http.StatusNoContent)
}

// getPMSIdentityListBindings serves the binding list for a policy, which the policy
// resource uses to reconcile its inline bindings
func (mp *mockPlatform) getPMSIdentityListBindings(res http.ResponseWriter, req *http.Request) {
	policy := mux.Vars(req)["policy"]
	items := []*PMSIdentityListBindingAPIModel{}
	for _, binding := range mp.pmsIdentityListBindings {
		if binding.PolicyID == policy {
			items = append(items, binding)
		}
	}
	mp.respond(res, map[string]interface{}{"items": items, "count": len(items)}, http.StatusOK)
}
