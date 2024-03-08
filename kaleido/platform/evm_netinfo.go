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
	"encoding/json"
	"fmt"
	"io"
	"math/big"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kaleido-io/terraform-provider-kaleido/kaleido/kaleidobase"
)

type EVMNetInfoDatasourceModel struct {
	Environment types.String `tfsdk:"environment"`
	Service     types.String `tfsdk:"service"`
	JsonRpcURL  types.String `tfsdk:"json_rpc_url"`
	Username    types.String `tfsdk:"username"`
	Password    types.String `tfsdk:"password"`
	ChainID     types.Int64  `tfsdk:"chain_id"`
}

type RPCRequest struct {
	JSONRpc string        `json:"jsonrpc"`
	ID      interface{}   `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params,omitempty"`
}

type RPCResponse struct {
	JSONRpc string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

type RPCError struct {
	Code    int64       `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func EVMNetInfoDataSourceFactory() datasource.DataSource {
	return &evm_netinfoDatasource{}
}

type evm_netinfoDatasource struct {
	commonDataSource
}

func (r *evm_netinfoDatasource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_evm_netinfo"
}

func (r *evm_netinfoDatasource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"environment": &schema.StringAttribute{
				Optional: true,
			},
			"service": &schema.StringAttribute{
				Optional: true,
			},
			"json_rpc_url": &schema.StringAttribute{
				Optional: true,
			},
			"username": &schema.StringAttribute{
				Optional: true,
			},
			"password": &schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
			},
			"chain_id": &schema.Int64Attribute{
				Computed: true,
			},
		},
	}
}

func (r *evm_netinfoDatasource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {

	var data EVMNetInfoDatasourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	reqID := 0
	url := data.JsonRpcURL.ValueString()
	client := resty.New()
	if data.Username.ValueString() != "" && data.Password.ValueString() != "" {
		client = client.SetBasicAuth(data.Username.ValueString(), data.Password.ValueString())
	}
	if data.Environment.ValueString() != "" && data.Service.ValueString() != "" {
		client = r.ProviderData.Platform
		url = fmt.Sprintf("/endpoint/%s/%s/jsonrpc", data.Environment.ValueString(), data.Service.ValueString())
	}
	if url == "" {
		resp.Diagnostics.AddError("no endpoint specified", "either configure environment and service, or set json_rpc_url directly")
		return
	}
	var jRes RPCResponse
	_ = kaleidobase.Retry.Do(ctx, "eth_chainId", func(attempt int) (retry bool, err error) {
		reqID++
		req := client.R().
			SetBody(RPCRequest{
				JSONRpc: "2.0",
				ID:      reqID,
				Method:  "eth_chainId",
				Params:  []interface{}{},
			}).
			SetHeader("Content-Type", "application/json").
			SetDoNotParseResponse(true)
		res, err := req.Post(url)
		rawResponse := []byte{}
		if err == nil {
			rawResponse, err = io.ReadAll(res.RawBody())
			res.RawBody().Close()
		}
		if err == nil && res.IsSuccess() {
			parseErr := json.Unmarshal(rawResponse, &jRes)
			if parseErr == nil {
				var stringChainID string
				parseErr = json.Unmarshal(jRes.Result, &stringChainID)
				if parseErr == nil {
					chainID, ok := new(big.Int).SetString(stringChainID, 0)
					if !ok {
						// This upgrades to be the error
						err = fmt.Errorf("invalid chainId in response: %s", rawResponse)
					} else {
						// Success!
						data.ChainID = types.Int64Value(chainID.Int64())
						return false, nil
					}
				}
			}
		}
		// Retry on all paths where we've not parsed out a chain ID successfully
		if err != nil {
			return true, fmt.Errorf("JSON/RPC call to %s failed: %s", url, err)
		}
		return true, fmt.Errorf("JSON/RPC call to %s returned [%d]: %s", url, res.StatusCode(), rawResponse)
	})
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)

}
