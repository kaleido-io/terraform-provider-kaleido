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
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/aidarkhanov/nanoid"
	"github.com/gorilla/mux"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/assert"
)

var cms_buildStep1 = `
resource "kaleido_platform_cms_build" "cms_build1" {
    environment = "env1"
	service = "service1"
    type = "github"
    name = "build1"
    path = "some/path"
	github = {
		contract_url = "https://github.com/hyperledger/firefly/blob/main/smart_contracts/ethereum/solidity_firefly/contracts/Firefly.sol"
		contract_name = "Firefly"
		auth_token = "token12345"
	}
}
`

var cms_buildStep2 = `
resource "kaleido_platform_cms_build" "cms_build1" {
    environment = "env1"
	service = "service1"
    type = "github"
    name = "build1"
	description = "shiny contract that does things and stuff"
    path = "some/new/path"
	github = {
		contract_url = "https://github.com/hyperledger/firefly/blob/main/smart_contracts/ethereum/solidity_firefly/contracts/Firefly.sol"
		contract_name = "Firefly"
		auth_token = "token12345"
	}
}
`

func TestCMSBuild1(t *testing.T) {

	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"POST /endpoint/{env}/{service}/rest/api/v1/builds",
			"GET /endpoint/{env}/{service}/rest/api/v1/builds/{build}",
			"GET /endpoint/{env}/{service}/rest/api/v1/builds/{build}",
			"GET /endpoint/{env}/{service}/rest/api/v1/builds/{build}",
			"GET /endpoint/{env}/{service}/rest/api/v1/builds/{build}",
			"PATCH /endpoint/{env}/{service}/rest/api/v1/builds/{build}",
			"GET /endpoint/{env}/{service}/rest/api/v1/builds/{build}",
			"DELETE /endpoint/{env}/{service}/rest/api/v1/builds/{build}",
			"GET /endpoint/{env}/{service}/rest/api/v1/builds/{build}",
		})
		mp.server.Close()
	}()

	cms_build1Resource := "kaleido_platform_cms_build.cms_build1"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + cms_buildStep1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(cms_build1Resource, "id"),
					resource.TestCheckResourceAttr(cms_build1Resource, "name", `build1`),
					resource.TestCheckResourceAttr(cms_build1Resource, "type", `github`),
					resource.TestCheckResourceAttrSet(cms_build1Resource, "abi"),
					resource.TestCheckResourceAttrSet(cms_build1Resource, "bytecode"),
					resource.TestCheckResourceAttrSet(cms_build1Resource, "dev_docs"),
					resource.TestCheckResourceAttrSet(cms_build1Resource, "commit_hash"),
				),
			},
			{
				Config: providerConfig + cms_buildStep2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(cms_build1Resource, "id"),
					resource.TestCheckResourceAttr(cms_build1Resource, "name", `build1`),
					resource.TestCheckResourceAttr(cms_build1Resource, "type", `github`),
					resource.TestCheckResourceAttrSet(cms_build1Resource, "abi"),
					resource.TestCheckResourceAttrSet(cms_build1Resource, "bytecode"),
					resource.TestCheckResourceAttrSet(cms_build1Resource, "dev_docs"),
					resource.TestCheckResourceAttrSet(cms_build1Resource, "commit_hash"),
					func(s *terraform.State) error {
						// Compare the final result on the mock-server side
						id := s.RootModule().Resources[cms_build1Resource].Primary.Attributes["id"]
						obj := mp.cmsBuilds[fmt.Sprintf("env1/service1/%s", id)]
						testJSONEqual(t, obj, fmt.Sprintf(`
						{
							"id": "%[1]s",
							"created": "%[2]s",
							"updated": "%[3]s",
							"name": "build1",
							"path": "some/new/path",
							"description": "shiny contract that does things and stuff",
							"github": {
								"contractUrl": "https://github.com/hyperledger/firefly/blob/main/smart_contracts/ethereum/solidity_firefly/contracts/Firefly.sol",
								"contractName": "Firefly",
								"oauthToken": "token12345",
								"commitHash": "%[4]s"
							},
							"abi": "[{\"some\":\"abi\"}]",
							"bytecode": "0xAAABBBCCCDDD",
							"devDocs": "[\"some\":\"devdocs\"]",
							"status": "succeeded"
						}
						`,
							// generated fields that vary per test run
							id,
							obj.Created.UTC().Format(time.RFC3339Nano),
							obj.Updated.UTC().Format(time.RFC3339Nano),
							obj.GitHub.CommitHash,
						))
						return nil
					},
				),
			},
		},
	})
}

var cms_buildPrecompiled = `
resource "kaleido_platform_cms_build" "cms_build_precompiled" {
    environment = "env1"
	  service = "service1"
    type = "precompiled"
    name = "build2"
    path = "some/path"
	  precompiled = {
    bytecode = "0xB17EC0DE"
    abi = "[{\"some\":\"precompiled_abi\"}]"
	}
}
`

func TestCMSBuildPreCompiled(t *testing.T) {

	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"POST /endpoint/{env}/{service}/rest/api/v1/builds",
			"GET /endpoint/{env}/{service}/rest/api/v1/builds/{build}",
			"GET /endpoint/{env}/{service}/rest/api/v1/builds/{build}",
			"GET /endpoint/{env}/{service}/rest/api/v1/builds/{build}",
			"DELETE /endpoint/{env}/{service}/rest/api/v1/builds/{build}",
			"GET /endpoint/{env}/{service}/rest/api/v1/builds/{build}",
		})
		mp.server.Close()
	}()

	cms_buildPreCompiledResource := "kaleido_platform_cms_build.cms_build_precompiled"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + cms_buildPrecompiled,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(cms_buildPreCompiledResource, "id"),
					resource.TestCheckResourceAttr(cms_buildPreCompiledResource, "name", `build2`),
					resource.TestCheckResourceAttr(cms_buildPreCompiledResource, "type", `precompiled`),
					resource.TestCheckResourceAttrSet(cms_buildPreCompiledResource, "abi"),
					resource.TestCheckResourceAttrSet(cms_buildPreCompiledResource, "bytecode"),
					resource.TestCheckResourceAttrSet(cms_buildPreCompiledResource, "dev_docs"),
					func(s *terraform.State) error {
						// Compare the final result on the mock-server side
						assert.NotNil(t, s.RootModule().Resources[cms_buildPreCompiledResource])
						id := s.RootModule().Resources[cms_buildPreCompiledResource].Primary.Attributes["id"]
						obj := mp.cmsBuilds[fmt.Sprintf("env1/service1/%s", id)]
						testJSONEqual(t, obj, fmt.Sprintf(`
						{
							"id": "%[1]s",
							"created": "%[2]s",
							"updated": "%[3]s",
							"name": "build2",
							"path": "some/path",
              "abi": [{"some":"precompiled_abi"}],
              "bytecode": "0xB17EC0DE",
							"devDocs": "[\"some\":\"devdocs\"]",
							"status": "succeeded"
						}
						`,
							// generated fields that vary per test run
							id,
							obj.Created.UTC().Format(time.RFC3339Nano),
							obj.Updated.UTC().Format(time.RFC3339Nano),
						))
						return nil
					},
				),
			},
		},
	})
}

func (mp *mockPlatform) getCMSBuild(res http.ResponseWriter, req *http.Request) {
	obj := mp.cmsBuilds[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["build"]]
	if obj == nil {
		mp.respond(res, nil, 404)
	} else {
		mp.respond(res, obj, 200)
		// Next time we'll complete the build
		obj.Status = "succeeded"

		if obj.ABI == nil {
			obj.ABI = `[{"some":"abi"}]`
		}
		if obj.Bytecode == "" {
			obj.Bytecode = `0xAAABBBCCCDDD`
		}
		obj.DevDocs = `["some":"devdocs"]`
		if obj.GitHub != nil {
			obj.GitHub.CommitHash = nanoid.New()
		}
	}
}

func (mp *mockPlatform) postCMSBuild(res http.ResponseWriter, req *http.Request) {
	var obj CMSBuildAPIModel
	mp.getBody(req, &obj)
	obj.ID = nanoid.New()
	now := time.Now().UTC()
	obj.Created = &now
	obj.Updated = &now
	mp.cmsBuilds[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+obj.ID] = &obj
	mp.respond(res, &obj, 201)
}

func (mp *mockPlatform) patchCMSBuild(res http.ResponseWriter, req *http.Request) {
	obj := mp.cmsBuilds[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["build"]] // expected behavior of provider is PUT only on exists
	assert.NotNil(mp.t, obj)
	var updates CMSBuildAPIModel
	mp.getBody(req, &updates)
	assert.Empty(mp.t, updates.ID)
	now := time.Now().UTC()
	obj.Updated = &now
	obj.Name = updates.Name
	obj.Path = updates.Path
	obj.Description = updates.Description
	mp.respond(res, &obj, 200)
}

func (mp *mockPlatform) deleteCMSBuild(res http.ResponseWriter, req *http.Request) {
	obj := mp.cmsBuilds[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["build"]]
	assert.NotNil(mp.t, obj)
	delete(mp.cmsBuilds, mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["build"])
	mp.respond(res, nil, 204)
}
