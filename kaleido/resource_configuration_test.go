// Copyright © Kaleido, Inc. 2018, 2024

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
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	kaleido "github.com/kaleido-io/kaleido-sdk-go/kaleido"
	gock "gopkg.in/h2non/gock.v1"
)

func TestKaleidoConfigResource(t *testing.T) {
	consortium := kaleido.NewConsortium("terraConfig", "terraforming")
	membership := kaleido.NewMembership("kaleido")
	environment := kaleido.NewEnvironment("configEnv", "terraforming", "quorum", "raft", false, 0, map[string]string{
		"3ae37053826acbf0cf8dbc5c2ff344a9576b9cf5": "1000000000000000000000000000",
	}, 0)
	ezone := kaleido.NewEZone("configZone", "us-east-2", "aws")
	configuration := kaleido.NewConfiguration("theConfig", "member1", "node_config", map[string]interface {
	}{
		"gas_price": "1",
	})
	node := kaleido.NewNode("node1", "member1", "zone1")
	node.NodeConfigID = "cfg1"
	node.Role = "validator"

	consResource := "kaleido_consortium." + consortium.Name
	membershipResource := "kaleido_membership." + membership.OrgName
	envResource := "kaleido_environment." + environment.Name
	configurationResource := "kaleido_configuration.theConfig"

	defer gock.Off()
	testNodeGocks(&node)
	testConfigurationGocks(&configuration)
	testEZoneGocks(&ezone)
	testEnvironmentGocks(&environment)
	testMembershipGocks(&membership)
	testConsortiumGocks(&consortium)
	testDebugGocks()

	gock.Observe(gock.DumpRequest)

	os.Setenv("KALEIDO_API", "http://example.com/api/v1")
	os.Setenv("KALEIDO_API_KEY", "ut_apikey")

	resource.Test(t, resource.TestCase{
		IsUnitTest:                true,
		PreventPostDestroyRefresh: true,
		PreCheck:                  func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories:  testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccConfigConfig_basic(&consortium, &membership, &environment),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckConfigExists(consResource, membershipResource, envResource, configurationResource),
					// TODO: REINSTATE BELOW AFTER https://github.com/hashicorp/terraform-plugin-framework/pull/931 AVAILABLE
					// resource.TestMatchResourceAttr(configurationResource, "details.gas_price", regexp.MustCompile("1")),
				),
			},
		},
	})

}

func testAccCheckConfigExists(consResource, membershipResource, envResource, configResource string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		configRs, ok := s.RootModule().Resources[configResource]

		if !ok {
			return fmt.Errorf("Not found: %s", configResource)
		}

		configID := configRs.Primary.ID
		if configID == "" {
			return fmt.Errorf("No terraform resource instance for %s", configResource)
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

		client := newTestProviderData().BaaS
		var config kaleido.Configuration
		res, err := client.GetConfiguration(consID, envID, configID, &config)

		if err != nil {
			return err
		}

		status := res.StatusCode()
		if status != 200 {
			msg := "Did not find config %s in consortia %s and environment %s, status was: %d"
			return fmt.Errorf(msg, configID, consID, envID, status)
		}

		return nil
	}
}

func testAccConfigConfig_basic(consortium *kaleido.Consortium, membership *kaleido.Membership, environment *kaleido.Environment) string {
	return fmt.Sprintf(`resource "kaleido_consortium" "terraConfig" {
    name = "%s"
    description = "%s"
    }
    resource "kaleido_membership" "kaleido" {
      consortium_id = "${kaleido_consortium.terraConfig.id}"
      org_name = "%s"
    }

    resource "kaleido_environment" "configEnv" {
      consortium_id = "${kaleido_consortium.terraConfig.id}"
      name = "%s"
      description = "%s"
      env_type = "%s"
      consensus_type = "%s"
			prefunded_accounts = {
				"3ae37053826acbf0cf8dbc5c2ff344a9576b9cf5" = "1000000000000000000000000000"
			}
    }

		resource "kaleido_ezone" "theZone" {
			name = "configZone"
			consortium_id = "${kaleido_consortium.terraConfig.id}"
			environment_id = "${kaleido_environment.configEnv.id}"
			cloud = "aws"
			region = "us-east-2"
		}

    resource "kaleido_configuration" "theConfig" {
      consortium_id = "${kaleido_consortium.terraConfig.id}"
      environment_id = "${kaleido_environment.configEnv.id}"
      membership_id = "${kaleido_membership.kaleido.id}"
      type = "node_config"
			name = "theConfig"
			# TODO: REINSTATE DETAILS ONCE https://github.com/hashicorp/terraform-plugin-framework/pull/931 AVAILABLE
			details_json = jsonencode({
				gas_price = "1"
			})
		}
		
    resource "kaleido_node" "theNode" {
			consortium_id = "${kaleido_consortium.terraConfig.id}"
			environment_id = "${kaleido_environment.configEnv.id}"
			membership_id = "${kaleido_membership.kaleido.id}"
			node_config_id = "${kaleido_configuration.theConfig.id}"
			zone_id = "${kaleido_ezone.theZone.id}"
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
