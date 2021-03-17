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
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	gock "gopkg.in/h2non/gock.v1"
)

func TestKaleidoPrivateStackBridgeResource(t *testing.T) {
	defer gock.Off()

	gock.Observe(gock.DumpRequest)

	os.Setenv("KALEIDO_API", "http://api.example.com/api/v1")
	os.Setenv("KALEIDO_API_KEY", "ut_apikey")

	mockConfig := map[string]interface{}{
		"nested": map[string]interface{}{
			"arrays": []int{
				1, 2, 3, 4, 5,
			},
		},
		"simple": "stringvalue",
	}

	gock.New("http://api.example.com").
		Get("/api/v1/consortia/cons1/environments/env1/services/svc1/tunneler_config").
		Persist().
		Reply(200).
		JSON(mockConfig)

	pstackBridgeResource := "data.kaleido_privatestack_bridge.test"

	expectedConf, _ := json.MarshalIndent(mockConfig, "", "  ")

	resource.Test(t, resource.TestCase{
		IsUnitTest:                true,
		PreventPostDestroyRefresh: true,
		PreCheck:                  func() { testAccPreCheck(t) },
		Providers:                 testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testPrivateStackBridgeConfigFetch_tf(),
				Check: resource.ComposeAggregateTestCheckFunc(
					testPrivateStackBridgeConfigFetch(pstackBridgeResource),
					resource.TestCheckResourceAttr(pstackBridgeResource, "config_json", string(expectedConf)),
				),
			},
		},
	})

}

func testPrivateStackBridgeConfigFetch(pstackBridgeResource string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		r := s.RootModule().Resources
		fmt.Printf("RETURNED: %+v", r)

		configRs, ok := s.RootModule().Resources[pstackBridgeResource]

		if !ok {
			return fmt.Errorf("Not found: %s", pstackBridgeResource)
		}

		if configRs.Primary.ID != "svc1" {
			return fmt.Errorf("Invalid ID: %+v", configRs.Primary)
		}

		return nil
	}
}

func testPrivateStackBridgeConfigFetch_tf() string {
	return fmt.Sprintf(`
	
		data "kaleido_privatestack_bridge" "test" {
			consortium_id = "cons1"
			environment_id = "env1"
			service_id = "svc1"
		}
	
    `)
}
