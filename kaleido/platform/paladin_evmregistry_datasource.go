// Copyright © Kaleido, Inc. 2025

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

	"github.com/kaleido-io/terraform-provider-kaleido/kaleido/kaleidobase"
)

type PaladinEVMRegistryDatasourceModel struct {
	Environment types.String `tfsdk:"environment"`
	Network     types.String `tfsdk:"network"`
	Address     types.String `tfsdk:"address"`
	BlockNumber types.Int64  `tfsdk:"block_number"`
}

// toPaladinEVMRegistryData parses the network response into the datasource model.
// Returns true when statusDetails.evmRegistry is fully populated and the caller
// can stop polling; false when the caller should retry. Terminal failures
// (e.g. wrong network type) are reported via d.AddError — check d.HasError()
// to distinguish a hard failure from "not ready yet".
func (m *NetworkAPIModel) toPaladinEVMRegistryData(_ context.Context, data *PaladinEVMRegistryDatasourceModel, d *diag.Diagnostics) (complete bool) {
	if m.Type != "PaladinNetwork" {
		d.AddError("invalid type for retrieved network", "network type must be PaladinNetwork")
		return false
	}
	data.Network = types.StringValue(m.ID)

	evmRegistry, ok := m.StatusDetails["evmRegistry"].(map[string]any)
	if !ok {
		return false
	}
	address, ok := evmRegistry["address"].(string)
	if !ok {
		return false
	}
	deployTransaction, ok := evmRegistry["deployTransaction"].(map[string]any)
	if !ok {
		return false
	}
	blockNumber, ok := deployTransaction["blockNumber"].(float64)
	if !ok {
		return false
	}

	data.Address = types.StringValue(address)
	data.BlockNumber = types.Int64Value(int64(blockNumber))
	return true
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
	if resp.Diagnostics.HasError() {
		return
	}

	apiPath := s.apiPath(data)
	cancelInfo := APICancelInfo()
	cancelInfo.CancelInfo = "(waiting for paladin evm registry)"
	removed := false
	_ = kaleidobase.Retry.Do(ctx, fmt.Sprintf("paladin-evm-registry %s", apiPath), func(attempt int) (retry bool, err error) {
		api := &NetworkAPIModel{}
		ok, status := s.apiRequest(ctx, http.MethodGet, apiPath, nil, api, &resp.Diagnostics, Allow404(), cancelInfo)
		if !ok {
			return false, fmt.Errorf("read failed") // diag already set
		}
		if status == 404 {
			removed = true
			return false, nil
		}
		if api.toPaladinEVMRegistryData(ctx, data, &resp.Diagnostics) {
			return false, nil
		}
		if resp.Diagnostics.HasError() {
			return false, fmt.Errorf("terminal failure") // diag already set
		}
		cancelInfo.CancelInfo = fmt.Sprintf("(waiting for paladin evm registry - network status: %s)", api.Status)
		return true, fmt.Errorf("evmRegistry not yet populated in network status details")
	})

	if resp.Diagnostics.HasError() {
		return
	}
	if removed {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}
