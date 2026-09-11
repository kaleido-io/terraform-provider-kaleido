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
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccPlatformIAM(t *testing.T) {
	suffix := testAccNameSuffix()
	envName := fmt.Sprintf("tf-acc-env-%s", suffix)
	groupName := fmt.Sprintf("tf-acc-group-%s", suffix)
	userName := fmt.Sprintf("tf-acc-user-%s", suffix)
	userEmail := fmt.Sprintf("tf-acc-%s@example.com", suffix)
	userSub := fmt.Sprintf("tf-acc-sub-%s", suffix)
	appName := fmt.Sprintf("tf-acc-app-%s", suffix)
	apiKeyName := fmt.Sprintf("tf-acc-key-%s", suffix)

	envResource := "kaleido_platform_environment.acc"
	groupResource := "kaleido_platform_group.acc"
	userResource := "kaleido_platform_user.acc"
	membershipResource := "kaleido_platform_group_membership.acc"
	appResource := "kaleido_platform_application.acc"
	apiKeyResource := "kaleido_platform_api_key.acc"

	config := testAccPlatformProviderConfig() + fmt.Sprintf(`
resource "kaleido_platform_environment" "acc" {
  name = %q
}

resource "kaleido_platform_group" "acc" {
  name = %q
}

resource "kaleido_platform_user" "acc" {
  name     = %q
  email    = %q
  sub      = %q
  is_admin = true
}

resource "kaleido_platform_group_membership" "acc" {
  group_id = kaleido_platform_group.acc.id
  user_id  = kaleido_platform_user.acc.id
}

resource "kaleido_platform_application" "acc" {
  name          = %q
  admin_enabled = true
  oauth_enabled = false
}

resource "kaleido_platform_api_key" "acc" {
  name           = %q
  application_id = kaleido_platform_application.acc.id
  no_expiry      = true
}
`, envName, groupName, userName, userEmail, userSub, appName, apiKeyName)

	configUpdate := testAccPlatformProviderConfig() + fmt.Sprintf(`
resource "kaleido_platform_environment" "acc" {
  name = %q
}

resource "kaleido_platform_group" "acc" {
  name = %q
}

resource "kaleido_platform_user" "acc" {
  name     = %q
  email    = %q
  sub      = %q
  # Admin so destroy can remove the last group membership (KA038302 for non-admins).
  is_admin = true
}

resource "kaleido_platform_group_membership" "acc" {
  group_id = kaleido_platform_group.acc.id
  user_id  = kaleido_platform_user.acc.id
}

resource "kaleido_platform_application" "acc" {
  name          = %q
  admin_enabled = true
  oauth_enabled = false
}

resource "kaleido_platform_api_key" "acc" {
  name           = %q
  application_id = kaleido_platform_application.acc.id
  no_expiry      = true
}
`, envName+"-renamed", groupName+"-renamed", userName, userEmail, userSub, appName+"-renamed", apiKeyName+"-renamed")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPlatformPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviders,
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCheckPlatformAPIGone(func(s *terraform.State) (string, error) {
				id, err := testAccResourceID(s, envResource)
				if err != nil {
					return "", nil // already gone from state
				}
				return fmt.Sprintf("/api/v1/environments/%s", id), nil
			}),
			testAccCheckPlatformAPIGone(func(s *terraform.State) (string, error) {
				id, err := testAccResourceID(s, groupResource)
				if err != nil {
					return "", nil
				}
				return fmt.Sprintf("/api/v1/groups/%s", id), nil
			}),
			testAccCheckPlatformAPIGone(func(s *terraform.State) (string, error) {
				id, err := testAccResourceID(s, userResource)
				if err != nil {
					return "", nil
				}
				return fmt.Sprintf("/api/v1/users/%s", id), nil
			}),
			testAccCheckPlatformAPIGone(func(s *terraform.State) (string, error) {
				id, err := testAccResourceID(s, appResource)
				if err != nil {
					return "", nil
				}
				return fmt.Sprintf("/api/v1/applications/%s", id), nil
			}),
		),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(envResource, "id"),
					resource.TestCheckResourceAttr(envResource, "name", envName),
					resource.TestCheckResourceAttrSet(groupResource, "id"),
					resource.TestCheckResourceAttr(groupResource, "name", groupName),
					resource.TestCheckResourceAttrSet(userResource, "id"),
					resource.TestCheckResourceAttr(userResource, "email", userEmail),
					resource.TestCheckResourceAttr(userResource, "is_admin", "true"),
					resource.TestCheckResourceAttrSet(membershipResource, "id"),
					resource.TestCheckResourceAttrSet(appResource, "id"),
					resource.TestCheckResourceAttr(appResource, "name", appName),
					resource.TestCheckResourceAttrSet(apiKeyResource, "id"),
					resource.TestCheckResourceAttr(apiKeyResource, "name", apiKeyName),
					resource.TestCheckResourceAttrSet(apiKeyResource, "secret"),
				),
			},
			// Catch omit-optional / platform-default drift after create.
			// PlanOnly fails unless the plan is empty
			{
				Config:   config,
				PlanOnly: true,
			},
			{
				Config: configUpdate,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(envResource, "name", envName+"-renamed"),
					resource.TestCheckResourceAttr(groupResource, "name", groupName+"-renamed"),
					resource.TestCheckResourceAttr(appResource, "name", appName+"-renamed"),
					resource.TestCheckResourceAttr(apiKeyResource, "name", apiKeyName+"-renamed"),
				),
			},
			{
				Config:   configUpdate,
				PlanOnly: true,
			},
		},
	})
}
