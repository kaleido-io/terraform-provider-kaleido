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
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/kaleido-io/terraform-provider-kaleido/kaleido/kaleidobase"
	"github.com/kaleido-io/terraform-provider-kaleido/kaleido/platform"
)

type baasBaseResource struct {
	*kaleidobase.ProviderData
}

type baasBaseDatasource struct {
	*kaleidobase.ProviderData
}

func (r *baasBaseResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.ProviderData = kaleidobase.ConfigureProviderData(req.ProviderData, &resp.Diagnostics)
}

func (d *baasBaseDatasource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.ProviderData = kaleidobase.ConfigureProviderData(req.ProviderData, &resp.Diagnostics)
}

func newTestProviderData() *kaleidobase.ProviderData {
	return kaleidobase.NewProviderData(context.Background(), &kaleidobase.ProviderModel{})
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return kaleidobase.New(
			version,
			append([]func() resource.Resource{
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
			}, platform.Resources()...),
			append([]func() datasource.DataSource{
				DatasourcePrivateStackBridgeFactory,
			}, platform.DataSources()...),
		)
	}
}
