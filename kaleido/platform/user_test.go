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
	"context"
	"fmt"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var userWithSubStep = `
resource "kaleido_platform_user" "user1" {
  name     = "preferred-username"
  email    = "user@example.com"
  sub      = "user-subject-identifier"
  is_admin = false
}
`

var userWithSubUpdatedStep = `
resource "kaleido_platform_user" "user1" {
  name     = "preferred-username-renamed"
  email    = "user-renamed@example.com"
  sub      = "user-subject-identifier-2"
  is_admin = true
}
`

var userWithoutSubStep = `
resource "kaleido_platform_user" "user1" {
  name     = "display-name"
  email    = "user-unbound@example.com"
  is_admin = true
}
`

var userWithoutSubRenamedStep = `
resource "kaleido_platform_user" "user1" {
  name     = "display-name-renamed"
  email    = "user-unbound@example.com"
  is_admin = true
}
`

func TestUserCRUD(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer mp.server.Close()

	userResource := "kaleido_platform_user.user1"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + userWithSubStep,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(userResource, "id"),
					resource.TestCheckResourceAttr(userResource, "account", "a:test"),
					resource.TestCheckResourceAttr(userResource, "name", "preferred-username"),
					resource.TestCheckResourceAttr(userResource, "email", "user@example.com"),
					resource.TestCheckResourceAttr(userResource, "sub", "user-subject-identifier"),
					resource.TestCheckResourceAttr(userResource, "is_admin", "false"),
				),
			},
			{
				Config: providerConfig + userWithSubUpdatedStep,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						// account is immutable / UseStateForUnknown — must stay known in the plan
						plancheck.ExpectKnownValue(userResource, tfjsonpath.New("account"), knownvalue.StringExact("a:test")),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(userResource, "account", "a:test"),
					resource.TestCheckResourceAttr(userResource, "name", "preferred-username-renamed"),
					resource.TestCheckResourceAttr(userResource, "email", "user-renamed@example.com"),
					resource.TestCheckResourceAttr(userResource, "sub", "user-subject-identifier-2"),
					resource.TestCheckResourceAttr(userResource, "is_admin", "true"),
					func(s *terraform.State) error {
						id := s.RootModule().Resources[userResource].Primary.Attributes["id"]
						u := mp.users[id]
						require.NotNil(t, u)
						assert.Equal(t, "preferred-username-renamed", u.Name)
						assert.Equal(t, "user-renamed@example.com", u.Email)
						assert.Equal(t, "user-subject-identifier-2", u.Sub)
						require.NotNil(t, u.IsAdmin)
						assert.True(t, *u.IsAdmin)
						return nil
					},
				),
			},
		},
	})
}

func TestUserRetainsPlatformBoundSub(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer mp.server.Close()

	const boundSub = "oidc-sub-bound-1"
	userResource := "kaleido_platform_user.user1"

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + userWithoutSubStep,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(userResource, "id"),
					resource.TestCheckNoResourceAttr(userResource, "sub"),
				),
			},
			{
				// Simulate OIDC login binding a subject outside Terraform
				PreConfig: func() {
					for _, u := range mp.users {
						u.Sub = boundSub
					}
				},
				Config:             providerConfig + userWithoutSubStep,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				// Refresh should retain the platform-bound sub without planning null
				Config: providerConfig + userWithoutSubStep,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(userResource, "sub", boundSub),
					func(s *terraform.State) error {
						id := s.RootModule().Resources[userResource].Primary.Attributes["id"]
						u := mp.users[id]
						require.NotNil(t, u)
						assert.Equal(t, boundSub, u.Sub)
						return nil
					},
				),
			},
		},
	})
}

func TestUserNameUpdateKeepsAccountKnown(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer mp.server.Close()

	userResource := "kaleido_platform_user.user1"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + userWithoutSubStep,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(userResource, "account", "a:test"),
					resource.TestCheckResourceAttr(userResource, "name", "display-name"),
				),
			},
			{
				Config: providerConfig + userWithoutSubRenamedStep,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectKnownValue(userResource, tfjsonpath.New("account"), knownvalue.StringExact("a:test")),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(userResource, "account", "a:test"),
					resource.TestCheckResourceAttr(userResource, "name", "display-name-renamed"),
				),
			},
		},
	})
}

func TestUserModifyPlanWarnsOnNameChange(t *testing.T) {
	ctx := context.Background()
	r := &userResource{}

	schemaResp := &fwresource.SchemaResponse{}
	r.Schema(ctx, fwresource.SchemaRequest{}, schemaResp)

	state := tfsdk.State{Schema: schemaResp.Schema}
	require.False(t, state.Set(ctx, &UserResourceModel{
		ID:      types.StringValue("u:testuser1"),
		Account: types.StringValue("a:test"),
		Name:    types.StringValue("user-unbound@example.com"),
		Email:   types.StringValue("user-unbound@example.com"),
		Sub:     types.StringValue("oidc-sub-bound-1"),
		IsAdmin: types.BoolValue(true),
	}).HasError())

	plan := tfsdk.Plan{Schema: schemaResp.Schema}
	require.False(t, plan.Set(ctx, &UserResourceModel{
		ID:      types.StringValue("u:testuser1"),
		Account: types.StringValue("a:test"),
		Name:    types.StringValue("display-name"),
		Email:   types.StringValue("user-unbound@example.com"),
		Sub:     types.StringValue("oidc-sub-bound-1"),
		IsAdmin: types.BoolValue(true),
	}).HasError())

	var resp fwresource.ModifyPlanResponse
	r.ModifyPlan(ctx, fwresource.ModifyPlanRequest{State: state, Plan: plan}, &resp)

	require.Len(t, resp.Diagnostics.Warnings(), 1)
	warn := resp.Diagnostics.Warnings()[0]
	assert.Equal(t, "User name drift after platform login sync", warn.Summary())
	assert.Contains(t, warn.Detail(), `user-unbound@example.com`)
	assert.Contains(t, warn.Detail(), `display-name`)
	assert.Contains(t, warn.Detail(), "Terraform is planning to change name")
}

func TestUserModifyPlanNoWarningWhenNameUnchanged(t *testing.T) {
	ctx := context.Background()
	r := &userResource{}

	schemaResp := &fwresource.SchemaResponse{}
	r.Schema(ctx, fwresource.SchemaRequest{}, schemaResp)

	model := &UserResourceModel{
		ID:      types.StringValue("u:testuser1"),
		Account: types.StringValue("a:test"),
		Name:    types.StringValue("user-unbound@example.com"),
		Email:   types.StringValue("user-unbound@example.com"),
	}

	state := tfsdk.State{Schema: schemaResp.Schema}
	require.False(t, state.Set(ctx, model).HasError())
	plan := tfsdk.Plan{Schema: schemaResp.Schema}
	require.False(t, plan.Set(ctx, model).HasError())

	var resp fwresource.ModifyPlanResponse
	r.ModifyPlan(ctx, fwresource.ModifyPlanRequest{State: state, Plan: plan}, &resp)
	assert.Empty(t, resp.Diagnostics.Warnings())
}

func TestUserModifyPlanNoWarningOnCreate(t *testing.T) {
	ctx := context.Background()
	r := &userResource{}

	schemaResp := &fwresource.SchemaResponse{}
	r.Schema(ctx, fwresource.SchemaRequest{}, schemaResp)

	plan := tfsdk.Plan{Schema: schemaResp.Schema}
	require.False(t, plan.Set(ctx, &UserResourceModel{
		Name: types.StringValue("display-name"),
	}).HasError())

	var resp fwresource.ModifyPlanResponse
	r.ModifyPlan(ctx, fwresource.ModifyPlanRequest{
		State: tfsdk.State{Schema: schemaResp.Schema}, // Raw null = create
		Plan:  plan,
	}, &resp)
	assert.Empty(t, resp.Diagnostics.Warnings(), fmt.Sprintf("%v", resp.Diagnostics))
}
