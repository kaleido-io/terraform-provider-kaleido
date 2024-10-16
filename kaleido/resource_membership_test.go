// Copyright © Kaleido, Inc. 2018, 2021

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package kaleido

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	kaleido "github.com/kaleido-io/kaleido-sdk-go/kaleido"
)

func TestKaleidoMembershipResource(t *testing.T) {
	consortium := kaleido.NewConsortium("terraMembers", "terraforming")
	membership := kaleido.NewMembership("kaleido")
	membershipResource := "kaleido_membership." + membership.OrgName

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccMembershipConfig_basic(&consortium, &membership),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMembershipExists(membershipResource),
				),
			},
		},
	})
}

func testAccMembershipConfig_basic(consortium *kaleido.Consortium, membership *kaleido.Membership) string {
	return fmt.Sprintf(`resource "kaleido_consortium" "terraMembers" {
    name = "%s"
    description = "%s"
  }
  resource "kaleido_membership" "kaleido" {
    consortium_id = "${kaleido_consortium.terraMembers.id}"
    org_name = "%s"
  }
  `,
		consortium.Name,
		consortium.Description,
		membership.OrgName)
}

func testAccCheckMembershipExists(memResName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		memRs, ok := s.RootModule().Resources[memResName]

		if !ok {
			return fmt.Errorf("Not found: %s", memResName)
		}

		if memRs.Primary.ID == "" {
			return fmt.Errorf("No terraform resource instance for %s", memResName)
		}

		client := newTestProviderData().BaaS
		consortiaID := memRs.Primary.Attributes["consortium_id"]
		var membership kaleido.Membership
		res, err := client.GetMembership(consortiaID, memRs.Primary.ID, &membership)

		if err != nil {
			return err
		}

		if res.StatusCode() != 200 {
			return fmt.Errorf("Could not find membership %s in consortia %s, status: %d",
				memRs.Primary.ID,
				consortiaID,
				res.StatusCode())
		}

		return nil
	}
}
