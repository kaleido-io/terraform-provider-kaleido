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
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	kaleido "github.com/kaleido-io/kaleido-sdk-go/kaleido"
	gock "gopkg.in/h2non/gock.v1"
)

func TestKaleidoServiceResource(t *testing.T) {
	consortium := kaleido.NewConsortium("terraService", "terraforming")
	membership := kaleido.NewMembership("kaleido")
	environment := kaleido.NewEnvironment("serviceEnv", "terraforming", "quorum", "raft", false, 0, map[string]string{
		"3ae37053826acbf0cf8dbc5c2ff344a9576b9cf5": "1000000000000000000000000000",
	}, 0)
	ezone := kaleido.NewEZone("serviceZone", "us-east-2", "aws")
	service := kaleido.NewService("service1", "hdwallet", "member1", "zone1", map[string]interface{}{
		"backup_id":     "backupid1",
		"kms_id":        "kms1",
		"networking_id": "networking1",
	})
	ipfs_service := kaleido.NewService("ipfs_service1", "ipfs", "member1", "zone1", map[string]interface{}{})
	node := kaleido.NewNode("node1", "member1", "zone1")
	node.Role = "validator"

	consResource := "kaleido_consortium." + consortium.Name
	membershipResource := "kaleido_membership." + membership.OrgName
	envResource := "kaleido_environment." + environment.Name
	serviceResource := "kaleido_service.theService"
	ipfsServiceResource := "kaleido_service.ipfs_service"

	defer gock.Off()
	testNodeGocks(&node)
	testServiceGocks(&service)
	testIPFSServiceGocks(&ipfs_service)
	testEZoneGocks(&ezone)
	testEnvironmentGocks(&environment)
	testMembershipGocks(&membership)
	testConsortiumGocks(&consortium)
	testDebugGocks()

	gock.Observe(gock.DumpRequest)

	os.Setenv("KALEIDO_API", "http://api.example.com/api/v1")
	os.Setenv("KALEIDO_API_KEY", "ut_apikey")

	resource.Test(t, resource.TestCase{
		IsUnitTest:                true,
		PreventPostDestroyRefresh: true,
		PreCheck:                  func() { testAccPreCheck(t) },
		Providers:                 testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccServiceConfig_basic(&consortium, &membership, &environment),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckServiceExists(consResource, membershipResource, envResource, serviceResource),
					testAccCheckServiceExists(consResource, membershipResource, envResource, ipfsServiceResource),
					resource.TestMatchResourceAttr(serviceResource, "details.kms_id", regexp.MustCompile("kms1")),
					resource.TestMatchResourceAttr(serviceResource, "details.backup_id", regexp.MustCompile("backupid1")),
					resource.TestMatchResourceAttr(serviceResource, "details.networking_id", regexp.MustCompile("networking1")),
				),
			},
		},
	})

}

func testAccCheckServiceExists(consResource, membershipResource, envResource, serviceResource string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		serviceRs, ok := s.RootModule().Resources[serviceResource]

		if !ok {
			return fmt.Errorf("Not found: %s", serviceResource)
		}

		serviceID := serviceRs.Primary.ID
		if serviceID == "" {
			return fmt.Errorf("No terraform resource instance for %s", serviceResource)
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
		var service kaleido.Service
		res, err := client.GetService(consID, envID, serviceID, &service)

		if err != nil {
			return err
		}

		status := res.StatusCode()
		if status != 200 {
			msg := "Did not find service %s in consortia %s and environment %s with status %d: %s"
			return fmt.Errorf(msg, serviceID, consID, envID, status, res.String())
		}

		return nil
	}
}

func testAccServiceConfig_basic(consortium *kaleido.Consortium, membership *kaleido.Membership, environment *kaleido.Environment) string {
	return fmt.Sprintf(`resource "kaleido_consortium" "terraService" {
    name = "%s"
    description = "%s"
    }
    resource "kaleido_membership" "kaleido" {
      consortium_id = "${kaleido_consortium.terraService.id}"
      org_name = "%s"
    }

    resource "kaleido_environment" "serviceEnv" {
      consortium_id = "${kaleido_consortium.terraService.id}"
      name = "%s"
      description = "%s"
      env_type = "%s"
      consensus_type = "%s"
			prefunded_accounts = {
				"3ae37053826acbf0cf8dbc5c2ff344a9576b9cf5" = "1000000000000000000000000000"
			}
    }

		resource "kaleido_ezone" "theZone" {
			name = "serviceZone"
			consortium_id = "${kaleido_consortium.terraService.id}"
			environment_id = "${kaleido_environment.serviceEnv.id}"
			cloud = "aws"
			region = "us-east-2"
		}

    resource "kaleido_node" "theNode" {
        consortium_id = "${kaleido_consortium.terraService.id}"
        environment_id = "${kaleido_environment.serviceEnv.id}"
        membership_id = "${kaleido_membership.kaleido.id}"
				zone_id = "${kaleido_ezone.theZone.id}"
        name = "node1"
		}
		
    resource "kaleido_service" "theService" {
      consortium_id = "${kaleido_consortium.terraService.id}"
      environment_id = "${kaleido_environment.serviceEnv.id}"
      membership_id = "${kaleido_membership.kaleido.id}"
			zone_id = "${kaleido_ezone.theZone.id}"
      service_type = "hdwallet"
			name = "service1"
			details = {
				backup_id = "backupid1"
				kms_id = "kms1"
				networking_id = "networking1"
			}
    }
    
	resource "kaleido_service" "ipfs_service" {
		consortium_id = "${kaleido_consortium.terraService.id}"
		environment_id = "${kaleido_environment.serviceEnv.id}"
		membership_id = "${kaleido_membership.kaleido.id}"
			  zone_id = "${kaleido_ezone.theZone.id}"
		service_type = "ipfs"
			  name = "ipfs_service1"
	}
    `, consortium.Name,
		consortium.Description,
		membership.OrgName,
		environment.Name,
		environment.Description,
		environment.Provider,
		environment.ConsensusType)
}
