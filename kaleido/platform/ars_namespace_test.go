// Copyright © Kaleido, Inc. 2024-2025

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
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/aidarkhanov/nanoid"
	"github.com/gorilla/mux"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/assert"
)

var arsNamespaceStep1 = `
resource "kaleido_platform_ars_namespace" "ns1" {
  environment = "env1"
  service     = "svc1"
  name        = "ns1"
  artifact_family   = "custom-providers"
  description       = "stuff"
}
`

func TestARSNamespace(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.server.Close()
	}()

	nsResource := "kaleido_platform_ars_namespace.ns1"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + arsNamespaceStep1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(nsResource, "id"),
					resource.TestCheckResourceAttr(nsResource, "name", "ns1"),
					resource.TestCheckResourceAttr(nsResource, "environment", "env1"),
					resource.TestCheckResourceAttr(nsResource, "service", "svc1"),
					resource.TestCheckResourceAttr(nsResource, "description", "stuff"),
					resource.TestCheckResourceAttr(nsResource, "auto_create_repos", "true"),
					resource.TestCheckResourceAttr(nsResource, "force_destroy", "false"),
					resource.TestCheckResourceAttr(nsResource, "artifact_family", "custom-providers"),
					func(s *terraform.State) error {
						id := s.RootModule().Resources[nsResource].Primary.Attributes["id"]
						obj := mp.arsNamespaces[fmt.Sprintf("env1/svc1/%s", id)]
						assert.NotNil(t, obj)
						assert.Equal(t, "ns1", obj.Name)
						assert.Equal(t, true, obj.AutoCreateRepos) // default
						assert.Equal(t, "stuff", obj.Description)
						assert.Equal(t, "custom-providers", obj.ArtifactFamily)
						return nil
					},
				),
			},
		},
	})
}

func TestARSNamespaceOptionalDescription(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.server.Close()
	}()

	config := `
resource "kaleido_platform_ars_namespace" "unset" {
  environment     = "env1"
  service         = "svc1"
  name            = "unset"
  artifact_family = "custom-providers"
}

resource "kaleido_platform_ars_namespace" "empty" {
  environment     = "env1"
  service         = "svc1"
  name            = "empty"
  artifact_family = "file"
  description     = ""
}
`
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("kaleido_platform_ars_namespace.unset", "description"),
					resource.TestCheckResourceAttr("kaleido_platform_ars_namespace.unset", "artifact_family", "custom-providers"),
					resource.TestCheckResourceAttr("kaleido_platform_ars_namespace.empty", "description", ""),
				),
			},
			{
				Config:   providerConfig + config,
				PlanOnly: true,
			},
		},
	})
	assert.Empty(t, mp.arsNamespaces)
}

func TestARSNamespaceDeleteBlockedByRepositories(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.server.Close()
	}()

	filePath := filepath.Join(t.TempDir(), "artifact.json")
	assert.NoError(t, os.WriteFile(filePath, []byte(`{"rev": 1}`), 0644))
	orphanRepo := "env1/svc1/files/orphan.json"

	config := `
resource "kaleido_platform_ars_namespace" "files" {
  environment     = "env1"
  service         = "svc1"
  name            = "files"
  artifact_family = "file"
  description     = "files"
}

resource "kaleido_platform_ars_file_artifact" "file1" {
  environment = "env1"
  service     = "svc1"
  namespace   = kaleido_platform_ars_namespace.files.name
  name        = "tracked.json"
  file_path   = "` + filePath + `"
  type        = "json"
  tag         = "rel1"
}
`

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + config,
			},
			{
				// The tracked artifact cleans up its own repository, but a repository the
				// configuration does not track (e.g. pushed outside Terraform) blocks the
				// namespace: the destroy fails immediately rather than waiting for removal
				PreConfig: func() {
					mp.arsRepos[orphanRepo] = &ARSRepositoryAPIModel{Name: "orphan.json"}
				},
				Config:      providerConfig,
				ExpectError: regexp.MustCompile(`(?s)Namespace still contains repositories.*contains 1 repositories.*force_destroy`),
			},
			{
				PreConfig: func() {
					delete(mp.arsRepos, orphanRepo)
				},
				Config: providerConfig,
			},
		},
	})
	assert.Empty(t, mp.arsNamespaces)
	assert.Empty(t, mp.arsRepos)
}

func TestARSNamespaceForceDestroy(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.server.Close()
	}()

	config := func(forceDestroy bool) string {
		return fmt.Sprintf(`
resource "kaleido_platform_ars_namespace" "files" {
  environment     = "env1"
  service         = "svc1"
  name            = "files"
  artifact_family = "file"
  force_destroy   = %t
}
`, forceDestroy)
	}

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + config(false),
			},
			{
				// force_destroy is provider-side only, so it updates in place
				Config: providerConfig + config(true),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("kaleido_platform_ars_namespace.files", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.TestCheckResourceAttr("kaleido_platform_ars_namespace.files", "force_destroy", "true"),
			},
			{
				// A leftover repository with a version no longer blocks the destroy
				PreConfig: func() {
					mp.arsRepos["env1/svc1/files/leftover.json"] = &ARSRepositoryAPIModel{Name: "leftover.json"}
					mp.arsFiles["env1/svc1/files/leftover.json:v1"] = &ARSFileArtifactAPIModel{Repository: "leftover.json", Tag: "v1"}
				},
				Config: providerConfig,
			},
		},
	})
	assert.Empty(t, mp.arsNamespaces)
	assert.Empty(t, mp.arsRepos)
	assert.Empty(t, mp.arsFiles)
}

func (mp *mockPlatform) getARSNamespace(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	key := vars["env"] + "/" + vars["service"] + "/" + vars["ns"]
	obj := mp.arsNamespaces[key]
	if obj == nil {
		mp.respond(res, nil, 404)
	} else {
		mp.respond(res, obj, 200)
	}
}

func (mp *mockPlatform) postARSNamespace(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	var obj ARSNamespaceAPIModel
	mp.getBody(req, &obj)
	obj.ID = nanoid.New()
	mp.arsNamespaces[vars["env"]+"/"+vars["service"]+"/"+obj.ID] = &obj
	mp.respond(res, &obj, 201)
}

func (mp *mockPlatform) deleteARSNamespace(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	key := vars["env"] + "/" + vars["service"] + "/" + vars["ns"]
	obj := mp.arsNamespaces[key]
	assert.NotNil(mp.t, obj)
	nsPrefix := vars["env"] + "/" + vars["service"] + "/" + obj.Name + "/"
	repoCount := 0
	for repoKey := range mp.arsRepos {
		if strings.HasPrefix(repoKey, nsPrefix) {
			repoCount++
		}
	}
	if repoCount > 0 && req.URL.Query().Get("force") != "true" {
		mp.respond(res, map[string]string{
			"error": fmt.Sprintf("KA250001: Namespace cannot be deleted as it contains %d repositories", repoCount),
		}, 400)
		return
	}
	// A forced delete removes every repository and version in the namespace
	for repoKey := range mp.arsRepos {
		if strings.HasPrefix(repoKey, nsPrefix) {
			delete(mp.arsRepos, repoKey)
		}
	}
	for fileKey := range mp.arsFiles {
		if strings.HasPrefix(fileKey, nsPrefix) {
			delete(mp.arsFiles, fileKey)
		}
	}
	delete(mp.arsNamespaces, key)
	mp.respond(res, nil, 204)
}
