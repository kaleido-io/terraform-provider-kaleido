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
	"testing"
	"time"

	"github.com/aidarkhanov/nanoid"
	"github.com/gorilla/mux"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/assert"
)

var groupMembershipStep1 = `
resource "kaleido_platform_group" "group1" {
  name = "group1"
}

resource "kaleido_platform_user" "user1" {
  name     = "user1"
  email    = "user1@example.com"
  sub      = "user1-sub"
  is_admin = false
}

resource "kaleido_platform_group_membership" "membership1" {
  group_id = kaleido_platform_group.group1.id
  user_id  = kaleido_platform_user.user1.id
}
`

func TestGroupMembershipReadFiltersByUserID(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer mp.server.Close()

	membershipResource := "kaleido_platform_group_membership.membership1"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + groupMembershipStep1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(membershipResource, "id"),
					resource.TestCheckResourceAttrSet(membershipResource, "group_id"),
					resource.TestCheckResourceAttrSet(membershipResource, "user_id"),
				),
			},
			{
				Config:             providerConfig + groupMembershipStep1,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				ResourceName:      membershipResource,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources[membershipResource]
					if !ok {
						return "", fmt.Errorf("resource not found: %s", membershipResource)
					}
					groupID := rs.Primary.Attributes["group_id"]
					userID := rs.Primary.Attributes["user_id"]
					if groupID == "" || userID == "" {
						return "", fmt.Errorf("missing group_id or user_id on %s", membershipResource)
					}
					return groupID + "/" + userID, nil
				},
			},
		},
	})
}

func (mp *mockPlatform) postUser(res http.ResponseWriter, req *http.Request) {
	var rt UserAPIModel
	mp.getBody(req, &rt)
	rt.ID = "u:" + nanoid.New()
	rt.Account = "a:test"
	now := time.Now().UTC()
	rt.Created = &now
	rt.Updated = &now
	mp.users[rt.ID] = &rt
	mp.respond(res, &rt, 201)
}

func (mp *mockPlatform) getUser(res http.ResponseWriter, req *http.Request) {
	rt := mp.users[mux.Vars(req)["user"]]
	if rt == nil {
		mp.respond(res, nil, 404)
		return
	}
	mp.respond(res, rt, 200)
}

func (mp *mockPlatform) patchUser(res http.ResponseWriter, req *http.Request) {
	rt := mp.users[mux.Vars(req)["user"]]
	assert.NotNil(mp.t, rt)
	var patch UserAPIModel
	mp.getBody(req, &patch)
	if patch.Name != "" {
		rt.Name = patch.Name
	}
	if patch.Email != "" {
		rt.Email = patch.Email
	}
	if patch.Sub != "" {
		rt.Sub = patch.Sub
	}
	if patch.IsAdmin != nil {
		rt.IsAdmin = patch.IsAdmin
	}
	now := time.Now().UTC()
	rt.Updated = &now
	mp.respond(res, rt, 200)
}

func (mp *mockPlatform) deleteUser(res http.ResponseWriter, req *http.Request) {
	userID := mux.Vars(req)["user"]
	assert.NotNil(mp.t, mp.users[userID])
	delete(mp.users, userID)
	mp.respond(res, nil, 204)
}

func (mp *mockPlatform) postGroupMember(res http.ResponseWriter, req *http.Request) {
	groupID := mux.Vars(req)["group"]
	group := mp.groups[groupID]
	assert.NotNil(mp.t, group)

	var create GroupMembershipCreateAPIModel
	mp.getBody(req, &create)
	user := mp.users[create.UserID]
	assert.NotNil(mp.t, user)

	now := time.Now().UTC()
	rt := &GroupMembershipAPIModel{
		ID:        "gpm:" + nanoid.New(),
		Created:   &now,
		Updated:   &now,
		GroupID:   groupID,
		UserID:    create.UserID,
		GroupName: group.Name,
		UserName:  user.Name,
		Account:   user.Account,
	}
	mp.groupMembers[groupID] = append(mp.groupMembers[groupID], rt)
	mp.respond(res, rt, 201)
}

func (mp *mockPlatform) listGroupMembers(res http.ResponseWriter, req *http.Request) {
	groupID := mux.Vars(req)["group"]
	userid := req.URL.Query().Get("userid")

	if userid == "" {
		const pageSize = 25
		decoys := make([]GroupMembershipAPIModel, pageSize)
		for i := 0; i < pageSize; i++ {
			decoys[i] = GroupMembershipAPIModel{
				ID:      fmt.Sprintf("gpm:decoy%d", i),
				GroupID: groupID,
				UserID:  fmt.Sprintf("u:decoy%d", i),
			}
		}
		mp.t.Errorf("listGroupMembers called without userid filter; provider Read must filter by userid")
		mp.respond(res, GroupMembershipListAPIModel{Count: pageSize, Items: decoys}, 200)
		return
	}

	var filtered []GroupMembershipAPIModel
	for _, m := range mp.groupMembers[groupID] {
		if m.UserID == userid {
			filtered = append(filtered, *m)
		}
	}
	mp.respond(res, GroupMembershipListAPIModel{Count: len(filtered), Items: filtered}, 200)
}

func (mp *mockPlatform) deleteGroupMember(res http.ResponseWriter, req *http.Request) {
	groupID := mux.Vars(req)["group"]
	memberID := mux.Vars(req)["member"]
	members := mp.groupMembers[groupID]
	for i, m := range members {
		if m.ID == memberID {
			mp.groupMembers[groupID] = append(members[:i], members[i+1:]...)
			mp.respond(res, m, 200)
			return
		}
	}
	mp.respond(res, nil, 404)
}
