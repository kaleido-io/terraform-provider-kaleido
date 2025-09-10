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

	_ "embed"
)

var pms_identity_list_step1 = `

resource "kaleido_platform_pms_identity_list" "test_identity_list" {
  environment = "test-env"
  service = "test-service"
  name = "test-identity-list"
  description = "Test identity list for policy management"
  identities = [
    "pmi:12345abcde",
    "pmi:67890fghij",
    "pmi:12345fghij"
  ]
}
`

var pms_identity_list_step2 = `

resource "kaleido_platform_pms_identity_list" "test_identity_list" {
  environment = "test-env"
  service = "test-service"
  name = "test-identity-list"
  description = "Test identity list for policy management"
  identities = [
    "pmi:abcde12345",
    "pmi:fghij67890",
    "pmi:fghij12345"
  ]
}
`

func TestPMSIdentityList1(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"PUT /endpoint/{env}/{service}/rest/api/v1/identity-lists/{identityList}",
			"POST /endpoint/{env}/{service}/rest/api/v1/identity-lists/{identityList}/versions",
			"GET /endpoint/{env}/{service}/rest/api/v1/identity-lists/{identityList}",
			"GET /endpoint/{env}/{service}/rest/api/v1/identity-lists/{identityList}",
			"GET /endpoint/{env}/{service}/rest/api/v1/identity-lists/{identityList}",
			"PATCH /endpoint/{env}/{service}/rest/api/v1/identity-lists/{identityList}",
			"POST /endpoint/{env}/{service}/rest/api/v1/identity-lists/{identityList}/versions",
			"GET /endpoint/{env}/{service}/rest/api/v1/identity-lists/{identityList}",
			"GET /endpoint/{env}/{service}/rest/api/v1/identity-lists/{identityList}",
			"DELETE /endpoint/{env}/{service}/rest/api/v1/identity-lists/{identityList}",
			"GET /endpoint/{env}/{service}/rest/api/v1/identity-lists/{identityList}",
		})
		mp.server.Close()
	}()

	pms_identity_list_resource := "kaleido_platform_pms_identity_list.test_identity_list"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + pms_identity_list_step1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(pms_identity_list_resource, "id"),
					resource.TestCheckResourceAttr(pms_identity_list_resource, "environment", "test-env"),
					resource.TestCheckResourceAttr(pms_identity_list_resource, "service", "test-service"),
					resource.TestCheckResourceAttr(pms_identity_list_resource, "name", "test-identity-list"),
					resource.TestCheckResourceAttr(pms_identity_list_resource, "description", "Test identity list for policy management"),
					resource.TestCheckResourceAttr(pms_identity_list_resource, "identities.0", "pmi:12345abcde"),
					resource.TestCheckResourceAttr(pms_identity_list_resource, "identities.1", "pmi:67890fghij"),
					resource.TestCheckResourceAttr(pms_identity_list_resource, "identities.2", "pmi:12345fghij"),
					resource.TestCheckResourceAttrSet(pms_identity_list_resource, "applied_version"),
					resource.TestCheckResourceAttrSet(pms_identity_list_resource, "created"),
					resource.TestCheckResourceAttrSet(pms_identity_list_resource, "updated"),
				),
			},
			{
				Config: providerConfig + pms_identity_list_step2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(pms_identity_list_resource, "id"),
					resource.TestCheckResourceAttr(pms_identity_list_resource, "environment", "test-env"),
					resource.TestCheckResourceAttr(pms_identity_list_resource, "service", "test-service"),
					resource.TestCheckResourceAttr(pms_identity_list_resource, "name", "test-identity-list"),
					resource.TestCheckResourceAttr(pms_identity_list_resource, "description", "Test identity list for policy management"),
					resource.TestCheckResourceAttr(pms_identity_list_resource, "identities.0", "pmi:abcde12345"),
					resource.TestCheckResourceAttr(pms_identity_list_resource, "identities.1", "pmi:fghij67890"),
					resource.TestCheckResourceAttr(pms_identity_list_resource, "identities.2", "pmi:fghij12345"),
					resource.TestCheckResourceAttrSet(pms_identity_list_resource, "applied_version"),
					resource.TestCheckResourceAttrSet(pms_identity_list_resource, "created"),
					resource.TestCheckResourceAttrSet(pms_identity_list_resource, "updated"),
				),
			},
		},
	})
}

// PMS Identity List handlers
func (mp *mockPlatform) putPMSIdentityList(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	identityListName := vars["identityList"]
	var identityList PMSIdentityListAPIModel
	mp.getBody(req, &identityList)
	identityList.ID = nanoid.New()
	now := time.Now().UTC()
	identityList.Created = &now
	identityList.Updated = &now
	// Store by both ID and name for lookup
	mp.pmsIdentityLists[identityList.ID] = &identityList
	mp.pmsIdentityLists[identityListName] = &identityList
	mp.respond(res, &identityList, http.StatusOK)
}

func (mp *mockPlatform) getPMSIdentityList(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	identityListID := vars["identityList"]
	identityList, exists := mp.pmsIdentityLists[identityListID]
	if !exists {
		mp.respond(res, nil, http.StatusNotFound)
		return
	}
	identityListVersions, versionsExist := mp.pmsIdentityListVersions[identityListID]
	if versionsExist {
		identityListVersion, versionExists := identityListVersions[identityList.CurrentVersion]
		if versionExists {
			identityList.Identities = identityListVersion.Identities
		}
	}
	mp.respond(res, identityList, http.StatusOK)
}

func (mp *mockPlatform) patchPMSIdentityList(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	identityListID := vars["identityList"]
	identityList, exists := mp.pmsIdentityLists[identityListID]
	if !exists {
		mp.respond(res, nil, http.StatusNotFound)
		return
	}
	var updates PMSIdentityListAPIModel
	mp.getBody(req, &updates)
	if updates.Description != "" {
		identityList.Description = updates.Description
	}
	now := time.Now().UTC()
	identityList.Updated = &now
	// Update both ID and name entries
	mp.pmsIdentityLists[identityList.ID] = identityList
	mp.pmsIdentityLists[identityListID] = identityList
	mp.respond(res, identityList, http.StatusOK)
}

func (mp *mockPlatform) deletePMSIdentityList(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	identityListID := vars["identityList"]
	identityList, exists := mp.pmsIdentityLists[identityListID]
	if exists {
		// Delete both ID and name entries
		delete(mp.pmsIdentityLists, identityList.ID)
		delete(mp.pmsIdentityListVersions, identityList.ID)
	}
	delete(mp.pmsIdentityLists, identityListID)
	delete(mp.pmsIdentityListVersions, identityListID)
	mp.respond(res, nil, http.StatusNoContent)
}

func (mp *mockPlatform) postPMSIdentityListVersion(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	identityListID := vars["identityList"]
	var version PMSIdentityListVersionAPIModel
	mp.getBody(req, &version)
	version.ID = nanoid.New()
	version.IdentityListID = identityListID
	now := time.Now().UTC()
	version.Created = &now
	version.Updated = &now

	// Initialize versions map if it doesn't exist
	if mp.pmsIdentityListVersions[identityListID] == nil {
		mp.pmsIdentityListVersions[identityListID] = make(map[string]*PMSIdentityListVersionAPIModel)
	}
	mp.pmsIdentityListVersions[identityListID][version.ID] = &version

	// Update the identity list's current version
	if identityList, exists := mp.pmsIdentityLists[identityListID]; exists {
		identityList.CurrentVersion = version.ID
		// Update both ID and name entries
		mp.pmsIdentityLists[identityList.ID] = identityList
		mp.pmsIdentityLists[identityListID] = identityList
	}

	mp.respond(res, &version, http.StatusOK)
}
