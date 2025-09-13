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

var wfe_workflow_step1 = `
resource "kaleido_platform_wfe_workflow" "test_workflow" {
  environment = "test-env"
  service = "test-service"
  name = "test-workflow"
  description = "Test workflow for workflow engine"
  flow_yaml = yamlencode({
    "version" = "1.0"
    "name" = "test-workflow"
    "description" = "A simple test workflow"
    "steps" = [
      {
        "id" = "step1"
        "type" = "task"
        "name" = "First Step"
        "action" = "log"
        "config" = {
          "message" = "Hello from workflow"
        }
      }
    ]
  })
  handler_bindings_json = jsonencode({
    "evm" = {
      "provider" = "s-12345abcde"
      "providerHandler" = "evm"
    }
  })
}
`

var wfe_workflow_step2 = `
resource "kaleido_platform_wfe_workflow" "test_workflow" {
  environment = "test-env"
  service = "test-service"
  name = "test-workflow"
  description = "Test workflow for workflow engine - updated"
  flow_yaml = yamlencode({
    "version" = "1.1"
    "name" = "test-workflow"
    "description" = "An updated test workflow"
    "steps" = [
      {
        "id" = "step1"
        "type" = "task"
        "name" = "First Step"
        "action" = "log"
        "config" = {
          "message" = "Hello from workflow"
        }
      },
      {
        "id" = "step2"
        "type" = "task"
        "name" = "Second Step"
        "action" = "log"
        "config" = {
          "message" = "Goodbye from workflow"
        }
      }
    ]
  })
  handler_bindings_json = jsonencode({
    "evm" = {
      "provider" = "s-12345abcde"
      "providerHandler" = "evm"
    }
  })	
}
`

func TestWFEWorkflow1(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"PUT /endpoint/{env}/{service}/rest/api/v1/workflows/{workflow}",
			"POST /endpoint/{env}/{service}/rest/api/v1/workflows/{workflow}/versions",
			"GET /endpoint/{env}/{service}/rest/api/v1/workflows/{workflow}",
			"GET /endpoint/{env}/{service}/rest/api/v1/workflows/{workflow}",
			"DELETE /endpoint/{env}/{service}/rest/api/v1/workflows/{workflow}",
			"GET /endpoint/{env}/{service}/rest/api/v1/workflows/{workflow}",
		})
		mp.server.Close()
	}()

	wfe_workflow_resource := "kaleido_platform_wfe_workflow.test_workflow"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + wfe_workflow_step1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(wfe_workflow_resource, "id"),
					resource.TestCheckResourceAttr(wfe_workflow_resource, "environment", "test-env"),
					resource.TestCheckResourceAttr(wfe_workflow_resource, "service", "test-service"),
					resource.TestCheckResourceAttr(wfe_workflow_resource, "name", "test-workflow"),
					resource.TestCheckResourceAttr(wfe_workflow_resource, "description", "Test workflow for workflow engine"),
					resource.TestCheckResourceAttrSet(wfe_workflow_resource, "applied_version"),
					resource.TestCheckResourceAttrSet(wfe_workflow_resource, "created"),
					resource.TestCheckResourceAttrSet(wfe_workflow_resource, "updated"),
				),
			},
		},
	})
}

func TestWFEWorkflow2(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"PUT /endpoint/{env}/{service}/rest/api/v1/workflows/{workflow}",
			"POST /endpoint/{env}/{service}/rest/api/v1/workflows/{workflow}/versions",
			"GET /endpoint/{env}/{service}/rest/api/v1/workflows/{workflow}",
			"GET /endpoint/{env}/{service}/rest/api/v1/workflows/{workflow}",
			"GET /endpoint/{env}/{service}/rest/api/v1/workflows/{workflow}",
			"PATCH /endpoint/{env}/{service}/rest/api/v1/workflows/{workflow}",
			"POST /endpoint/{env}/{service}/rest/api/v1/workflows/{workflow}/versions",
			"GET /endpoint/{env}/{service}/rest/api/v1/workflows/{workflow}",
			"GET /endpoint/{env}/{service}/rest/api/v1/workflows/{workflow}",
			"DELETE /endpoint/{env}/{service}/rest/api/v1/workflows/{workflow}",
			"GET /endpoint/{env}/{service}/rest/api/v1/workflows/{workflow}",
		})
		mp.server.Close()
	}()

	wfe_workflow_resource := "kaleido_platform_wfe_workflow.test_workflow"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + wfe_workflow_step1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(wfe_workflow_resource, "id"),
					resource.TestCheckResourceAttr(wfe_workflow_resource, "environment", "test-env"),
					resource.TestCheckResourceAttr(wfe_workflow_resource, "service", "test-service"),
					resource.TestCheckResourceAttr(wfe_workflow_resource, "name", "test-workflow"),
					resource.TestCheckResourceAttr(wfe_workflow_resource, "description", "Test workflow for workflow engine"),
					resource.TestCheckResourceAttrSet(wfe_workflow_resource, "applied_version"),
					resource.TestCheckResourceAttrSet(wfe_workflow_resource, "created"),
					resource.TestCheckResourceAttrSet(wfe_workflow_resource, "updated"),
				),
			},
			{
				Config: providerConfig + wfe_workflow_step2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(wfe_workflow_resource, "id"),
					resource.TestCheckResourceAttr(wfe_workflow_resource, "environment", "test-env"),
					resource.TestCheckResourceAttr(wfe_workflow_resource, "service", "test-service"),
					resource.TestCheckResourceAttr(wfe_workflow_resource, "name", "test-workflow"),
					resource.TestCheckResourceAttr(wfe_workflow_resource, "description", "Test workflow for workflow engine - updated"),
					resource.TestCheckResourceAttrSet(wfe_workflow_resource, "applied_version"),
					resource.TestCheckResourceAttrSet(wfe_workflow_resource, "created"),
					resource.TestCheckResourceAttrSet(wfe_workflow_resource, "updated"),
				),
			},
		},
	})
}

// WFE Workflow handlers
func (mp *mockPlatform) putWFEWorkflow(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	workflowNameOrID := vars["workflow"]
	workflow, exists := mp.wfeWorkflows[workflowNameOrID]
	newWorkflow := new(WFEWorkflowAPIModel)
	mp.getBody(req, newWorkflow)
	now := time.Now().UTC()
	if !exists {
		newWorkflow.ID = nanoid.New()
		newWorkflow.Created = &now
	} else {
		newWorkflow.ID = workflow.ID
		newWorkflow.Name = workflow.Name
	}
	newWorkflow.Updated = &now
	//Store by both ID and name for lookup
	mp.wfeWorkflows[newWorkflow.Name] = newWorkflow
	mp.wfeWorkflows[newWorkflow.ID] = newWorkflow
	mp.respond(res, &workflow, http.StatusOK)
}

func (mp *mockPlatform) getWFEWorkflow(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	workflowID := vars["workflow"]
	workflow, exists := mp.wfeWorkflows[workflowID]
	if !exists {
		// Try to find by name
		for _, w := range mp.wfeWorkflows {
			if w.Name == workflowID {
				workflow = w
				exists = true
				break
			}
		}
	}
	if !exists {
		mp.respond(res, nil, 404)
		return
	}

	mp.respond(res, workflow, http.StatusOK)
}

func (mp *mockPlatform) patchWFEWorkflow(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	workflowNameOrID := vars["workflow"]
	workflow, exists := mp.wfeWorkflows[workflowNameOrID]
	if !exists {
		mp.respond(res, nil, 404)
		return
	}
	var workflowUpdates WFEWorkflowAPIModel
	mp.getBody(req, &workflowUpdates)
	if workflowUpdates.Name != "" {
		workflow.Name = workflowUpdates.Name
	}
	if workflowUpdates.Description != "" {
		workflow.Description = workflowUpdates.Description
	}
	now := time.Now().UTC()
	workflow.Updated = &now
	//Store by both ID and name for lookup
	mp.wfeWorkflows[workflow.Name] = workflow
	mp.wfeWorkflows[workflow.ID] = workflow
	mp.respond(res, workflow, http.StatusOK)
}

func (mp *mockPlatform) deleteWFEWorkflow(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	workflowID := vars["workflow"]
	workflow, exists := mp.wfeWorkflows[workflowID]
	if !exists {
		mp.respond(res, nil, 404)
		return
	}
	delete(mp.wfeWorkflows, workflowID)
	delete(mp.wfeWorkflows, workflow.Name)
	delete(mp.wfeWorkflowVersions, workflow.ID)
	mp.respond(res, nil, http.StatusNoContent)
}

func (mp *mockPlatform) postWFEWorkflowVersion(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	workflowID := vars["workflow"]
	workflow, exists := mp.wfeWorkflows[workflowID]
	if !exists {
		mp.respond(res, nil, 404)
		return
	}
	var version WFEWorkflowVersionAPIModel
	mp.getBody(req, &version)
	version.ID = nanoid.New()
	version.WorkflowID = workflow.ID
	now := time.Now().UTC()
	version.Created = &now
	version.Updated = &now
	if mp.wfeWorkflowVersions[workflow.ID] == nil {
		mp.wfeWorkflowVersions[workflow.ID] = make(map[string]*WFEWorkflowVersionAPIModel)
	}
	mp.wfeWorkflowVersions[workflow.ID][version.ID] = &version
	workflow.CurrentVersion = version.ID
	mp.respond(res, &version, http.StatusOK)
}

func (mp *mockPlatform) getWFEWorkflowVersion(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	workflowID := vars["workflow"]
	versionID := vars["version"]

	workflow, exists := mp.wfeWorkflows[workflowID]
	if !exists {
		// Try to find by name
		for _, w := range mp.wfeWorkflows {
			if w.Name == workflowID {
				workflow = w
				exists = true
				break
			}
		}
	}
	if !exists {
		mp.respond(res, nil, 404)
		return
	}

	if versions, exists := mp.wfeWorkflowVersions[workflow.ID]; exists {
		if version, exists := versions[versionID]; exists {
			mp.respond(res, version, http.StatusOK)
			return
		}
	}

	mp.respond(res, nil, 404)
}
