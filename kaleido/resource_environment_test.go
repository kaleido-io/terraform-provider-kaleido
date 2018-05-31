package kaleido

import (
	"fmt"
	"testing"

	kaleido "github.com/kaleido-io/kaleido-sdk-go/kaleido"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestKaleidoEnvironmentResource(t *testing.T) {
	consortium := kaleido.NewConsortium("terraformConsortEnv", "terraforming", "single-org")
	environment := kaleido.NewEnvironment("terraEnv", "terraforming", "quorum", "raft")
	envResourceName := "kaleido_environment.basicEnv"
	consortiumResourceName := "kaleido_consortium.basic"
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

func testAccEnvironmentConfig_basic(consortium *kaleido.Consortium, environ *kaleido.Environment) string {
	return fmt.Sprintf(`resource "kaleido_consortium" "basic" {
		name = "%s"
		description = "%s"
		mode = "%s"
		}
		resource "kaleido_environment" "basicEnv" {
			consortium_id = "${kaleido_consortium.basic.id}"
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

		client := testAccProvider.Meta().(kaleido.KaleidoClient)
		var environment kaleido.Environment
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
