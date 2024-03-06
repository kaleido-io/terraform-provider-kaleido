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
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
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
	kaleidoProvider := New(
		"0.0.1-unittest",
	)
	testAccProviders = map[string]func() (tfprotov6.ProviderServer, error){
		"kaleido": providerserver.NewProtocol6WithError(kaleidoProvider),
	}
}
