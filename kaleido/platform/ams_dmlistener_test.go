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
	"time"

	"github.com/aidarkhanov/nanoid"
	"github.com/gorilla/mux"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"

	_ "embed"
)

var ams_dmlistenerStep1 = `
resource "kaleido_platform_ams_dmlistener" "ams_dmlistener1" {
    environment = "env1"
	service = "service1"
	name = "listener1"
	task_id = "task1"
	topic_filter = "trigger-.*"
}
`

func TestAMSDMListener1(t *testing.T) {

	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"PUT /endpoint/{env}/{service}/rest/api/v1/listeners/datamodel/{listener}", // by name initially
			"GET /endpoint/{env}/{service}/rest/api/v1/listeners/datamodel/{listener}",
			"DELETE /endpoint/{env}/{service}/rest/api/v1/listeners/datamodel/{listener}",
			"GET /endpoint/{env}/{service}/rest/api/v1/listeners/datamodel/{listener}",
		})
		mp.server.Close()
	}()

	ams_dmlistener1Resource := "kaleido_platform_ams_dmlistener.ams_dmlistener1"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + ams_dmlistenerStep1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(ams_dmlistener1Resource, "id"),
				),
			},
		},
	})
}

func (mp *mockPlatform) getAMSDMListener(res http.ResponseWriter, req *http.Request) {
	obj := mp.amsDMListeners[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["listener"]]
	if obj == nil {
		mp.respond(res, nil, 404)
	} else {
		mp.respond(res, obj, 200)
	}
}

func (mp *mockPlatform) putAMSDMListener(res http.ResponseWriter, req *http.Request) {
	now := time.Now().UTC()
	obj := mp.amsDMListeners[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["listener"]] // expected behavior of provider is PUT only on exists
	var newObj AMSDMListenerAPIModel
	mp.getBody(req, &newObj)
	if obj == nil {
		assert.Equal(mp.t, newObj.Name, mux.Vars(req)["listener"])
		newObj.ID = nanoid.New()
		newObj.Created = now.Format(time.RFC3339Nano)
	} else {
		assert.Equal(mp.t, obj.ID, mux.Vars(req)["listener"])
		newObj.ID = mux.Vars(req)["listener"]
		newObj.Created = obj.Created
	}
	newObj.Updated = now.Format(time.RFC3339Nano)
	mp.amsDMListeners[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+newObj.ID] = &newObj
	mp.respond(res, &newObj, 200)
}

func (mp *mockPlatform) deleteAMSDMListener(res http.ResponseWriter, req *http.Request) {
	obj := mp.amsDMListeners[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["listener"]]
	assert.NotNil(mp.t, obj)
	delete(mp.amsDMListeners, mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["listener"])
	mp.respond(res, nil, 204)
}
