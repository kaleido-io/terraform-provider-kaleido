// Copyright 2020 Kaleido, a ConsenSys business

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
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/kaleido-io/terraform-provider-kaleido/kaleido/kaleidobase"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

var testAccProviders map[string]func() (tfprotov6.ProviderServer, error)

func testSetup(t *testing.T) (mp *mockPlatform, providerConfig string) {
	mp = startMockPlatformServer(t)
	providerConfig = fmt.Sprintf(`
provider "kaleido" {
	platform_api = "%s"
}
`,
		mp.server.URL)
	return
}

func init() {
	kaleidoProvider := kaleidobase.New(
		"0.0.1-unittest",
		"",
		Resources(),
		DataSources(),
	)
	testAccProviders = map[string]func() (tfprotov6.ProviderServer, error){
		"kaleido": providerserver.NewProtocol6WithError(kaleidoProvider),
	}
}

func testJSONEqual(t *testing.T, obj interface{}, expected string) {
	assert.NotNil(t, obj)
	jsonObj, err := json.Marshal(obj)
	assert.NoError(t, err)
	t.Logf("%s\n", jsonObj)
	assert.JSONEq(t, expected, string(jsonObj))
}

func testYAMLEqual(t *testing.T, obj interface{}, expected string) {
	assert.NotNil(t, obj)
	yamlObj, err := yaml.Marshal(obj)
	assert.NoError(t, err)
	assert.YAMLEq(t, expected, string(yamlObj))
}

func TestPatchString(t *testing.T) {
	assert.Nil(t, patchString(types.StringValue("a"), types.StringValue("a")), "unchanged is left out")
	assert.Nil(t, patchString(types.StringUnknown(), types.StringValue("a")), "unknown is left out")
	assert.Nil(t, patchString(types.StringNull(), types.StringNull()), "still unset is left out")
	if changed := patchString(types.StringValue("b"), types.StringValue("a")); assert.NotNil(t, changed) {
		assert.Equal(t, "b", *changed)
	}
	if removed := patchString(types.StringNull(), types.StringValue("a")); assert.NotNil(t, removed) {
		assert.Equal(t, "", *removed, "removed is sent empty to clear it")
	}
}

func TestPatchChanged(t *testing.T) {
	list := func(values ...string) types.List {
		elements := make([]attr.Value, len(values))
		for i, v := range values {
			elements[i] = types.StringValue(v)
		}
		return types.ListValueMust(types.StringType, elements)
	}
	assert.False(t, patchChanged(list("a"), list("a")))
	assert.False(t, patchChanged(types.ListUnknown(types.StringType), list("a")))
	assert.True(t, patchChanged(list("a", "b"), list("a")))
	assert.True(t, patchChanged(types.ListNull(types.StringType), list("a")))
}
