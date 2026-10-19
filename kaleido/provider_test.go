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

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

var testAccProviders map[string]func() (tfprotov6.ProviderServer, error)

func init() {
	testAccProviders = map[string]func() (tfprotov6.ProviderServer, error){
		"kaleido": providerserver.NewProtocol6WithError(New("0.0.1-unittest", "")()),
	}
}

func testAccPreCheck(t *testing.T) {
	// Allow platform-only ACC runs (TF_ACC_PLATFORM) without requiring KALEIDO_API.
	if os.Getenv("TF_ACC_PLATFORM") != "" && os.Getenv("KALEIDO_API") == "" {
		t.Skip("skipping acceptance tests that require KALEIDO_API during TF_ACC_PLATFORM run")
	}
	if v := os.Getenv("KALEIDO_API"); v == "" {
		t.Fatal("KALEIDO_API must be set for acceptance tests")
	}
	if v := os.Getenv("KALEIDO_API_KEY"); v == "" {
		t.Fatal("KALEIDO_API_KEY must be set for acceptance tests")
	}
}
