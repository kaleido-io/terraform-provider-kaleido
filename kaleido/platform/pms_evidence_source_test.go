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
  parameter = [
    { name = "amount", type = "number", description = "The amount to approve" }
  ]
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
  parameter = [
    { name = "amount", type = "number", description = "The amount to approve" }
  ]
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
			"PUT /endpoint/{env}/{service}/rest/api/v2/evidence-sources/{evidenceSource}",
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
					resource.TestCheckResourceAttr(esResource, "parameter.0.name", "amount"),
					// The schema of an approval source is derived by the server from the typed data
					resource.TestCheckResourceAttr(esResource, "schema_json", `{"properties":{"amount":{"type":"string"}},"type":"object"}`),
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
  schema_json = jsonencode({
    type = "object"
    properties = { id = { type = "string" }, name = { type = "string" } }
  })
  payload_jsonata = "body"
  attestation_jsonata = "body.signature"
  parameter = [
    { name = "walletNameOrId", type = "string", display_name = "Wallet" },
    { name = "limit", type = "number", default_json = jsonencode(10), enum_json = jsonencode([10, 20]) },
  ]
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
					resource.TestCheckResourceAttr(esResource, "payload_jsonata", "body"),
					resource.TestCheckResourceAttr(esResource, "attestation_jsonata", "body.signature"),
					resource.TestCheckResourceAttr(esResource, "parameter.#", "2"),
					resource.TestCheckResourceAttr(esResource, "parameter.0.display_name", "Wallet"),
					resource.TestCheckResourceAttr(esResource, "parameter.1.default_json", "10"),
					resource.TestCheckResourceAttr(esResource, "parameter.1.enum_json", "[10,20]"),
					resource.TestCheckResourceAttr(esResource, "schema_json", `{"properties":{"id":{"type":"string"},"name":{"type":"string"}},"type":"object"}`),
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
					resource.TestCheckNoResourceAttr(esResource, "schema_json"),
					resource.TestCheckNoResourceAttr(esResource, "payload_jsonata"),
					resource.TestCheckNoResourceAttr(esResource, "parameter.#"),
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

// An attachment source requests nothing: it only describes the shape of evidence that is
// pushed in, so it needs no configuration block and only a schema.
var pms_es_attachment = `
resource "kaleido_platform_pms_evidence_source" "documents" {
  environment = "test-env"
  service = "test-service"
  name = "documents"
  type = "attachment"
  schema_json = jsonencode({
    type = "object"
    properties = { document = { type = "string" } }
  })
  payload_jsonata = "body.document"
  attestation_jsonata = "body.signature"
}
`

func TestPMSEvidenceSourceAttachment(t *testing.T) {
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

	esResource := "kaleido_platform_pms_evidence_source.documents"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + pms_es_attachment,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(esResource, "type", "attachment"),
					resource.TestCheckResourceAttr(esResource, "payload_jsonata", "body.document"),
					resource.TestCheckNoResourceAttr(esResource, "approval.%"),
					resource.TestCheckNoResourceAttr(esResource, "service_request.%"),
					resource.TestCheckNoResourceAttr(esResource, "workflow.%"),
				),
			},
			{
				Config:             providerConfig + pms_es_attachment,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

var pms_es_attachment_without_schema = `
resource "kaleido_platform_pms_evidence_source" "documents" {
  environment = "test-env"
  service = "test-service"
  name = "documents"
  type = "attachment"
}
`

var pms_es_approval_with_schema = `
resource "kaleido_platform_pms_evidence_source" "approval" {
  environment = "test-env"
  service = "test-service"
  name = "approval"
  type = "approval"
  schema_json = jsonencode({ type = "object" })
  approval = {
    approve = {
      primary_type = "Approval"
      types_json = jsonencode({ Approval = [{ name = "amount", type = "uint256" }] })
    }
  }
}
`

var pms_es_approval_with_mapping = `
resource "kaleido_platform_pms_evidence_source" "approval" {
  environment = "test-env"
  service = "test-service"
  name = "approval"
  type = "approval"
  payload_jsonata = "body"
  approval = {
    approve = {
      primary_type = "Approval"
      types_json = jsonencode({ Approval = [{ name = "amount", type = "uint256" }] })
    }
  }
}
`

func TestPMSEvidenceSourceTypeConsistency(t *testing.T) {
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
				Config:      providerConfig + pms_es_attachment_without_schema,
				ExpectError: regexp.MustCompile(`schema_json must be set when type is "attachment"`),
			},
			{
				Config:      providerConfig + pms_es_approval_with_schema,
				ExpectError: regexp.MustCompile(`schema_json must not be set when type is "approval"`),
			},
			{
				Config:      providerConfig + pms_es_approval_with_mapping,
				ExpectError: regexp.MustCompile(`(?s)payload_jsonata and\s+attestation_jsonata must not be set`),
			},
		},
	})
}

// deriveApprovalSchema mirrors the server, which derives the schema of an approval source
// from the typed data of its responses: here, a string property per member of the
// approve response's primary type.
func deriveApprovalSchema(source *PMSEvidenceSourceAPIModel) {
	if source.Approval == nil || source.Approval.Responses == nil || source.Approval.Responses.Approve == nil || source.Approval.Responses.Approve.TypedDataV4 == nil {
		return
	}
	typed := source.Approval.Responses.Approve.TypedDataV4
	properties := map[string]interface{}{}
	for _, member := range typed.Types[typed.PrimaryType] {
		properties[member.Name] = map[string]interface{}{"type": "string"}
	}
	source.Schema = map[string]interface{}{"type": "object", "properties": properties}
}

func (mp *mockPlatform) postPMSEvidenceSource(res http.ResponseWriter, req *http.Request) {
	var source PMSEvidenceSourceAPIModel
	mp.getBody(req, &source)
	now := time.Now().UTC()
	source.ID = "pes:" + nanoid.New()
	// The server stores an FFEnum lowercased.
	source.Type = strings.ToLower(source.Type)
	deriveApprovalSchema(&source)
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

// putPMSEvidenceSource mirrors the server: the body replaces the whole source, keeping
// its ID, immutable type and timestamps.
func (mp *mockPlatform) putPMSEvidenceSource(res http.ResponseWriter, req *http.Request) {
	existing := mp.lookupPMSEvidenceSource(mux.Vars(req)["evidenceSource"])
	if existing == nil {
		mp.respond(res, nil, 404)
		return
	}
	var source PMSEvidenceSourceAPIModel
	mp.getBody(req, &source)
	source.ID = existing.ID
	source.Type = existing.Type
	source.Created = existing.Created
	deriveApprovalSchema(&source)
	now := time.Now().UTC()
	source.Updated = &now
	mp.pmsEvidenceSources[source.ID] = &source
	mp.respond(res, &source, http.StatusOK)
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
