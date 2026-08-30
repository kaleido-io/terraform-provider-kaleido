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
	"regexp"
	"testing"
	"time"

	"github.com/aidarkhanov/nanoid"
	"github.com/gorilla/mux"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	_ "embed"
)

var pms_esb_approval_step1 = `
resource "kaleido_platform_pms_policy_evidence_source_binding" "approvers" {
  environment = "test-env"
  service = "test-service"
  policy = "test-policy"
  name = "approvers"
  type = "approval"
  approval = {
    approval = {
      payload_type = "TypedDataV4"
      payload_template_jsonata = "$.decision.approve"
    }
    rejection = {
      payload_type = "TypedDataV4"
      payload_template_jsonata = "$.decision.reject"
    }
    identity_list_version = {
      id = "pmil:12345abcde"
      version = "2026.1.0"
    }
  }
}
`

var pms_esb_approval_step2 = `
resource "kaleido_platform_pms_policy_evidence_source_binding" "approvers" {
  environment = "test-env"
  service = "test-service"
  policy = "test-policy"
  name = "approvers"
  type = "approval"
  approval = {
    approval = {
      payload_type = "TypedDataV4"
      payload_template_jsonata = "$.decision.approve"
    }
    rejection = {
      payload_type = "TypedDataV4"
      payload_template_jsonata = "$.decision.reject"
    }
    identity_list_version = {
      id = "pmil:12345abcde"
      version = "2026.2.0"
    }
  }
}
`

func TestPMSEvidenceSourceBindingApproval(t *testing.T) {
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
				Config: providerConfig + pms_esb_approval_step1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(esbResource, "id"),
					resource.TestCheckResourceAttr(esbResource, "name", "approvers"),
					resource.TestCheckResourceAttr(esbResource, "type", "approval"),
					resource.TestCheckResourceAttr(esbResource, "approval.approval.payload_type", "TypedDataV4"),
					resource.TestCheckResourceAttr(esbResource, "approval.rejection.payload_template_jsonata", "$.decision.reject"),
					resource.TestCheckResourceAttr(esbResource, "approval.identity_list_version.version", "2026.1.0"),
				),
			},
			{
				Config: providerConfig + pms_esb_approval_step2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(esbResource, "approval.identity_list_version.version", "2026.2.0"),
				),
			},
		},
	})
}

var pms_esb_attachment = `
resource "kaleido_platform_pms_policy_evidence_source_binding" "documents" {
  environment = "test-env"
  service = "test-service"
  policy = "test-policy"
  name = "documents"
  type = "attachment"
  attachment = {
    payload_jsonata = "$.input.document"
    attestation_jsonata = "$.input.signature"
  }
}
`

func TestPMSEvidenceSourceBindingAttachment(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"POST /endpoint/{env}/{service}/rest/api/v2/policies/{policy}/evidence-source-bindings",
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
				Config: providerConfig + pms_esb_attachment,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(esbResource, "type", "attachment"),
					resource.TestCheckResourceAttr(esbResource, "attachment.payload_jsonata", "$.input.document"),
					resource.TestCheckResourceAttr(esbResource, "attachment.attestation_jsonata", "$.input.signature"),
				),
			},
		},
	})
}

var pms_esb_type_mismatch = `
resource "kaleido_platform_pms_policy_evidence_source_binding" "mismatch" {
  environment = "test-env"
  service = "test-service"
  policy = "test-policy"
  name = "mismatch"
  type = "approval"
  attachment = {
    payload_jsonata = "$.input.document"
  }
}
`

func TestPMSEvidenceSourceBindingTypeMismatch(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{})
		mp.server.Close()
	}()

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      providerConfig + pms_esb_type_mismatch,
				ExpectError: regexp.MustCompile(`the approval block must be set when type is`),
			},
		},
	})
}

func TestPMSEvidenceSourceBindingAttachmentEmptyApprovalEchoed(t *testing.T) {
	mp, providerConfig := testSetup(t)
	// Reproduce a server that echoes back empty objects for the type-specific fields
	// the binding does not use - these must not surface as non-null blocks in state
	mp.echoEmptyEvidenceSourceBindingBlocks = true
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
				Config: providerConfig + pms_esb_attachment,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(esbResource, "attachment.payload_jsonata", "$.input.document"),
					resource.TestCheckNoResourceAttr(esbResource, "approval.%"),
				),
			},
			{
				// The echoed empty blocks must not produce a perpetual diff
				Config:             providerConfig + pms_esb_attachment,
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
	if mp.echoEmptyEvidenceSourceBindingBlocks {
		if binding.Approval == nil {
			binding.Approval = &PMSApprovalEvidenceSourceBindingAPIModel{
				IdentityListVersion: &PMSIdentityListVersionReferenceAPIModel{},
			}
		}
		if binding.Attachment == nil {
			binding.Attachment = &PMSAttachmentEvidenceSourceBindingAPIModel{}
		}
	}
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

func (mp *mockPlatform) patchPMSEvidenceSourceBinding(res http.ResponseWriter, req *http.Request) {
	binding := mp.pmsEvidenceSourceBindings[mux.Vars(req)["binding"]]
	if binding == nil {
		mp.respond(res, nil, 404)
		return
	}
	var updates PMSEvidenceSourceBindingPatchAPIModel
	mp.getBody(req, &updates)
	binding.Approval = updates.Approval
	binding.Attachment = updates.Attachment
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
