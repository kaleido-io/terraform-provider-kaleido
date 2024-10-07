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
	"net/http"
	"testing"

	"github.com/gorilla/mux"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	_ "embed"
)

var networkBoostrapStep1 = `
data "kaleido_platform_network_bootstrap_data" "boot1" {
    environment = "env1"
	network = "net1"
}
`

var networkBoostrapStep2 = `
data "kaleido_platform_network_bootstrap_data" "boot1" {
    environment = "env1"
	network = "net1"
	bootstrap_files = jsonencode({
		"files": {
			"genesis.json": {
				"data": {
					"text": "{\"alloc\":{\"0x12F62772C4652280d06E64CfBC9033d409559aD4\":{\"balance\":\"0x111111111111\"}},\"coinbase\":\"0x0000000000000000000000000000000000000000\",\"config\":{\"berlinBlock\":0,\"chainId\":12345,\"contractSizeLimit\":98304,\"qbft\":{\"blockperiodseconds\":5,\"epochlength\":30000,\"requesttimeoutseconds\":10},\"shanghaiTime\":0,\"zeroBaseFee\":true},\"difficulty\":\"0x1\",\"extraData\":\"0xf83aa00000000000000000000000000000000000000000000000000000000000000000d594ca45971aa3e3eb66e8cd4ad6ed491520b601bf6cc080c0\",\"gasLimit\":\"0x2fefd800\",\"mixHash\":\"0x63746963616c2062797a616e74696e65206661756c7420746f6c6572616e6365\"}"
				},
				"type": "json"
			}
		},
		"name": "init"
	})
}
`

func TestBootstrap1(t *testing.T) {

	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"GET /api/v1/environments/{env}/networks/{network}/initdata",
			"GET /api/v1/environments/{env}/networks/{network}/initdata",
			"GET /api/v1/environments/{env}/networks/{network}/initdata",
		})
		mp.server.Close()
	}()

	boot1Data := "data.kaleido_platform_network_bootstrap_data.boot1"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + networkBoostrapStep1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(boot1Data, "environment", `env1`),
				),
			},
			{
				Config: providerConfig + networkBoostrapStep2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(boot1Data, "environment", `env1`),
				),
			},
		},
	})
}

func (mp *mockPlatform) getNetworkInitData(res http.ResponseWriter, req *http.Request) {
	bd := mp.networkinitdatas[mux.Vars(req)["env"]+"/"+mux.Vars(req)["network"]+"/"+"initdata"]
	if bd == nil {
		mp.respond(res, nil, 404)
	} else {
		mp.respond(res, bd, 200)
	}
}
