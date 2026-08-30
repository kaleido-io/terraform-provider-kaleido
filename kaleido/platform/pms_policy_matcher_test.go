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

var pms_policy_matcher_step1 = `
resource "kaleido_platform_pms_policy_matcher" "test_matcher" {
  environment = "test-env"
  service = "test-service"
  policy = "test-policy"
  enforcement_point = "wfe-hook"
  match_json = jsonencode({
    equal = [
      {
        field = "label.assetType"
        value = "bond"
      }
    ]
  })
  parameters_json = jsonencode({
    threshold = 100
  })
  evidence = [
    {
      slot = "documents"
      payload_jsonata = "$.input.document"
      attestation_jsonata = "$.input.signature"
    }
  ]
}
`

var pms_policy_matcher_step2 = `
resource "kaleido_platform_pms_policy_matcher" "test_matcher" {
  environment = "test-env"
  service = "test-service"
  policy = "test-policy"
  enforcement_point = "wfe-hook"
  match_json = jsonencode({
    equal = [
      {
        field = "label.assetType"
        value = "equity"
      }
    ]
  })
  parameters_json = jsonencode({
    threshold = 200
  })
  evidence = [
    {
      slot = "documents"
      payload_jsonata = "$.input.document"
      attestation_jsonata = "$.input.signature"
    }
  ]
}
`

func TestPMSPolicyMatcher1(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"POST /endpoint/{env}/{service}/rest/api/v2/policies/{policy}/matchers",
			"GET /endpoint/{env}/{service}/rest/api/v2/policies/{policy}/matchers/{matcher}",
			"GET /endpoint/{env}/{service}/rest/api/v2/policies/{policy}/matchers/{matcher}",
			"PATCH /endpoint/{env}/{service}/rest/api/v2/policies/{policy}/matchers/{matcher}",
			"GET /endpoint/{env}/{service}/rest/api/v2/policies/{policy}/matchers/{matcher}",
			"DELETE /endpoint/{env}/{service}/rest/api/v2/policies/{policy}/matchers/{matcher}",
		})
		mp.server.Close()
	}()

	matcherResource := "kaleido_platform_pms_policy_matcher.test_matcher"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + pms_policy_matcher_step1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(matcherResource, "id"),
					resource.TestCheckResourceAttr(matcherResource, "policy", "test-policy"),
					resource.TestCheckResourceAttr(matcherResource, "enforcement_point", "wfe-hook"),
					resource.TestCheckResourceAttr(matcherResource, "evidence.0.slot", "documents"),
					resource.TestCheckResourceAttr(matcherResource, "evidence.0.payload_jsonata", "$.input.document"),
					resource.TestCheckResourceAttr(matcherResource, "evidence.0.attestation_jsonata", "$.input.signature"),
				),
			},
			{
				Config: providerConfig + pms_policy_matcher_step2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(matcherResource, "id"),
					resource.TestCheckResourceAttr(matcherResource, "enforcement_point", "wfe-hook"),
				),
			},
		},
	})
}

func (mp *mockPlatform) postPMSPolicyMatcher(res http.ResponseWriter, req *http.Request) {
	var matcher PMSPolicyMatcherAPIModel
	mp.getBody(req, &matcher)
	now := time.Now().UTC()
	matcher.ID = nanoid.New()
	matcher.PolicyID = mux.Vars(req)["policy"]
	matcher.Created = &now
	matcher.Updated = &now
	mp.pmsPolicyMatchers[matcher.ID] = &matcher
	mp.respond(res, &matcher, http.StatusCreated)
}

func (mp *mockPlatform) getPMSPolicyMatcher(res http.ResponseWriter, req *http.Request) {
	matcher := mp.pmsPolicyMatchers[mux.Vars(req)["matcher"]]
	if matcher == nil {
		mp.respond(res, nil, 404)
		return
	}
	mp.respond(res, matcher, http.StatusOK)
}

func (mp *mockPlatform) patchPMSPolicyMatcher(res http.ResponseWriter, req *http.Request) {
	matcher := mp.pmsPolicyMatchers[mux.Vars(req)["matcher"]]
	if matcher == nil {
		mp.respond(res, nil, 404)
		return
	}
	var updates PMSPolicyMatcherPatchAPIModel
	mp.getBody(req, &updates)
	matcher.Match = updates.Match
	matcher.Parameters = updates.Parameters
	now := time.Now().UTC()
	matcher.Updated = &now
	mp.respond(res, matcher, http.StatusOK)
}

func (mp *mockPlatform) deletePMSPolicyMatcher(res http.ResponseWriter, req *http.Request) {
	id := mux.Vars(req)["matcher"]
	if mp.pmsPolicyMatchers[id] == nil {
		mp.respond(res, nil, 404)
		return
	}
	delete(mp.pmsPolicyMatchers, id)
	mp.respond(res, nil, http.StatusNoContent)
}
