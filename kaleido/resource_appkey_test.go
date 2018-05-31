package kaleido

import (
	"fmt"
	"testing"

	kaleido "github.com/kaleido-io/kaleido-sdk-go/kaleido"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestKaleidoAppKeyResource(t *testing.T) {
	consortium := kaleido.NewConsortium("terrAppKey", "appkey", "single-org")
	membership := kaleido.NewMembership("kaleido")
	environment := kaleido.NewEnvironment("appKeyEnv", "appKey", "quorum", "raft")

	consResource := "kaleido_consortium." + consortium.Name
	membershipResource := "kaleido_membership." + membership.OrgName
	envResource := "kaleido_environment." + environment.Name
	appKeyResource := "kaleido_app_key.key"

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccAppKeyConfig_basic(&consortium, &membership, &environment),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAppKeyExists(consResource, membershipResource, envResource, appKeyResource),
					resource.TestCheckResourceAttrSet(appKeyResource, "username"),
					resource.TestCheckResourceAttrSet(appKeyResource, "password"),
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

		client := testAccProvider.Meta().(kaleido.KaleidoClient)
		var appKey kaleido.AppKey
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

func testAccAppKeyConfig_basic(consortium *kaleido.Consortium, membership *kaleido.Membership, environment *kaleido.Environment) string {
	return fmt.Sprintf(`resource "kaleido_consortium" "terrAppKey" {
    name = "%s",
    description = "%s",
    mode = "%s"
    }
    resource "kaleido_membership" "kaleido" {
      consortium_id = "${kaleido_consortium.terrAppKey.id}"
      org_name = "%s"
    }

    resource "kaleido_environment" "appKeyEnv" {
      consortium_id = "${kaleido_consortium.terrAppKey.id}"
      name = "%s"
      description = "%s"
      env_type = "%s"
      consensus_type = "%s"
    }

    resource "kaleido_app_key" "key" {
      consortium_id = "${kaleido_consortium.terrAppKey.id}"
      environment_id = "${kaleido_environment.appKeyEnv.id}"
      membership_id = "${kaleido_membership.kaleido.id}"
    }
    `, consortium.Name, consortium.Description, consortium.Mode,
		membership.OrgName,
		environment.Name, environment.Description, environment.Provider, environment.ConsensusType)
}
