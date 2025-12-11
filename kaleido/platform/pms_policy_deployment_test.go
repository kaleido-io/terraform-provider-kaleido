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
	"time"

	"github.com/aidarkhanov/nanoid"
	"github.com/gorilla/mux"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	_ "embed"
)

var pms_policy_deployment_step1 = `

resource "kaleido_platform_pms_policy_deployment" "test_policy_deployment" {
  environment = "test-env"
  service = "test-service"
  name = "test-policy-deployment"
  description = "Test policy deployment for policy management"
  policy = "kaleido.policy.approval"
  policy_version = "25.6.0"
  config = jsonencode({
    "maxApprovals" = 2
    "timeout" = "24h"
  })
}
`

var pms_policy_deployment_step2 = `

resource "kaleido_platform_pms_policy_deployment" "test_policy_deployment" {
  environment = "test-env"
  service = "test-service"
  name = "test-policy-deployment"
  description = "Test policy deployment for policy management"
  policy = "kaleido.policy.approval"
  policy_version = "25.6.0"
  config = jsonencode({
    "maxApprovals" = 3
    "timeout" = "48h"
  })
}
`

func TestPMSPolicyDeployment1(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"PUT /endpoint/{env}/{service}/rest/api/v1/policy-deployments/{policyDeployment}",
			"POST /endpoint/{env}/{service}/rest/api/v1/policy-deployments/{policyDeployment}/versions",
			"GET /endpoint/{env}/{service}/rest/api/v1/policy-deployments/{policyDeployment}",
			"GET /endpoint/{env}/{service}/rest/api/v1/policy-deployments/{policyDeployment}",
			"GET /endpoint/{env}/{service}/rest/api/v1/policy-deployments/{policyDeployment}",
			"PATCH /endpoint/{env}/{service}/rest/api/v1/policy-deployments/{policyDeployment}",
			"POST /endpoint/{env}/{service}/rest/api/v1/policy-deployments/{policyDeployment}/versions",
			"GET /endpoint/{env}/{service}/rest/api/v1/policy-deployments/{policyDeployment}",
			"GET /endpoint/{env}/{service}/rest/api/v1/policy-deployments/{policyDeployment}",
			"DELETE /endpoint/{env}/{service}/rest/api/v1/policy-deployments/{policyDeployment}",
			"GET /endpoint/{env}/{service}/rest/api/v1/policy-deployments/{policyDeployment}",
		})
		mp.server.Close()
	}()

	pms_policy_deployment_resource := "kaleido_platform_pms_policy_deployment.test_policy_deployment"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + pms_policy_deployment_step1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(pms_policy_deployment_resource, "id"),
					resource.TestCheckResourceAttr(pms_policy_deployment_resource, "environment", "test-env"),
					resource.TestCheckResourceAttr(pms_policy_deployment_resource, "service", "test-service"),
					resource.TestCheckResourceAttr(pms_policy_deployment_resource, "name", "test-policy-deployment"),
					resource.TestCheckResourceAttr(pms_policy_deployment_resource, "description", "Test policy deployment for policy management"),
					resource.TestCheckResourceAttr(pms_policy_deployment_resource, "policy", "kaleido.policy.approval"),
					resource.TestCheckResourceAttr(pms_policy_deployment_resource, "policy_version", "25.6.0"),
					resource.TestCheckResourceAttr(pms_policy_deployment_resource, "config", `{"maxApprovals":2,"timeout":"24h"}`),
					resource.TestCheckResourceAttrSet(pms_policy_deployment_resource, "current_version"),
					resource.TestCheckResourceAttrSet(pms_policy_deployment_resource, "created"),
					resource.TestCheckResourceAttrSet(pms_policy_deployment_resource, "updated"),
				),
			},
			{
				Config: providerConfig + pms_policy_deployment_step2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(pms_policy_deployment_resource, "id"),
					resource.TestCheckResourceAttr(pms_policy_deployment_resource, "environment", "test-env"),
					resource.TestCheckResourceAttr(pms_policy_deployment_resource, "service", "test-service"),
					resource.TestCheckResourceAttr(pms_policy_deployment_resource, "name", "test-policy-deployment"),
					resource.TestCheckResourceAttr(pms_policy_deployment_resource, "description", "Test policy deployment for policy management"),
					resource.TestCheckResourceAttr(pms_policy_deployment_resource, "policy", "kaleido.policy.approval"),
					resource.TestCheckResourceAttr(pms_policy_deployment_resource, "policy_version", "25.6.0"),
					resource.TestCheckResourceAttr(pms_policy_deployment_resource, "config", `{"maxApprovals":3,"timeout":"48h"}`),
					resource.TestCheckResourceAttrSet(pms_policy_deployment_resource, "current_version"),
					resource.TestCheckResourceAttrSet(pms_policy_deployment_resource, "created"),
					resource.TestCheckResourceAttrSet(pms_policy_deployment_resource, "updated"),
				),
			},
		},
	})
}

// PMS Policy Deployment handlers
func (mp *mockPlatform) putPMSPolicyDeployment(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	policyDeploymentName := vars["policyDeployment"]
	var policyDeployment PMSPolicyDeploymentAPIModel
	mp.getBody(req, &policyDeployment)
	policyDeployment.ID = nanoid.New()
	now := time.Now().UTC()
	policyDeployment.Created = &now
	policyDeployment.Updated = &now
	// Store by both ID and name for lookup
	mp.pmsPolicyDeployments[policyDeployment.ID] = &policyDeployment
	mp.pmsPolicyDeployments[policyDeploymentName] = &policyDeployment
	mp.respond(res, &policyDeployment, http.StatusOK)
}

func (mp *mockPlatform) getPMSPolicyDeployment(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	policyDeploymentID := vars["policyDeployment"]
	policyDeployment, exists := mp.pmsPolicyDeployments[policyDeploymentID]
	if !exists {
		mp.respond(res, nil, http.StatusNotFound)
		return
	}
	policyDeploymentVersions, versionsExist := mp.pmsPolicyDeploymentVersions[policyDeploymentID]
	if versionsExist {
		policyDeploymentVersion, versionExists := policyDeploymentVersions[policyDeployment.CurrentVersion]
		if versionExists {
			policyDeployment.Policy = policyDeploymentVersion.Policy
			policyDeployment.PolicyVersion = policyDeploymentVersion.PolicyVersion
			policyDeployment.Config = policyDeploymentVersion.Config
		}
	}
	mp.respond(res, policyDeployment, http.StatusOK)
}

func (mp *mockPlatform) patchPMSPolicyDeployment(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	policyDeploymentID := vars["policyDeployment"]
	policyDeployment, exists := mp.pmsPolicyDeployments[policyDeploymentID]
	if !exists {
		mp.respond(res, nil, http.StatusNotFound)
		return
	}
	var updates PMSPolicyDeploymentAPIModel
	mp.getBody(req, &updates)
	if updates.Description != "" {
		policyDeployment.Description = updates.Description
	}
	now := time.Now().UTC()
	policyDeployment.Updated = &now
	// Update both ID and name entries
	mp.pmsPolicyDeployments[policyDeployment.ID] = policyDeployment
	mp.pmsPolicyDeployments[policyDeploymentID] = policyDeployment
	mp.respond(res, policyDeployment, http.StatusOK)
}

func (mp *mockPlatform) deletePMSPolicyDeployment(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	policyDeploymentID := vars["policyDeployment"]
	policyDeployment, exists := mp.pmsPolicyDeployments[policyDeploymentID]
	if exists {
		// Delete both ID and name entries
		delete(mp.pmsPolicyDeployments, policyDeployment.ID)
		delete(mp.pmsPolicyDeploymentVersions, policyDeployment.ID)
	}
	delete(mp.pmsPolicyDeployments, policyDeploymentID)
	delete(mp.pmsPolicyDeploymentVersions, policyDeploymentID)
	mp.respond(res, nil, http.StatusNoContent)
}

func (mp *mockPlatform) postPMSPolicyDeploymentVersion(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	policyDeploymentID := vars["policyDeployment"]
	var version PMSPolicyDeploymentVersionAPIModel
	mp.getBody(req, &version)
	version.ID = nanoid.New()
	version.PolicyDeploymentID = policyDeploymentID
	now := time.Now().UTC()
	version.Created = &now
	version.Updated = &now

	// Initialize versions map if it doesn't exist
	if mp.pmsPolicyDeploymentVersions[policyDeploymentID] == nil {
		mp.pmsPolicyDeploymentVersions[policyDeploymentID] = make(map[string]*PMSPolicyDeploymentVersionAPIModel)
	}
	mp.pmsPolicyDeploymentVersions[policyDeploymentID][version.ID] = &version

	// Update the policy deployment's current version
	if policyDeployment, exists := mp.pmsPolicyDeployments[policyDeploymentID]; exists {
		policyDeployment.CurrentVersion = version.ID
		// Update both ID and name entries
		mp.pmsPolicyDeployments[policyDeployment.ID] = policyDeployment
		mp.pmsPolicyDeployments[policyDeploymentID] = policyDeployment
	}

	mp.respond(res, &version, http.StatusOK)
}
