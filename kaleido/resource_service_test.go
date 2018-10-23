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

func TestKaleidoServiceResource(t *testing.T) {
	consortium := kaleido.NewConsortium("terraService", "terraforming", "single-org")
	membership := kaleido.NewMembership("kaleido")
	environment := kaleido.NewEnvironment("serviceEnv", "terraforming", "quorum", "raft")

	consResource := "kaleido_consortium." + consortium.Name
	membershipResource := "kaleido_membership." + membership.OrgName
	envResource := "kaleido_environment." + environment.Name
	serviceResource := "kaleido_service.theService"

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccServiceConfig_basic(&consortium, &membership, &environment),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckServiceExists(consResource, membershipResource, envResource, serviceResource),
					resource.TestCheckResourceAttrSet(serviceResource, "https_url"),
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

		serviceId := serviceRs.Primary.ID
		if serviceId == "" {
			return fmt.Errorf("No terraform resource instance for %s", serviceResource)
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

		client := testAccProvider.Meta().(kaleido.KaleidoClient)
		var service kaleido.Service
		res, err := client.GetService(consId, envId, serviceId, &service)

		if err != nil {
			return err
		}

		status := res.StatusCode()
		if status != 200 {
			msg := "Did not find service %s in consortia %s and environment %s, status was: %d"
			return fmt.Errorf(msg, serviceId, consId, envId, status)
		}

		return nil
	}
}

func testAccServiceConfig_basic(consortium *kaleido.Consortium, membership *kaleido.Membership, environment *kaleido.Environment) string {
	return fmt.Sprintf(`resource "kaleido_consortium" "terraService" {
    name = "%s"
    description = "%s"
    mode = "%s"
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
    }

    resource "kaleido_node" "theNode" {
        consortium_id = "${kaleido_consortium.terraService.id}"
        environment_id = "${kaleido_environment.serviceEnv.id}"
        membership_id = "${kaleido_membership.kaleido.id}"
        name = "node1"
    }

    resource "kaleido_service" "theService" {
      consortium_id = "${kaleido_consortium.terraService.id}"
      environment_id = "${kaleido_environment.serviceEnv.id}"
      membership_id = "${kaleido_membership.kaleido.id}"
      service_type = "hdwallet"
      name = "service1"
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
