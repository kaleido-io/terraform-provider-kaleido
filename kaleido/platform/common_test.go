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

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
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
