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

var pms_policy_step1 = `
resource "kaleido_platform_pms_policy" "test_policy" {
  environment = "test-env"
  service = "test-service"
  name = "test-policy"
  description = "Test policy"
}
`

var pms_policy_step2 = `
resource "kaleido_platform_pms_policy" "test_policy" {
  environment = "test-env"
  service = "test-service"
  name = "test-policy"
  description = "Test policy - updated"
}
`

var pms_policy_step3 = `
resource "kaleido_platform_pms_policy" "test_policy" {
  environment = "test-env"
  service = "test-service"
  name = "test-policy"
}
`

func TestPMSPolicy1(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"PUT /endpoint/{env}/{service}/rest/api/v2/policies/{policy}",
			"GET /endpoint/{env}/{service}/rest/api/v2/policies/{policy}",
			"GET /endpoint/{env}/{service}/rest/api/v2/policies/{policy}",
			"PATCH /endpoint/{env}/{service}/rest/api/v2/policies/{policy}",
			"GET /endpoint/{env}/{service}/rest/api/v2/policies/{policy}",
			"GET /endpoint/{env}/{service}/rest/api/v2/policies/{policy}",
			"PATCH /endpoint/{env}/{service}/rest/api/v2/policies/{policy}",
			"GET /endpoint/{env}/{service}/rest/api/v2/policies/{policy}",
			"DELETE /endpoint/{env}/{service}/rest/api/v2/policies/{policy}",
			"GET /endpoint/{env}/{service}/rest/api/v2/policies/{policy}",
		})
		mp.server.Close()
	}()

	pms_policy_resource := "kaleido_platform_pms_policy.test_policy"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + pms_policy_step1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(pms_policy_resource, "id"),
					resource.TestCheckResourceAttr(pms_policy_resource, "environment", "test-env"),
					resource.TestCheckResourceAttr(pms_policy_resource, "service", "test-service"),
					resource.TestCheckResourceAttr(pms_policy_resource, "name", "test-policy"),
					resource.TestCheckResourceAttr(pms_policy_resource, "description", "Test policy"),
					resource.TestCheckResourceAttr(pms_policy_resource, "applied_version", ""),
					resource.TestCheckResourceAttrSet(pms_policy_resource, "created"),
					resource.TestCheckResourceAttrSet(pms_policy_resource, "updated"),
					func(s *terraform.State) error {
						policyID := s.RootModule().Resources[pms_policy_resource].Primary.Attributes["id"]
						assert.Empty(t, mp.pmsPolicyVersions[policyID], "creating a policy must not create a version")
						return nil
					},
				),
			},
			{
				Config: providerConfig + pms_policy_step2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(pms_policy_resource, "description", "Test policy - updated"),
					func(s *terraform.State) error {
						assert.Equal(t, map[string]interface{}{"description": "Test policy - updated"}, mp.lastPMSPatchBody())
						return nil
					},
				),
			},
			{
				Config: providerConfig + pms_policy_step3,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr(pms_policy_resource, "description"),
					func(s *terraform.State) error {
						assert.Equal(t, map[string]interface{}{"description": ""}, mp.lastPMSPatchBody(),
							"a removed description is cleared by sending it empty")
						return nil
					},
				),
			},
		},
	})
}

func (mp *mockPlatform) putPMSPolicy(res http.ResponseWriter, req *http.Request) {
	nameOrID := mux.Vars(req)["policy"]
	existing := mp.pmsPolicies[nameOrID]
	var policy PMSPolicyAPIModel
	rawBody := mp.peekBody(req, &policy)
	var wire map[string]interface{}
	assert.NoError(mp.t, json.Unmarshal(rawBody, &wire))
	for field := range wire {
		assert.Contains(mp.t, []string{"name", "description"}, field, "a policy is created without bindings or a version")
	}

	now := time.Now().UTC()
	if existing == nil {
		policy.ID = nanoid.New()
		policy.Created = &now
	} else {
		policy.ID = existing.ID
		policy.Created = existing.Created
		policy.CurrentVersion = existing.CurrentVersion
	}
	policy.Updated = &now
	// Stored by both ID and name, as the resource addresses by name on create and by ID after
	mp.pmsPolicies[policy.Name] = &policy
	mp.pmsPolicies[policy.ID] = &policy
	mp.respond(res, &policy, http.StatusOK)
}

func (mp *mockPlatform) getPMSPolicy(res http.ResponseWriter, req *http.Request) {
	policy := mp.pmsPolicies[mux.Vars(req)["policy"]]
	if policy == nil {
		mp.respond(res, nil, 404)
		return
	}
	mp.respond(res, policy, http.StatusOK)
}

func (mp *mockPlatform) patchPMSPolicy(res http.ResponseWriter, req *http.Request) {
	policy := mp.pmsPolicies[mux.Vars(req)["policy"]]
	if policy == nil {
		mp.respond(res, nil, 404)
		return
	}
	var updates PMSPolicyPatchAPIModel
	mp.recordPMSPatchBody(req, &updates)
	if updates.Description != nil {
		policy.Description = *updates.Description
	}
	now := time.Now().UTC()
	policy.Updated = &now
	mp.respond(res, policy, http.StatusOK)
}

func (mp *mockPlatform) deletePMSPolicy(res http.ResponseWriter, req *http.Request) {
	nameOrID := mux.Vars(req)["policy"]
	policy := mp.pmsPolicies[nameOrID]
	if policy == nil {
		mp.respond(res, nil, 404)
		return
	}
	delete(mp.pmsPolicies, policy.ID)
	delete(mp.pmsPolicies, policy.Name)
	delete(mp.pmsPolicyVersions, policy.ID)
	mp.respond(res, nil, http.StatusNoContent)
}

func (mp *mockPlatform) postPMSPolicyVersion(res http.ResponseWriter, req *http.Request) {
	policy := mp.pmsPolicies[mux.Vars(req)["policy"]]
	if policy == nil {
		mp.respond(res, nil, 404)
		return
	}
	var version PMSPolicyVersionAPIModel
	rawBody := mp.peekBody(req, &version)
	// The definition must arrive at the top level of the version body, alongside the
	// metadata - not nested under a wrapper field
	var wire map[string]interface{}
	assert.NoError(mp.t, json.Unmarshal(rawBody, &wire))
	assert.Contains(mp.t, wire, "evidence", "policy definition must be sent at the top level, got: %s", rawBody)
	assert.Contains(mp.t, wire, "decision", "policy definition must be sent at the top level, got: %s", rawBody)
	now := time.Now().UTC()
	version.ID = nanoid.New()
	version.PolicyID = policy.ID
	if version.Name == "" {
		version.Name = version.ID
	}
	version.Created = &now
	version.Updated = &now
	if mp.pmsPolicyVersions[policy.ID] == nil {
		mp.pmsPolicyVersions[policy.ID] = map[string]*PMSPolicyVersionAPIModel{}
	}
	mp.pmsPolicyVersions[policy.ID][version.Name] = &version
	// Posting a version activates it
	policy.CurrentVersion = version.Name
	policy.Updated = &now
	mp.respond(res, &version, http.StatusCreated)
}
