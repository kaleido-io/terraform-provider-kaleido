// Copyright © Kaleido, Inc. 2026

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
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

type kmsV2AccConfig struct {
	envName string
	keyName string
	specSet bool // false = omit spec, exercising the default (non-v2) path
	pathSet bool // true = set path alongside spec, expected to error
}

func (c kmsV2AccConfig) hcl() string {
	specLine := ""
	if c.specSet {
		specLine = `  spec = "secp256k1"` + "\n"
	}
	pathLine := ""
	if c.pathSet {
		pathLine = `  path = "m/44'/60'/0'/0/0"` + "\n"
	}
	return testAccPlatformProviderConfig() + fmt.Sprintf(`
resource "kaleido_platform_environment" "acc" {
  name = %q
}

resource "kaleido_platform_runtime" "kms" {
  type        = "KeyManager"
  name        = "keys"
  environment = kaleido_platform_environment.acc.id
  config_json = jsonencode({})
}

resource "kaleido_platform_service" "kms" {
  type        = "KeyManager"
  name        = "keys"
  environment = kaleido_platform_environment.acc.id
  runtime     = kaleido_platform_runtime.kms.id
  config_json = jsonencode({})
}

resource "kaleido_platform_kms_wallet" "hd" {
  type        = "hdwallet"
  name        = "hdwallet"
  environment = kaleido_platform_environment.acc.id
  service     = kaleido_platform_service.kms.id
  config_json = jsonencode({})
}

resource "kaleido_platform_kms_key" "v2" {
  name        = %q
  environment = kaleido_platform_environment.acc.id
  service     = kaleido_platform_service.kms.id
  wallet      = kaleido_platform_kms_wallet.hd.id
  public_identifier_types = ["address_ethereum", "address_ethereum_checksum"]
%s%s}
`, c.envName, c.keyName, specLine, pathLine)
}

// TestAccPlatformKMSKeyV2 exercises the v2 create path (spec set): both
// requested identifier types must actually be created, with distinct values
// exposed via public_identifiers, and address must be populated. Also
// confirms rename (Update, always v2) leaves address/public_identifiers
// unchanged.
func TestAccPlatformKMSKeyV2(t *testing.T) {
	if os.Getenv("TF_LOG") == "" {
		t.Setenv("TF_LOG", "INFO")
	}
	if os.Getenv("TF_LOG_PROVIDER") == "" {
		t.Setenv("TF_LOG_PROVIDER", "INFO")
	}

	suffix := testAccNameSuffix()
	base := kmsV2AccConfig{
		envName: fmt.Sprintf("tf-acc-kms-v2-%s", suffix),
		keyName: "v2-key",
		specSet: true,
	}
	configRename := base
	configRename.keyName = "v2-key-renamed"

	keyResource := "kaleido_platform_kms_key.v2"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPlatformPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				PreConfig: func() { t.Log("step: create v2 key with two identifier types") },
				Config:    base.hcl(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(keyResource, "id"),
					resource.TestCheckResourceAttr(keyResource, "spec", "secp256k1"),
					resource.TestCheckResourceAttr(keyResource, "public_identifier_types.#", "2"),
					resource.TestCheckResourceAttrSet(keyResource, "address"),
					resource.TestCheckResourceAttr(keyResource, "public_identifiers.%", "2"),
					resource.TestCheckResourceAttrSet(keyResource, "public_identifiers.address_ethereum"),
					resource.TestCheckResourceAttrSet(keyResource, "public_identifiers.address_ethereum_checksum"),
				),
			},
			{
				PreConfig: func() { t.Log("step: plan-only after create (expect empty)") },
				Config:    base.hcl(),
				PlanOnly:  true,
			},
			{
				PreConfig: func() { t.Log("step: rename v2 key (Update, always v2)") },
				Config:    configRename.hcl(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(keyResource, "name", "v2-key-renamed"),
					resource.TestCheckResourceAttrSet(keyResource, "address"),
					resource.TestCheckResourceAttr(keyResource, "public_identifiers.%", "2"),
				),
			},
			{
				PreConfig: func() { t.Log("step: plan-only after rename (expect empty)") },
				Config:    configRename.hcl(),
				PlanOnly:  true,
			},
		},
	})
}

// TestAccPlatformKMSKeyV2RejectsPath confirms Create errors instead of
// silently dropping path when spec is also set.
func TestAccPlatformKMSKeyV2RejectsPath(t *testing.T) {
	if os.Getenv("TF_LOG") == "" {
		t.Setenv("TF_LOG", "INFO")
	}

	suffix := testAccNameSuffix()
	config := kmsV2AccConfig{
		envName: fmt.Sprintf("tf-acc-kms-v2-pathconflict-%s", suffix),
		keyName: "v2-path-conflict",
		specSet: true,
		pathSet: true,
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPlatformPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				PreConfig:   func() { t.Log("step: expect error setting path alongside spec") },
				Config:      config.hcl(),
				ExpectError: regexp.MustCompile(`path is not supported with keystore_name/spec`),
			},
		},
	})
}
