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

var pms_of_step1 = `
resource "kaleido_platform_pms_output_formatter" "evm" {
  environment = "test-env"
  service = "test-service"
  name = "evmTransfer"
  description = "Shapes an EVM transfer for the wallet manager"
  type = "kaleido.policy.evm.v1"
  mapping_rego = "{\"to\": parameters.to, \"amount\": parameters.amount}"
  parameter = [
    { name = "to", type = "string", description = "Recipient address" },
    { name = "amount", type = "number", default_json = jsonencode(0) },
  ]
}
`

var pms_of_step2 = `
resource "kaleido_platform_pms_output_formatter" "evm" {
  environment = "test-env"
  service = "test-service"
  name = "evmTransfer"
  type = "kaleido.policy.evm.v1"
  mapping_rego = "{\"to\": parameters.to}"
  parameter = [
    { name = "to", type = "string", description = "Recipient address" },
  ]
}
`

func TestPMSOutputFormatter1(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"POST /endpoint/{env}/{service}/rest/api/v2/output-formatters",
			"GET /endpoint/{env}/{service}/rest/api/v2/output-formatters/{outputFormatter}",
			"GET /endpoint/{env}/{service}/rest/api/v2/output-formatters/{outputFormatter}",
			// update replaces the whole formatter, so the dropped description and parameter go away
			"PUT /endpoint/{env}/{service}/rest/api/v2/output-formatters/{outputFormatter}",
			"GET /endpoint/{env}/{service}/rest/api/v2/output-formatters/{outputFormatter}",
			"DELETE /endpoint/{env}/{service}/rest/api/v2/output-formatters/{outputFormatter}",
		})
		mp.server.Close()
	}()

	ofResource := "kaleido_platform_pms_output_formatter.evm"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + pms_of_step1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(ofResource, "id"),
					resource.TestCheckResourceAttr(ofResource, "name", "evmTransfer"),
					resource.TestCheckResourceAttr(ofResource, "type", "kaleido.policy.evm.v1"),
					resource.TestCheckResourceAttr(ofResource, "mapping_rego", `{"to": parameters.to, "amount": parameters.amount}`),
					resource.TestCheckResourceAttr(ofResource, "parameter.#", "2"),
					resource.TestCheckResourceAttr(ofResource, "parameter.0.name", "to"),
					resource.TestCheckResourceAttr(ofResource, "parameter.1.default_json", "0"),
					resource.TestCheckResourceAttrSet(ofResource, "created"),
				),
			},
			{
				Config: providerConfig + pms_of_step2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr(ofResource, "description"),
					resource.TestCheckResourceAttr(ofResource, "parameter.#", "1"),
					resource.TestCheckResourceAttr(ofResource, "mapping_rego", `{"to": parameters.to}`),
				),
			},
		},
	})
}

var pms_of_duplicate_parameter = `
resource "kaleido_platform_pms_output_formatter" "dup" {
  environment = "test-env"
  service = "test-service"
  name = "dup"
  type = "kaleido.policy.evm.v1"
  mapping_rego = "parameters.to"
  parameter = [
    { name = "to" },
    { name = "to" },
  ]
}
`

func TestPMSOutputFormatterDuplicateParameter(t *testing.T) {
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
				Config:      providerConfig + pms_of_duplicate_parameter,
				ExpectError: regexp.MustCompile(`parameter "to" is declared more than once`),
			},
		},
	})
}

func (mp *mockPlatform) postPMSOutputFormatter(res http.ResponseWriter, req *http.Request) {
	var formatter PMSOutputFormatterAPIModel
	mp.getBody(req, &formatter)
	now := time.Now().UTC()
	formatter.ID = "pof:" + nanoid.New()
	formatter.Created = &now
	formatter.Updated = &now
	mp.pmsOutputFormatters[formatter.ID] = &formatter
	mp.respond(res, &formatter, http.StatusCreated)
}

func (mp *mockPlatform) lookupPMSOutputFormatter(nameOrID string) *PMSOutputFormatterAPIModel {
	if formatter := mp.pmsOutputFormatters[nameOrID]; formatter != nil {
		return formatter
	}
	for _, formatter := range mp.pmsOutputFormatters {
		if formatter.Name == nameOrID {
			return formatter
		}
	}
	return nil
}

func (mp *mockPlatform) getPMSOutputFormatter(res http.ResponseWriter, req *http.Request) {
	formatter := mp.lookupPMSOutputFormatter(mux.Vars(req)["outputFormatter"])
	if formatter == nil {
		mp.respond(res, nil, 404)
		return
	}
	mp.respond(res, formatter, http.StatusOK)
}

// putPMSOutputFormatter mirrors the server: the body replaces the whole formatter,
// keeping its ID, immutable type and timestamps.
func (mp *mockPlatform) putPMSOutputFormatter(res http.ResponseWriter, req *http.Request) {
	existing := mp.lookupPMSOutputFormatter(mux.Vars(req)["outputFormatter"])
	if existing == nil {
		mp.respond(res, nil, 404)
		return
	}
	var formatter PMSOutputFormatterAPIModel
	mp.getBody(req, &formatter)
	formatter.ID = existing.ID
	formatter.Type = existing.Type
	formatter.Created = existing.Created
	now := time.Now().UTC()
	formatter.Updated = &now
	mp.pmsOutputFormatters[formatter.ID] = &formatter
	mp.respond(res, &formatter, http.StatusOK)
}

func (mp *mockPlatform) deletePMSOutputFormatter(res http.ResponseWriter, req *http.Request) {
	formatter := mp.lookupPMSOutputFormatter(mux.Vars(req)["outputFormatter"])
	if formatter == nil {
		mp.respond(res, nil, 404)
		return
	}
	delete(mp.pmsOutputFormatters, formatter.ID)
	mp.respond(res, nil, http.StatusNoContent)
}
