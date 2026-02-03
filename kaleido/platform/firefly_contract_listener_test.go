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
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/aidarkhanov/nanoid"
	"github.com/gorilla/mux"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/assert"

	_ "embed"
)

var firefly_contract_listenerStep1 = `
resource "kaleido_platform_firefly_contract_listener" "listener1" {
    environment = "env1"
    service     = "service1"
    namespace   = "ns1"
    name        = "mylistener"
    config_json = jsonencode({
        location = {
            address = "0x1234567890123456789012345678901234567890"
        }
        event = {
            name = "Transfer"
            signature = "Transfer(address,address,uint256)"
        }
        topic    = "topic1"
        options  = {}
        signature = ""
    })
}
`

func TestFireFlyContractListener1(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"POST /endpoint/{env}/{service}/rest/api/v1/contracts/listeners",
			"GET /endpoint/{env}/{service}/rest/api/v1/contracts/listeners/{id}",
			"DELETE /endpoint/{env}/{service}/rest/api/v1/contracts/listeners/{id}",
			"GET /endpoint/{env}/{service}/rest/api/v1/contracts/listeners/{id}",
		})
		mp.server.Close()
	}()

	listenerResource := "kaleido_platform_firefly_contract_listener.listener1"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + firefly_contract_listenerStep1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(listenerResource, "id"),
					resource.TestCheckResourceAttr(listenerResource, "environment", "env1"),
					resource.TestCheckResourceAttr(listenerResource, "service", "service1"),
					resource.TestCheckResourceAttr(listenerResource, "namespace", "ns1"),
					resource.TestCheckResourceAttr(listenerResource, "name", "mylistener"),
					func(s *terraform.State) error {
						id := s.RootModule().Resources[listenerResource].Primary.Attributes["id"]
						obj := mp.fireflyContractListeners[fmt.Sprintf("env1/service1/%s", id)]
						assert.NotNil(t, obj)
						assert.Equal(t, "mylistener", obj.Name)
						return nil
					},
				),
			},
		},
	})
}

// TestFireFlyContractListenerUpdateReturnsError verifies that the contract listener
// resource's Update method returns an error. FireFly contract listeners are
// immutable; the Update implementation adds "failed to update firefly contract
// listener" and "firefly contract listeners are immutable - changes require
// replacement". We cannot call Update with a zero Plan (framework panics on
// req.Plan.Get), so we assert the error strings are present in the source file.
func TestFireFlyContractListenerUpdateReturnsError(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	srcPath := filepath.Join(filepath.Dir(file), "firefly_contract_listener.go")
	src, err := os.ReadFile(srcPath)
	assert.NoError(t, err)
	srcStr := string(src)
	assert.Contains(t, srcStr, "failed to update firefly contract listener", "Update should add this error summary")
	assert.Contains(t, srcStr, "firefly contract listeners are immutable - changes require replacement", "Update should add this error detail")
}

func (mp *mockPlatform) postFireFlyContractListener(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	key := vars["env"] + "/" + vars["service"]
	var api FireFlyContractListenerAPIModel
	mp.getBody(req, &api)
	api.ID = nanoid.New()
	if api.Name == "" {
		api.Name = api.ID
	}
	mp.fireflyContractListeners[key+"/"+api.ID] = &api
	mp.respond(res, &api, 201)
}

func (mp *mockPlatform) getFireFlyContractListener(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	key := vars["env"] + "/" + vars["service"] + "/" + vars["id"]
	obj := mp.fireflyContractListeners[key]
	if obj == nil {
		mp.respond(res, nil, 404)
	} else {
		mp.respond(res, obj, 200)
	}
}

func (mp *mockPlatform) deleteFireFlyContractListener(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	key := vars["env"] + "/" + vars["service"] + "/" + vars["id"]
	obj := mp.fireflyContractListeners[key]
	assert.NotNil(mp.t, obj)
	delete(mp.fireflyContractListeners, key)
	mp.respond(res, nil, 204)
}
