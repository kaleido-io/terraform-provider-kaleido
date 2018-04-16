package photic

import (
	"fmt"
	"testing"

	photic "github.com/Consensys/photic-sdk-go/kaleido"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestPhoticAppKeyResource(t *testing.T) {
	consortium := photic.NewConsortium("terrAppKey", "appkey", "single-org")
	membership := photic.NewMembership("kaleido")
	environment := photic.NewEnvironment("appKeyEnv", "appKey", "quorum", "raft")

	consResource := "photic_consortium." + consortium.Name
	membershipResource := "photic_membership." + membership.OrgName
	envResource := "photic_environment." + environment.Name
	appKeyResource := "photic_app_key.key"

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccAppKeyConfig_basic(&consortium, &membership, &environment),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAppKeyExists(consResource, membershipResource, envResource, appKeyResource),
				),
			},
		},
	})
}

func testAccCheckAppKeyExists(consResource, membershipResource, envResource, appKeyResource string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		appKeyRs, ok := s.RootModule().Resources[appKeyResource]

		if !ok {
			return fmt.Errorf("Not found: %s.", appKeyResource)
		}

		consortRs, ok := s.RootModule().Resources[consResource]

		if !ok {
			return fmt.Errorf("Not found: %s.", consResource)
		}

		envRs, ok := s.RootModule().Resources[envResource]

		if !ok {
			return fmt.Errorf("Not found: %s.", envResource)
		}

		client := testAccProvider.Meta().(photic.KaleidoClient)
		var appKey photic.AppKey
		res, err := client.GetAppKey(consortRs.Primary.ID, envRs.Primary.ID, appKeyRs.Primary.ID, &appKey)
		if err != nil {
			return err
		}

		if res.StatusCode() != 200 {
			msg := "Could not find AppKey %s in consortium %s in environment %s. Status: %d"
			return fmt.Errorf(msg, appKey.Id, consortRs.Primary.ID, envRs.Primary.ID, res.StatusCode())
		}

		return nil
	}
}

func testAccAppKeyConfig_basic(consortium *photic.Consortium, membership *photic.Membership, environment *photic.Environment) string {
	return fmt.Sprintf(`resource "photic_consortium" "terrAppKey" {
    name = "%s",
    description = "%s",
    mode = "%s"
    }
    resource "photic_membership" "kaleido" {
      consortium_id = "${photic_consortium.terrAppKey.id}"
      org_name = "%s"
    }

    resource "photic_environment" "appKeyEnv" {
      consortium_id = "${photic_consortium.terrAppKey.id}"
      name = "%s"
      description = "%s"
      env_type = "%s"
      consensus_type = "%s"
    }

    resource "photic_app_key" "key" {
      consortium_id = "${photic_consortium.terrAppKey.id}"
      environment_id = "${photic_environment.appKeyEnv.id}"
      membership_id = "${photic_membership.kaleido.id}"
    }
    `, consortium.Name, consortium.Description, consortium.Mode,
		membership.OrgName,
		environment.Name, environment.Description, environment.Provider, environment.ConsensusType)
}
