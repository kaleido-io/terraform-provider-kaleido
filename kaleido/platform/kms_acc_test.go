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
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

type kmsAccConfig struct {
	envName           string
	attributedKeyName string
	folderPath        string
	attributesHCL     string // empty = omit attributes block
}

func (c kmsAccConfig) hcl() string {
	attrsBlock := ""
	if c.attributesHCL != "" {
		attrsBlock = fmt.Sprintf("\n  attributes = {\n%s\n  }", c.attributesHCL)
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

# Minimal key 
resource "kaleido_platform_kms_key" "bare" {
  name        = "bare"
  environment = kaleido_platform_environment.acc.id
  service     = kaleido_platform_service.kms.id
  wallet      = kaleido_platform_kms_wallet.hd.id
}

# Folder path only
resource "kaleido_platform_kms_key" "in_folder" {
  name        = "in-folder"
  folder_path = "ops/hot"
  environment = kaleido_platform_environment.acc.id
  service     = kaleido_platform_service.kms.id
  wallet      = kaleido_platform_kms_wallet.hd.id
  public_identifier_types = ["address_ethereum"]
}

# Folder path + attributes + public identifiers 
resource "kaleido_platform_kms_key" "attributed" {
  name        = %q
  folder_path = %q
  environment = kaleido_platform_environment.acc.id
  service     = kaleido_platform_service.kms.id
  wallet      = kaleido_platform_kms_wallet.hd.id
  public_identifier_types = ["address_ethereum"]%s
}
`, c.envName, c.attributedKeyName, c.folderPath, attrsBlock)
}

func TestAccPlatformKMSHDWallet(t *testing.T) {
	if os.Getenv("TF_LOG") == "" {
		t.Setenv("TF_LOG", "INFO")
	}
	if os.Getenv("TF_LOG_PROVIDER") == "" {
		t.Setenv("TF_LOG_PROVIDER", "INFO")
	}

	suffix := testAccNameSuffix()
	base := kmsAccConfig{
		envName:           fmt.Sprintf("tf-acc-kms-%s", suffix),
		attributedKeyName: "attributed",
		folderPath:        "treasury/hot",
		attributesHCL:     `    derivation = "bip44"`,
	}

	config := base.hcl()
	configRename := base
	configRename.attributedKeyName = "attributed-renamed"

	configBadFolder := configRename
	configBadFolder.folderPath = "treasury/cold"

	configBadAttrs := configRename
	configBadAttrs.attributesHCL = `    derivation = "bip32"`

	envResource := "kaleido_platform_environment.acc"
	walletResource := "kaleido_platform_kms_wallet.hd"
	bareKey := "kaleido_platform_kms_key.bare"
	folderKey := "kaleido_platform_kms_key.in_folder"
	attributedKey := "kaleido_platform_kms_key.attributed"

	immutableErr := regexp.MustCompile(`Immutable attribute cannot be updated`)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPlatformPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviders,
		CheckDestroy: testAccCheckPlatformAPIGone(func(s *terraform.State) (string, error) {
			id, err := testAccResourceID(s, envResource)
			if err != nil {
				return "", nil
			}
			return fmt.Sprintf("/api/v1/environments/%s", id), nil
		}),
		Steps: []resource.TestStep{
			{
				PreConfig: func() { t.Log("step: create KeyManager + HD wallet keys (bare / folder / attributed)") },
				Config:    config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(envResource, "id"),
					resource.TestCheckResourceAttrSet(walletResource, "id"),
					resource.TestCheckResourceAttr(walletResource, "type", "hdwallet"),
					resource.TestCheckResourceAttrSet(bareKey, "id"),
					resource.TestCheckResourceAttrSet(bareKey, "path"),
					resource.TestCheckResourceAttrSet(folderKey, "id"),
					resource.TestCheckResourceAttr(folderKey, "folder_path", "ops/hot"),
					resource.TestCheckResourceAttr(folderKey, "public_identifier_types.0", "address_ethereum"),
					resource.TestCheckResourceAttrSet(folderKey, "address"),
					resource.TestCheckResourceAttrSet(attributedKey, "id"),
					resource.TestCheckResourceAttr(attributedKey, "folder_path", "treasury/hot"),
					resource.TestCheckResourceAttr(attributedKey, "attributes.derivation", "bip44"),
					resource.TestCheckResourceAttr(attributedKey, "public_identifier_types.0", "address_ethereum"),
					resource.TestCheckResourceAttrSet(attributedKey, "address"),
					resource.TestCheckResourceAttrSet(attributedKey, "uri"),
				),
			},
			{
				PreConfig: func() { t.Log("step: plan-only after create (expect empty)") },
				Config:    config,
				PlanOnly:  true,
			},
			{
				PreConfig: func() { t.Log("step: rename attributed key (mutable)") },
				Config:    configRename.hcl(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(attributedKey, "name", "attributed-renamed"),
					resource.TestCheckResourceAttr(attributedKey, "folder_path", "treasury/hot"),
					resource.TestCheckResourceAttr(attributedKey, "attributes.derivation", "bip44"),
				),
			},
			{
				PreConfig: func() { t.Log("step: plan-only after rename (expect empty)") },
				Config:    configRename.hcl(),
				PlanOnly:  true,
			},
			{
				PreConfig:   func() { t.Log("step: expect error changing folder_path (RequireRecreate)") },
				Config:      configBadFolder.hcl(),
				ExpectError: immutableErr,
			},
			{
				PreConfig:   func() { t.Log("step: expect error changing attributes (RequireRecreate)") },
				Config:      configBadAttrs.hcl(),
				ExpectError: immutableErr,
			},
		},
	})
}
