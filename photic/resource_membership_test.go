package photic

import (
	"fmt"
	"testing"

	photic "github.com/Consensys/photic-sdk-go/kaleido"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestPhoticMembershipResource(t *testing.T) {
	consortium := photic.NewConsortium("terraMembers", "terraforming", "single-org")
	membership := photic.NewMembership("kaleido")
	consResource := "photic_consortium." + consortium.Name
	membershipResource := "photic_membership." + membership.OrgName

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

func testAccMembershipConfig_basic(consortium *photic.Consortium, membership *photic.Membership) string {
	return fmt.Sprintf(`resource "photic_consortium" "terraMembers" {
    name = "%s"
    description = "%s"
    mode = "%s"
  }
  resource "photic_membership" "kaleido" {
    consortium_id = "${photic_consortium.terraMembers.id}"
    org_name = "%s"
  }
  `,
		consortium.Name,
		consortium.Description,
		consortium.Mode,
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

		client := testAccProvider.Meta().(photic.KaleidoClient)
		consortiaId := memRs.Primary.Attributes["consortium_id"]
		var membership photic.Membership
		res, err := client.GetMembership(consortiaId, memRs.Primary.ID, &membership)

		if err != nil {
			return err
		}

		if res.StatusCode() != 200 {
			return fmt.Errorf("Could not find membership %s in consortia %s, status: %d",
				memRs.Primary.ID,
				consortiaId,
				res.StatusCode())
		}

		return nil
	}
}
