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
	"regexp"
	"testing"
	"time"

	"github.com/aidarkhanov/nanoid"
	"github.com/gorilla/mux"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/assert"
)

// A set with a param, granted to a group; an inline set granted to an application, whose grant
// gives nothing yet; and a built-in set read by name, granted with its param.
var grantStep1 = permissionSetStep1 + `
resource "kaleido_platform_grant" "fx" {
	name              = "fx-desk"
	group_id          = "g:fx"
	permission_set_id = kaleido_platform_permission_set.desk.id
	params_json       = jsonencode({ desk = "fx" })
	labels            = { owner = "ops" }
}

resource "kaleido_platform_grant" "inline" {
	name                = "dead-end"
	application_id      = "ap:1234"
	permission_set_json = jsonencode({ permissions = [{ actions = ["read"] }] })
}

data "kaleido_platform_permission_set" "full" {
	name = "full-access"
}

resource "kaleido_platform_grant" "full" {
	name              = "km1-full"
	group_id          = "g:fx"
	permission_set_id = data.kaleido_platform_permission_set.full.id
	params_json       = jsonencode({ service = "s:km1" })
}
`

// Step 2 gives the fx grant a list for its param, and drops its labels.
var grantStep2 = permissionSetStep1 + `
resource "kaleido_platform_grant" "fx" {
	name              = "fx-desk"
	group_id          = "g:fx"
	permission_set_id = kaleido_platform_permission_set.desk.id
	params_json       = jsonencode({ desk = ["fx", "rates"] })
}

resource "kaleido_platform_grant" "inline" {
	name                = "dead-end"
	application_id      = "ap:1234"
	permission_set_json = jsonencode({ permissions = [{ actions = ["read"] }] })
}

data "kaleido_platform_permission_set" "full" {
	name = "full-access"
}

resource "kaleido_platform_grant" "full" {
	name              = "km1-full"
	group_id          = "g:fx"
	permission_set_id = data.kaleido_platform_permission_set.full.id
	params_json       = jsonencode({ service = "s:km1" })
}
`

func TestGrant1(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.server.Close()
	}()

	fx := "kaleido_platform_grant.fx"
	inline := "kaleido_platform_grant.inline"
	full := "kaleido_platform_grant.full"
	fullSet := "data.kaleido_platform_permission_set.full"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + grantStep1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(fx, "id"),
					resource.TestCheckResourceAttrPair(fx, "permission_set_id", "kaleido_platform_permission_set.desk", "id"),
					resource.TestCheckResourceAttrSet(fx, "effective_permissions_json"),
					resource.TestCheckResourceAttr(fx, "warnings.#", "0"),
					resource.TestMatchResourceAttr(inline, "permission_set_id", regexp.MustCompile(`^ps:`)),
					resource.TestCheckResourceAttr(inline, "warnings.#", "1"),
					resource.TestCheckResourceAttr(fullSet, "id", "ps:fullaccess0"),
					resource.TestCheckResourceAttr(fullSet, "builtin", "true"),
					resource.TestCheckResourceAttr(full, "permission_set_id", "ps:fullaccess0"),
				),
			},
			{
				Config: providerConfig + grantStep2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr(fx, "labels.owner"),
					func(s *terraform.State) error {
						id := s.RootModule().Resources[fx].Primary.Attributes["id"]
						g := mp.grants[id]
						testJSONEqual(t, json.RawMessage(g.Params), `{"desk": ["fx", "rates"]}`)
						assert.Empty(t, g.Labels)
						return nil
					},
				),
			},
		},
	})

	// Everything is gone, the grants' inline set with them; the built-in set stays.
	assert.Empty(t, mp.grants)
	assert.Len(t, mp.permissionSets, 1)
	assert.Contains(t, mp.permissionSets, "ps:fullaccess0")
}

// mockGrantWarning is the warning the mock gives a grant named dead-end, as Access Manager warns
// of a grant whose permissions give nothing yet.
const mockGrantWarning = "the grant's permissions select nothing in this account yet"

// mockGrantResponse is a grant as the API returns it after a write: with its warnings.
func mockGrantResponse(g *GrantAPIModel) *GrantAPIModel {
	resp := *g
	if g.Name != nil && *g.Name == "dead-end" {
		resp.Warnings = []string{mockGrantWarning}
	}
	return &resp
}

func (mp *mockPlatform) findGrant(nameOrID string) *GrantAPIModel {
	if g := mp.grants[nameOrID]; g != nil {
		return g
	}
	for _, g := range mp.grants {
		if g.Name != nil && *g.Name == nameOrID {
			return g
		}
	}
	return nil
}

func (mp *mockPlatform) postGrant(res http.ResponseWriter, req *http.Request) {
	var g GrantAPIModel
	mp.getBody(req, &g)
	assert.True(mp.t, (g.GroupID == "") != (g.ApplicationID == ""), "one subject")
	assert.True(mp.t, (g.PermissionSetID == "") != (len(g.PermissionSet) == 0), "one set")
	if len(g.PermissionSet) > 0 {
		// An inline set of the grant's own. The API doesn't return it with the grant.
		var inline PermissionSetAPIModel
		assert.NoError(mp.t, json.Unmarshal(g.PermissionSet, &inline))
		isInline := true
		inline.Inline = &isInline
		g.PermissionSetID = mp.newPermissionSet(&inline).ID
		g.PermissionSet = nil
	}
	set := mp.permissionSets[g.PermissionSetID]
	if set == nil {
		mp.respond(res, map[string]string{"error": "permission set not found"}, 400)
		return
	}
	g.ID = "gr:" + nanoid.New()
	now := time.Now().UTC()
	g.Created = &now
	g.Params = mockNormalizeJSON(g.Params, `{}`)
	g.EffectivePermissions = set.Permissions
	mp.grants[g.ID] = &g
	mp.respond(res, mockGrantResponse(&g), 201)
}

func (mp *mockPlatform) getGrant(res http.ResponseWriter, req *http.Request) {
	g := mp.findGrant(mux.Vars(req)["grant"])
	if g == nil {
		mp.respond(res, nil, 404)
		return
	}
	mp.respond(res, g, 200)
}

// patchGrant merges what it's sent: a grant's name, params and labels can change.
func (mp *mockPlatform) patchGrant(res http.ResponseWriter, req *http.Request) {
	g := mp.findGrant(mux.Vars(req)["grant"])
	if g == nil {
		mp.respond(res, nil, 404)
		return
	}
	var patch GrantPatchAPIModel
	mp.getBody(req, &patch)
	if patch.Name != nil {
		g.Name = patch.Name
	}
	if patch.Params != nil {
		g.Params = mockNormalizeJSON(patch.Params, `{}`)
	}
	if patch.Labels != nil {
		g.Labels = patch.Labels
	}
	now := time.Now().UTC()
	g.Updated = &now
	mp.respond(res, mockGrantResponse(g), 200)
}

func (mp *mockPlatform) deleteGrant(res http.ResponseWriter, req *http.Request) {
	g := mp.findGrant(mux.Vars(req)["grant"])
	assert.NotNil(mp.t, g)
	delete(mp.grants, g.ID)
	if set := mp.permissionSets[g.PermissionSetID]; set != nil && set.Inline != nil && *set.Inline {
		delete(mp.permissionSets, set.ID)
	}
	mp.respond(res, nil, 204)
}
