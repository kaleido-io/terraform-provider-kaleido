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
package kaleidobase

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

type kaleidoProvider struct {
	version     string
	resources   []func() resource.Resource
	datasources []func() datasource.DataSource
}

var (
	_ provider.Provider = &kaleidoProvider{}
)

// Metadata returns the provider type name.
func (p *kaleidoProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "kaleido"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *kaleidoProvider) Schema(ctx context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The Kaleido Terraform Provider supports the next-generation Kaleido platform including hosted, dedicated and software offerings.",
		Attributes: map[string]schema.Attribute{
			"platform_api": schema.StringAttribute{
				Required:    true,
				Description: "For resources prefixed with `platform_`",
			},
			"platform_username": schema.StringAttribute{
				Optional:    true,
				Description: "For resources prefixed with `platform_`",
			},
			"platform_password": schema.StringAttribute{
				Sensitive:   true,
				Optional:    true,
				Description: "For resources prefixed with `platform_`",
			},
			"platform_bearer_token": schema.StringAttribute{
				Sensitive:   true,
				Optional:    true,
				Description: "For resources prefixed with `platform_`",
			},
		},
	}
}

func (p *kaleidoProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data ProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	pd := NewProviderData(ctx, &data)
	resp.DataSourceData = pd
	resp.ResourceData = pd
}

// DataSources defines the data sources implemented in the provider.
func (p *kaleidoProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return p.datasources
}

// Resources defines the resources implemented in the provider.
func (p *kaleidoProvider) Resources(_ context.Context) []func() resource.Resource {
	return p.resources
}

func New(version string, resources []func() resource.Resource, datasources []func() datasource.DataSource) provider.Provider {
	return &kaleidoProvider{
		version:     version,
		resources:   resources,
		datasources: datasources,
	}
}
