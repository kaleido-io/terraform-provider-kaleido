// Copyright © Kaleido, Inc. 2018, 2021

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

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
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

type baasBaseResource struct {
	*kaleidoProviderData
}

func (r *baasBaseResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	var ok bool
	r.kaleidoProviderData, ok = req.ProviderData.(*kaleidoProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected %T, got: %T. Please report this issue to the provider developers.", r.kaleidoProviderData, req.ProviderData),
		)
		return
	}
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
	kc := kaleido.NewClient(api, apiKey)
	pd := &kaleidoProviderData{client: &kc}
	resp.DataSourceData = pd
	resp.ResourceData = pd
}

// DataSources defines the data sources implemented in the provider.
func (p *kaleidoProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		"kaleido_privatestack_bridge": resourcePrivateStackBridge,
	}
}

// Resources defines the resources implemented in the provider.
func (p *kaleidoProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		"kaleido_consortium":    resourceConsortium,
		"kaleido_environment":   resourceEnvironment,
		"kaleido_membership":    resourceMembership,
		"kaleido_node":          resourceNode,
		"kaleido_service":       NewResourceService,
		"kaleido_app_creds":     resourceAppCreds,
		"kaleido_invitation":    resourceInvitation,
		"kaleido_czone":         resourceCZone,
		"kaleido_ezone":         resourceEZone,
		"kaleido_configuration": resourceConfiguration,
		"kaleido_destination":   resourceDestination,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &kaleidoProvider{}
	}
}

func providerConfigure(d *resource.ResourceData) (interface{}, error) {
	api := d.Get("api").(string)
	apiKey := d.Get("api_key").(string)
	client := kaleido.NewClient(api, apiKey)
	return client, nil
}
