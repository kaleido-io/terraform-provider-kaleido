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

	"github.com/gorilla/mux"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// A policy created as an empty container, with its version and the binding that version
// depends on declared as their own resources. Terraform orders the binding ahead of the
// version, which is what the API requires of a definition whose constants initialize
// from an identity list binding.
var pms_policy_version_step1 = `
resource "kaleido_platform_pms_policy" "container" {
	environment = "env1"
	service     = "pms1"
	name        = "dual_approval"
	description = "container for separately declared bindings"
}

resource "kaleido_platform_pms_policy_identity_list_binding" "treasury" {
	environment              = "env1"
	service                  = "pms1"
	policy                   = kaleido_platform_pms_policy.container.id
	attester_label           = "treasuryOperations"
	identity_list_version_id = "ilv12345"
}

resource "kaleido_platform_pms_policy_version" "v1" {
	environment     = "env1"
	service         = "pms1"
	policy          = kaleido_platform_pms_policy.container.id
	name            = "v1"
	definition_yaml = yamlencode({
		evidence = [{ name = "approvals", source = "approvers" }]
		decision = { gate = { evidence = ["approvals"] } }
	})

	depends_on = [kaleido_platform_pms_policy_identity_list_binding.treasury]
}
`

func TestPMSPolicyVersion1(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			// create
			"PUT /endpoint/{env}/{service}/rest/api/v2/policies/{policy}",
			"GET /endpoint/{env}/{service}/rest/api/v2/policies/{policy}",
			"POST /endpoint/{env}/{service}/rest/api/v2/policies/{policy}/identity-list-bindings",
			"POST /endpoint/{env}/{service}/rest/api/v2/policies/{policy}/versions",
			// refresh before destroy
			"GET /endpoint/{env}/{service}/rest/api/v2/policies/{policy}",
			"GET /endpoint/{env}/{service}/rest/api/v2/policies/{policy}/identity-list-bindings/{binding}",
			"GET /endpoint/{env}/{service}/rest/api/v2/policies/{policy}/versions/{version}",
			// destroy
			"DELETE /endpoint/{env}/{service}/rest/api/v2/policies/{policy}/versions/{version}",
			"DELETE /endpoint/{env}/{service}/rest/api/v2/policies/{policy}/identity-list-bindings/{binding}",
			"DELETE /endpoint/{env}/{service}/rest/api/v2/policies/{policy}",
			"GET /endpoint/{env}/{service}/rest/api/v2/policies/{policy}",
		})
		mp.server.Close()
	}()

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + pms_policy_version_step1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("kaleido_platform_pms_policy_version.v1", "id"),
					resource.TestCheckResourceAttr("kaleido_platform_pms_policy_version.v1", "name", "v1"),
					// The container policy is created without a version of its own, and
					// picks the separately declared one up as its current version
					resource.TestCheckResourceAttr("kaleido_platform_pms_policy.container", "applied_version", ""),
				),
			},
		},
	})
}

// resolvePMSPolicyVersion looks a version up by name or ID within a policy, as the API does
func (mp *mockPlatform) resolvePMSPolicyVersion(req *http.Request) *PMSPolicyVersionAPIModel {
	policy := mp.pmsPolicies[mux.Vars(req)["policy"]]
	if policy == nil {
		return nil
	}
	nameOrID := mux.Vars(req)["version"]
	versions := mp.pmsPolicyVersions[policy.ID]
	if version, found := versions[nameOrID]; found {
		return version
	}
	for _, version := range versions {
		if version.ID == nameOrID {
			return version
		}
	}
	return nil
}

func (mp *mockPlatform) getPMSPolicyVersion(res http.ResponseWriter, req *http.Request) {
	version := mp.resolvePMSPolicyVersion(req)
	if version == nil {
		mp.respond(res, nil, 404)
		return
	}
	mp.respond(res, version, http.StatusOK)
}

func (mp *mockPlatform) patchPMSPolicyVersion(res http.ResponseWriter, req *http.Request) {
	version := mp.resolvePMSPolicyVersion(req)
	if version == nil {
		mp.respond(res, nil, 404)
		return
	}
	var updates PMSPolicyVersionPatchAPIModel
	mp.getBody(req, &updates)
	version.Description = updates.Description
	now := time.Now().UTC()
	version.Updated = &now
	mp.respond(res, version, http.StatusOK)
}

func (mp *mockPlatform) deletePMSPolicyVersion(res http.ResponseWriter, req *http.Request) {
	version := mp.resolvePMSPolicyVersion(req)
	if version == nil {
		mp.respond(res, nil, 404)
		return
	}
	policy := mp.pmsPolicies[mux.Vars(req)["policy"]]
	if policy != nil && policy.CurrentVersion == version.Name {
		// The server refuses to delete the version the policy is currently on
		mp.respond(res, map[string]interface{}{"error": "cannot delete the current version"}, http.StatusConflict)
		return
	}
	delete(mp.pmsPolicyVersions[version.PolicyID], version.Name)
	mp.respond(res, nil, http.StatusNoContent)
}

// A definition_yaml with its own top level description sets the version's description
// even though no description attribute was configured. The attribute is Computed so that
// terraform accepts the value the server returns.
var pms_policy_version_yaml_description = `
resource "kaleido_platform_pms_policy" "described" {
	environment = "env1"
	service     = "pms1"
	name        = "described-policy"
}

resource "kaleido_platform_pms_policy_version" "approval" {
	environment     = "env1"
	service         = "pms1"
	policy          = kaleido_platform_pms_policy.described.id
	definition_yaml = yamlencode({
		description = "A policy for approving any request"
		evidence    = [{ name = "request" }]
		decision    = { cases = [{ allow = { rego = "true" } }] }
	})
}
`

func TestPMSPolicyVersionDescriptionFromDefinition(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer mp.server.Close()

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + pms_policy_version_yaml_description,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kaleido_platform_pms_policy_version.approval",
						"description", "A policy for approving any request"),
				),
			},
			{
				// and the value the server supplied must not show as a diff
				Config:             providerConfig + pms_policy_version_yaml_description,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}
