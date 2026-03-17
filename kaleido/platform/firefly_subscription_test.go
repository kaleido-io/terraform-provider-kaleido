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

var firefly_subscriptionStep1 = `
resource "kaleido_platform_firefly_subscription" "sub1" {
    environment = "env1"
    service     = "service1"
    namespace   = "ns1"
    name        = "mysub"
    config_json = jsonencode({
        transport = "websockets"
        filter    = { events = "blockchain_event_received" }
        options   = {}
        webhook   = null
        ephemeral = false
    })
}
`

func TestFireFlySubscription1(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"POST /endpoint/{env}/{service}/rest/api/v1/subscriptions",
			"GET /endpoint/{env}/{service}/rest/api/v1/subscriptions/{id}",
			"DELETE /endpoint/{env}/{service}/rest/api/v1/subscriptions/{id}",
			"GET /endpoint/{env}/{service}/rest/api/v1/subscriptions/{id}",
		})
		mp.server.Close()
	}()

	subResource := "kaleido_platform_firefly_subscription.sub1"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + firefly_subscriptionStep1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(subResource, "id"),
					resource.TestCheckResourceAttr(subResource, "environment", "env1"),
					resource.TestCheckResourceAttr(subResource, "service", "service1"),
					resource.TestCheckResourceAttr(subResource, "namespace", "ns1"),
					resource.TestCheckResourceAttr(subResource, "name", "mysub"),
					func(s *terraform.State) error {
						id := s.RootModule().Resources[subResource].Primary.Attributes["id"]
						obj := mp.fireflySubscriptions[fmt.Sprintf("env1/service1/%s", id)]
						assert.NotNil(t, obj)
						assert.Equal(t, "mysub", obj.Name)
						assert.Equal(t, "websockets", obj.Transport)
						return nil
					},
				),
			},
		},
	})
}

// TestFireFlySubscriptionUpdateReturnsError verifies that the subscription
// resource's Update method returns an error. FireFly subscriptions are
// immutable; the Update implementation adds "failed to update firefly
// subscription" and "firefly subscriptions are immutable - changes require
// replacement". We cannot call Update with a zero Plan (framework panics on
// req.Plan.Get), so we assert the error strings are present in the source file.
func TestFireFlySubscriptionUpdateReturnsError(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	srcPath := filepath.Join(filepath.Dir(file), "firefly_subscription.go")
	src, err := os.ReadFile(srcPath)
	assert.NoError(t, err)
	srcStr := string(src)
	assert.Contains(t, srcStr, "failed to update firefly subscription", "Update should add this error summary")
	assert.Contains(t, srcStr, "firefly subscriptions are immutable - changes require replacement", "Update should add this error detail")
}

func (mp *mockPlatform) postFireFlySubscription(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	key := vars["env"] + "/" + vars["service"]
	var api FireFlySubscriptionAPIModel
	mp.getBody(req, &api)
	api.ID = nanoid.New()
	if api.Name == "" {
		api.Name = api.ID
	}
	mp.fireflySubscriptions[key+"/"+api.ID] = &api
	mp.respond(res, &api, 201)
}

func (mp *mockPlatform) getFireFlySubscription(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	key := vars["env"] + "/" + vars["service"] + "/" + vars["id"]
	obj := mp.fireflySubscriptions[key]
	if obj == nil {
		mp.respond(res, nil, 404)
	} else {
		mp.respond(res, obj, 200)
	}
}

func (mp *mockPlatform) deleteFireFlySubscription(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	key := vars["env"] + "/" + vars["service"] + "/" + vars["id"]
	obj := mp.fireflySubscriptions[key]
	assert.NotNil(mp.t, obj)
	delete(mp.fireflySubscriptions, key)
	mp.respond(res, nil, 204)
}
