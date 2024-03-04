// Copyright © Kaleido, Inc. 2018, 2024

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package kaleido

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

var testAccProvider provider.Provider
var testAccProviders map[string]provider.Provider

func init() {
	testAccProvider = New("0.0.1-unittest")()
	testAccProviders = map[string]provider.Provider{
		"kaleido": testAccProvider,
	}
}

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("KALEIDO_API"); v == "" {
		t.Fatal("KALEIDO_API must be set for acceptance tests")
	}
	if v := os.Getenv("KALEIDO_API_KEY"); v == "" {
		t.Fatal("KALEIDO_API_KEY must be set for acceptance tests")
	}

	err := testAccProvider.Configure(terraform.NewResourceConfigRaw(nil))
	if err != nil {
		t.Fatal(err)
	}
}
