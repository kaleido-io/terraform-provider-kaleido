// Copyright © Kaleido, Inc. 2025

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package platform

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"
)

var secp256k1NodeKeyStep = `
resource "kaleido_platform_secp256k1_node_key" "secp256k1_node_key1" {}
`

func TestSecp256k1NodeKeyResource(t *testing.T) {
	var (
		privateKey string
		publicKey  string
		address      string
	)
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: secp256k1NodeKeyStep,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("kaleido_platform_secp256k1_node_key.secp256k1_node_key1", "private_key"),
					resource.TestCheckResourceAttrWith("kaleido_platform_secp256k1_node_key.secp256k1_node_key1", "private_key", func(value string) error {
						t.Logf("privateKey: %s", value)
						privateKey = value
						return nil
					}),
					resource.TestCheckResourceAttrSet("kaleido_platform_secp256k1_node_key.secp256k1_node_key1", "public_key"),
					resource.TestCheckResourceAttrWith("kaleido_platform_secp256k1_node_key.secp256k1_node_key1", "public_key", func(value string) error {
						t.Logf("publicKey: %s", value)
						publicKey = value
						return nil
					}),
					resource.TestCheckResourceAttrSet("kaleido_platform_secp256k1_node_key.secp256k1_node_key1", "address"),
					resource.TestCheckResourceAttrWith("kaleido_platform_secp256k1_node_key.secp256k1_node_key1", "address", func(value string) error {
						t.Logf("address: %s", value)
						address = value
						return nil
					}),
				),
			},
			// ensures that the key is the same as the one generated in the first step
			{
				Config: secp256k1NodeKeyStep,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrWith("kaleido_platform_secp256k1_node_key.secp256k1_node_key1", "private_key", func(value string) error {
						t.Logf("privateKey: %s", value)
						assert.Equal(t, privateKey, value)
						return nil
					}),
					resource.TestCheckResourceAttrWith("kaleido_platform_secp256k1_node_key.secp256k1_node_key1", "public_key", func(value string) error {
						t.Logf("publicKey: %s", value)
						assert.Equal(t, publicKey, value)
						return nil
					}),
					resource.TestCheckResourceAttrWith("kaleido_platform_secp256k1_node_key.secp256k1_node_key1", "address", func(value string) error {
						t.Logf("address: %s", value)
						assert.Equal(t, address, value)
						return nil
					}),
				),
			},
		},
	})
}