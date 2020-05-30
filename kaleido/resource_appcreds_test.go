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

func TestKaleidoAppKeyResource(t *testing.T) {
	consortium := kaleido.NewConsortium("terrAppKey", "appkey")
	membership := kaleido.NewMembership("kaleido")
	environment := kaleido.NewEnvironment("appKeyEnv", "appKey", "quorum", "raft", false, 0)

	consResource := "kaleido_consortium." + consortium.Name
	membershipResource := "kaleido_membership." + membership.OrgName
	envResource := "kaleido_environment." + environment.Name
	appKeyResource := "kaleido_app_creds.key"

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
		var appKey kaleido.AppCreds
		res, err := client.GetAppCreds(consortRs.Primary.ID, envRs.Primary.ID, appKeyRs.Primary.ID, &appKey)
		if err != nil {
			return err
		}

		if res.StatusCode() != 200 {
			msg := "Could not find AppKey %s in consortium %s in environment %s. Status: %d"
			return fmt.Errorf(msg, appKey.ID, consortRs.Primary.ID, envRs.Primary.ID, res.StatusCode())
		}

		return nil
	}
}

func testAccAppKeyConfig_basic(consortium *kaleido.Consortium, membership *kaleido.Membership, environment *kaleido.Environment) string {
	return fmt.Sprintf(`resource "kaleido_consortium" "terrAppKey" {
    name = "%s",
    description = "%s",
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

    resource "kaleido_app_creds" "key" {
      consortium_id = "${kaleido_consortium.terrAppKey.id}"
      environment_id = "${kaleido_environment.appKeyEnv.id}"
      membership_id = "${kaleido_membership.kaleido.id}"
    }
    `, consortium.Name, consortium.Description,
		membership.OrgName,
		environment.Name, environment.Description, environment.Provider, environment.ConsensusType)
}
