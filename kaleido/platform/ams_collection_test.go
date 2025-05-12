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
	_ "embed"
	"net/http"
	"testing"
	"time"

	"github.com/aidarkhanov/nanoid"
	"github.com/gorilla/mux"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"
)

var ams_collectionStep1 = `
  resource "kaleido_platform_ams_collection" "ams_collection1" {
  environment = "env1"
  service = "service1"
  name = "my_collection"
  display_name = "my_collection"
}
`

var ams_collectionUpdateDescription = `
  resource "kaleido_platform_ams_collection" "ams_collection1" {
  environment = "env1"
  service = "service1"
  name = "my_collection"
  display_name = "my_collection"
  description = "this is my updated description"
}
`

func TestAMSCollection1(t *testing.T) {

	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"PUT /endpoint/{env}/{service}/rest/api/v1/collections/{collection}", // by name initially
			"GET /endpoint/{env}/{service}/rest/api/v1/collections/{collection}",
			"GET /endpoint/{env}/{service}/rest/api/v1/collections/{collection}",
			"PUT /endpoint/{env}/{service}/rest/api/v1/collections/{collection}", // by name initially
			"GET /endpoint/{env}/{service}/rest/api/v1/collections/{collection}",
			"DELETE /endpoint/{env}/{service}/rest/api/v1/collections/{collection}",
			"GET /endpoint/{env}/{service}/rest/api/v1/collections/{collection}",
		})
		mp.server.Close()
	}()

	ams_collection1Resource := "kaleido_platform_ams_collection.ams_collection1"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + ams_collectionStep1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(ams_collection1Resource, "id"),
					resource.TestCheckResourceAttr(ams_collection1Resource, "display_name", "my_collection"),
					resource.TestCheckNoResourceAttr(ams_collection1Resource, "description"),
				),
			},
			{
				Config: providerConfig + ams_collectionUpdateDescription,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(ams_collection1Resource, "id"),
					resource.TestCheckResourceAttr(ams_collection1Resource, "display_name", "my_collection"),
					resource.TestCheckResourceAttr(ams_collection1Resource, "description", "this is my updated description"),
				),
			},
		},
	})
}

func (mp *mockPlatform) getAMSCollection(res http.ResponseWriter, req *http.Request) {
	obj := mp.amsCollections[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["collection"]]
	if obj == nil {
		mp.respond(res, nil, 404)
	} else {
		mp.respond(res, obj, 200)
	}
}

func (mp *mockPlatform) putAMSCollection(res http.ResponseWriter, req *http.Request) {
	now := time.Now().UTC()
	obj := mp.amsCollections[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["collection"]]
	var newObj AMSCollectionAPIModel
	mp.getBody(req, &newObj)
	if obj == nil {
		assert.Equal(mp.t, newObj.Name, mux.Vars(req)["collection"])
		newObj.ID = nanoid.New()
		newObj.Created = &now
	} else {
		assert.Equal(mp.t, obj.ID, mux.Vars(req)["collection"])
		newObj.ID = mux.Vars(req)["collection"]
		newObj.Created = obj.Created
	}
	newObj.Updated = &now
	mp.amsCollections[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+newObj.ID] = &newObj
	mp.respond(res, &newObj, 200)
}

func (mp *mockPlatform) deleteAMSCollection(res http.ResponseWriter, req *http.Request) {
	obj := mp.amsCollections[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["collection"]]
	assert.NotNil(mp.t, obj)
	delete(mp.amsCollections, mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["collection"])
	mp.respond(res, nil, 204)
}
