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
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/aidarkhanov/nanoid"
	"github.com/gorilla/mux"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/assert"
)

var permissionSetStep1 = `
resource "kaleido_platform_permission_set" "desk" {
	name        = "desk-trader"
	description = "Trade on one desk"
	permissions_json = jsonencode([{
		actions = ["read", "write"]
		resourceSelector = [
			{ field = "resource.type", op = "eq", values = ["subledger.accounting.account"] },
			{ field = "resource.within.book.labels.desk", op = "eq", param = "desk" },
		]
	}])
	params_json = jsonencode([{ name = "desk", description = "The desk" }])
	labels = { team = "fx" }
}
`

// Step 2 changes the permissions, and drops the description and the labels.
var permissionSetStep2 = `
resource "kaleido_platform_permission_set" "desk" {
	name = "desk-trader"
	permissions_json = jsonencode([{
		actions = ["read", "write", "configure"]
		resourceSelector = [
			{ field = "resource.type", op = "eq", values = ["subledger.accounting.account"] },
			{ field = "resource.within.book.labels.desk", op = "eq", param = "desk" },
		]
	}])
	params_json = jsonencode([{ name = "desk", description = "The desk" }])
}
`

func TestPermissionSet1(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"POST /api/v1/permissionsets",
			"GET /api/v1/permissionsets/{set}",
			"GET /api/v1/permissionsets/{set}",
			"PATCH /api/v1/permissionsets/{set}",
			"GET /api/v1/permissionsets/{set}",
			"DELETE /api/v1/permissionsets/{set}",
			"GET /api/v1/permissionsets/{set}",
		})
		mp.server.Close()
	}()

	desk := "kaleido_platform_permission_set.desk"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + permissionSetStep1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(desk, "id"),
					resource.TestCheckResourceAttr(desk, "labels.team", "fx"),
				),
			},
			{
				Config: providerConfig + permissionSetStep2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr(desk, "description"),
					resource.TestCheckNoResourceAttr(desk, "labels.team"),
					func(s *terraform.State) error {
						id := s.RootModule().Resources[desk].Primary.Attributes["id"]
						set := mp.permissionSets[id]
						assert.Equal(t, "", *set.Description)
						assert.Empty(t, set.Labels)
						testJSONEqual(t, json.RawMessage(set.Permissions), `[{
							"actions": ["read", "write", "configure"],
							"resourceSelector": [
								{"field": "resource.type", "op": "eq", "values": ["subledger.accounting.account"]},
								{"field": "resource.within.book.labels.desk", "op": "eq", "param": "desk"}
							]
						}]`)
						return nil
					},
				),
			},
		},
	})
}

// mockBuiltinPermissionSets are the built-in sets the mock serves: read-only, found by name.
func mockBuiltinPermissionSets() map[string]*PermissionSetAPIModel {
	name, builtin := "full-access", true
	return map[string]*PermissionSetAPIModel{
		"ps:fullaccess0": {
			ID:          "ps:fullaccess0",
			Name:        &name,
			Builtin:     &builtin,
			Permissions: json.RawMessage(`[{"actions":["*"],"resourceSelector":[{"field":"resource.within.service.id","op":"eq","param":"service"}]}]`),
			Params:      json.RawMessage(`[{"name":"service"}]`),
		},
	}
}

// mockNormalizeJSON re-encodes JSON as a server may store it: keys sorted, no whitespace. The
// provider must see no drift from that.
func mockNormalizeJSON(raw json.RawMessage, empty string) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage(empty)
	}
	var v interface{}
	if err := json.Unmarshal(raw, &v); err != nil {
		return raw
	}
	b, _ := json.Marshal(v)
	return b
}

func (mp *mockPlatform) findPermissionSet(nameOrID string) *PermissionSetAPIModel {
	if set := mp.permissionSets[nameOrID]; set != nil {
		return set
	}
	for _, set := range mp.permissionSets {
		if set.Name != nil && *set.Name == nameOrID && (set.Inline == nil || !*set.Inline) {
			return set
		}
	}
	return nil
}

func (mp *mockPlatform) newPermissionSet(set *PermissionSetAPIModel) *PermissionSetAPIModel {
	set.ID = "ps:" + nanoid.New()
	now := time.Now().UTC()
	set.Created = &now
	set.Permissions = mockNormalizeJSON(set.Permissions, `[]`)
	set.Params = mockNormalizeJSON(set.Params, `[]`)
	mp.permissionSets[set.ID] = set
	return set
}

func (mp *mockPlatform) postPermissionSet(res http.ResponseWriter, req *http.Request) {
	var set PermissionSetAPIModel
	mp.getBody(req, &set)
	mp.respond(res, mp.newPermissionSet(&set), 201)
}

func (mp *mockPlatform) getPermissionSet(res http.ResponseWriter, req *http.Request) {
	set := mp.findPermissionSet(mux.Vars(req)["set"])
	if set == nil {
		mp.respond(res, nil, 404)
		return
	}
	mp.respond(res, set, 200)
}

// patchPermissionSet merges what it's sent, as Access Manager does.
func (mp *mockPlatform) patchPermissionSet(res http.ResponseWriter, req *http.Request) {
	set := mp.findPermissionSet(mux.Vars(req)["set"])
	if set == nil {
		mp.respond(res, nil, 404)
		return
	}
	var patch PermissionSetAPIModel
	mp.getBody(req, &patch)
	if patch.Name != nil {
		set.Name = patch.Name
	}
	if patch.Description != nil {
		set.Description = patch.Description
	}
	if patch.Permissions != nil {
		set.Permissions = mockNormalizeJSON(patch.Permissions, `[]`)
	}
	if patch.Params != nil {
		set.Params = mockNormalizeJSON(patch.Params, `[]`)
	}
	if patch.Labels != nil {
		set.Labels = patch.Labels
	}
	now := time.Now().UTC()
	set.Updated = &now
	mp.respond(res, set, 200)
}

func (mp *mockPlatform) deletePermissionSet(res http.ResponseWriter, req *http.Request) {
	set := mp.findPermissionSet(mux.Vars(req)["set"])
	assert.NotNil(mp.t, set)
	delete(mp.permissionSets, set.ID)
	mp.respond(res, nil, 204)
}
