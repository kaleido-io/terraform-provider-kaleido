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

func TestKaleidoConsortiumResource(t *testing.T) {
	consortium := kaleido.NewConsortium("terraformConsort", "terraforming")
	resourceName := "kaleido_consortium.basic"
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckConsortiumDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccConsortiumConfig_basic(&consortium),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckConsortiumExists(resourceName, &consortium),
					resource.TestCheckResourceAttr(resourceName, "name", consortium.Name),
					resource.TestCheckResourceAttr(resourceName, "description", consortium.Description),
				),
			},
		},
	})
}

func testAccConsortiumConfig_basic(consortium *kaleido.Consortium) string {
	return fmt.Sprintf(`resource "kaleido_consortium" "basic" {
    name = "%s"
    description = "%s"
    }`, consortium.Name, consortium.Description)
}

func testAccCheckConsortiumDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(kaleido.KaleidoClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kaleido_consortium" || rs.Primary.ID == "" {
			continue
		}
		client.DeleteConsortium(rs.Primary.ID)
	}
	return nil
}

// testAccCheckConsortiumExists
func testAccCheckConsortiumExists(resourceName string, consortium *kaleido.Consortium) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No terraform resource instance for consortium.")
		}

		consortiumID := rs.Primary.Attributes["id"]
		if consortiumID == "" {
			return fmt.Errorf("Terraform resource instance missing consortium id.")
		}

		client := testAccProvider.Meta().(kaleido.KaleidoClient)
		var consortium kaleido.Consortium
		res, err := client.GetConsortium(consortiumID, &consortium)

		if err != nil {
			return err
		}

		if res.StatusCode() != 200 {
			return fmt.Errorf("Could not fetch Consortium %s, status: %d", consortiumID, res.StatusCode())
		}

		return nil
	}
}
