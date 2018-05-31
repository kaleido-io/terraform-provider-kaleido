package kaleido

import (
	"fmt"
	"testing"

	kaleido "github.com/kaleido-io/kaleido-sdk-go/kaleido"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestKaleidoConsortiumResource(t *testing.T) {
	consortium := kaleido.NewConsortium("terraformConsort", "terraforming", "single-org")
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
					resource.TestCheckResourceAttr(resourceName, "mode", consortium.Mode),
				),
			},
		},
	})
}

func testAccConsortiumConfig_basic(consortium *kaleido.Consortium) string {
	return fmt.Sprintf(`resource "kaleido_consortium" "basic" {
    name = "%s"
    description = "%s"
    mode = "%s"
    }`, consortium.Name, consortium.Description, consortium.Mode)
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

		consortiumId := rs.Primary.Attributes["id"]
		if consortiumId == "" {
			return fmt.Errorf("Terraform resource instance missing consortium id.")
		}

		client := testAccProvider.Meta().(kaleido.KaleidoClient)
		var consortium kaleido.Consortium
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
