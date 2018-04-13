package photic

import (
	"fmt"
	"testing"

	photic "github.com/Consensys/photic-sdk-go/kaleido"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestPhoticNodeResource(t *testing.T) {
	consortium := photic.NewConsortium("terraNode", "terraforming", "single-org")
	membership := photic.NewMembership("kaleido")
	environment := photic.NewEnvironment("terraNode", "terraforming", "quorum", "raft")

	consResource := "photic_consortium." + consortium.Name
	membershipResource := "photic_membership." + membership.OrgName
	envResource := "photic_environment." + environment.Name
	nodeResource := "photic_node.basicNode"

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccNodeConfig_basic(&consortium, &membership, &environment),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckNodeExists(consResource, membershipResource, envResource, nodeResource),
				),
			},
		},
	})
}

func testAccCheckNodeExists(consResource, membershipResource, envResource, nodeResource string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		nodeRs, ok := s.RootModule().Resources[nodeResource]

		if !ok {
			return fmt.Errorf("Not found: %s", nodeResource)
		}

		nodeId := nodeRs.Primary.ID
		if nodeId == "" {
			return fmt.Errorf("No terraform resource instance for %s", nodeResource)
		}

		consRs, ok := s.RootModule().Resources[consResource]
		if !ok {
			return fmt.Errorf("Not found: %s", consResource)
		}
		consId := consRs.Primary.ID
		if consId == "" {
			return fmt.Errorf("No terraform resource instance for %s", consResource)
		}

		envRs, ok := s.RootModule().Resources[envResource]
		if !ok {
			return fmt.Errorf("Not found: %s", envResource)
		}
		envId := envRs.Primary.ID
		if envId == "" {
			return fmt.Errorf("No terraform resource instance for %s", envResource)
		}

		membershipRs, ok := s.RootModule().Resources[membershipResource]
		if !ok {
			return fmt.Errorf("Not found: %s", envResource)
		}
		membershipId := membershipRs.Primary.ID
		if membershipId == "" {
			return fmt.Errorf("No terraform resource instance for %s", membershipResource)
		}

		client := testAccProvider.Meta().(photic.KaleidoClient)
		var node photic.Node
		res, err := client.GetNode(consId, envId, nodeId, &node)

		if err != nil {
			return err
		}

		status := res.StatusCode()
		if status != 200 {
			msg := "Did not find node %s in consortia %s and environment %s, status was: %d"
			return fmt.Errorf(msg, nodeId, consId, envId, status)
		}

		return nil
	}
}

func testAccNodeConfig_basic(consortium *photic.Consortium, membership *photic.Membership, environment *photic.Environment) string {
	return fmt.Sprintf(`resource "photic_consortium" "terraNode" {
    name = "%s"
    description = "%s"
    mode = "%s"
    }
    resource "photic_membership" "kaleido" {
      consortium_id = "${photic_consortium.terraNode.id}"
      org_name = "%s"
    }

    resource "photic_environment" "nodeEnv" {
      consortium_id = "${photic_consortium.terraNode.id}"
      name = "%s"
      description = "%s"
      env_type = "%s"
      consensus_type = "%s"
    }

    resource "photic_node" "theNode" {
      consortium_id = "${photic_consortium.terraNode.id}"
      environment_id = "${photic_environment.nodeEnv.id}"
      membership_id = "${photic_membership.kaleido.id}"
      name = "node1"
    }
    `, consortium.Name,
		consortium.Description,
		consortium.Mode,
		membership.OrgName,
		environment.Name,
		environment.Description,
		environment.Provider,
		environment.ConsensusType)
}
