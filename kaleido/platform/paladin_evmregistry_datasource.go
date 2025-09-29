// Copyright © Kaleido, Inc. 2024

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
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type PaladinEVMRegistryDatasourceModel struct {
	Environment types.String `tfsdk:"environment"`
	Network     types.String `tfsdk:"network"`
	Address     types.String `tfsdk:"address"`
	BlockNumber types.Int64  `tfsdk:"block_number"`
}

func (m *NetworkAPIModel) toPaladinEVMRegistryData(_ context.Context, data *PaladinEVMRegistryDatasourceModel, d *diag.Diagnostics) {
	if m.Type != "PaladinNetwork" {
		d.AddError("invalid type for retrieved network", "network type must be PaladinNetwork")
		return
	}
	data.Network = types.StringValue(m.ID)

	_, evmRegistryExist := m.StatusDetails["evmRegistry"]
	if !evmRegistryExist {
		d.AddError("evm registry not found", "evmRegistry not found in network status details, please wait and try again")
		return
	}
	evmRegistry, ok := m.StatusDetails["evmRegistry"].(map[string]interface{})
	if !ok {
		d.AddError("invalid evm registry", "evmRegistry must be a JSON object")
		return
	}
	data.Address = types.StringValue(evmRegistry["address"].(string))

	deployTransaction, ok := evmRegistry["deployTransaction"].(map[string]interface{})
	if !ok {
		d.AddError("invalid deploy transaction", "deployTransaction must be a JSON object")
		return
	}

	data.BlockNumber = types.Int64Value(int64(deployTransaction["blockNumber"].(float64)))
}

func PaladinEVMRegistryDatasourceModelFactory() datasource.DataSource {
	return &paladinEVMRegistryDatasource{}
}

type paladinEVMRegistryDatasource struct {
	commonDataSource
}

func (r *paladinEVMRegistryDatasource) apiPath(data *PaladinEVMRegistryDatasourceModel) string {
	return fmt.Sprintf("/api/v1/environments/%s/networks/%s", data.Environment.ValueString(), data.Network.ValueString())
}

func (s paladinEVMRegistryDatasource) Metadata(ctx context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_paladin_evm_registry"
}

func (s *paladinEVMRegistryDatasource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetch the Paladin EVM registry details",
		Attributes: map[string]schema.Attribute{
			"environment": &schema.StringAttribute{
				Required: true,
			},
			"network": &schema.StringAttribute{
				Required: true,
			},
			"address": &schema.StringAttribute{
				Computed: true,
			},
			"block_number": &schema.Int64Attribute{
				Computed: true,
			},
		},
	}
}

func (s *paladinEVMRegistryDatasource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	data := &PaladinEVMRegistryDatasourceModel{}
	resp.Diagnostics.Append(req.Config.Get(ctx, data)...)

	api := &NetworkAPIModel{}
	ok, status := s.apiRequest(ctx, http.MethodGet, s.apiPath(data), nil, &api, &resp.Diagnostics, Allow404())
	if !ok {
		return
	}
	if status == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	// TODO poll for the network to be ready before hand ??

	api.toPaladinEVMRegistryData(ctx, data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}
