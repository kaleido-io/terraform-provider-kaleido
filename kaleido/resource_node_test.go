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

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	kaleido "github.com/kaleido-io/kaleido-sdk-go/kaleido"
)

func TestKaleidoNodeResource(t *testing.T) {
	consortium := kaleido.NewConsortium("terraNode", "terraforming")
	membership := kaleido.NewMembership("kaleido")
	environment := kaleido.NewEnvironment("nodeEnv", "terraforming", "quorum", "raft", false, 0, map[string]string{}, 0)

	consResource := "kaleido_consortium." + consortium.Name
	membershipResource := "kaleido_membership." + membership.OrgName
	envResource := "kaleido_environment." + environment.Name
	nodeResource := "kaleido_node.theNode"

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccNodeConfig_basic(&consortium, &membership, &environment),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckNodeExists(consResource, membershipResource, envResource, nodeResource),
					resource.TestCheckResourceAttrSet(nodeResource, "https_url"),
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

		nodeID := nodeRs.Primary.ID
		if nodeID == "" {
			return fmt.Errorf("No terraform resource instance for %s", nodeResource)
		}

		consRs, ok := s.RootModule().Resources[consResource]
		if !ok {
			return fmt.Errorf("Not found: %s", consResource)
		}
		consID := consRs.Primary.ID
		if consID == "" {
			return fmt.Errorf("No terraform resource instance for %s", consResource)
		}

		envRs, ok := s.RootModule().Resources[envResource]
		if !ok {
			return fmt.Errorf("Not found: %s", envResource)
		}
		envID := envRs.Primary.ID
		if envID == "" {
			return fmt.Errorf("No terraform resource instance for %s", envResource)
		}

		membershipRs, ok := s.RootModule().Resources[membershipResource]
		if !ok {
			return fmt.Errorf("Not found: %s", envResource)
		}
		membershipID := membershipRs.Primary.ID
		if membershipID == "" {
			return fmt.Errorf("No terraform resource instance for %s", membershipResource)
		}

		client := testAccProvider.Meta().(kaleido.KaleidoClient)
		var node kaleido.Node
		res, err := client.GetNode(consID, envID, nodeID, &node)

		if err != nil {
			return err
		}

		status := res.StatusCode()
		if status != 200 {
			msg := "Did not find node %s in consortia %s and environment %s with status %d: %s"
			return fmt.Errorf(msg, nodeID, consID, envID, status, res.String())
		}

		return nil
	}
}

func testAccNodeConfig_basic(consortium *kaleido.Consortium, membership *kaleido.Membership, environment *kaleido.Environment) string {
	return fmt.Sprintf(`resource "kaleido_consortium" "terraNode" {
    name = "%s"
    description = "%s"
    }
    resource "kaleido_membership" "kaleido" {
      consortium_id = "${kaleido_consortium.terraNode.id}"
      org_name = "%s"
    }

    resource "kaleido_environment" "nodeEnv" {
      consortium_id = "${kaleido_consortium.terraNode.id}"
      name = "%s"
      description = "%s"
      env_type = "%s"
      consensus_type = "%s"
    }

    resource "kaleido_node" "theNode" {
      consortium_id = "${kaleido_consortium.terraNode.id}"
      environment_id = "${kaleido_environment.nodeEnv.id}"
      membership_id = "${kaleido_membership.kaleido.id}"
      name = "node1"
    }
    `, consortium.Name,
		consortium.Description,
		membership.OrgName,
		environment.Name,
		environment.Description,
		environment.Provider,
		environment.ConsensusType)
}
