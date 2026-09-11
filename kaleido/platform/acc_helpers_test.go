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
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/kaleido-io/terraform-provider-kaleido/kaleido/kaleidobase"
)

func testAccPlatformPreCheck(t *testing.T) {
	t.Helper()
	if os.Getenv("TF_ACC_PLATFORM") == "" {
		t.Skip("TF_ACC_PLATFORM must be set to run Platform acceptance tests")
	}
	if os.Getenv("KALEIDO_PLATFORM_API") == "" {
		t.Fatal("KALEIDO_PLATFORM_API must be set for Platform acceptance tests")
	}
	hasBasic := os.Getenv("KALEIDO_PLATFORM_USERNAME") != "" && os.Getenv("KALEIDO_PLATFORM_PASSWORD") != ""
	hasBearer := os.Getenv("KALEIDO_PLATFORM_BEARER_TOKEN") != ""
	if !hasBasic && !hasBearer {
		t.Fatal("set KALEIDO_PLATFORM_USERNAME/KALEIDO_PLATFORM_PASSWORD or KALEIDO_PLATFORM_BEARER_TOKEN for Platform acceptance tests")
	}
}

func testAccPlatformProviderConfig() string {
	var b strings.Builder
	b.WriteString("provider \"kaleido\" {\n")
	b.WriteString(fmt.Sprintf("  platform_api = %q\n", os.Getenv("KALEIDO_PLATFORM_API")))
	if u := os.Getenv("KALEIDO_PLATFORM_USERNAME"); u != "" {
		b.WriteString(fmt.Sprintf("  platform_username = %q\n", u))
	}
	if p := os.Getenv("KALEIDO_PLATFORM_PASSWORD"); p != "" {
		b.WriteString(fmt.Sprintf("  platform_password = %q\n", p))
	}
	if tok := os.Getenv("KALEIDO_PLATFORM_BEARER_TOKEN"); tok != "" {
		b.WriteString(fmt.Sprintf("  platform_bearer_token = %q\n", tok))
	}
	b.WriteString("}\n")
	return b.String()
}

func testAccPlatformClient() *resty.Client {
	return kaleidobase.NewProviderData(context.Background(), &kaleidobase.ProviderModel{}, "acc-test", "").Platform
}

// testAccCheckPlatformAPIGone expects GET path to eventually be 404 after destroy.
func testAccCheckPlatformAPIGone(pathFn func(s *terraform.State) (string, error)) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		path, err := pathFn(s)
		if err != nil {
			return err
		}
		if path == "" {
			return nil
		}
		client := testAccPlatformClient()
		res, err := client.R().Get(path)
		if err != nil {
			return err
		}
		if res.StatusCode() != http.StatusNotFound {
			return fmt.Errorf("expected %s to be gone (404), got %d", path, res.StatusCode())
		}
		return nil
	}
}

func testAccResourceID(s *terraform.State, name string) (string, error) {
	rs, ok := s.RootModule().Resources[name]
	if !ok {
		return "", fmt.Errorf("resource not found in state: %s", name)
	}
	if rs.Primary.ID == "" {
		return "", fmt.Errorf("resource %s has empty ID", name)
	}
	return rs.Primary.ID, nil
}
