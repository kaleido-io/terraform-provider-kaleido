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
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	kaleido "github.com/kaleido-io/kaleido-sdk-go/kaleido"
)

type kaleidoProvider struct {
	version string
}

var (
	_ provider.Provider = &kaleidoProvider{}
)

type kaleidoProviderData struct {
	client *kaleido.KaleidoClient
}

type ProviderModel struct {
	API    types.String `tfsdk:"api"`
	APIKey types.String `tfsdk:"api_key"`
}

type baasBaseResource struct {
	*kaleidoProviderData
}

type baasBaseDatasource struct {
	*kaleidoProviderData
}

func (r *baasBaseResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.kaleidoProviderData = configureProviderData(req.ProviderData, resp.Diagnostics)
}

func (d *baasBaseDatasource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.kaleidoProviderData = configureProviderData(req.ProviderData, resp.Diagnostics)
}

func configureProviderData(providerData any, diagnostics diag.Diagnostics) *kaleidoProviderData {
	kaleidoProviderData, ok := providerData.(*kaleidoProviderData)
	if !ok {
		diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected %T, got: %T. Please report this issue to the provider developers.", kaleidoProviderData, providerData),
		)
		return nil
	}
	return kaleidoProviderData
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
			"api":     schema.StringAttribute{},
			"api_key": schema.StringAttribute{},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.BlockAll(ctx),
		},
	}
}

// Configure prepares a HashiCups API client for data sources and resources.
func (p *kaleidoProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data ProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	api := data.API.ValueString()
	if api == "" {
		api = os.Getenv("KALEIDO_API")
	}
	apiKey := data.APIKey.ValueString()
	if apiKey == "" {
		apiKey = os.Getenv("KALEIDO_API_KEY")
	}
	kc := kaleido.NewClient(api, apiKey)
	pd := &kaleidoProviderData{client: &kc}
	resp.DataSourceData = pd
	resp.ResourceData = pd
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
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &kaleidoProvider{}
	}
}
