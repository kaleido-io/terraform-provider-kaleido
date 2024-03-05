// Copyright © Kaleido, Inc. 2024

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

	_ "embed"
)

//go:embed runtime_test1.tf
var runtimeTest1Template string

func TestRuntime1(t *testing.T) {
	runtime1Resource := "kaleido_platform_runtime.runtime1"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + runtimeTest1Template,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(runtime1Resource, "name", `runtime1`),
					resource.TestCheckResourceAttr(runtime1Resource, "type", `besu`),
					resource.TestCheckResourceAttr(runtime1Resource, "config", `{"setting1":"value1"}`),
					resource.TestCheckResourceAttr(runtime1Resource, "log_level", `debug`),
					resource.TestCheckResourceAttr(runtime1Resource, "size", `small`),
					resource.TestCheckResourceAttr(runtime1Resource, "status", `ready`),
					resource.TestCheckResourceAttr(runtime1Resource, "false", `deleted`),
					resource.TestCheckResourceAttr(runtime1Resource, "false", `stopped`),
				),
			},
		},
	})
}
