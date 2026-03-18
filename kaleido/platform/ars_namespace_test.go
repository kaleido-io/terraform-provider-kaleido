// Copyright © Kaleido, Inc. 2024-2025

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
	"testing"

	"github.com/aidarkhanov/nanoid"
	"github.com/gorilla/mux"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/assert"
)

var arsNamespaceStep1 = `
resource "kaleido_platform_ars_namespace" "ns1" {
  environment = "env1"
  service     = "svc1"
  name        = "ns1"
  auto_create_repos = true
  allowed_types = [
    "application/vnd.docker.distribution.manifest.v2+json",
    "application/vnd.kaleido.json-schema.v1+json"
  ]
  description = "stuff"
}
`

func TestARSNamespace(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.server.Close()
	}()

	nsResource := "kaleido_platform_ars_namespace.ns1"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + arsNamespaceStep1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(nsResource, "id"),
					resource.TestCheckResourceAttr(nsResource, "name", "ns1"),
					resource.TestCheckResourceAttr(nsResource, "environment", "env1"),
					resource.TestCheckResourceAttr(nsResource, "service", "svc1"),
					resource.TestCheckResourceAttr(nsResource, "description", "stuff"),
					resource.TestCheckResourceAttr(nsResource, "auto_create_repos", "true"),
					resource.TestCheckResourceAttr(nsResource, "allowed_types.#", "2"),
					resource.TestCheckResourceAttr(nsResource, "allowed_types.0", "application/vnd.docker.distribution.manifest.v2+json"),
					resource.TestCheckResourceAttr(nsResource, "allowed_types.1", "application/vnd.kaleido.json-schema.v1+json"),
					func(s *terraform.State) error {
						id := s.RootModule().Resources[nsResource].Primary.Attributes["id"]
						obj := mp.arsNamespaces[fmt.Sprintf("env1/svc1/%s", id)]
						assert.NotNil(t, obj)
						assert.Equal(t, "ns1", obj.Name)
						assert.Equal(t, true, obj.AutoCreateRepos)
						assert.Equal(t, "stuff", obj.Description)
						assert.Len(t, obj.AllowedTypes, 2)
						return nil
					},
				),
			},
		},
	})
}

func (mp *mockPlatform) getARSNamespace(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	key := vars["env"] + "/" + vars["service"] + "/" + vars["ns"]
	obj := mp.arsNamespaces[key]
	if obj == nil {
		mp.respond(res, nil, 404)
	} else {
		mp.respond(res, obj, 200)
	}
}

func (mp *mockPlatform) postARSNamespace(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	var obj ARSNamespaceAPIModel
	mp.getBody(req, &obj)
	obj.ID = nanoid.New()
	mp.arsNamespaces[vars["env"]+"/"+vars["service"]+"/"+obj.ID] = &obj
	mp.respond(res, &obj, 201)
}

func (mp *mockPlatform) deleteARSNamespace(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	key := vars["env"] + "/" + vars["service"] + "/" + vars["ns"]
	obj := mp.arsNamespaces[key]
	assert.NotNil(mp.t, obj)
	delete(mp.arsNamespaces, key)
	mp.respond(res, nil, 204)
}
