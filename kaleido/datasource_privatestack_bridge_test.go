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
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	gock "gopkg.in/h2non/gock.v1"
)

func TestKaleidoPrivateStackBridgeResource(t *testing.T) {
	defer gock.Off()

	gock.Observe(gock.DumpRequest)

	os.Setenv("KALEIDO_API", "http://api.example.com/api/v1")
	os.Setenv("KALEIDO_API_KEY", "ut_apikey")

	mockConfigJSON := []byte(`
	{
		"id": "zzqbpgi4xb-zzsvij4k8v-client",
		"nodes": [
			{
				"role": "server",
				"id": "zzqbpgi4xb-zzsvij4k8v-server",
				"address": "wss://zzqbpgi4xb-zzsvij4k8v.xyz.kaleido.io",
				"auth": {
					"user": "xxx",
					"secret": "yyy"
				}
			}
		]
	}`)
	var mockConfig map[string]interface{}
	json.Unmarshal(mockConfigJSON, &mockConfig)

	gock.New("http://api.example.com").
		Get("/api/v1/consortia/cons1/environments/env1/services/svc1/tunneler_config").
		Persist().
		Reply(200).
		JSON(mockConfig)

	pstackBridgeResource := "data.kaleido_privatestack_bridge.test"

	expectedConfJSONUnFormatted := []byte(`
	{
		"id": "zzqbpgi4xb-zzsvij4k8v-client",
		"nodes": [
			{
				"role": "server",
				"id": "zzqbpgi4xb-zzsvij4k8v-server",
				"address": "wss://zzqbpgi4xb-zzsvij4k8v.xyz.kaleido.io",
				"auth": {
					"user": "user_abc",
					"secret": "password_abc"
				}
			}
		]
	}`)
	var expectedConfig map[string]interface{}
	json.Unmarshal(expectedConfJSONUnFormatted, &expectedConfig)
	expectedConfJSON, _ := json.MarshalIndent(expectedConfig, "", "  ")

	resource.Test(t, resource.TestCase{
		IsUnitTest:                true,
		PreventPostDestroyRefresh: true,
		PreCheck:                  func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories:  testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testPrivateStackBridgeConfigFetch_tf(),
				Check: resource.ComposeAggregateTestCheckFunc(
					testPrivateStackBridgeConfigFetch(pstackBridgeResource),
					resource.TestCheckResourceAttr(pstackBridgeResource, "config_json", string(expectedConfJSON)),
				),
			},
		},
	})

}

func testPrivateStackBridgeConfigFetch(pstackBridgeResource string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

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
	return `
	
		data "kaleido_privatestack_bridge" "test" {
			consortium_id = "cons1"
			environment_id = "env1"
			service_id = "svc1"
			appcred_id = "user_abc"
			appcred_secret = "password_abc"
		}
	
    `
}
