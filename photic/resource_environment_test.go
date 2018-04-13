package photic

import (
	"fmt"
	"testing"

	photic "github.com/Consensys/photic-sdk-go/kaleido"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestPhoticEnvironmentResource(t *testing.T) {
	consortium := photic.NewConsortium("terraformConsortEnv", "terraforming", "single-org")
	environment := photic.NewEnvironment("terraEnv", "terraforming", "quorum", "raft")
	envResourceName := "photic_environment.basicEnv"
	consortiumResourceName := "photic_consortium.basic"
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckConsortiumDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEnvironmentConfig_basic(&consortium, &environment),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckEnvironmentExists(envResourceName, consortiumResourceName),
				),
			},
		},
	})
}

func testAccEnvironmentConfig_basic(consortium *photic.Consortium, environ *photic.Environment) string {
	return fmt.Sprintf(`resource "photic_consortium" "basic" {
		name = "%s"
		description = "%s"
		mode = "%s"
		}
		resource "photic_environment" "basicEnv" {
			consortium_id = "${photic_consortium.basic.id}"
			name = "%s"
			description = "%s"
			env_type = "%s"
			consensus_type = "%s"
		}
		`, consortium.Name,
		consortium.Description,
		consortium.Mode,
		environ.Name,
		environ.Description,
		environ.Provider,
		environ.ConsensusType)
}
func testAccCheckEnvironmentExists(envResource, consortiumResource string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[envResource]

		if !ok {
			return fmt.Errorf("Not found: %s", envResource)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("Not terraform resource instance for %s", envResource)
		}

		consortium, ok := s.RootModule().Resources[consortiumResource]
		if !ok {
			return fmt.Errorf("Not found: %s", envResource)
		}

		if consortium.Primary.ID == "" {
			return fmt.Errorf("Not terraform resource instance for %s", envResource)
		}

		envId := rs.Primary.Attributes["id"]
		if envId != rs.Primary.ID {
			return fmt.Errorf("Terraform id mismatch for environment %s and %s", envId, rs.Primary.ID)
		}

		client := testAccProvider.Meta().(photic.KaleidoClient)
		var environment photic.Environment
		res, err := client.GetEnvironment(consortium.Primary.ID, envId, &environment)

		if err != nil {
			return err
		}

		if res.StatusCode() != 200 {
			return fmt.Errorf("Expected environment with id %s, status was: %d.", envId, res.StatusCode())
		}

		return nil
	}
}
