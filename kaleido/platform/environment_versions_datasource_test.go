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
	"net/http"
	"testing"

	"github.com/gorilla/mux"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"
)

var environmentVersionsStep1 = `
resource "kaleido_platform_environment" "environment1" {
    name = "environment1"
	version = "26.1.0"
	update_strategy = "manual"
}

data "kaleido_platform_environment_versions" "environment1" {
	environment = kaleido_platform_environment.environment1.id
}
`

func TestEnvironmentVersions(t *testing.T) {

	mp, providerConfig := testSetup(t)
	mp.environmentVersions = &EnvironmentVersionsAPIModel{
		PlatformVersion: VersionIdentifierAPIModel{Version: "26.2.0"},
		EnvironmentVersions: EnvironmentVersionListAPIModel{
			LatestVersion: "26.2.0",
			Versions: []VersionIdentifierAPIModel{
				{
					Version: "26.2.0",
					Tag:     "v26.2.0",
					Migrations: []VersionMigrationAPIModel{
						{Summary: "besu fast sync resync", Details: "re-sync affected nodes", Required: true, Overridable: true},
					},
				},
				{Version: "26.1.0", Tag: "v26.1.0"},
			},
		},
	}
	defer func() {
		mp.server.Close()
	}()

	versionsDatasource := "data.kaleido_platform_environment_versions.environment1"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + environmentVersionsStep1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kaleido_platform_environment.environment1", "version", `26.1.0`),
					resource.TestCheckResourceAttr(versionsDatasource, "platform_version", `26.2.0`),
					resource.TestCheckResourceAttr(versionsDatasource, "latest_version", `26.2.0`),
					resource.TestCheckResourceAttr(versionsDatasource, "upgrade_available", `true`),
					resource.TestCheckResourceAttr(versionsDatasource, "available_versions.#", `2`),
					resource.TestCheckResourceAttr(versionsDatasource, "available_versions.0.version", `26.2.0`),
					resource.TestCheckResourceAttr(versionsDatasource, "available_versions.0.tag", `v26.2.0`),
					resource.TestCheckResourceAttr(versionsDatasource, "available_versions.0.blocks_upgrade", `true`),
					resource.TestCheckResourceAttr(versionsDatasource, "available_versions.0.migrations.#", `1`),
					resource.TestCheckResourceAttr(versionsDatasource, "available_versions.0.migrations.0.summary", `besu fast sync resync`),
					resource.TestCheckResourceAttr(versionsDatasource, "available_versions.0.migrations.0.details", `re-sync affected nodes`),
					resource.TestCheckResourceAttr(versionsDatasource, "available_versions.0.migrations.0.required", `true`),
					resource.TestCheckResourceAttr(versionsDatasource, "available_versions.0.migrations.0.overridable", `true`),
					resource.TestCheckResourceAttr(versionsDatasource, "available_versions.1.version", `26.1.0`),
					resource.TestCheckResourceAttr(versionsDatasource, "available_versions.1.blocks_upgrade", `false`),
					resource.TestCheckResourceAttr(versionsDatasource, "available_versions.1.migrations.#", `0`),
				),
			},
		},
	})
}

func TestEnvironmentVersionsUpToDate(t *testing.T) {
	api := &EnvironmentVersionsAPIModel{
		PlatformVersion: VersionIdentifierAPIModel{Version: "26.2.0"},
	}
	var data EnvironmentVersionsDatasourceModel
	api.toData(&data)

	assert.Nil(t, api.latest())
	assert.True(t, data.LatestVersion.IsNull())
	assert.False(t, data.UpgradeAvailable.ValueBool())
	assert.Empty(t, data.AvailableVersions)
}

func TestEnvironmentVersionsLatestNotInList(t *testing.T) {
	api := &EnvironmentVersionsAPIModel{
		EnvironmentVersions: EnvironmentVersionListAPIModel{LatestVersion: "26.2.0"},
	}
	latest := api.latest()
	assert.NotNil(t, latest)
	assert.Equal(t, "1.2.0", latest.Version)
	assert.False(t, latest.blocksUpgrade())
}

func TestEnvironmentVersionsMigrationGating(t *testing.T) {
	notOverridable := &VersionIdentifierAPIModel{Migrations: []VersionMigrationAPIModel{
		{Summary: "storage format change"},
	}}
	assert.True(t, notOverridable.blocksUpgrade())

	// Confirmable by the platform, but not by this provider, so still blocking
	confirmable := &VersionIdentifierAPIModel{Migrations: []VersionMigrationAPIModel{
		{Summary: "resync nodes", Required: true, Overridable: true},
	}}
	assert.True(t, confirmable.blocksUpgrade())

	// Purely informational: the platform applies the upgrade without confirmation
	informational := &VersionIdentifierAPIModel{Migrations: []VersionMigrationAPIModel{
		{Summary: "Migration summary", Overridable: true},
	}}
	assert.False(t, informational.blocksUpgrade())
}

func (mp *mockPlatform) getEnvironmentVersions(res http.ResponseWriter, req *http.Request) {
	if mp.environments[mux.Vars(req)["env"]] == nil {
		mp.respond(res, nil, 404)
		return
	}
	versions := mp.environmentVersions
	if versions == nil {
		versions = &EnvironmentVersionsAPIModel{} // environment is up to date
	}
	mp.respond(res, versions, 200)
}
