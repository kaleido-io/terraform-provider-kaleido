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
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/aidarkhanov/nanoid"
	"github.com/gorilla/mux"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/assert"
)

const arsRepositoryRepoKey = "env1/svc1/ns1/path/to/myrepo"

func arsRepositoryConfig(description string) string {
	descriptionAttr := ""
	if description != "" {
		descriptionAttr = fmt.Sprintf("description = %q", description)
	}
	return fmt.Sprintf(`
resource "kaleido_platform_ars_repository" "repo1" {
  environment = "env1"
  service     = "svc1"
  namespace   = "ns1"
  name        = "path/to/myrepo"
  %s
}
`, descriptionAttr)
}

func TestARSRepository(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.server.Close()
	}()

	repoResource := "kaleido_platform_ars_repository.repo1"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + arsRepositoryConfig("my repo"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(repoResource, "id"),
					resource.TestCheckResourceAttr(repoResource, "name", "path/to/myrepo"),
					resource.TestCheckResourceAttr(repoResource, "namespace", "ns1"),
					resource.TestCheckResourceAttr(repoResource, "description", "my repo"),
					func(s *terraform.State) error {
						obj := mp.arsRepos[arsRepositoryRepoKey]
						assert.NotNil(t, obj)
						assert.Equal(t, "my repo", obj.Description)
						return nil
					},
				),
			},
			{
				Config:            providerConfig + arsRepositoryConfig("my repo"),
				ResourceName:      repoResource,
				ImportState:       true,
				ImportStateId:     "env1/svc1/ns1/path/to/myrepo",
				ImportStateVerify: true,
			},
			{
				// An unset description stays null across refreshes
				Config: providerConfig + arsRepositoryConfig(""),
				Check:  resource.TestCheckNoResourceAttr(repoResource, "description"),
			},
			{
				Config:   providerConfig + arsRepositoryConfig(""),
				PlanOnly: true,
			},
			{
				// Deleted outside Terraform: refresh drops it from state and plans a re-create
				PreConfig: func() {
					delete(mp.arsRepos, arsRepositoryRepoKey)
				},
				Config:             providerConfig + arsRepositoryConfig(""),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
	assert.Empty(t, mp.arsRepos)
}

func TestARSRepositoryDestroyWithVersionsWarns(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.server.Close()
	}()

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + arsRepositoryConfig(""),
			},
			{
				// A version not tracked by Terraform keeps the repository in the registry,
				// but the destroy succeeds with a warning
				PreConfig: func() {
					mp.arsFiles[arsRepositoryRepoKey+":v1"] = &ARSFileArtifactAPIModel{Repository: "path/to/myrepo", Tag: "v1"}
				},
				Config: providerConfig,
			},
		},
	})
	assert.NotNil(t, mp.arsRepos[arsRepositoryRepoKey])
	assert.NotNil(t, mp.arsFiles[arsRepositoryRepoKey+":v1"])
}

func TestARSRepositoryAlreadyExists(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.server.Close()
	}()

	// e.g. auto-created by an artifact push
	mp.arsRepos[arsRepositoryRepoKey] = &ARSRepositoryAPIModel{ID: "existing", NamespaceName: "ns1", Name: "path/to/myrepo"}

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      providerConfig + arsRepositoryConfig(""),
				ExpectError: regexp.MustCompile(`(?s)Repository already exists.*terraform import`),
			},
		},
	})
}

func (mp *mockPlatform) postARSRepository(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	var obj ARSRepositoryAPIModel
	mp.getBody(req, &obj)
	vars["name"] = obj.Name
	repoKey := mp.arsRepoKey(vars)
	if mp.arsRepos[repoKey] != nil {
		mp.respond(res, map[string]string{
			"error": fmt.Sprintf("repository %q already exists in namespace %q", obj.Name, vars["ns"]),
		}, 500)
		return
	}
	obj.ID = nanoid.New()
	obj.NamespaceName = vars["ns"]
	mp.arsRepos[repoKey] = &obj
	mp.respond(res, &obj, 201)
}

func (mp *mockPlatform) getARSRepository(res http.ResponseWriter, req *http.Request) {
	obj := mp.arsRepos[mp.arsRepoKey(mux.Vars(req))]
	if obj == nil {
		mp.respond(res, map[string]string{"error": "repository not found"}, 500)
		return
	}
	mp.respond(res, obj, 200)
}
