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

func TestKaleidoInvitationResource(t *testing.T) {
	consortium := kaleido.NewConsortium("terraInvitations", "terraforming", "single-org")
	invitation := kaleido.NewInvitation("Invited Test Org", "ricardo@larosasanoja.com")
	consResource := "kaleido_consortium." + consortium.Name
	invitationResource := "kaleido_invitation." + invitation.OrgName

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccInvitationConfig_basic(&consortium, &invitation),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckInvitationExists(consResource, invitationResource),
				),
			},
		},
	})
}

func testAccInvitationConfig_basic(consortium *kaleido.Consortium, invitation *kaleido.Invitation) string {
	return fmt.Sprintf(`resource "kaleido_consortium" "terraInvitations" {
    name = "%s"
    description = "%s"
    mode = "%s"
  }
  resource "kaleido_invitation" "kaleido" {
    consortium_id = "${kaleido_consortium.terraInvitations.id}"
	org_name = "%s"
	email = "%s"
  }
  `,
		consortium.Name,
		consortium.Description,
		consortium.Mode,
		invitation.OrgName,
		invitation.Email,
	)
}

func testAccCheckInvitationExists(consResName, invResName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		invRs, ok := s.RootModule().Resources[invResName]

		if !ok {
			return fmt.Errorf("Not found: %s", invResName)
		}

		if invRs.Primary.ID == "" {
			return fmt.Errorf("No terraform resource instance for %s", invResName)
		}

		client := testAccProvider.Meta().(kaleido.KaleidoClient)
		consortiaId := invRs.Primary.Attributes["consortium_id"]
		var invitation kaleido.Invitation
		res, err := client.GetInvitation(consortiaId, invRs.Primary.ID, &invitation)

		if err != nil {
			return err
		}

		if res.StatusCode() != 200 {
			return fmt.Errorf("Could not find invitation %s in consortia %s, status: %d",
				invRs.Primary.ID,
				consortiaId,
				res.StatusCode())
		}

		return nil
	}
}
