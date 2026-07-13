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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/assert"
)

func arsFileArtifactAutoConfig(filePath string, removeOldVersions bool) string {
	return fmt.Sprintf(`
resource "kaleido_platform_ars_file_artifact" "file1" {
  environment         = "env1"
  service             = "svc1"
  namespace           = "ns1"
  name                = "path/to/myfilename.ext"
  file_path           = "%s"
  type                = "json"
  version             = "v1.2.3.4"
  remove_old_versions = %t
}
`, filePath, removeOldVersions)
}

func arsFileArtifactExplicitConfig(filePath, tag string, removeOldVersions bool) string {
	return fmt.Sprintf(`
resource "kaleido_platform_ars_file_artifact" "file1" {
  environment         = "env1"
  service             = "svc1"
  namespace           = "ns1"
  name                = "path/to/myfilename.ext"
  file_path           = "%s"
  type                = "json"
  tag                 = "%s"
  remove_old_versions = %t
}
`, filePath, tag, removeOldVersions)
}

func sha256Hex(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func TestARSFileArtifactAutoTag(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.server.Close()
	}()

	filePath := filepath.Join(t.TempDir(), "artifact.json")
	content1 := []byte(`{"rev": 1}`)
	content2 := []byte(`{"rev": 2}`)
	content3 := []byte(`{"rev": 3}`)
	assert.NoError(t, os.WriteFile(filePath, content1, 0644))
	tag1 := "v1.2.3.4-" + sha256Hex(content1)[:8]
	tag2 := "v1.2.3.4-" + sha256Hex(content2)[:8]
	tag3 := "v1.2.3.4-" + sha256Hex(content3)[:8]
	key1 := "env1/svc1/ns1/path/to/myfilename.ext:" + tag1
	key2 := "env1/svc1/ns1/path/to/myfilename.ext:" + tag2
	key3 := "env1/svc1/ns1/path/to/myfilename.ext:" + tag3

	fileResource := "kaleido_platform_ars_file_artifact.file1"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + arsFileArtifactAutoConfig(filePath, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(fileResource, "id", "env1/svc1/ns1/path/to/myfilename.ext:"+tag1),
					resource.TestCheckResourceAttr(fileResource, "tag", tag1),
					resource.TestCheckResourceAttr(fileResource, "content_sha256", "sha256:"+sha256Hex(content1)),
					resource.TestCheckResourceAttr(fileResource, "size", fmt.Sprintf("%d", len(content1))),
					func(s *terraform.State) error {
						obj := mp.arsFiles[key1]
						assert.NotNil(t, obj)
						assert.Equal(t, "path/to/myfilename.ext", obj.Repository)
						assert.Equal(t, tag1, obj.Tag)
						assert.Equal(t, "json", obj.Kind)
						assert.Equal(t, "sha256:"+sha256Hex(content1), obj.LayerDigest)
						return nil
					},
				),
			},
			{
				// Changing the file content changes the derived tag => in-place upgrade
				// retaining the old version by default
				PreConfig: func() {
					assert.NoError(t, os.WriteFile(filePath, content2, 0644))
				},
				Config: providerConfig + arsFileArtifactAutoConfig(filePath, false),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(fileResource, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(fileResource, "id", "env1/svc1/ns1/path/to/myfilename.ext:"+tag2),
					resource.TestCheckResourceAttr(fileResource, "tag", tag2),
					resource.TestCheckResourceAttr(fileResource, "content_sha256", "sha256:"+sha256Hex(content2)),
					func(s *terraform.State) error {
						assert.NotNil(t, mp.arsFiles[key1]) // old version retained by default
						assert.NotNil(t, mp.arsFiles[key2])
						return nil
					},
				),
			},
			{
				// With remove_old_versions the upgrade untags the previous version
				PreConfig: func() {
					assert.NoError(t, os.WriteFile(filePath, content3, 0644))
				},
				Config: providerConfig + arsFileArtifactAutoConfig(filePath, true),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(fileResource, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(fileResource, "tag", tag3),
					func(s *terraform.State) error {
						assert.NotNil(t, mp.arsFiles[key1]) // retained from the earlier upgrade
						assert.Nil(t, mp.arsFiles[key2])    // untagged by remove_old_versions
						assert.NotNil(t, mp.arsFiles[key3])
						return nil
					},
				),
			},
		},
	})
	// Final destroy only untags the currently tracked version; tag1 was
	// retained by the first (default) upgrade
	assert.Nil(t, mp.arsFiles[key3])
	assert.NotNil(t, mp.arsFiles[key1])
	delete(mp.arsFiles, key1)
	assert.Empty(t, mp.arsFiles)
}

func TestARSFileArtifactExplicitTag(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.server.Close()
	}()

	filePath := filepath.Join(t.TempDir(), "artifact.json")
	content1 := []byte(`{"rev": 1}`)
	content2 := []byte(`{"rev": 2}`)
	assert.NoError(t, os.WriteFile(filePath, content1, 0644))
	key1 := "env1/svc1/ns1/path/to/myfilename.ext:rel1"
	key2 := "env1/svc1/ns1/path/to/myfilename.ext:rel2"

	fileResource := "kaleido_platform_ars_file_artifact.file1"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + arsFileArtifactExplicitConfig(filePath, "rel1", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(fileResource, "id", "env1/svc1/ns1/path/to/myfilename.ext:rel1"),
					resource.TestCheckResourceAttr(fileResource, "tag", "rel1"),
					resource.TestCheckResourceAttr(fileResource, "content_sha256", "sha256:"+sha256Hex(content1)),
					func(s *terraform.State) error {
						assert.NotNil(t, mp.arsFiles[key1])
						return nil
					},
				),
			},
			{
				// Explicit tag mode trusts the tag: local file changes produce no diff
				PreConfig: func() {
					assert.NoError(t, os.WriteFile(filePath, content2, 0644))
				},
				Config:   providerConfig + arsFileArtifactExplicitConfig(filePath, "rel1", false),
				PlanOnly: true,
			},
			{
				Config:                  providerConfig + arsFileArtifactExplicitConfig(filePath, "rel1", false),
				ResourceName:            fileResource,
				ImportState:             true,
				ImportStateId:           "env1/svc1/ns1/path/to/myfilename.ext:rel1",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"file_path"},
			},
			{
				// Changing the explicit tag re-uploads the (changed) file under the new
				// tag, and remove_old_versions untags the previous one
				Config: providerConfig + arsFileArtifactExplicitConfig(filePath, "rel2", true),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(fileResource, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(fileResource, "id", "env1/svc1/ns1/path/to/myfilename.ext:rel2"),
					resource.TestCheckResourceAttr(fileResource, "content_sha256", "sha256:"+sha256Hex(content2)),
					func(s *terraform.State) error {
						assert.Nil(t, mp.arsFiles[key1]) // untagged by remove_old_versions
						assert.NotNil(t, mp.arsFiles[key2])
						return nil
					},
				),
			},
		},
	})
	assert.Empty(t, mp.arsFiles)
}

func TestARSFileArtifactTagImmutable(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.server.Close()
	}()

	filePath := filepath.Join(t.TempDir(), "artifact.json")
	assert.NoError(t, os.WriteFile(filePath, []byte(`{"rev": 1}`), 0644))

	// The tag already exists in the registry with different content
	mp.arsFiles["env1/svc1/ns1/path/to/myfilename.ext:rel1"] = &ARSFileArtifactAPIModel{
		Namespace:   "ns1",
		Repository:  "path/to/myfilename.ext",
		Tag:         "rel1",
		Kind:        "json",
		LayerDigest: "sha256:" + sha256Hex([]byte(`different content`)),
		Size:        17,
	}
	defer delete(mp.arsFiles, "env1/svc1/ns1/path/to/myfilename.ext:rel1")

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      providerConfig + arsFileArtifactExplicitConfig(filePath, "rel1", false),
				ExpectError: regexp.MustCompile(`Tag is immutable`),
			},
		},
	})
}

func TestARSNamespaceFileFamily(t *testing.T) {
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
				Config: providerConfig + `
resource "kaleido_platform_ars_namespace" "ns1" {
  environment       = "env1"
  service           = "svc1"
  name              = "ns1"
  artifact_family   = "file"
  description       = "file artifacts"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(nsResource, "artifact_family", "file"),
				),
			},
		},
	})
}

func (mp *mockPlatform) arsFileKey(vars map[string]string) string {
	return vars["env"] + "/" + vars["service"] + "/" + vars["ns"] + "/" + vars["name"] + ":" + vars["tag"]
}

func (mp *mockPlatform) postARSFile(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	// The real server accepts multipart/form-data with the 'type' field
	// preceding the 'file' part
	assert.True(mp.t, strings.HasPrefix(req.Header.Get("Content-Type"), "multipart/form-data"))
	fileType := req.FormValue("type")
	assert.NotEmpty(mp.t, fileType)
	file, _, err := req.FormFile("file")
	assert.NoError(mp.t, err)
	body, err := io.ReadAll(file)
	assert.NoError(mp.t, err)
	digest := "sha256:" + sha256Hex(body)

	key := mp.arsFileKey(vars)
	if existing := mp.arsFiles[key]; existing != nil {
		if existing.LayerDigest != digest {
			// Tags are immutable server-side; identical re-push is idempotent-OK
			mp.respond(res, map[string]string{
				"code":    "TAG_IMMUTABLE",
				"message": "tag is immutable and already references a different manifest",
			}, 409)
			return
		}
		mp.respond(res, existing, 201)
		return
	}

	obj := &ARSFileArtifactAPIModel{
		Namespace:      vars["ns"],
		Repository:     vars["name"],
		Tag:            vars["tag"],
		Kind:           fileType,
		LayerDigest:    digest,
		ManifestDigest: "sha256:" + sha256Hex(append([]byte("manifest:"), body...)),
		Size:           int64(len(body)),
	}
	mp.arsFiles[key] = obj
	mp.respond(res, obj, 201)
}

func (mp *mockPlatform) getARSFile(res http.ResponseWriter, req *http.Request) {
	obj := mp.arsFiles[mp.arsFileKey(mux.Vars(req))]
	if obj == nil {
		mp.respond(res, nil, 404)
	} else {
		mp.respond(res, obj, 200)
	}
}

func (mp *mockPlatform) deleteARSFile(res http.ResponseWriter, req *http.Request) {
	key := mp.arsFileKey(mux.Vars(req))
	if mp.arsFiles[key] == nil {
		mp.respond(res, nil, 404)
	} else {
		delete(mp.arsFiles, key)
		mp.respond(res, nil, 204)
	}
}
