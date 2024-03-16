// Copyright © Kaleido, Inc. 2024

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
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"

	_ "embed"
)

var evm_netinfoStep1 = func(mp *mockPlatform) string {
	return fmt.Sprintf(`
data "kaleido_platform_evm_netinfo" "evm_netinfo1" {
    json_rpc_url = "%s/json_rpc"
}
`, mp.server.URL)
}

func TestEVMNetInfo1(t *testing.T) {

	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"POST /json_rpc",
			"POST /json_rpc",
			"POST /json_rpc",
			"POST /json_rpc",
			"POST /json_rpc",
		})
		mp.server.Close()
	}()

	mp.mockRPC("eth_chainId", []interface{}{}, "0xAB4130", 500, 502)

	evm_netinfo1Resource := "data.kaleido_platform_evm_netinfo.evm_netinfo1"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + evm_netinfoStep1(mp),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(evm_netinfo1Resource, "chain_id", `11223344`),
				),
			},
		},
	})
}

func (mp *mockPlatform) mockRPC(method string, params []interface{}, result interface{}, failures ...int) {
	callCount := 0
	mp.register("/json_rpc", http.MethodPost, func(res http.ResponseWriter, req *http.Request) {
		var jReq RPCRequest
		err := json.NewDecoder(req.Body).Decode(&jReq)
		assert.NoError(mp.t, err)

		assert.Equal(mp.t, "2.0", jReq.JSONRpc)
		assert.NotNil(mp.t, jReq.ID)
		assert.Equal(mp.t, method, jReq.Method)
		assert.Equal(mp.t, params, params)

		callCount++
		if len(failures) >= callCount {
			res.WriteHeader(failures[callCount-1])
		}

		resultBytes, err := json.Marshal(result)
		assert.NoError(mp.t, err)
		jRes := RPCResponse{
			JSONRpc: "2.0",
			ID:      jReq.ID,
			Result:  resultBytes,
		}
		res.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(res).Encode(jRes)
		assert.NoError(mp.t, err)
	})
}
