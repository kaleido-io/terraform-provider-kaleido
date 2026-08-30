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
	"fmt"
	"net/http"
	"strings"
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
  identity_list_binding = [
    {
      attester_label = "treasuryOperations"
      identity_list_version_id = "ilv:12345abcde"
    }
  ]
  definition_yaml = yamlencode({
    "evidence" = [
      {
        "name" = "approval"
        "source" = "approvers"
      }
    ]
    "decision" = {
      "gate" = {
        "evidence" = ["approval"]
      }
    }
  })
}
`

var pms_policy_step2 = `
resource "kaleido_platform_pms_policy" "test_policy" {
  environment = "test-env"
  service = "test-service"
  name = "test-policy"
  description = "Test policy - updated"
  identity_list_binding = [
    {
      attester_label = "treasuryOperations"
      identity_list_version_id = "ilv:67890fghij"
    }
  ]
  definition_yaml = yamlencode({
    "evidence" = [
      {
        "name" = "approval"
        "source" = "approvers"
      },
      {
        "name" = "attachment"
        "source" = "documents"
      }
    ]
    "decision" = {
      "gate" = {
        "evidence" = ["approval"]
      }
    }
  })
}
`

func TestPMSPolicy1(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			// create: the policy, its inline bindings and its first version are one call,
			// then the bindings are listed to pick up the IDs the write does not return
			"PUT /endpoint/{env}/{service}/rest/api/v2/policies/{policy}",
			"GET /endpoint/{env}/{service}/rest/api/v2/policies/{policy}",
			"GET /endpoint/{env}/{service}/rest/api/v2/policies/{policy}/identity-list-bindings",
			// refresh
			"GET /endpoint/{env}/{service}/rest/api/v2/policies/{policy}",
			"GET /endpoint/{env}/{service}/rest/api/v2/policies/{policy}/identity-list-bindings",
			"GET /endpoint/{env}/{service}/rest/api/v2/policies/{policy}",
			"GET /endpoint/{env}/{service}/rest/api/v2/policies/{policy}/identity-list-bindings",
			// update: the PATCH merges the binding changes, then the new version is posted
			// separately because a PATCH cannot carry one
			"PATCH /endpoint/{env}/{service}/rest/api/v2/policies/{policy}",
			"POST /endpoint/{env}/{service}/rest/api/v2/policies/{policy}/versions",
			"GET /endpoint/{env}/{service}/rest/api/v2/policies/{policy}",
			"GET /endpoint/{env}/{service}/rest/api/v2/policies/{policy}/identity-list-bindings",
			// refresh before destroy
			"GET /endpoint/{env}/{service}/rest/api/v2/policies/{policy}",
			"GET /endpoint/{env}/{service}/rest/api/v2/policies/{policy}/identity-list-bindings",
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
					resource.TestCheckResourceAttr(pms_policy_resource, "identity_list_binding.0.attester_label", "treasuryOperations"),
					resource.TestCheckResourceAttr(pms_policy_resource, "identity_list_binding.0.identity_list_version_id", "ilv:12345abcde"),
					resource.TestCheckResourceAttrSet(pms_policy_resource, "identity_list_binding.0.id"),
					resource.TestCheckResourceAttrSet(pms_policy_resource, "applied_version"),
					resource.TestCheckResourceAttrSet(pms_policy_resource, "created"),
					resource.TestCheckResourceAttrSet(pms_policy_resource, "updated"),
				),
			},
			{
				Config: providerConfig + pms_policy_step2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(pms_policy_resource, "description", "Test policy - updated"),
					resource.TestCheckResourceAttr(pms_policy_resource, "identity_list_binding.0.identity_list_version_id", "ilv:67890fghij"),
					resource.TestCheckResourceAttrSet(pms_policy_resource, "applied_version"),
				),
			},
		},
	})
}

var pms_policy_one_binding = `
resource "kaleido_platform_pms_policy" "coexist" {
  environment = "test-env"
  service = "test-service"
  name = "coexist-policy"
  identity_list_binding = [
    {
      attester_label = "managed"
      identity_list_version_id = "ilv:12345abcde"
    }
  ]
  definition_yaml = yamlencode({
    "evidence" = [{ "name" = "request" }]
    "decision" = { "cases" = [{ "allow" = { "rego" = "true" } }] }
  })
}
`

var pms_policy_no_bindings = `
resource "kaleido_platform_pms_policy" "coexist" {
  environment = "test-service-env"
  service = "test-service"
  name = "coexist-policy"
  definition_yaml = yamlencode({
    "evidence" = [{ "name" = "request" }]
    "decision" = { "cases" = [{ "allow" = { "rego" = "true" } }] }
  })
}
`

// Inline bindings manage only the labels they list. A binding created by something else
// on the same policy - the standalone resource, or by hand - must survive untouched.
func TestPMSPolicyInlineBindingsLeaveOthersAlone(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer mp.server.Close()

	policyResource := "kaleido_platform_pms_policy.coexist"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + pms_policy_one_binding,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(policyResource, "identity_list_binding.0.attester_label", "managed"),
					func(s *terraform.State) error {
						// Something else creates a binding on the same policy
						policyID := s.RootModule().Resources[policyResource].Primary.Attributes["id"]
						mp.pmsIdentityListBindings["foreign-binding"] = &PMSIdentityListBindingAPIModel{
							ID:                    "foreign-binding",
							PolicyID:              policyID,
							AttesterLabel:         "notManagedHere",
							IdentityListVersionID: "ilv:abcde12345",
						}
						return nil
					},
				),
			},
			{
				// Dropping the inline block deletes the binding it managed...
				Config: providerConfig + strings.Replace(pms_policy_no_bindings, "test-service-env", "test-env", 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr(policyResource, "identity_list_binding.0.attester_label"),
					func(s *terraform.State) error {
						labels := map[string]bool{}
						for _, b := range mp.pmsIdentityListBindings {
							labels[b.AttesterLabel] = true
						}
						assert.False(t, labels["managed"], "the inline binding should have been deleted when dropped from config")
						assert.True(t, labels["notManagedHere"], "a binding this resource never listed must be left alone")
						return nil
					},
				),
			},
		},
	})
}

// pmsPolicyWriteBody is the flattened policy write body. The binding maps are pointers
// so that a map the client did not send at all can be told apart from an empty one: an
// absent map leaves that kind of binding alone, a present one is authoritative.
type pmsPolicyWriteBody struct {
	PMSPolicyAPIModel
	Version                string                                       `json:"version,omitempty"`
	VersionDescription     string                                       `json:"versionDescription,omitempty"`
	IdentityListBindings   *map[string]pmsInlineIdentityListBinding     `json:"identityListBindings,omitempty"`
	EvidenceSourceBindings *map[string]PMSEvidenceSourceBindingAPIModel `json:"evidenceSourceBindings,omitempty"`
}

type pmsInlineIdentityListBinding struct {
	IdentityListVersionID string `json:"identityListVersionId,omitempty"`
}

// pmsPolicyMetadataFields are the fields of a policy write body that are not part of a
// policy definition. Anything else in the body means a version was inlined.
var pmsPolicyMetadataFields = map[string]bool{
	"id": true, "name": true, "description": true, "currentVersion": true,
	"version": true, "versionDescription": true, "created": true, "updated": true,
	"identityListBindings": true, "evidenceSourceBindings": true,
}

// writeInlinePolicyBindings mirrors the server: bindings are matched on name or attester
// label and keep their ID, and when replaceAll is set (a PUT, not a PATCH) a binding of
// a kind the body carries that is not named in it is deleted.
func (mp *mockPlatform) writeInlinePolicyBindings(policyID string, body *pmsPolicyWriteBody, replaceAll bool) {
	now := time.Now().UTC()

	if body.IdentityListBindings != nil {
		written := map[string]bool{}
		for label, target := range *body.IdentityListBindings {
			written[label] = true
			existing := (*PMSIdentityListBindingAPIModel)(nil)
			for _, binding := range mp.pmsIdentityListBindings {
				if binding.PolicyID == policyID && binding.AttesterLabel == label {
					existing = binding
				}
			}
			if existing == nil {
				existing = &PMSIdentityListBindingAPIModel{
					ID: nanoid.New(), PolicyID: policyID, AttesterLabel: label, Created: &now,
				}
				mp.pmsIdentityListBindings[existing.ID] = existing
			}
			existing.IdentityListVersionID = target.IdentityListVersionID
			existing.Updated = &now
		}
		if replaceAll {
			for id, binding := range mp.pmsIdentityListBindings {
				if binding.PolicyID == policyID && !written[binding.AttesterLabel] {
					delete(mp.pmsIdentityListBindings, id)
				}
			}
		}
	}

	if body.EvidenceSourceBindings != nil {
		written := map[string]bool{}
		for name, incoming := range *body.EvidenceSourceBindings {
			written[name] = true
			existing := (*PMSEvidenceSourceBindingAPIModel)(nil)
			for _, binding := range mp.pmsEvidenceSourceBindings {
				if binding.PolicyID == policyID && binding.Name == name {
					existing = binding
				}
			}
			if existing == nil {
				existing = &PMSEvidenceSourceBindingAPIModel{
					ID: nanoid.New(), PolicyID: policyID, Name: name, Created: &now,
				}
				mp.pmsEvidenceSourceBindings[existing.ID] = existing
			}
			existing.Type = incoming.Type
			existing.Approval = incoming.Approval
			existing.Attachment = incoming.Attachment
			existing.Updated = &now
		}
		if replaceAll {
			for id, binding := range mp.pmsEvidenceSourceBindings {
				if binding.PolicyID == policyID && !written[binding.Name] {
					delete(mp.pmsEvidenceSourceBindings, id)
				}
			}
		}
	}
}

// activateInlinePolicyVersion stores and activates a version supplied in a policy write
// body, if the body carried a definition at all.
func (mp *mockPlatform) activateInlinePolicyVersion(policy *PMSPolicyAPIModel, wire map[string]interface{}, versionName, versionDescription string) {
	definition := map[string]interface{}{}
	for field, value := range wire {
		if !pmsPolicyMetadataFields[field] {
			definition[field] = value
		}
	}
	if len(definition) == 0 {
		return
	}
	now := time.Now().UTC()
	version := &PMSPolicyVersionAPIModel{
		ID: nanoid.New(), PolicyID: policy.ID, Name: versionName,
		Description: versionDescription, Created: &now, Updated: &now,
	}
	if version.Name == "" {
		version.Name = version.ID
	}
	if mp.pmsPolicyVersions[policy.ID] == nil {
		mp.pmsPolicyVersions[policy.ID] = map[string]*PMSPolicyVersionAPIModel{}
	}
	mp.pmsPolicyVersions[policy.ID][version.Name] = version
	policy.CurrentVersion = version.Name
	policy.Updated = &now
}

func (mp *mockPlatform) putPMSPolicy(res http.ResponseWriter, req *http.Request) {
	nameOrID := mux.Vars(req)["policy"]
	existing := mp.pmsPolicies[nameOrID]
	body := new(pmsPolicyWriteBody)
	rawBody := mp.peekBody(req, body)
	var wire map[string]interface{}
	assert.NoError(mp.t, json.Unmarshal(rawBody, &wire))
	mp.pmsPolicyPutBodies = append(mp.pmsPolicyPutBodies, wire)

	newPolicy := &body.PMSPolicyAPIModel
	now := time.Now().UTC()
	if existing == nil {
		newPolicy.ID = nanoid.New()
		newPolicy.Created = &now
	} else {
		newPolicy.ID = existing.ID
		newPolicy.Created = existing.Created
		newPolicy.CurrentVersion = existing.CurrentVersion
	}
	newPolicy.Updated = &now
	// Stored by both ID and name, as the resource addresses by name on create and by ID after
	mp.pmsPolicies[newPolicy.Name] = newPolicy
	mp.pmsPolicies[newPolicy.ID] = newPolicy

	// The bindings are written before the version, so a version arriving in the same
	// call resolves against them
	mp.writeInlinePolicyBindings(newPolicy.ID, body, true)
	mp.activateInlinePolicyVersion(newPolicy, wire, body.Version, body.VersionDescription)

	mp.respond(res, newPolicy, http.StatusOK)
}

func (mp *mockPlatform) getPMSPolicy(res http.ResponseWriter, req *http.Request) {
	policy := mp.pmsPolicies[mux.Vars(req)["policy"]]
	if policy == nil {
		mp.respond(res, nil, 404)
		return
	}
	// withCurrentVersion embeds the version body, and the server rejects the request
	// when there is no version to embed. currentVersion itself is always returned, so
	// the provider has no reason to ask for the embedded body.
	if strings.EqualFold(req.URL.Query().Get("withCurrentVersion"), "true") && policy.CurrentVersion == "" {
		mp.respond(res, map[string]interface{}{
			"error": fmt.Sprintf("KA171405: Policy '%s' has no current version", policy.ID),
		}, http.StatusBadRequest)
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
	var updates pmsPolicyWriteBody
	mp.getBody(req, &updates)
	if updates.Description != "" {
		policy.Description = updates.Description
	}
	// A PATCH merges the binding maps rather than replacing the set
	mp.writeInlinePolicyBindings(policy.ID, &updates, false)
	now := time.Now().UTC()
	policy.Updated = &now
	mp.respond(res, policy, http.StatusOK)
}

// getPMSEvidenceSourceBindings serves the binding list for a policy, which the policy
// resource uses to pick up the IDs of the bindings it declares inline
func (mp *mockPlatform) getPMSEvidenceSourceBindings(res http.ResponseWriter, req *http.Request) {
	policy := mux.Vars(req)["policy"]
	items := []*PMSEvidenceSourceBindingAPIModel{}
	for _, binding := range mp.pmsEvidenceSourceBindings {
		if binding.PolicyID == policy {
			items = append(items, binding)
		}
	}
	mp.respond(res, map[string]interface{}{"items": items, "count": len(items)}, http.StatusOK)
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

// A fully wired policy - both kinds of inline binding and the first version - is created
// by a single call, which is what lets the definition reference a binding on the very
// first apply.
var pms_policy_inline_everything = `
resource "kaleido_platform_pms_policy" "wired" {
  environment = "test-env"
  service = "test-service"
  name = "wired-policy"
  identity_list_binding = [
    {
      attester_label = "treasuryOperations"
      identity_list_version_id = "ilv:12345abcde"
    }
  ]
  evidence_source_binding = [
    {
      name = "approvers"
      type = "approval"
      approval = {
        approval = {
          payload_type = "TypedDataV4"
        }
      }
    },
    {
      name = "documents"
      type = "attachment"
      attachment = {
        payload_jsonata = "$.input.document"
      }
    }
  ]
  definition_yaml = yamlencode({
    "evidence" = [{ "name" = "approval", "source" = "approvers" }]
    "decision" = { "gate" = { "evidence" = ["approval"] } }
  })
}
`

// Dropping one inline evidence source binding leaves the other in place.
var pms_policy_inline_one_esb = `
resource "kaleido_platform_pms_policy" "wired" {
  environment = "test-env"
  service = "test-service"
  name = "wired-policy"
  identity_list_binding = [
    {
      attester_label = "treasuryOperations"
      identity_list_version_id = "ilv:12345abcde"
    }
  ]
  evidence_source_binding = [
    {
      name = "approvers"
      type = "approval"
      approval = {
        approval = {
          payload_type = "TypedDataV4"
        }
      }
    }
  ]
  definition_yaml = yamlencode({
    "evidence" = [{ "name" = "approval", "source" = "approvers" }]
    "decision" = { "gate" = { "evidence" = ["approval"] } }
  })
}
`

func TestPMSPolicyInlineBindingsAndVersionInOneCall(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer mp.server.Close()

	policyResource := "kaleido_platform_pms_policy.wired"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + pms_policy_inline_everything,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(policyResource, "applied_version"),
					resource.TestCheckResourceAttrSet(policyResource, "evidence_source_binding.0.id"),
					resource.TestCheckResourceAttr(policyResource, "evidence_source_binding.0.name", "approvers"),
					resource.TestCheckResourceAttr(policyResource, "evidence_source_binding.1.name", "documents"),
					func(s *terraform.State) error {
						assert.Len(t, mp.pmsPolicyPutBodies, 1, "the policy, its bindings and its version must be created by one call")
						body := mp.pmsPolicyPutBodies[0]
						assert.Contains(t, body, "identityListBindings")
						assert.Contains(t, body, "evidenceSourceBindings")
						assert.Contains(t, body, "decision", "the definition must be sent at the top level")
						esbs := body["evidenceSourceBindings"].(map[string]interface{})
						assert.Contains(t, esbs, "approvers")
						assert.Contains(t, esbs, "documents")
						return nil
					},
				),
			},
			{
				Config: providerConfig + pms_policy_inline_one_esb,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(policyResource, "evidence_source_binding.#", "1"),
					func(s *terraform.State) error {
						names := map[string]bool{}
						for _, b := range mp.pmsEvidenceSourceBindings {
							names[b.Name] = true
						}
						assert.True(t, names["approvers"], "the binding still in the configuration must remain")
						assert.False(t, names["documents"], "the binding dropped from the configuration must be deleted")
						return nil
					},
				),
			},
		},
	})
}

// A policy with no definition_yaml is a container: it is created by a single call and no
// version is cut for it.
var pms_policy_container_only = `
resource "kaleido_platform_pms_policy" "container_only" {
  environment = "test-env"
  service = "test-service"
  name = "container-only-policy"
  description = "no version yet"
}
`

func TestPMSPolicyContainerWithoutDefinition(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			// create: one call, with no version posted for it
			"PUT /endpoint/{env}/{service}/rest/api/v2/policies/{policy}",
			"GET /endpoint/{env}/{service}/rest/api/v2/policies/{policy}",
			// refresh before destroy
			"GET /endpoint/{env}/{service}/rest/api/v2/policies/{policy}",
			// destroy
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
				Config: providerConfig + pms_policy_container_only,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kaleido_platform_pms_policy.container_only", "applied_version", ""),
					func(s *terraform.State) error {
						policyID := s.RootModule().Resources["kaleido_platform_pms_policy.container_only"].Primary.Attributes["id"]
						assert.Empty(t, mp.pmsPolicyVersions[policyID], "no version should have been created for a container policy")
						return nil
					},
				),
			},
		},
	})
}

// A definition_yaml with its own top level description sets the version's description,
// not the policy's. The policy attribute must stay null when it was never configured,
// or terraform rejects the apply as an inconsistent result.
var pms_policy_yaml_description = `
resource "kaleido_platform_pms_policy" "described" {
  environment = "test-env"
  service = "test-service"
  name = "described-policy"
  definition_yaml = yamlencode({
    "description" = "A policy for approving any request"
    "evidence" = [{ "name" = "request" }]
    "decision" = { "cases" = [{ "allow" = { "rego" = "true" } }] }
  })
}
`

func TestPMSPolicyDefinitionDescriptionDoesNotBecomePolicyDescription(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer mp.server.Close()

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + pms_policy_yaml_description,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("kaleido_platform_pms_policy.described", "description"),
					func(s *terraform.State) error {
						body := mp.pmsPolicyPutBodies[0]
						assert.Equal(t, "", body["description"], "the policy description must not be taken from the definition")
						assert.Equal(t, "A policy for approving any request", body["versionDescription"],
							"the definition's description belongs to the version")
						return nil
					},
				),
			},
		},
	})
}
