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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"testing"

	"github.com/gorilla/mux"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func connectorFlowConfig(version, bindings, profiles string) string {
	v := ""
	if version != "" {
		v = fmt.Sprintf("  version     = %q\n", version)
	}
	d := ""
	if profiles != "" {
		d = "  config_profiles = {\n" + profiles + "  }\n"
	}
	return `
resource "kaleido_platform_connector_flow" "submission" {
  environment = "test-env"
  service     = "test-service"
  name        = "submission"
` + v + bindingsBlock(bindings) + d + `}
`
}

const (
	bindGasStandard     = "    \"evm.gasPricing\" = \"gas-standard\"\n"
	bindGasFast         = "    \"evm.gasPricing\" = \"gas-fast\"\n"
	bindPrioritize      = "    \"evm.prioritization\" = \"prioritize\"\n"
	gasStandardID       = "fcp:gasStandrd" // a config profile ID: fcp: + ten alphanumerics
	gasSelectable       = "    \"evm.gasPricing\" = { profile_id = \"" + gasStandardID + "\", jsonata = \"state.input.options.gasPricing.configProfileName\" }\n"
	gasFixedOnly        = "    \"evm.gasPricing\" = { profile_id = \"" + gasStandardID + "\" }\n"
	prioritizeID        = "fcp:prioritiz0"
	prioritizeFixedOnly = "    \"evm.prioritization\" = { profile_id = \"" + prioritizeID + "\" }\n"
	flowResource        = "kaleido_platform_connector_flow.submission"
)

func bindingsBlock(bindings string) string {
	if bindings == "" {
		return ""
	}
	return "  config_type_bindings = {\n" + bindings + "  }\n"
}

// lastFlowRequest returns the body of the most recent /deploy or /upgrade.
func lastFlowRequest(mp *mockPlatform) map[string]interface{} {
	if len(mp.connectorFlowRequests) == 0 {
		return nil
	}
	return mp.connectorFlowRequests[len(mp.connectorFlowRequests)-1]
}

func sentBinding(mp *mockPlatform, configType string) (interface{}, bool) {
	b, _ := lastFlowRequest(mp)["configTypeBindings"].(map[string]interface{})
	v, ok := b[configType]
	return v, ok
}

// An omitted version is deployed at latest and then HELD: a later binding-only change must send
// the deployed version, never nothing - or the connector service would treat it as "upgrade to
// latest" and move the flow without anyone asking.
func TestConnectorFlowUnpinned(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer mp.server.Close()

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + connectorFlowConfig("", bindGasStandard, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(flowResource, "id"),
					resource.TestCheckResourceAttr(flowResource, "flow_type", "submission"),
					resource.TestCheckResourceAttr(flowResource, "version", "2026.03.0"),
					resource.TestCheckResourceAttr(flowResource, "current_version", "2026.03.0"),
					func(*terraform.State) error {
						_, sent := lastFlowRequest(mp)["version"]
						assert.False(t, sent, "an omitted version is not sent on deploy - the service deploys latest")
						return nil
					},
				),
			},
			{
				// A newer template is now stored, and only a binding changes.
				PreConfig: func() { mp.connectorFlowTemplateVersion = "2026.04.0" },
				Config:    providerConfig + connectorFlowConfig("", bindGasFast, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(flowResource, "version", "2026.03.0"),
					resource.TestCheckResourceAttr(flowResource, "config_profile_bindings.gasPricing.config_profile_id", "cfp-gas-fast"),
					func(*terraform.State) error {
						assert.Equal(t, "2026.03.0", lastFlowRequest(mp)["version"], "a binding-only change sends the held version")
						return nil
					},
				),
			},
		},
	})
}

// With several versions stored, a pin deploys that version - not latest - and raising the pin is
// a forward upgrade carrying the version.
func TestConnectorFlowPinnedUpgrade(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer mp.server.Close()
	mp.connectorFlowTemplateVersion = "2026.04.0"

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + connectorFlowConfig("2026.03.0", bindGasStandard, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(flowResource, "version", "2026.03.0"),
					resource.TestCheckResourceAttr(flowResource, "current_version", "2026.03.0"),
					func(*terraform.State) error {
						assert.Equal(t, "2026.03.0", lastFlowRequest(mp)["version"])
						return nil
					},
				),
			},
			{
				Config: providerConfig + connectorFlowConfig("2026.04.0", bindGasStandard, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(flowResource, "version", "2026.04.0"),
					resource.TestCheckResourceAttr(flowResource, "current_version", "2026.04.0"),
					func(*terraform.State) error {
						assert.Equal(t, "2026.04.0", lastFlowRequest(mp)["version"])
						return nil
					},
				),
			},
		},
	})
}

// Lowering the pin fails at plan time - no /upgrade is sent, and the flow is never replaced.
func TestConnectorFlowVersionCannotBeLowered(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer mp.server.Close()
	mp.connectorFlowTemplateVersion = "2026.04.0"

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{Config: providerConfig + connectorFlowConfig("2026.04.0", bindGasStandard, "")},
			{
				Config:      providerConfig + connectorFlowConfig("2026.03.0", bindGasStandard, ""),
				ExpectError: regexp.MustCompile(`Connector flow version cannot be lowered`),
				PlanOnly:    true,
			},
		},
	})
	assert.Len(t, mp.connectorFlowRequests, 1, "only the deploy reached the service")
}

// A version the service does not store is refused by the service itself.
func TestConnectorFlowUnstoredVersionRefused(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer mp.server.Close()

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      providerConfig + connectorFlowConfig("2026.09.0", bindGasStandard, ""),
				ExpectError: regexp.MustCompile(`KA190157`),
			},
		},
	})
}

// A connector runtime that predates version-aware deploy ignores `version` and deploys latest.
// The post-apply check turns that into a clear failure instead of recording a version that was
// never deployed.
func TestConnectorFlowLegacyRuntimeVersionMismatch(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer mp.server.Close()
	mp.connectorFlowTemplateVersion = "2026.04.0"
	mp.connectorFlowLegacyRuntime = true

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      providerConfig + connectorFlowConfig("2026.03.0", bindGasStandard, ""),
				ExpectError: regexp.MustCompile(`(?s)Connector flow version mismatch.*does\s+not\s+support\s+deploying\s+or\s+upgrading\s+to\s+a\s+specific\s+version`),
			},
		},
	})
}

// A config_profiles entry carries both halves of one binding: a fixed profile, and a JSONata
// selection that falls back to it. One call carries both, the service assigns namePrefix, and
// removing `jsonata` from the entry clears the selection while the fixed profile stays bound.
func TestConnectorFlowConfigProfilesSelection(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer mp.server.Close()

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + connectorFlowConfig("", "", gasSelectable),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(flowResource, "config_profile_bindings.gasPricing.config_profile_id", gasStandardID),
					resource.TestCheckResourceAttr(flowResource, "config_profile_bindings.gasPricing.jsonata", "state.input.options.gasPricing.configProfileName"),
					func(*terraform.State) error {
						b, _ := sentBinding(mp, "evm.gasPricing")
						entry := b.(map[string]interface{})
						assert.Equal(t, gasStandardID, entry["configProfile"], "the ID is sent as configProfile, so the connector still verifies the profile exists")
						dm := entry["dynamicMapping"].(map[string]interface{})
						assert.Equal(t, "state.input.options.gasPricing.configProfileName", dm["jsonata"])
						_, hasPrefix := dm["namePrefix"]
						assert.False(t, hasPrefix, "namePrefix is assigned by the service, never sent")
						return nil
					},
				),
			},
			{
				Config: providerConfig + connectorFlowConfig("", "", gasFixedOnly),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(flowResource, "config_profile_bindings.gasPricing.config_profile_id", gasStandardID),
					resource.TestCheckNoResourceAttr(flowResource, "config_profile_bindings.gasPricing.jsonata"),
					func(*terraform.State) error {
						b, _ := sentBinding(mp, "evm.gasPricing")
						_, hasMapping := b.(map[string]interface{})["dynamicMapping"]
						assert.False(t, hasMapping)
						return nil
					},
				),
			},
		},
	})
}

// The two maps are one binding set: a type still in the deprecated shorthand and one in
// config_profiles go out in the same call, and a type can move between them.
func TestConnectorFlowBothMapsTogether(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer mp.server.Close()

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + connectorFlowConfig("", bindPrioritize, gasSelectable),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(flowResource, "config_profile_bindings.prioritization.config_profile_id", "cfp-prioritize"),
					resource.TestCheckResourceAttr(flowResource, "config_profile_bindings.gasPricing.jsonata", "state.input.options.gasPricing.configProfileName"),
				),
			},
			{
				Config: providerConfig + connectorFlowConfig("", bindGasStandard+bindPrioritize, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(flowResource, "config_profile_bindings.gasPricing.config_profile_id", "cfp-gas-standard"),
					resource.TestCheckNoResourceAttr(flowResource, "config_profile_bindings.gasPricing.jsonata"),
				),
			},
		},
	})
}

func TestConnectorFlowConfigProfilesValidation(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer mp.server.Close()

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      providerConfig + connectorFlowConfig("", bindGasStandard, gasFixedOnly),
				ExpectError: regexp.MustCompile(`Config type configured twice`),
				PlanOnly:    true,
			},
			{
				Config:      providerConfig + connectorFlowConfig("", "", "    \"evm.gasPricing\" = {}\n"),
				ExpectError: regexp.MustCompile(`No config profile chosen`),
				PlanOnly:    true,
			},
			{
				// A name, not an ID: binding by name would silently go stale if the profile were
				// replaced under the same name.
				Config:      providerConfig + connectorFlowConfig("", "", "    \"evm.gasPricing\" = { profile_id = \"gas-standard\" }\n"),
				ExpectError: regexp.MustCompile(`Not a config profile ID`),
				PlanOnly:    true,
			},
			{
				// A config profile TYPE's ID (fct:) is well-formed but the wrong kind of thing.
				Config:      providerConfig + connectorFlowConfig("", "", "    \"evm.gasPricing\" = { profile_id = \"fct:gasPricing\" }\n"),
				ExpectError: regexp.MustCompile(`Not a config profile ID`),
				PlanOnly:    true,
			},
		},
	})
	assert.Empty(t, mp.connectorFlowRequests, "invalid configuration never reaches the service")
}

// /upgrade leaves alone any binding it is not sent, so removing an optional binding from
// configuration must send an explicit null for it.
func TestConnectorFlowUnbindOptional(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer mp.server.Close()

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + connectorFlowConfig("", bindGasStandard+bindPrioritize, ""),
				Check:  resource.TestCheckResourceAttr(flowResource, "config_profile_bindings.prioritization.config_profile_id", "cfp-prioritize"),
			},
			{
				Config: providerConfig + connectorFlowConfig("", bindGasStandard, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr(flowResource, "config_profile_bindings.prioritization.config_profile_id"),
					func(*terraform.State) error {
						v, sent := sentBinding(mp, "evm.prioritization")
						assert.True(t, sent, "the removed optional type is sent")
						assert.Nil(t, v, "as JSON null")
						return nil
					},
				),
			},
		},
	})
}

// Removing a REQUIRED type from configuration is refused at plan time: the flow cannot run without
// it, so it could never be unbound, and leaving it bound but unconfigured would be a diff no apply
// could resolve.
func TestConnectorFlowRemovingRequiredTypeRefusedAtPlan(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer mp.server.Close()

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{Config: providerConfig + connectorFlowConfig("", "", gasFixedOnly+prioritizeFixedOnly)},
			{
				Config:      providerConfig + connectorFlowConfig("", "", prioritizeFixedOnly),
				ExpectError: regexp.MustCompile(`Required config type cannot be removed`),
				PlanOnly:    true,
			},
		},
	})
	assert.Len(t, mp.connectorFlowRequests, 1, "only the deploy reached the service")
}

// On a connector runtime that predates stored template versions, removing a config type is refused
// at plan time with a message naming the fix - not a raw 404, and never an apply that would send a
// null binding the runtime cannot handle.
func TestConnectorFlowRemovingTypeRefusedOnLegacyRuntime(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer mp.server.Close()

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{Config: providerConfig + connectorFlowConfig("", "", gasFixedOnly+prioritizeFixedOnly)},
			{
				PreConfig:   func() { mp.connectorFlowLegacyRuntime = true },
				Config:      providerConfig + connectorFlowConfig("", "", gasFixedOnly),
				ExpectError: regexp.MustCompile(`Connector runtime cannot remove config types`),
				PlanOnly:    true,
			},
		},
	})
	assert.Len(t, mp.connectorFlowRequests, 1, "only the deploy reached the service")
}

// description: omitted takes the template's (and is not a perpetual diff), a change is sent to the
// workflow itself because /upgrade has no description, and once set it is drift-checked.
func TestConnectorFlowDescription(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer mp.server.Close()
	withDescription := func(d string) string {
		return providerConfig + `
resource "kaleido_platform_connector_flow" "submission" {
  environment     = "test-env"
  service         = "test-service"
  name            = "submission"
  description     = "` + d + `"
  config_profiles = {
` + gasFixedOnly + `  }
}
`
	}
	patched := func() int {
		n := 0
		for _, c := range mp.calls {
			if c == "PATCH /endpoint/{env}/{service}/rest/api/v1/connector-flows/{flow}" {
				n++
			}
		}
		return n
	}
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + connectorFlowConfig("", "", gasFixedOnly),
				Check:  resource.TestCheckResourceAttr(flowResource, "description", mockTemplateDescription),
			},
			{
				Config: withDescription("Our submission flow"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(flowResource, "description", "Our submission flow"),
					func(*terraform.State) error {
						assert.Equal(t, 1, patched(), "the description change was sent")
						assert.Equal(t, "Our submission flow", mp.connectorFlows["test-env/test-service/submission"].Description)
						return nil
					},
				),
			},
			{
				PreConfig:          func() { mp.connectorFlows["test-env/test-service/submission"].Description = "changed in the UI" },
				Config:             withDescription("Our submission flow"),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			{
				Config: withDescription("Our submission flow"),
				Check:  resource.TestCheckResourceAttr(flowResource, "description", "Our submission flow"),
			},
		},
	})
}

// outOfBand changes the mock service's stored bindings directly, the way the UI or the workflow
// engine's per-slot API would, bypassing Terraform.
func outOfBand(mp *mockPlatform, change func(b map[string]ConfigProfileBindingTargetAPIModel)) func() {
	return func() { change(mp.connectorFlows["test-env/test-service/submission"].ConfigProfileBindings) }
}

// Each out-of-band change to a binding config_profiles manages shows as a plan diff, and the next
// apply reverts it - the flow resource owns its bindings.
func TestConnectorFlowDriftIsDetectedAndReverted(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(b map[string]ConfigProfileBindingTargetAPIModel)
		check  resource.TestCheckFunc
	}{
		{
			name: "dynamic mapping added",
			change: func(b map[string]ConfigProfileBindingTargetAPIModel) {
				g := b["gasPricing"]
				g.DynamicMapping = &ConfigProfileDynamicMappingAPIModel{JSONata: "state.input.injected"}
				b["gasPricing"] = g
			},
			check: resource.TestCheckNoResourceAttr(flowResource, "config_profile_bindings.gasPricing.jsonata"),
		},
		{
			name: "profile re-pointed",
			change: func(b map[string]ConfigProfileBindingTargetAPIModel) {
				b["gasPricing"] = ConfigProfileBindingTargetAPIModel{ConfigProfileID: "fcp:someoneElse"}
			},
			check: resource.TestCheckResourceAttr(flowResource, "config_profile_bindings.gasPricing.config_profile_id", gasStandardID),
		},
		{
			name: "optional type bound",
			change: func(b map[string]ConfigProfileBindingTargetAPIModel) {
				b["prioritization"] = ConfigProfileBindingTargetAPIModel{ConfigProfileID: prioritizeID}
			},
			check: resource.TestCheckNoResourceAttr(flowResource, "config_profile_bindings.prioritization.config_profile_id"),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mp, providerConfig := testSetup(t)
			defer mp.server.Close()
			config := providerConfig + connectorFlowConfig("", "", gasFixedOnly)
			resource.Test(t, resource.TestCase{
				IsUnitTest:               true,
				ProtoV6ProviderFactories: testAccProviders,
				Steps: []resource.TestStep{
					{Config: config},
					{PreConfig: outOfBand(mp, tc.change), Config: config, PlanOnly: true, ExpectNonEmptyPlan: true},
					{Config: config, Check: tc.check},
				},
			})
		})
	}
}

// An optional binding config_profiles declares, unbound outside Terraform, is bound again.
func TestConnectorFlowDriftUnboundOptionalIsRebound(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer mp.server.Close()
	config := providerConfig + connectorFlowConfig("", "", gasFixedOnly+prioritizeFixedOnly)
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{Config: config},
			{
				PreConfig:          outOfBand(mp, func(b map[string]ConfigProfileBindingTargetAPIModel) { delete(b, "prioritization") }),
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			{Config: config, Check: resource.TestCheckResourceAttr(flowResource, "config_profile_bindings.prioritization.config_profile_id", prioritizeID)},
		},
	})
}

// Types in the deprecated config_type_bindings are not checked for drift: that map holds profile
// names, the service stores IDs, so there is nothing to compare. Pinned so a change is deliberate.
func TestConnectorFlowDriftNotCheckedForShorthand(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer mp.server.Close()
	config := providerConfig + connectorFlowConfig("", bindGasStandard, "")
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{Config: config},
			{
				PreConfig: outOfBand(mp, func(b map[string]ConfigProfileBindingTargetAPIModel) {
					b["gasPricing"] = ConfigProfileBindingTargetAPIModel{ConfigProfileID: "fcp:someoneElse"}
				}),
				Config:   config,
				PlanOnly: true, // and an empty plan
			},
		},
	})
}

// A connector runtime without stored template versions has no route to read, so its bindings are
// simply not checked - a refresh must never fail because of it.
func TestConnectorFlowDriftSkippedOnLegacyRuntime(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer mp.server.Close()
	config := providerConfig + connectorFlowConfig("", "", gasFixedOnly)
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{Config: config},
			{PreConfig: func() { mp.connectorFlowLegacyRuntime = true }, Config: config, PlanOnly: true},
		},
	})
}

// Slots of one config type normally agree - every shipped flow has one slot per type - but if they
// do not, the value chosen must differ from the expected entry so the plan shows the drift.
func TestObservedForType(t *testing.T) {
	a := observedBinding{profileID: "fcp:aaaaaaaaaa"}
	b := observedBinding{profileID: "fcp:bbbbbbbbbb"}
	assert.Equal(t, a, observedForType([]observedBinding{a}, &a), "agreeing slot: no drift")
	assert.Equal(t, b, observedForType([]observedBinding{a, b}, &a), "one slot drifted: its value surfaces")
	assert.Equal(t, b, observedForType([]observedBinding{b, a}, &a), "whichever order")
	assert.Equal(t, a, observedForType([]observedBinding{a, b}, nil), "nothing expected: first slot")
}

// The "configured twice" check is repeated at apply, where every value is known. ValidateConfig
// cannot see an overlap when config_profiles is unknown at plan time; this proves collectBindings
// refuses it rather than letting the config_profiles entry silently win.
func TestCollectBindingsRefusesTypeInBothMapsAtApply(t *testing.T) {
	ctx := context.Background()
	entryType := types.ObjectType{AttrTypes: configProfileEntryAttrTypes}
	data := ConnectorFlowResourceModel{
		ConfigTypeBindings: types.MapValueMust(types.StringType, map[string]attr.Value{
			"evm.gasPricing": types.StringValue("gas-standard"),
		}),
		ConfigProfiles: types.MapValueMust(entryType, map[string]attr.Value{
			"evm.gasPricing": types.ObjectValueMust(configProfileEntryAttrTypes, map[string]attr.Value{
				"profile_id": types.StringValue(gasStandardID),
				"jsonata":    types.StringNull(),
			}),
		}),
	}
	var diags diag.Diagnostics
	bindings := (&connectorFlowResource{}).collectBindings(ctx, &data, &diags)
	assert.Nil(t, bindings, "nothing is sent")
	require.True(t, diags.HasError())
	assert.Contains(t, diags.Errors()[0].Summary(), "Config type configured twice")
}

// Mock server: models a version-aware connector service. A single "submission" template is
// catalogued at two versions with the same binding shape - one required and one optional config
// type - and every catalogue version at or below mp.connectorFlowTemplateVersion is stored.

type mockFlowTemplate struct {
	required map[string]string // binding name -> config type
	optional map[string]string
}

var connectorFlowCatalogue = map[string]mockFlowTemplate{
	"2026.03.0": {required: map[string]string{"gasPricing": "evm.gasPricing"}, optional: map[string]string{"prioritization": "evm.prioritization"}},
	"2026.04.0": {required: map[string]string{"gasPricing": "evm.gasPricing"}, optional: map[string]string{"prioritization": "evm.prioritization"}},
}

func (mp *mockPlatform) storedFlowTemplate(v string) (mockFlowTemplate, bool) {
	tmpl, ok := connectorFlowCatalogue[v]
	if !ok {
		return tmpl, false
	}
	return tmpl, !version.Must(version.NewVersion(v)).GreaterThan(version.Must(version.NewVersion(mp.connectorFlowTemplateVersion)))
}

func (mp *mockPlatform) connectorFlowKey(vars map[string]string) string {
	return vars["env"] + "/" + vars["service"] + "/" + vars["flow"]
}

func (mp *mockPlatform) readFlowRequest(req *http.Request, into interface{}) {
	raw, err := io.ReadAll(req.Body)
	assert.NoError(mp.t, err)
	var generic map[string]interface{}
	assert.NoError(mp.t, json.Unmarshal(raw, &generic))
	mp.connectorFlowRequests = append(mp.connectorFlowRequests, generic)
	assert.NoError(mp.t, json.Unmarshal(raw, into))
}

func (mp *mockPlatform) flowError(res http.ResponseWriter, status int, msg string) {
	mp.respond(res, map[string]string{"error": msg}, status)
}

// applyFlowBindings upserts each named config type's target onto every binding name it fans out
// to - replacing the whole target, as the workflow engine does - and treats a nil entry as unbind.
func (mp *mockPlatform) applyFlowBindings(res http.ResponseWriter, flow *ConnectorFlowAPIModel, tmpl mockFlowTemplate, in map[string]*ConfigProfileBindingTargetInput) bool {
	for configType, target := range in {
		for _, set := range []map[string]string{tmpl.required, tmpl.optional} {
			for bindingName, ct := range set {
				if ct != configType {
					continue
				}
				if target == nil {
					if _, isRequired := tmpl.required[bindingName]; isRequired {
						mp.flowError(res, http.StatusBadRequest, fmt.Sprintf("KA190165: Config type '%s' is required by this connector flow and cannot be unbound", configType))
						return false
					}
					delete(flow.ConfigProfileBindings, bindingName)
					continue
				}
				b := ConfigProfileBindingTargetAPIModel{}
				if configProfileIDPattern.MatchString(target.ConfigProfile) {
					b.ConfigProfileID = target.ConfigProfile // already an ID
				} else if target.ConfigProfile != "" {
					b.ConfigProfileID = "cfp-" + target.ConfigProfile // a name, as the deprecated shorthand sends
				}
				if target.DynamicMapping != nil {
					b.DynamicMapping = &ConfigProfileDynamicMappingAPIModel{NamePrefix: "test-service/", JSONata: target.DynamicMapping.JSONata}
				}
				flow.ConfigProfileBindings[bindingName] = b
			}
		}
	}
	return true
}

func (mp *mockPlatform) deployConnectorFlow(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	var body ConnectorFlowDeployAPIModel
	mp.readFlowRequest(req, &body)
	v := body.Version
	if v == "" || mp.connectorFlowLegacyRuntime {
		v = mp.connectorFlowTemplateVersion
	}
	tmpl, ok := mp.storedFlowTemplate(v)
	if !ok {
		mp.flowError(res, http.StatusNotFound, fmt.Sprintf("KA190157: Version '%s' of connector_flows template '%s' not found", v, vars["flow"]))
		return
	}
	for _, ct := range tmpl.required {
		if body.ConfigTypeBindings[ct] == nil {
			mp.flowError(res, http.StatusBadRequest, fmt.Sprintf("KA190111: Missing config profile binding for config type '%s'", ct))
			return
		}
	}
	description := body.Description
	if description == "" {
		description = mockTemplateDescription // the server starts from the template's own description
	}
	flow := &ConnectorFlowAPIModel{
		ID:                    "wfl:" + vars["flow"],
		Name:                  vars["flow"],
		FlowType:              vars["flow"],
		CurrentVersion:        v,
		Description:           description,
		Labels:                map[string]string{connectorReferenceVersionLabel: v},
		ConfigProfileBindings: map[string]ConfigProfileBindingTargetAPIModel{},
	}
	if !mp.applyFlowBindings(res, flow, tmpl, body.ConfigTypeBindings) {
		return
	}
	mp.connectorFlows[mp.connectorFlowKey(vars)] = flow
	mp.respond(res, flow, http.StatusOK)
}

func (mp *mockPlatform) upgradeConnectorFlow(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	flow := mp.connectorFlows[mp.connectorFlowKey(vars)]
	if flow == nil {
		mp.respond(res, nil, http.StatusNotFound)
		return
	}
	var body ConnectorFlowUpgradeAPIModel
	mp.readFlowRequest(req, &body)
	v := body.Version
	if v == "" || mp.connectorFlowLegacyRuntime {
		v = mp.connectorFlowTemplateVersion
	}
	tmpl, ok := mp.storedFlowTemplate(v)
	if !ok {
		mp.flowError(res, http.StatusNotFound, fmt.Sprintf("KA190157: Version '%s' of connector_flows template '%s' not found", v, vars["flow"]))
		return
	}
	target := version.Must(version.NewVersion(v))
	current := version.Must(version.NewVersion(flow.Labels[connectorReferenceVersionLabel]))
	if target.LessThan(current) {
		mp.flowError(res, http.StatusBadRequest, fmt.Sprintf("KA190159: Connector flow '%s' is at version '%s'; cannot upgrade to older version '%s'", flow.Name, current, v))
		return
	}
	versionChanged := target.GreaterThan(current)
	if !versionChanged && (len(body.ConfigTypeBindings) == 0 || mp.connectorFlowLegacyRuntime) {
		mp.respond(res, flow, http.StatusOK) // no-op - or, on a legacy runtime, bindings silently dropped (BUG-2)
		return
	}
	if !mp.applyFlowBindings(res, flow, tmpl, body.ConfigTypeBindings) {
		return
	}
	if versionChanged {
		flow.CurrentVersion = v
		flow.Labels[connectorReferenceVersionLabel] = v
	}
	mp.respond(res, flow, http.StatusOK)
}

func (mp *mockPlatform) getConnectorFlowTemplate(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	if mp.connectorFlowLegacyRuntime && vars["version"] != "" {
		mp.respond(res, nil, http.StatusNotFound) // no stored template versions on a legacy runtime
		return
	}
	v := vars["version"]
	if v == "" {
		v = mp.connectorFlowTemplateVersion
	}
	tmpl, ok := mp.storedFlowTemplate(v)
	if !ok {
		mp.flowError(res, http.StatusNotFound, fmt.Sprintf("KA190157: Version '%s' of connector_flows template '%s' not found", v, vars["flow"]))
		return
	}
	mp.respond(res, &ConnectorFlowTemplateAPIModel{Version: v, RequiredConfigProfiles: tmpl.required, OptionalConfigProfiles: tmpl.optional}, http.StatusOK)
}

const mockTemplateDescription = "Submits transactions to the chain"

// patchConnectorFlow is the workflow engine's sparse update of a deployed flow.
func (mp *mockPlatform) patchConnectorFlow(res http.ResponseWriter, req *http.Request) {
	flow := mp.connectorFlows[mp.connectorFlowKey(mux.Vars(req))]
	if flow == nil {
		mp.respond(res, nil, http.StatusNotFound)
		return
	}
	var body map[string]string
	mp.getBody(req, &body)
	if d, ok := body["description"]; ok {
		flow.Description = d
	}
	mp.respond(res, flow, http.StatusOK)
}

func (mp *mockPlatform) getConnectorFlow(res http.ResponseWriter, req *http.Request) {
	flow := mp.connectorFlows[mp.connectorFlowKey(mux.Vars(req))]
	if flow == nil {
		mp.respond(res, nil, http.StatusNotFound)
		return
	}
	mp.respond(res, flow, http.StatusOK)
}

func (mp *mockPlatform) deleteConnectorFlow(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	key := mp.connectorFlowKey(vars)
	if mp.connectorFlows[key] == nil {
		mp.respond(res, nil, http.StatusNotFound)
		return
	}
	delete(mp.connectorFlows, key)
	mp.respond(res, nil, http.StatusNoContent)
}
