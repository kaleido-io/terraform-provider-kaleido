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

var pms_ofb_step1 = `
resource "kaleido_platform_pms_policy_output_formatter_binding" "evm" {
  environment = "test-env"
  service = "test-service"
  policy = "test-policy"
  policy_output_formatter = "evmOutput"
  output_formatter_id = "pof:12345abcde"
}
`

var pms_ofb_step2 = `
resource "kaleido_platform_pms_policy_output_formatter_binding" "evm" {
  environment = "test-env"
  service = "test-service"
  policy = "test-policy"
  policy_output_formatter = "evmOutput"
  output_formatter_id = "pof:67890fghij"
}
`

func TestPMSOutputFormatterBinding1(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"POST /endpoint/{env}/{service}/rest/api/v2/policies/{policy}/output-formatter-bindings",
			"GET /endpoint/{env}/{service}/rest/api/v2/policies/{policy}/output-formatter-bindings/{binding}",
			"GET /endpoint/{env}/{service}/rest/api/v2/policies/{policy}/output-formatter-bindings/{binding}",
			"PATCH /endpoint/{env}/{service}/rest/api/v2/policies/{policy}/output-formatter-bindings/{binding}",
			"GET /endpoint/{env}/{service}/rest/api/v2/policies/{policy}/output-formatter-bindings/{binding}",
			"DELETE /endpoint/{env}/{service}/rest/api/v2/policies/{policy}/output-formatter-bindings/{binding}",
		})
		mp.server.Close()
	}()

	ofbResource := "kaleido_platform_pms_policy_output_formatter_binding.evm"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + pms_ofb_step1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(ofbResource, "id"),
					resource.TestCheckResourceAttr(ofbResource, "policy_output_formatter", "evmOutput"),
					resource.TestCheckResourceAttr(ofbResource, "output_formatter_id", "pof:12345abcde"),
				),
			},
			{
				Config: providerConfig + pms_ofb_step2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(ofbResource, "output_formatter_id", "pof:67890fghij"),
				),
			},
		},
	})
}

func (mp *mockPlatform) postPMSOutputFormatterBinding(res http.ResponseWriter, req *http.Request) {
	var binding PMSOutputFormatterBindingAPIModel
	mp.getBody(req, &binding)
	now := time.Now().UTC()
	binding.ID = nanoid.New()
	binding.PolicyID = mux.Vars(req)["policy"]
	binding.Created = &now
	binding.Updated = &now
	mp.pmsOutputFormatterBindings[binding.ID] = &binding
	mp.respond(res, &binding, http.StatusCreated)
}

func (mp *mockPlatform) getPMSOutputFormatterBinding(res http.ResponseWriter, req *http.Request) {
	binding := mp.pmsOutputFormatterBindings[mux.Vars(req)["binding"]]
	if binding == nil {
		mp.respond(res, nil, 404)
		return
	}
	mp.respond(res, binding, http.StatusOK)
}

func (mp *mockPlatform) patchPMSOutputFormatterBinding(res http.ResponseWriter, req *http.Request) {
	binding := mp.pmsOutputFormatterBindings[mux.Vars(req)["binding"]]
	if binding == nil {
		mp.respond(res, nil, 404)
		return
	}
	var updates PMSOutputFormatterBindingTargetAPIModel
	mp.getBody(req, &updates)
	if updates.OutputFormatterID != "" {
		binding.OutputFormatterID = updates.OutputFormatterID
	}
	now := time.Now().UTC()
	binding.Updated = &now
	mp.respond(res, binding, http.StatusOK)
}

func (mp *mockPlatform) deletePMSOutputFormatterBinding(res http.ResponseWriter, req *http.Request) {
	id := mux.Vars(req)["binding"]
	if mp.pmsOutputFormatterBindings[id] == nil {
		mp.respond(res, nil, 404)
		return
	}
	delete(mp.pmsOutputFormatterBindings, id)
	mp.respond(res, nil, http.StatusNoContent)
}
