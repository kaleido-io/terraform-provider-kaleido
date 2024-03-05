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
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/kaleido-io/terraform-provider-kaleido/kaleido/kaleidobase"
	"github.com/kaleido-io/terraform-provider-kaleido/kaleido/platform"
)

type kaleidoProvider struct {
	version string
}

var (
	_ provider.Provider = &kaleidoProvider{}
)

type baasBaseResource struct {
	*kaleidobase.ProviderData
}

type baasBaseDatasource struct {
	*kaleidobase.ProviderData
}

func (r *baasBaseResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.ProviderData = kaleidobase.ConfigureProviderData(req.ProviderData, resp.Diagnostics)
}

func (d *baasBaseDatasource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.ProviderData = kaleidobase.ConfigureProviderData(req.ProviderData, resp.Diagnostics)
}

// Metadata returns the provider type name.
func (p *kaleidoProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "kaleido"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *kaleidoProvider) Schema(ctx context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"api": schema.StringAttribute{
				Optional: true,
			},
			"api_key": schema.StringAttribute{
				Sensitive: true,
				Optional:  true,
			},
			"platform_api": schema.StringAttribute{
				Optional: true,
			},
			"platform_username": schema.StringAttribute{
				Optional: true,
			},
			"platform_password": schema.StringAttribute{
				Sensitive: true,
				Optional:  true,
			},
		},
	}
}

// Configure prepares a HashiCups API client for data sources and resources.
func (p *kaleidoProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data kaleidobase.ProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	pd := kaleidobase.NewProviderData(&data)
	resp.DataSourceData = pd
	resp.ResourceData = pd
}

func newTestProviderData() *kaleidobase.ProviderData {
	return kaleidobase.NewProviderData(&kaleidobase.ProviderModel{})
}

// DataSources defines the data sources implemented in the provider.
func (p *kaleidoProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		DatasourcePrivateStackBridgeFactory,
	}
}

// Resources defines the resources implemented in the provider.
func (p *kaleidoProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		// BaaS
		ResourceConsortiumFactory,
		ResourceEnvironmentFactory,
		ResourceMembershipFactory,
		ResourceNodeFactory,
		ResourceServiceFactory,
		ResourceAppCredsFactory,
		ResourceInvitationFactory,
		ResourceCZoneFactory,
		ResourceEZoneFactory,
		ResourceConfigurationFactory,
		ResourceDestinationFactory,
		// Platform
		platform.RuntimeResourceFactory,
	}
}

func New(version string) provider.Provider {
	return &kaleidoProvider{version: version}
}
