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
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"
)

var evmConnectorContractDeployStep1 = `
resource "kaleido_platform_evm_connector_contract_deploy" "deploy1" {
    environment = "env1"
    service = "service1"
    api = "evm"
    key = "kld:///keystore/k1/key/signer1"
    abi = jsonencode([{"type": "constructor", "inputs": [{"name": "initialOwner", "type": "address"}]}])
    bytecode = "0x6080604052"
    params_json = jsonencode(["0x1234567890123456789012345678901234567890"])
}
`

var evmConnectorContractDeployStep2 = `
resource "kaleido_platform_evm_connector_contract_deploy" "deploy1" {
    environment = "env1"
    service = "service1"
    api = "evm"
    key = "kld:///keystore/k1/key/signer1"
    abi = jsonencode([{"type": "constructor", "inputs": [{"name": "initialOwner", "type": "address"}]}])
    bytecode = "0x6080604052"
    params_json = jsonencode(["0x1234567890123456789012345678901234567890"])
    wait_timeout = "1m"
}
`

// mockEVMConnectorDeploy simulates the connector's standard EVM API deploy operation and
// the workflow-engine transaction APIs it drives
type mockEVMConnectorDeploy struct {
	lock          sync.Mutex
	txn           *EVMConnectorTransactionAPIModel
	lastSubmit    *EVMConnectorDeploySubmitAPIModel
	submitStatus  int // 0 means accept the submission
	deleted       bool
	submitCount   int
	waitCount     int
	pendingWaits  int // number of wait calls that report pending before completion
	failCompleted bool
}

func (md *mockEVMConnectorDeploy) register(mp *mockPlatform) {
	mp.router.HandleFunc("/endpoint/{env}/{service}/rest/api/v1/apis/{api}/api/contract/deploy", func(res http.ResponseWriter, req *http.Request) {
		md.lock.Lock()
		defer md.lock.Unlock()
		md.submitCount++
		var submit EVMConnectorDeploySubmitAPIModel
		_ = json.NewDecoder(req.Body).Decode(&submit)
		md.lastSubmit = &submit
		if md.submitStatus != 0 {
			res.WriteHeader(md.submitStatus)
			_, _ = res.Write([]byte(`{"error":"KA140401: Idempotency key '` + submit.IdempotencyKey + `' already used for transaction '` + md.txn.ID + `'"}`))
			return
		}
		md.txn.IdempotencyKey = submit.IdempotencyKey
		res.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(res).Encode(&EVMConnectorSubmitResultAPIModel{
			ID:             md.txn.ID,
			IdempotencyKey: submit.IdempotencyKey,
		})
	}).Methods(http.MethodPost)

	mp.router.HandleFunc("/endpoint/{env}/{service}/rest/api/v1/transactions/{idOrKey}", func(res http.ResponseWriter, req *http.Request) {
		md.lock.Lock()
		defer md.lock.Unlock()
		idOrKey := mux.Vars(req)["idOrKey"]
		if req.Method == http.MethodDelete {
			md.deleted = true
			res.WriteHeader(http.StatusNoContent)
			return
		}
		if md.deleted || (idOrKey != md.txn.ID && idOrKey != md.txn.IdempotencyKey) {
			res.WriteHeader(http.StatusNotFound)
			_, _ = res.Write([]byte(`{"error":"not found"}`))
			return
		}
		_ = json.NewEncoder(res).Encode(md.txn)
	}).Methods(http.MethodGet, http.MethodDelete)

	mp.router.HandleFunc("/endpoint/{env}/{service}/rest/api/v1/transactions/{idOrKey}/wait", func(res http.ResponseWriter, req *http.Request) {
		md.lock.Lock()
		defer md.lock.Unlock()
		md.waitCount++
		if md.waitCount <= md.pendingWaits {
			// simulate the server-side wait timeout expiring before completion
			res.WriteHeader(http.StatusInternalServerError)
			_, _ = res.Write([]byte(`{"error":"KA140272: Timed out waiting for transaction to complete after 2m0s"}`))
			return
		}
		if md.failCompleted {
			md.txn.Status = "failure"
			md.txn.Stage = "failed"
			md.txn.Error = "insufficient funds for gas"
		} else {
			md.txn.Status = "success"
			md.txn.Stage = "succeeded"
			md.txn.Output = &EVMConnectorDeployOutputAPIModel{
				Receipt: &EVMConnectorDeployReceiptAPIModel{
					TransactionHash: "0xaabbcc",
					ContractAddress: "0x77b7f4d0d90fbba59fb520d7c9275b2820d31234",
					BlockNumber:     json.RawMessage(`"0x888"`),
				},
			}
		}
		_ = json.NewEncoder(res).Encode(md.txn)
	}).Methods(http.MethodGet)
}

func TestEVMConnectorContractDeploy(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer mp.server.Close()

	md := &mockEVMConnectorDeploy{
		txn: &EVMConnectorTransactionAPIModel{
			ID:     "txn-0001",
			Status: "pending",
			Stage:  "submit",
		},
	}
	md.register(mp)

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + evmConnectorContractDeployStep1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kaleido_platform_evm_connector_contract_deploy.deploy1", "id", "txn-0001"),
					resource.TestCheckResourceAttr("kaleido_platform_evm_connector_contract_deploy.deploy1", "contract_address", "0x77b7f4d0d90fbba59fb520d7c9275b2820d31234"),
					resource.TestCheckResourceAttr("kaleido_platform_evm_connector_contract_deploy.deploy1", "transaction_hash", "0xaabbcc"),
					resource.TestCheckResourceAttr("kaleido_platform_evm_connector_contract_deploy.deploy1", "block_number", "0x888"),
					resource.TestCheckResourceAttrWith("kaleido_platform_evm_connector_contract_deploy.deploy1", "idempotency_key", func(value string) error {
						assert.True(t, strings.HasPrefix(value, "tfdeploy-"))
						return nil
					}),
				),
			},
			{
				// only wait_timeout changes - must update in place without resubmitting
				Config: providerConfig + evmConnectorContractDeployStep2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kaleido_platform_evm_connector_contract_deploy.deploy1", "id", "txn-0001"),
					resource.TestCheckResourceAttr("kaleido_platform_evm_connector_contract_deploy.deploy1", "contract_address", "0x77b7f4d0d90fbba59fb520d7c9275b2820d31234"),
				),
			},
		},
	})

	// The deployment must only ever have been submitted once, and destroy must have
	// deleted the transaction record to free the idempotency key
	assert.Equal(t, 1, md.submitCount)
	assert.True(t, md.deleted)

	// Verify the submitted operation input carried through all the fields
	assert.True(t, strings.HasPrefix(md.lastSubmit.IdempotencyKey, "tfdeploy-"))
	testJSONEqual(t, md.lastSubmit.Input, `{
		"key": "kld:///keystore/k1/key/signer1",
		"abi": [{"type": "constructor", "inputs": [{"name": "initialOwner", "type": "address"}]}],
		"bytecode": "0x6080604052",
		"params": ["0x1234567890123456789012345678901234567890"]
	}`)
}

func TestEVMConnectorContractDeployIdempotencyConflictAutoImport(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer mp.server.Close()

	// The idempotency key is already bound to txn-0002 (e.g. a previous partial apply) -
	// the provider must recover the transaction ID from a GET by idempotency key
	md := &mockEVMConnectorDeploy{
		submitStatus: http.StatusConflict,
		txn: &EVMConnectorTransactionAPIModel{
			ID:     "txn-0002",
			Status: "pending",
			Stage:  "submit",
		},
	}
	md.register(mp)

	deployConfig := `
resource "kaleido_platform_evm_connector_contract_deploy" "deploy1" {
    environment = "env1"
    service = "service1"
    api = "evm"
    key = "signer1"
    abi = jsonencode([{"type": "constructor"}])
    bytecode = "0x6080604052"
    idempotency_key = "my-deploy-1"
}
`
	md.txn.IdempotencyKey = "my-deploy-1"

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + deployConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kaleido_platform_evm_connector_contract_deploy.deploy1", "id", "txn-0002"),
					resource.TestCheckResourceAttr("kaleido_platform_evm_connector_contract_deploy.deploy1", "idempotency_key", "my-deploy-1"),
					resource.TestCheckResourceAttr("kaleido_platform_evm_connector_contract_deploy.deploy1", "contract_address", "0x77b7f4d0d90fbba59fb520d7c9275b2820d31234"),
				),
			},
		},
	})
}

func TestEVMConnectorContractDeployFailedTransaction(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer mp.server.Close()

	md := &mockEVMConnectorDeploy{
		failCompleted: true,
		txn: &EVMConnectorTransactionAPIModel{
			ID:     "txn-0003",
			Status: "pending",
			Stage:  "submit",
		},
	}
	md.register(mp)

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      providerConfig + evmConnectorContractDeployStep1,
				ExpectError: regexp.MustCompile(`insufficient funds for gas`),
			},
		},
	})
}
