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
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"github.com/stretchr/testify/assert"
)

func TestDeepMergeJSON(t *testing.T) {
	for _, tc := range []struct {
		name, base, overlay, expected string
	}{
		{"overlay replaces a scalar, keeps the rest", `{"count":12,"resubmission":{"enabled":true,"timeout":"5m"}}`, `{"count":20}`, `{"count":20,"resubmission":{"enabled":true,"timeout":"5m"}}`},
		{"nested objects merge", `{"count":12,"resubmission":{"enabled":true}}`, `{"resubmission":{"timeout":"1m"}}`, `{"count":12,"resubmission":{"enabled":true,"timeout":"1m"}}`},
		{"null at any level is not set", `{"count":12,"resubmission":{"enabled":true}}`, `{"count":null,"resubmission":{"enabled":null,"timeout":null}}`, `{"count":12,"resubmission":{"enabled":true}}`},
		{"a null overlay keeps base", `{"count":12}`, `null`, `{"count":12}`},
		{"false, zero and empty string are set", `{"enabled":true,"count":12,"name":"x"}`, `{"enabled":false,"count":0,"name":""}`, `{"enabled":false,"count":0,"name":""}`},
		{"a list replaces the whole list", `{"tiers":[{"l":"a"},{"l":"b"}]}`, `{"tiers":[{"l":"c"}]}`, `{"tiers":[{"l":"c"}]}`},
		{"an empty list is set", `{"tiers":[{"l":"a"}]}`, `{"tiers":[]}`, `{"tiers":[]}`},
		{"maps merge key by key", `{"httpHeaders":{"X-A":"1","X-B":"2"}}`, `{"httpHeaders":{"X-B":"9","X-C":"3"}}`, `{"httpHeaders":{"X-A":"1","X-B":"9","X-C":"3"}}`},
		{"no depth limit", `{"a":{"b":{"c":{"d":{"e":{"f":{"g":{"h":{"i":{"j":{"k":{"l":1,"m":2}}}}}}}}}}}}`, `{"a":{"b":{"c":{"d":{"e":{"f":{"g":{"h":{"i":{"j":{"k":{"l":9}}}}}}}}}}}}`, `{"a":{"b":{"c":{"d":{"e":{"f":{"g":{"h":{"i":{"j":{"k":{"l":9,"m":2}}}}}}}}}}}}`},
		{"an object replaces a scalar, without its nulls", `{"source":"none"}`, `{"source":{"rpcEndpoint":{"cache":null}}}`, `{"source":{"rpcEndpoint":{}}}`},
		{"a scalar replaces an object", `{"source":{"rpcEndpoint":{}}}`, `{"source":"none"}`, `{"source":"none"}`},
		{"empty base", `{}`, `{"count":3,"resubmission":null}`, `{"count":3}`},
		{"empty overlay", `{"count":12}`, `{}`, `{"count":12}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			base, err := decodeJSONExact(tc.base)
			assert.NoError(t, err)
			overlay, err := decodeJSONExact(tc.overlay)
			assert.NoError(t, err)
			testJSONEqual(t, deepMergeJSON(base, overlay), tc.expected)
		})
	}
}

func TestMergeJSONFunction(t *testing.T) {
	config := func(base, overlay string) string {
		return fmt.Sprintf(`
output "merged" {
  value = provider::kaleido::merge_json(%q, %q)
}
`, base, overlay)
	}
	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_8_0)},
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: config(`{"count":12,"resubmission":{"enabled":true}}`, `{"count":20,"resubmission":{"timeout":"1m"}}`),
				Check:  resource.TestCheckOutput("merged", `{"count":20,"resubmission":{"enabled":true,"timeout":"1m"}}`),
			},
			{
				// A number beyond float64 precision comes through exactly.
				Config: config(`{"caps":{"maxFeePerGas":1}}`, `{"caps":{"maxFeePerGas":123456789012345678901234567890}}`),
				Check:  resource.TestCheckOutput("merged", `{"caps":{"maxFeePerGas":123456789012345678901234567890}}`),
			},
			{
				Config:      config(`{"count":`, `{}`),
				ExpectError: regexp.MustCompile(`base is not valid JSON`),
			},
			{
				Config:      config(`{}`, `{} {}`),
				ExpectError: regexp.MustCompile(`overlay is not valid JSON`),
			},
		},
	})
}
