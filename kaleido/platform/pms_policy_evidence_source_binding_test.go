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

var pms_esb_sourced_step1 = `
resource "kaleido_platform_pms_policy_evidence_source_binding" "approvers" {
  environment = "test-env"
  service = "test-service"
  policy = "test-policy"
  policy_evidence_source = "treasuryApproval"
  evidence_source_id = "pes:12345abcde"
  attesters = "treasuryOperations"
}
`

var pms_esb_sourced_step2 = `
resource "kaleido_platform_pms_policy_evidence_source_binding" "approvers" {
  environment = "test-env"
  service = "test-service"
  policy = "test-policy"
  policy_evidence_source = "treasuryApproval"
  evidence_source_id = "pes:12345abcde"
  attesters = "treasuryExecutives"
  run_as = "ap:12345abcde"
}
`

func TestPMSEvidenceSourceBindingSourced(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"POST /endpoint/{env}/{service}/rest/api/v2/policies/{policy}/evidence-source-bindings",
			"GET /endpoint/{env}/{service}/rest/api/v2/policies/{policy}/evidence-source-bindings/{binding}",
			"GET /endpoint/{env}/{service}/rest/api/v2/policies/{policy}/evidence-source-bindings/{binding}",
			"PATCH /endpoint/{env}/{service}/rest/api/v2/policies/{policy}/evidence-source-bindings/{binding}",
			"GET /endpoint/{env}/{service}/rest/api/v2/policies/{policy}/evidence-source-bindings/{binding}",
			"DELETE /endpoint/{env}/{service}/rest/api/v2/policies/{policy}/evidence-source-bindings/{binding}",
		})
		mp.server.Close()
	}()

	esbResource := "kaleido_platform_pms_policy_evidence_source_binding.approvers"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + pms_esb_sourced_step1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(esbResource, "id"),
					resource.TestCheckResourceAttr(esbResource, "policy_evidence_source", "treasuryApproval"),
					resource.TestCheckResourceAttr(esbResource, "evidence_source_id", "pes:12345abcde"),
					resource.TestCheckResourceAttr(esbResource, "attesters", "treasuryOperations"),
					resource.TestCheckNoResourceAttr(esbResource, "run_as"),
				),
			},
			{
				Config: providerConfig + pms_esb_sourced_step2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(esbResource, "attesters", "treasuryExecutives"),
					resource.TestCheckResourceAttr(esbResource, "run_as", "ap:12345abcde"),
				),
			},
		},
	})
}

// A binding with no evidence source is a slot whose evidence is pushed in; only the
// ingress mappings apply.
var pms_esb_sourceless = `
resource "kaleido_platform_pms_policy_evidence_source_binding" "documents" {
  environment = "test-env"
  service = "test-service"
  policy = "test-policy"
  policy_evidence_source = "documents"
  payload_jsonata = "body.document"
  attestation_jsonata = "body.signature"
}
`

func TestPMSEvidenceSourceBindingSourceless(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"POST /endpoint/{env}/{service}/rest/api/v2/policies/{policy}/evidence-source-bindings",
			"GET /endpoint/{env}/{service}/rest/api/v2/policies/{policy}/evidence-source-bindings/{binding}",
			"GET /endpoint/{env}/{service}/rest/api/v2/policies/{policy}/evidence-source-bindings/{binding}",
			"DELETE /endpoint/{env}/{service}/rest/api/v2/policies/{policy}/evidence-source-bindings/{binding}",
		})
		mp.server.Close()
	}()

	esbResource := "kaleido_platform_pms_policy_evidence_source_binding.documents"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + pms_esb_sourceless,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr(esbResource, "evidence_source_id"),
					resource.TestCheckResourceAttr(esbResource, "payload_jsonata", "body.document"),
					resource.TestCheckResourceAttr(esbResource, "attestation_jsonata", "body.signature"),
				),
			},
			{
				Config:             providerConfig + pms_esb_sourceless,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func (mp *mockPlatform) postPMSEvidenceSourceBinding(res http.ResponseWriter, req *http.Request) {
	var binding PMSEvidenceSourceBindingAPIModel
	mp.getBody(req, &binding)
	now := time.Now().UTC()
	binding.ID = nanoid.New()
	binding.PolicyID = mux.Vars(req)["policy"]
	binding.Created = &now
	binding.Updated = &now
	mp.pmsEvidenceSourceBindings[binding.ID] = &binding
	mp.respond(res, &binding, http.StatusCreated)
}

func (mp *mockPlatform) getPMSEvidenceSourceBinding(res http.ResponseWriter, req *http.Request) {
	binding := mp.pmsEvidenceSourceBindings[mux.Vars(req)["binding"]]
	if binding == nil {
		mp.respond(res, nil, 404)
		return
	}
	mp.respond(res, binding, http.StatusOK)
}

// patchPMSEvidenceSourceBinding mirrors the server: a field the patch carries replaces
// the stored one; a field it omits is left alone.
func (mp *mockPlatform) patchPMSEvidenceSourceBinding(res http.ResponseWriter, req *http.Request) {
	binding := mp.pmsEvidenceSourceBindings[mux.Vars(req)["binding"]]
	if binding == nil {
		mp.respond(res, nil, 404)
		return
	}
	var updates PMSEvidenceSourceBindingTargetAPIModel
	mp.getBody(req, &updates)
	if updates.EvidenceSourceID != "" {
		binding.EvidenceSourceID = updates.EvidenceSourceID
	}
	if updates.Attesters != "" {
		binding.Attesters = updates.Attesters
	}
	if updates.RunAs != "" {
		binding.RunAs = updates.RunAs
	}
	if updates.PayloadMapping != nil {
		binding.PayloadMapping = updates.PayloadMapping
	}
	if updates.AttestationMapping != nil {
		binding.AttestationMapping = updates.AttestationMapping
	}
	now := time.Now().UTC()
	binding.Updated = &now
	mp.respond(res, binding, http.StatusOK)
}

func (mp *mockPlatform) deletePMSEvidenceSourceBinding(res http.ResponseWriter, req *http.Request) {
	id := mux.Vars(req)["binding"]
	if mp.pmsEvidenceSourceBindings[id] == nil {
		mp.respond(res, nil, 404)
		return
	}
	delete(mp.pmsEvidenceSourceBindings, id)
	mp.respond(res, nil, http.StatusNoContent)
}
