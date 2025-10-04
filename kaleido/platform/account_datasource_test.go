// Copyright © Kaleido, Inc. 2025

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
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"net/http"
	"testing"
)

var accountDataStep1 = `
data "kaleido_platform_account" "account" {}
`

func TestAccountData(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"GET /api/v1/self/identity", // read for initial state
			"GET /api/v1/self/identity", // get data
			"GET /api/v1/self/identity", // read for destroy / final state
		})
		mp.server.Close()
	}()

	accountData := "data.kaleido_platform_account.account"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + accountDataStep1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(accountData, "account_id", "an-account"),
				),
			},
		},
	})
}

func (mp *mockPlatform) getSelfIdentity(res http.ResponseWriter, req *http.Request) {
	sid := &SelfIdentityAPIModel{
		AccountID: "an-account",
	}
	mp.respond(res, sid, 200)
}
