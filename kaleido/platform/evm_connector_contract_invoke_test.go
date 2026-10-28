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
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/gorilla/mux"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"
)

var evmConnectorContractInvokeStep1 = `
resource "kaleido_platform_evm_connector_contract_invoke" "invoke1" {
    environment = "env1"
    service = "service1"
    api = "evm"
    key = "kld:///keystore/k1/key/signer1"
    to = "0x77b7f4d0d90fbba59fb520d7c9275b2820d31234"
    abi = jsonencode([{
      "type": "function",
      "name": "registerImplementation",
      "inputs": [
        {"name": "name", "type": "string"},
        {"name": "implementation", "type": "tuple", "components": [
          {"name": "implementation", "type": "address"},
          {"name": "verifier", "type": "address"}
        ]}
      ]
    }])
    method = "registerImplementation"
    params_json = jsonencode({
      name = "Zeto_Anon"
      implementation = {
        implementation = "0x1234567890123456789012345678901234567890"
        verifier = "0x0987654321098765432109876543210987654321"
      }
    })
}
`

var evmConnectorContractInvokeStep2 = strings.Replace(evmConnectorContractInvokeStep1,
	`method = "registerImplementation"`,
	`method = "registerImplementation"
    wait_timeout = "1m"`, 1)

// mockEVMConnectorInvoke simulates the connector's standard EVM API invoke operation and
// the workflow-engine transaction APIs it drives
type mockEVMConnectorInvoke struct {
	lock          sync.Mutex
	txn           *EVMConnectorTransactionAPIModel
	lastSubmit    *EVMConnectorInvokeSubmitAPIModel
	submitStatus  int // 0 means accept the submission
	deleted       bool
	submitCount   int
	waitCount     int
	pendingWaits  int // number of wait calls that report pending before completion
	failCompleted bool
}

func (mi *mockEVMConnectorInvoke) register(mp *mockPlatform) {
	mp.router.HandleFunc("/endpoint/{env}/{service}/rest/api/v1/apis/{api}/api/contract/invoke", func(res http.ResponseWriter, req *http.Request) {
		mi.lock.Lock()
		defer mi.lock.Unlock()
		mi.submitCount++
		var submit EVMConnectorInvokeSubmitAPIModel
		_ = json.NewDecoder(req.Body).Decode(&submit)
		mi.lastSubmit = &submit
		if mi.submitStatus != 0 {
			res.WriteHeader(mi.submitStatus)
			_, _ = res.Write([]byte(`{"error":"KA140401: Idempotency key '` + submit.IdempotencyKey + `' already used for transaction '` + mi.txn.ID + `'"}`))
			return
		}
		mi.txn.IdempotencyKey = submit.IdempotencyKey
		res.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(res).Encode(&EVMConnectorSubmitResultAPIModel{
			ID:             mi.txn.ID,
			IdempotencyKey: submit.IdempotencyKey,
		})
	}).Methods(http.MethodPost)

	mp.router.HandleFunc("/endpoint/{env}/{service}/rest/api/v1/transactions/{idOrKey}", func(res http.ResponseWriter, req *http.Request) {
		mi.lock.Lock()
		defer mi.lock.Unlock()
		idOrKey := mux.Vars(req)["idOrKey"]
		if req.Method == http.MethodDelete {
			mi.deleted = true
			res.WriteHeader(http.StatusNoContent)
			return
		}
		if mi.deleted || (idOrKey != mi.txn.ID && idOrKey != mi.txn.IdempotencyKey) {
			res.WriteHeader(http.StatusNotFound)
			_, _ = res.Write([]byte(`{"error":"not found"}`))
			return
		}
		_ = json.NewEncoder(res).Encode(mi.txn)
	}).Methods(http.MethodGet, http.MethodDelete)

	mp.router.HandleFunc("/endpoint/{env}/{service}/rest/api/v1/transactions/{idOrKey}/wait", func(res http.ResponseWriter, req *http.Request) {
		mi.lock.Lock()
		defer mi.lock.Unlock()
		mi.waitCount++
		if mi.waitCount <= mi.pendingWaits {
			// simulate the server-side wait timeout expiring before completion
			res.WriteHeader(http.StatusInternalServerError)
			_, _ = res.Write([]byte(`{"error":"KA140272: Timed out waiting for transaction to complete after 2m0s"}`))
			return
		}
		if mi.failCompleted {
			mi.txn.Status = "failure"
			mi.txn.Stage = "failed"
			mi.txn.Error = "execution reverted: Ownable: caller is not the owner"
		} else {
			mi.txn.Status = "success"
			mi.txn.Stage = "succeeded"
			mi.txn.Output = &EVMConnectorDeployOutputAPIModel{
				Receipt: &EVMConnectorDeployReceiptAPIModel{
					TransactionHash: "0xddeeff",
					BlockNumber:     json.RawMessage(`999`),
				},
			}
		}
		_ = json.NewEncoder(res).Encode(mi.txn)
	}).Methods(http.MethodGet)
}

func TestEVMConnectorContractInvoke(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer mp.server.Close()

	mi := &mockEVMConnectorInvoke{
		pendingWaits: 1,
		txn: &EVMConnectorTransactionAPIModel{
			ID:     "txn-1001",
			Status: "pending",
			Stage:  "submit",
		},
	}
	mi.register(mp)

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + evmConnectorContractInvokeStep1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kaleido_platform_evm_connector_contract_invoke.invoke1", "id", "txn-1001"),
					resource.TestCheckResourceAttr("kaleido_platform_evm_connector_contract_invoke.invoke1", "transaction_hash", "0xddeeff"),
					resource.TestCheckResourceAttr("kaleido_platform_evm_connector_contract_invoke.invoke1", "block_number", "999"),
					resource.TestCheckResourceAttrWith("kaleido_platform_evm_connector_contract_invoke.invoke1", "idempotency_key", func(value string) error {
						assert.True(t, strings.HasPrefix(value, "tfinvoke-"))
						return nil
					}),
				),
			},
			{
				// only wait_timeout changes - must update in place without resubmitting
				Config: providerConfig + evmConnectorContractInvokeStep2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kaleido_platform_evm_connector_contract_invoke.invoke1", "id", "txn-1001"),
					resource.TestCheckResourceAttr("kaleido_platform_evm_connector_contract_invoke.invoke1", "transaction_hash", "0xddeeff"),
				),
			},
		},
	})

	// The invocation must only ever have been submitted once (surviving a wait timeout),
	// and destroy must have deleted the transaction record to free the idempotency key
	assert.Equal(t, 1, mi.submitCount)
	assert.True(t, mi.deleted)

	// Verify the submitted operation input carried through all the fields, including the
	// nested tuple parameter
	assert.True(t, strings.HasPrefix(mi.lastSubmit.IdempotencyKey, "tfinvoke-"))
	testJSONEqual(t, mi.lastSubmit.Input, `{
		"key": "kld:///keystore/k1/key/signer1",
		"to": "0x77b7f4d0d90fbba59fb520d7c9275b2820d31234",
		"abi": [{
			"type": "function",
			"name": "registerImplementation",
			"inputs": [
				{"name": "name", "type": "string"},
				{"name": "implementation", "type": "tuple", "components": [
					{"name": "implementation", "type": "address"},
					{"name": "verifier", "type": "address"}
				]}
			]
		}],
		"method": "registerImplementation",
		"params": {
			"name": "Zeto_Anon",
			"implementation": {
				"implementation": "0x1234567890123456789012345678901234567890",
				"verifier": "0x0987654321098765432109876543210987654321"
			}
		}
	}`)
}

func TestEVMConnectorContractInvokeIdempotencyConflict(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer mp.server.Close()

	// The idempotency key is already bound to another transaction - the apply must fail for
	// the user to resolve, never adopting the existing transaction
	mi := &mockEVMConnectorInvoke{
		submitStatus: http.StatusConflict,
		txn: &EVMConnectorTransactionAPIModel{
			ID:             "txn-1002",
			IdempotencyKey: "my-invoke-1",
			Status:         "pending",
			Stage:          "submit",
		},
	}
	mi.register(mp)

	invokeConfig := `
resource "kaleido_platform_evm_connector_contract_invoke" "invoke1" {
    environment = "env1"
    service = "service1"
    api = "evm"
    key = "signer1"
    to = "0x77b7f4d0d90fbba59fb520d7c9275b2820d31234"
    abi = jsonencode([{"type": "function", "name": "initialize", "inputs": []}])
    method = "initialize"
    idempotency_key = "my-invoke-1"
}
`

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      providerConfig + invokeConfig,
				ExpectError: regexp.MustCompile(`idempotency key 'my-invoke-1' is already in use`),
			},
		},
	})

	// The provider must not have waited on or adopted the conflicting transaction
	assert.Equal(t, 0, mi.waitCount)
}

func TestEVMConnectorContractInvokeFailedTransaction(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer mp.server.Close()

	mi := &mockEVMConnectorInvoke{
		failCompleted: true,
		txn: &EVMConnectorTransactionAPIModel{
			ID:     "txn-1003",
			Status: "pending",
			Stage:  "submit",
		},
	}
	mi.register(mp)

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      providerConfig + evmConnectorContractInvokeStep1,
				ExpectError: regexp.MustCompile(`caller is not the owner`),
			},
		},
	})
}

func TestEVMConnectorContractInvokeIdempotencyKeyDerivation(t *testing.T) {
	// The derived key must be stable for identical inputs, change with any input, and be
	// distinct from a deploy key so the two resources can never collide
	base := func() *EVMConnectorContractInvokeResourceModel {
		var data EVMConnectorContractInvokeResourceModel
		data.Environment = types.StringValue("env1")
		data.Service = types.StringValue("service1")
		data.API = types.StringValue("evm")
		data.Key = types.StringValue("signer1")
		data.To = types.StringValue("0x77b7f4d0d90fbba59fb520d7c9275b2820d31234")
		data.ABI = types.StringValue(`[]`)
		data.Method = types.StringValue("initialize")
		return &data
	}
	k1 := base().deriveIdempotencyKey()
	assert.Equal(t, k1, base().deriveIdempotencyKey())
	assert.True(t, strings.HasPrefix(k1, "tfinvoke-"))

	changed := base()
	changed.ParamsJSON = types.StringValue(`["x"]`)
	assert.NotEqual(t, k1, changed.deriveIdempotencyKey())

	changed = base()
	changed.To = types.StringValue("0x0000000000000000000000000000000000000001")
	assert.NotEqual(t, k1, changed.deriveIdempotencyKey())
}
