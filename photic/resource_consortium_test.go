package photic

import (
	"fmt"
	"testing"

	photic "github.com/Consensys/photic-sdk-go/kaleido"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestPhoticConsortiumResource(t *testing.T) {
	consortium := photic.NewConsortium("terraformConsort", "terraforming", "single-org")
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccConsortiumConfig_basic(&consortium),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckConsortiumExists("photic_consortium.testConsort", &consortium),
				),
			},
		},
	})
}

func testAccConsortiumConfig_basic(consortium *photic.Consortium) string {
	return fmt.Sprintf(`resource "photic_consortium" "testConsort" {
    name = "%s"
    description = "%s"
    mode = "%s"
    }`, consortium.Name, consortium.Description, consortium.Mode)
}

func testAccCheckConsortiumDestroy(s *terraform.State) error {
	return nil
}

// testAccCheckConsortiumExists
func testAccCheckConsortiumExists(resourceName string, consortium *photic.Consortium) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No terraform resource instance for consortium.")
		}

		consortiumId := rs.Primary.Attributes["id"]
		if consortiumId == "" {
			return fmt.Errorf("Terraform resource instance missing consortium id.")
		}

		client := testAccProvider.Meta().(photic.KaleidoClient)
		var consortium photic.Consortium
		res, err := client.GetConsortium(consortiumId, &consortium)

		if err != nil {
			return err
		}

		if res.StatusCode() != 200 {
			return fmt.Errorf("Could not fetch Consortium %s, status: %d", consortiumId, res.StatusCode())
		}

		return nil
	}
}
