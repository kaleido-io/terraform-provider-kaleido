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
	"strings"
	"testing"
	"time"

	"github.com/aidarkhanov/nanoid"
	"github.com/gorilla/mux"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	_ "embed"
)

var pms_es_approval_step1 = `
resource "kaleido_platform_pms_evidence_source" "transfer_approval" {
  environment = "test-env"
  service = "test-service"
  name = "transferApproval"
  description = "asks the attesters to approve a transfer"
  type = "approval"
  request_schema_json = jsonencode({
    type = "object"
    properties = { amount = { type = "integer" } }
  })
  approval = {
    approve = {
      primary_type = "Approval"
      types_json = jsonencode({
        Approval = [{ name = "amount", type = "uint256" }]
      })
      message_jsonata = "{\"amount\": request.amount}"
    }
  }
}
`

var pms_es_approval_step2 = `
resource "kaleido_platform_pms_evidence_source" "transfer_approval" {
  environment = "test-env"
  service = "test-service"
  name = "transferApproval"
  description = "asks the attesters to approve or reject a transfer"
  type = "approval"
  request_schema_json = jsonencode({
    type = "object"
    properties = { amount = { type = "integer" } }
  })
  approval = {
    approve = {
      primary_type = "Approval"
      types_json = jsonencode({
        Approval = [{ name = "amount", type = "uint256" }]
      })
      message_jsonata = "{\"amount\": request.amount}"
    }
    reject = {
      primary_type = "Rejection"
      types_json = jsonencode({
        Rejection = [{ name = "amount", type = "uint256" }]
      })
      message_jsonata = "{\"amount\": request.amount}"
    }
  }
}
`

func TestPMSEvidenceSourceApproval(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"POST /endpoint/{env}/{service}/rest/api/v2/evidence-sources",
			"GET /endpoint/{env}/{service}/rest/api/v2/evidence-sources/{evidenceSource}",
			"GET /endpoint/{env}/{service}/rest/api/v2/evidence-sources/{evidenceSource}",
			"PATCH /endpoint/{env}/{service}/rest/api/v2/evidence-sources/{evidenceSource}",
			"GET /endpoint/{env}/{service}/rest/api/v2/evidence-sources/{evidenceSource}",
			"DELETE /endpoint/{env}/{service}/rest/api/v2/evidence-sources/{evidenceSource}",
		})
		mp.server.Close()
	}()

	esResource := "kaleido_platform_pms_evidence_source.transfer_approval"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + pms_es_approval_step1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(esResource, "id"),
					resource.TestCheckResourceAttr(esResource, "name", "transferApproval"),
					resource.TestCheckResourceAttr(esResource, "type", "approval"),
					resource.TestCheckResourceAttr(esResource, "approval.approve.primary_type", "Approval"),
					resource.TestCheckResourceAttr(esResource, "approval.approve.message_jsonata", `{"amount": request.amount}`),
					resource.TestCheckNoResourceAttr(esResource, "approval.reject.%"),
					resource.TestCheckNoResourceAttr(esResource, "service_request.%"),
				),
			},
			{
				Config: providerConfig + pms_es_approval_step2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(esResource, "description", "asks the attesters to approve or reject a transfer"),
					resource.TestCheckResourceAttr(esResource, "approval.reject.primary_type", "Rejection"),
				),
			},
		},
	})
}

var pms_es_service_request = `
resource "kaleido_platform_pms_evidence_source" "wallet_lookup" {
  environment = "test-env"
  service = "test-service"
  name = "walletLookup"
  type = "serviceRequest"
  service_request = {
    service = "myWalletManager"
    type = "WalletManagerService"
    options_json = jsonencode({ method = "GET", endpoint = "rest" })
    dynamic_options = {
      path_jsonata = "\"/wallets/\" & request.walletNameOrId"
    }
  }
}
`

func TestPMSEvidenceSourceServiceRequest(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"POST /endpoint/{env}/{service}/rest/api/v2/evidence-sources",
			"GET /endpoint/{env}/{service}/rest/api/v2/evidence-sources/{evidenceSource}",
			"GET /endpoint/{env}/{service}/rest/api/v2/evidence-sources/{evidenceSource}",
			"DELETE /endpoint/{env}/{service}/rest/api/v2/evidence-sources/{evidenceSource}",
		})
		mp.server.Close()
	}()

	esResource := "kaleido_platform_pms_evidence_source.wallet_lookup"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + pms_es_service_request,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(esResource, "type", "serviceRequest"),
					resource.TestCheckResourceAttr(esResource, "service_request.service", "myWalletManager"),
					resource.TestCheckResourceAttr(esResource, "service_request.dynamic_options.path_jsonata", `"/wallets/" & request.walletNameOrId`),
				),
			},
			{
				// The server lowercases the type and re-serialises the JSON; neither may
				// produce a perpetual diff.
				Config:             providerConfig + pms_es_service_request,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// A source declared without the optional JSON blobs must read them back as absent, not as
// the string "null".
var pms_es_service_request_minimal = `
resource "kaleido_platform_pms_evidence_source" "asset_lookup" {
  environment = "test-env"
  service = "test-service"
  name = "assetLookup"
  type = "serviceRequest"
  service_request = {
    service = "myWalletManager"
  }
}
`

func TestPMSEvidenceSourceServiceRequestMinimal(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer mp.server.Close()

	esResource := "kaleido_platform_pms_evidence_source.asset_lookup"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + pms_es_service_request_minimal,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(esResource, "service_request.service", "myWalletManager"),
					resource.TestCheckNoResourceAttr(esResource, "service_request.options_json"),
					resource.TestCheckNoResourceAttr(esResource, "service_request.dynamic_options.%"),
					resource.TestCheckNoResourceAttr(esResource, "request_schema_json"),
				),
			},
		},
	})
}

var pms_es_workflow = `
resource "kaleido_platform_pms_evidence_source" "screening" {
  environment = "test-env"
  service = "test-service"
  name = "screeningScore"
  type = "workflow"
  workflow = {
    transaction_template_json = jsonencode({
      workflow = "flw:9kxviy9izf"
      operation = "getScore"
      jsonata = "{\"input\": {\"ethAddress\": request.ethAddress}}"
    })
  }
}
`

func TestPMSEvidenceSourceWorkflow(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"POST /endpoint/{env}/{service}/rest/api/v2/evidence-sources",
			"GET /endpoint/{env}/{service}/rest/api/v2/evidence-sources/{evidenceSource}",
			"DELETE /endpoint/{env}/{service}/rest/api/v2/evidence-sources/{evidenceSource}",
		})
		mp.server.Close()
	}()

	esResource := "kaleido_platform_pms_evidence_source.screening"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + pms_es_workflow,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(esResource, "type", "workflow"),
					resource.TestCheckResourceAttrSet(esResource, "workflow.transaction_template_json"),
				),
			},
		},
	})
}

var pms_es_type_mismatch = `
resource "kaleido_platform_pms_evidence_source" "mismatch" {
  environment = "test-env"
  service = "test-service"
  name = "mismatch"
  type = "approval"
  workflow = {
    workflow = "flw:9kxviy9izf"
  }
}
`

func TestPMSEvidenceSourceTypeMismatch(t *testing.T) {
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
				Config:      providerConfig + pms_es_type_mismatch,
				ExpectError: regexp.MustCompile(`the workflow block must not be set when type is "approval"`),
			},
		},
	})
}

func (mp *mockPlatform) postPMSEvidenceSource(res http.ResponseWriter, req *http.Request) {
	var source PMSEvidenceSourceAPIModel
	mp.getBody(req, &source)
	now := time.Now().UTC()
	source.ID = "pes:" + nanoid.New()
	// The server stores an FFEnum lowercased.
	source.Type = strings.ToLower(source.Type)
	source.Created = &now
	source.Updated = &now
	mp.pmsEvidenceSources[source.ID] = &source
	mp.respond(res, &source, http.StatusCreated)
}

func (mp *mockPlatform) lookupPMSEvidenceSource(nameOrID string) *PMSEvidenceSourceAPIModel {
	if source := mp.pmsEvidenceSources[nameOrID]; source != nil {
		return source
	}
	for _, source := range mp.pmsEvidenceSources {
		if source.Name == nameOrID {
			return source
		}
	}
	return nil
}

func (mp *mockPlatform) getPMSEvidenceSource(res http.ResponseWriter, req *http.Request) {
	source := mp.lookupPMSEvidenceSource(mux.Vars(req)["evidenceSource"])
	if source == nil {
		mp.respond(res, nil, 404)
		return
	}
	mp.respond(res, source, http.StatusOK)
}

func (mp *mockPlatform) patchPMSEvidenceSource(res http.ResponseWriter, req *http.Request) {
	source := mp.lookupPMSEvidenceSource(mux.Vars(req)["evidenceSource"])
	if source == nil {
		mp.respond(res, nil, 404)
		return
	}
	var updates PMSEvidenceSourcePatchAPIModel
	mp.getBody(req, &updates)
	if updates.Description != "" {
		source.Description = updates.Description
	}
	if updates.RequestSchema != nil {
		source.RequestSchema = updates.RequestSchema
	}
	if updates.Approval != nil {
		source.Approval = updates.Approval
	}
	if updates.ServiceRequest != nil {
		source.ServiceRequest = updates.ServiceRequest
	}
	if updates.Workflow != nil {
		source.Workflow = updates.Workflow
	}
	now := time.Now().UTC()
	source.Updated = &now
	mp.respond(res, source, http.StatusOK)
}

func (mp *mockPlatform) deletePMSEvidenceSource(res http.ResponseWriter, req *http.Request) {
	source := mp.lookupPMSEvidenceSource(mux.Vars(req)["evidenceSource"])
	if source == nil {
		mp.respond(res, nil, 404)
		return
	}
	delete(mp.pmsEvidenceSources, source.ID)
	mp.respond(res, nil, http.StatusNoContent)
}
