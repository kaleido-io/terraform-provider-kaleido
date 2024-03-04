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

func TestKaleidoInvitationResource(t *testing.T) {
	consortium := kaleido.NewConsortium("terraInvitations", "terraforming")
	invitation := kaleido.NewInvitation("Invited Test Org", "someone@example.com")
	invitationResource := "kaleido_invitation." + invitation.OrgName

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccInvitationConfig_basic(&consortium, &invitation),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckInvitationExists(invitationResource),
				),
			},
		},
	})
}

func testAccInvitationConfig_basic(consortium *kaleido.Consortium, invitation *kaleido.Invitation) string {
	return fmt.Sprintf(`resource "kaleido_consortium" "terraInvitations" {
    name = "%s"
    description = "%s"
  }
  resource "kaleido_invitation" "kaleido" {
    consortium_id = "${kaleido_consortium.terraInvitations.id}"
	org_name = "%s"
	email = "%s"
  }
  `,
		consortium.Name,
		consortium.Description,
		invitation.OrgName,
		invitation.Email,
	)
}

func testAccCheckInvitationExists(invResName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		invRs, ok := s.RootModule().Resources[invResName]

		if !ok {
			return fmt.Errorf("Not found: %s", invResName)
		}

		if invRs.Primary.ID == "" {
			return fmt.Errorf("No terraform resource instance for %s", invResName)
		}

		client := newProviderData("", "").baas
		consortiaID := invRs.Primary.Attributes["consortium_id"]
		var invitation kaleido.Invitation
		res, err := client.GetInvitation(consortiaID, invRs.Primary.ID, &invitation)

		if err != nil {
			return err
		}

		if res.StatusCode() != 200 {
			return fmt.Errorf("Could not find invitation %s in consortia %s, status: %d",
				invRs.Primary.ID,
				consortiaID,
				res.StatusCode())
		}

		return nil
	}
}
