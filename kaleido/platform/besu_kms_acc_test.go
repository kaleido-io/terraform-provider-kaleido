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
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

type besuKMSAccConfig struct {
	envName, netName, stackName string
	chainID                     int
	keyName                     string
	forceDelete                 bool
}

func (c besuKMSAccConfig) hcl() string {
	forceNet, forceRT, forceSvc := "", "", ""
	if c.forceDelete {
		forceNet = "\n  force_delete = true"
		forceRT = "\n  force_delete = true"
		forceSvc = "\n  force_delete = true"
	}
	return testAccPlatformProviderConfig() + fmt.Sprintf(`
resource "kaleido_platform_environment" "acc" {
  name = %q
}

resource "kaleido_platform_network" "besu" {
  type        = "BesuNetwork"
  name        = %q
  environment = kaleido_platform_environment.acc.id
  init_mode   = "automated"%s

  config_json = jsonencode({
    chainID = %d
    bootstrapOptions = {
      qbft = {
        blockperiodseconds = 5
      }
      eipBlockConfig = {
        shanghaiTime = 0, 
        osakaTime = 0
      }
      blockConfigFlags = {
        zeroBaseFee = true
      }
    }
  })
}

resource "kaleido_platform_stack" "besu" {
  environment = kaleido_platform_environment.acc.id
  name        = %q
  type        = "chain_infrastructure"
  sub_type    = "BesuStack"
  network_id  = kaleido_platform_network.besu.id
}

resource "kaleido_platform_runtime" "besu" {
  type        = "BesuNode"
  name        = "node-0"
  environment = kaleido_platform_environment.acc.id
  config_json = jsonencode({})
  stack_id    = kaleido_platform_stack.besu.id%s
}

resource "kaleido_platform_service" "besu" {
  type        = "BesuNode"
  name        = "node-0"
  environment = kaleido_platform_environment.acc.id
  runtime     = kaleido_platform_runtime.besu.id
  stack_id    = kaleido_platform_stack.besu.id%s
  config_json = jsonencode({
    network = {
      id = kaleido_platform_network.besu.id
    }
    signer            = true
    customBesuArgs    = ["--revert-reason-enabled"]
    dataStorageFormat = "BONSAI"
    logLevel          = "INFO"
  })
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

resource "kaleido_platform_kms_wallet" "acc" {
  type        = "hdwallet"
  name        = "infra"
  environment = kaleido_platform_environment.acc.id
  service     = kaleido_platform_service.kms.id
  config_json = jsonencode({})
}

resource "kaleido_platform_kms_key" "acc" {
  name        = %q
  environment = kaleido_platform_environment.acc.id
  service     = kaleido_platform_service.kms.id
  wallet      = kaleido_platform_kms_wallet.acc.id
}
`, c.envName, c.netName, forceNet, c.chainID, c.stackName, forceRT, forceSvc, c.keyName)
}

func TestAccPlatformBesuAndKMS(t *testing.T) {
	// Stream Terraform CLI + provider logs
	// Callers can override with a noisier level (e.g. TF_LOG=DEBUG).
	if os.Getenv("TF_LOG") == "" {
		t.Setenv("TF_LOG", "INFO")
	}
	if os.Getenv("TF_LOG_PROVIDER") == "" {
		t.Setenv("TF_LOG_PROVIDER", "INFO")
	}

	suffix := testAccNameSuffix()
	base := besuKMSAccConfig{
		envName:   fmt.Sprintf("tf-acc-besu-%s", suffix),
		netName:   fmt.Sprintf("tf-acc-net-%s", suffix),
		stackName: fmt.Sprintf("tf-acc-stack-%s", suffix),
		chainID:   10000 + (int(suffix[0]) % 20000),
		keyName:   "deployer",
	}

	config := base.hcl()
	configUpdate := base
	configUpdate.keyName = "deployer-renamed"
	configForceDelete := configUpdate
	configForceDelete.forceDelete = true

	envResource := "kaleido_platform_environment.acc"
	netResource := "kaleido_platform_network.besu"
	stackResource := "kaleido_platform_stack.besu"
	besuRuntime := "kaleido_platform_runtime.besu"
	besuService := "kaleido_platform_service.besu"
	kmsRuntime := "kaleido_platform_runtime.kms"
	kmsService := "kaleido_platform_service.kms"
	walletResource := "kaleido_platform_kms_wallet.acc"
	keyResource := "kaleido_platform_kms_key.acc"

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
				PreConfig: func() { t.Log("step: create Besu network/node + KMS wallet/key") },
				Config:    config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(envResource, "id"),
					resource.TestCheckResourceAttrSet(netResource, "id"),
					resource.TestCheckResourceAttr(netResource, "type", "BesuNetwork"),
					resource.TestCheckResourceAttrSet(stackResource, "id"),
					resource.TestCheckResourceAttr(stackResource, "type", "chain_infrastructure"),
					resource.TestCheckResourceAttrSet(besuRuntime, "id"),
					resource.TestCheckResourceAttr(besuRuntime, "type", "BesuNode"),
					resource.TestCheckResourceAttrSet(besuService, "id"),
					resource.TestCheckResourceAttr(besuService, "type", "BesuNode"),
					resource.TestCheckResourceAttrSet(kmsRuntime, "id"),
					resource.TestCheckResourceAttr(kmsRuntime, "type", "KeyManager"),
					resource.TestCheckResourceAttrSet(kmsService, "id"),
					resource.TestCheckResourceAttr(kmsService, "type", "KeyManager"),
					resource.TestCheckResourceAttrSet(walletResource, "id"),
					resource.TestCheckResourceAttr(walletResource, "type", "hdwallet"),
					resource.TestCheckResourceAttrSet(keyResource, "id"),
					resource.TestCheckResourceAttr(keyResource, "name", "deployer"),
				),
			},
			// Catch omit-optional / platform-default drift after create.
			{
				PreConfig: func() { t.Log("step: plan-only after create (expect empty)") },
				Config:    config,
				PlanOnly:  true,
			},
			{
				PreConfig: func() { t.Log("step: rename KMS key") },
				Config:    configUpdate.hcl(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(keyResource, "name", "deployer-renamed"),
				),
			},
			{
				PreConfig: func() { t.Log("step: plan-only after rename (expect empty)") },
				Config:    configUpdate.hcl(),
				PlanOnly:  true,
			},
			// Besu validators / networks refuse delete unless force_delete is applied first as a safety precaution.
			{
				PreConfig: func() {
					t.Log("step: apply force_delete=true on Besu network/runtime/service before destroy")
				},
				Config: configForceDelete.hcl(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(netResource, "force_delete", "true"),
					resource.TestCheckResourceAttr(besuRuntime, "force_delete", "true"),
					resource.TestCheckResourceAttr(besuService, "force_delete", "true"),
				),
			},
		},
	})
}
