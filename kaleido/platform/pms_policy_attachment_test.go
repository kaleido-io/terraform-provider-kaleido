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
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	_ "embed"
)

var pms_policy_attachment_step1 = `
resource "kaleido_platform_pms_policy_attachment" "test_attachment" {
  environment = "test-env"
  service = "test-service"
  policy_deployment_id = "pmd123"
  type = "wallet_id"
  attachment_point = "wal:1234567890"
}
`

var pms_policy_attachment_step2 = `
resource "kaleido_platform_pms_policy_attachment" "test_attachment" {
  environment = "test-env"
  service = "test-service"
  policy_deployment_id = "pmd123"
  type = "asset_id"
  attachment_point = "was:1234567890"
}
`

func TestPMSPolicyAttachment1(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"POST /endpoint/{env}/{service}/rest/api/v1/policy-deployments/{policyDeploymentId}/add-attachment",
			"POST /endpoint/{env}/{service}/rest/api/v1/policy-deployments/{policyDeploymentId}/remove-attachment",
		})
		mp.server.Close()
	}()

	// Add attachment point handlers
	mp.register("/endpoint/{env}/{service}/rest/api/v1/policy-deployments/{policyDeploymentId}/add-attachment", "POST", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mp.register("/endpoint/{env}/{service}/rest/api/v1/policy-deployments/{policyDeploymentId}/remove-attachment", "POST", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	pms_policy_attachment_resource := "kaleido_platform_pms_policy_attachment.test_attachment"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + pms_policy_attachment_step1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(pms_policy_attachment_resource, "id", "pmd123:wallet_id:wal:1234567890"),
					resource.TestCheckResourceAttr(pms_policy_attachment_resource, "environment", "test-env"),
					resource.TestCheckResourceAttr(pms_policy_attachment_resource, "service", "test-service"),
					resource.TestCheckResourceAttr(pms_policy_attachment_resource, "policy_deployment_id", "pmd123"),
					resource.TestCheckResourceAttr(pms_policy_attachment_resource, "type", "wallet_id"),
					resource.TestCheckResourceAttr(pms_policy_attachment_resource, "name", "wal:1234567890"),
				),
			},
		},
	})
}

func TestPMSPolicyAttachment2(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"POST /endpoint/{env}/{service}/rest/api/v1/policy-deployments/{policyDeploymentId}/add-attachment",
			"POST /endpoint/{env}/{service}/rest/api/v1/policy-deployments/{policyDeploymentId}/remove-attachment",
			"POST /endpoint/{env}/{service}/rest/api/v1/policy-deployments/{policyDeploymentId}/add-attachment",
			"POST /endpoint/{env}/{service}/rest/api/v1/policy-deployments/{policyDeploymentId}/remove-attachment",
		})
		mp.server.Close()
	}()

	// Add attachment point handlers
	mp.register("/endpoint/{env}/{service}/rest/api/v1/policy-deployments/{policyDeploymentId}/add-attachment", "POST", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mp.register("/endpoint/{env}/{service}/rest/api/v1/policy-deployments/{policyDeploymentId}/remove-attachment", "POST", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	pms_policy_attachment_resource := "kaleido_platform_pms_policy_attachment.test_attachment"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + pms_policy_attachment_step1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(pms_policy_attachment_resource, "id", "pmd123:wallet_id:wal:1234567890"),
					resource.TestCheckResourceAttr(pms_policy_attachment_resource, "environment", "test-env"),
					resource.TestCheckResourceAttr(pms_policy_attachment_resource, "service", "test-service"),
					resource.TestCheckResourceAttr(pms_policy_attachment_resource, "policy_deployment_id", "pmd123"),
					resource.TestCheckResourceAttr(pms_policy_attachment_resource, "type", "wallet_id"),
					resource.TestCheckResourceAttr(pms_policy_attachment_resource, "name", "wal:1234567890"),
				),
			},
			{
				Config: providerConfig + pms_policy_attachment_step2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(pms_policy_attachment_resource, "id", "pmd123:asset_id:was:1234567890"),
					resource.TestCheckResourceAttr(pms_policy_attachment_resource, "environment", "test-env"),
					resource.TestCheckResourceAttr(pms_policy_attachment_resource, "service", "test-service"),
					resource.TestCheckResourceAttr(pms_policy_attachment_resource, "policy_deployment_id", "pmd123"),
					resource.TestCheckResourceAttr(pms_policy_attachment_resource, "type", "asset_id"),
					resource.TestCheckResourceAttr(pms_policy_attachment_resource, "name", "was:1234567890"),
				),
			},
		},
	})
}
