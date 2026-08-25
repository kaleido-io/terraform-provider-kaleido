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

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type EnvironmentVersionsDatasourceModel struct {
	Environment       types.String                       `tfsdk:"environment"`
	PlatformVersion   types.String                       `tfsdk:"platform_version"`
	LatestVersion     types.String                       `tfsdk:"latest_version"`
	UpgradeAvailable  types.Bool                         `tfsdk:"upgrade_available"`
	AvailableVersions []EnvironmentAvailableVersionModel `tfsdk:"available_versions"`
}

type EnvironmentAvailableVersionModel struct {
	Version              types.String                `tfsdk:"version"`
	Tag                  types.String                `tfsdk:"tag"`
	BlocksUpgrade        types.Bool                  `tfsdk:"blocks_upgrade"`
	RequiresConfirmation types.Bool                  `tfsdk:"requires_confirmation"`
	Migrations           []EnvironmentMigrationModel `tfsdk:"migrations"`
}

type EnvironmentMigrationModel struct {
	Summary     types.String `tfsdk:"summary"`
	Details     types.String `tfsdk:"details"`
	Required    types.Bool   `tfsdk:"required"`
	Overridable types.Bool   `tfsdk:"overridable"`
}

type EnvironmentVersionsAPIModel struct {
	PlatformVersion     VersionIdentifierAPIModel      `json:"platformVersion,omitempty"`
	EnvironmentVersions EnvironmentVersionListAPIModel `json:"environmentVersions,omitempty"`
}

type EnvironmentVersionListAPIModel struct {
	LatestVersion string                      `json:"latestVersion,omitempty"`
	Versions      []VersionIdentifierAPIModel `json:"versions,omitempty"`
}

type VersionIdentifierAPIModel struct {
	Version    string                     `json:"version,omitempty"`
	Tag        string                     `json:"tag,omitempty"`
	Migrations []VersionMigrationAPIModel `json:"migrations,omitempty"`
}

type VersionMigrationAPIModel struct {
	Summary     string `json:"summary,omitempty"`
	Details     string `json:"details,omitempty"`
	Required    bool   `json:"required,omitempty"`
	Overridable bool   `json:"overridable,omitempty"`
}

func environmentVersionsAPIPath(environment string) string {
	return fmt.Sprintf("/api/v1/environments/%s/versions", environment)
}

// latest returns the version identifier the environment would move to if upgraded
// to the newest available version, or nil when the environment is already current.
func (api *EnvironmentVersionsAPIModel) latest() *VersionIdentifierAPIModel {
	if api.EnvironmentVersions.LatestVersion == "" {
		return nil
	}
	for i, v := range api.EnvironmentVersions.Versions {
		if v.Version == api.EnvironmentVersions.LatestVersion {
			return &api.EnvironmentVersions.Versions[i]
		}
	}
	return &VersionIdentifierAPIModel{Version: api.EnvironmentVersions.LatestVersion}
}

// blocksUpgrade reports whether the platform refuses to move an environment to
// this version. Migrations returned for an environment have already been
// filtered to the ones that apply to it, and a migration that is not
// explicitly overridable can only be cleared by changing the environment's
// configuration until the migration no longer matches.
func (v *VersionIdentifierAPIModel) blocksUpgrade() bool {
	for _, m := range v.Migrations {
		if !m.Overridable {
			return true
		}
	}
	return false
}

// requiresConfirmation reports whether the upgrade is permitted but has to be
// explicitly confirmed, which the platform accepts as ?confirmed=true on the
// environment update.
func (v *VersionIdentifierAPIModel) requiresConfirmation() bool {
	if v.blocksUpgrade() {
		return false // blocked outright, so confirmation is not on offer
	}
	for _, m := range v.Migrations {
		if m.Required {
			return true
		}
	}
	return false
}

func (v *VersionIdentifierAPIModel) migrationSummaries() []string {
	summaries := make([]string, 0, len(v.Migrations))
	for _, m := range v.Migrations {
		if m.Summary != "" {
			summaries = append(summaries, m.Summary)
		}
	}
	return summaries
}

func (api *EnvironmentVersionsAPIModel) toData(data *EnvironmentVersionsDatasourceModel) {
	if api.PlatformVersion.Version != "" {
		data.PlatformVersion = types.StringValue(api.PlatformVersion.Version)
	} else {
		data.PlatformVersion = types.StringNull()
	}

	if api.EnvironmentVersions.LatestVersion != "" {
		data.LatestVersion = types.StringValue(api.EnvironmentVersions.LatestVersion)
	} else {
		data.LatestVersion = types.StringNull()
	}
	data.UpgradeAvailable = types.BoolValue(api.EnvironmentVersions.LatestVersion != "")

	data.AvailableVersions = make([]EnvironmentAvailableVersionModel, 0, len(api.EnvironmentVersions.Versions))
	for _, v := range api.EnvironmentVersions.Versions {
		available := EnvironmentAvailableVersionModel{
			Version:              types.StringValue(v.Version),
			BlocksUpgrade:        types.BoolValue(v.blocksUpgrade()),
			RequiresConfirmation: types.BoolValue(v.requiresConfirmation()),
			Migrations:           make([]EnvironmentMigrationModel, 0, len(v.Migrations)),
		}
		if v.Tag != "" {
			available.Tag = types.StringValue(v.Tag)
		} else {
			available.Tag = types.StringNull()
		}
		for _, m := range v.Migrations {
			migration := EnvironmentMigrationModel{
				Required:    types.BoolValue(m.Required),
				Overridable: types.BoolValue(m.Overridable),
			}
			if m.Summary != "" {
				migration.Summary = types.StringValue(m.Summary)
			} else {
				migration.Summary = types.StringNull()
			}
			if m.Details != "" {
				migration.Details = types.StringValue(m.Details)
			} else {
				migration.Details = types.StringNull()
			}
			available.Migrations = append(available.Migrations, migration)
		}
		data.AvailableVersions = append(data.AvailableVersions, available)
	}
}

func EnvironmentVersionsDatasourceModelFactory() datasource.DataSource {
	return &environmentVersionsDatasource{}
}

type environmentVersionsDatasource struct {
	commonDataSource
}

func (s *environmentVersionsDatasource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_environment_versions"
}

func (s *environmentVersionsDatasource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetch the platform versions an environment can be upgraded to. Only versions newer than the version the environment is currently running are returned, so this can be used to detect - and notify - a pending upgrade of an environment with `update_strategy = \"manual\"` without proposing any change to it.",
		Attributes: map[string]schema.Attribute{
			"environment": &schema.StringAttribute{
				Required:    true,
				Description: "ID or name of the environment",
			},
			"platform_version": &schema.StringAttribute{
				Computed:    true,
				Description: "Kaleido platform version",
			},
			"latest_version": &schema.StringAttribute{
				Computed:    true,
				Description: "Newest version the environment can be upgraded to. Null when the environment is already running the newest version available to it",
			},
			"upgrade_available": &schema.BoolAttribute{
				Computed:    true,
				Description: "True when at least one newer version is available to the environment",
			},
			"available_versions": &schema.ListNestedAttribute{
				Computed:    true,
				Description: "Versions the environment can be upgraded to, newest first",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"version": &schema.StringAttribute{
							Computed:    true,
							Description: "Version to set on the environment to upgrade to this release",
						},
						"tag": &schema.StringAttribute{
							Computed:    true,
							Description: "Release tag associated with the version",
						},
						"blocks_upgrade": &schema.BoolAttribute{
							Computed:    true,
							Description: "True when the platform rejects an upgrade to this version until the environment configuration changes so that the migrations below no longer apply to it",
						},
						"requires_confirmation": &schema.BoolAttribute{
							Computed:    true,
							Description: "True when the upgrade is permitted, but the platform requires confirmation before it is applied",
						},
						"migrations": &schema.ListNestedAttribute{
							Computed:    true,
							Description: "Migrations that apply to the resources in this environment when upgrading to this version",
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"summary": &schema.StringAttribute{
										Computed:    true,
										Description: "Short description of the migration",
									},
									"details": &schema.StringAttribute{
										Computed:    true,
										Description: "Instructions for carrying out the migration",
									},
									"required": &schema.BoolAttribute{
										Computed:    true,
										Description: "The migration needs action or acknowledgement before the upgrade proceeds",
									},
									"overridable": &schema.BoolAttribute{
										Computed:    true,
										Description: "The upgrade can still proceed with confirmation. When false the environment configuration must change until this migration no longer applies",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (s *environmentVersionsDatasource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data EnvironmentVersionsDatasourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var api EnvironmentVersionsAPIModel
	if ok, _ := s.apiRequest(ctx, http.MethodGet, environmentVersionsAPIPath(data.Environment.ValueString()), nil, &api, &resp.Diagnostics); !ok {
		return
	}

	api.toData(&data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}
