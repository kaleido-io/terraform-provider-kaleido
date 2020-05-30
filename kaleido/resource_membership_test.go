// Copyright 2018 Kaleido, a ConsenSys business

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

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	kaleido "github.com/kaleido-io/kaleido-sdk-go/kaleido"
)

func TestKaleidoMembershipResource(t *testing.T) {
	consortium := kaleido.NewConsortium("terraMembers", "terraforming")
	membership := kaleido.NewMembership("kaleido")
	consResource := "kaleido_consortium." + consortium.Name
	membershipResource := "kaleido_membership." + membership.OrgName

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccMembershipConfig_basic(&consortium, &membership),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMembershipExists(consResource, membershipResource),
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

func testAccCheckMembershipExists(consResName, memResName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		memRs, ok := s.RootModule().Resources[memResName]

		if !ok {
			return fmt.Errorf("Not found: %s", memResName)
		}

		if memRs.Primary.ID == "" {
			return fmt.Errorf("No terraform resource instance for %s", memResName)
		}

		client := testAccProvider.Meta().(kaleido.KaleidoClient)
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
