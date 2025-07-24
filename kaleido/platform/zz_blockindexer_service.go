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

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type BlockIndexerServiceResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Environment         types.String `tfsdk:"environment"`
	EnvironmentMemberID types.String `tfsdk:"environment_member_id"`
	Runtime             types.String `tfsdk:"runtime"`
	Name                types.String `tfsdk:"name"`
	StackID             types.String `tfsdk:"stack_id"`
	Contractmanager         types.String `tfsdk:"contractmanager"`
	Enabletracetransactions types.Bool `tfsdk:"enabletracetransactions"`
	Evmgateway              types.String `tfsdk:"evmgateway"`
	Maxworkqueuesize        types.Int64 `tfsdk:"maxworkqueuesize"`
	Node                    types.String `tfsdk:"node"`
	Requiredconfirmations   types.Int64 `tfsdk:"requiredconfirmations"`
	Rpcclient               types.String `tfsdk:"rpcclient"`
	Startingblock           types.String `tfsdk:"startingblock"`
	Ui                      types.String `tfsdk:"ui"`
	ForceDelete         types.Bool   `tfsdk:"force_delete"`
}

func BlockIndexerServiceResourceFactory() resource.Resource {
	return &blockindexerserviceResource{}
}

type blockindexerserviceResource struct {
	commonResource
}

func (r *blockindexerserviceResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_blockindexer"
}

func (r *blockindexerserviceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A Block Indexer service that indexes and queries blockchain transaction data.",
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"environment": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Environment ID where the BlockIndexer service will be deployed",
			},
			"environment_member_id": &schema.StringAttribute{
				Computed: true,
			},
			"runtime": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Runtime ID where the BlockIndexer service will be deployed",
			},
			"name": &schema.StringAttribute{
				Required:    true,
				Description: "Display name for the BlockIndexer service",
			},
			"stack_id": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Stack ID where the BlockIndexer service belongs",
			},
			"contractmanager": &schema.StringAttribute{
				Optional:    true,
				Description: "The Smart Contract Manager will link your custom contracts with the Block Indexer to be able to decode transactions.",
			},
			"enabletracetransactions": &schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Enables indexing of internal transactions",
			},
			"evmgateway": &schema.StringAttribute{
				Optional:    true,
				Description: "An EVM gateway providing a JSONRPC endpoint. Specify this instead of providing a node or a JSONRPC endpoint.",
			},
			"maxworkqueuesize": &schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(5),
				Description: "Maximum number of workers for indexing blocks concurrently",
			},
			"node": &schema.StringAttribute{
				Optional:    true,
				Description: "A node providing a JSONRPC endpoint. Specify this instead of providing an EVM gateway or a JSONRPC endpoint.",
			},
			"requiredconfirmations": &schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
				Description: "Number of confirmations required to index a block",
			},
			"rpcclient": &schema.StringAttribute{
				Optional:    true,
				Description: "The JSON/RPC endpoint connection. Specify this connection instead of providing an EVM gateway or a node.",
			},
			"startingblock": &schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("0"),
				Description: "The block to start indexing from. Provide a block number or 'latest'",
			},
			"ui": &schema.StringAttribute{
				Optional:    true,
				Description: "UI branding & style configurations",
			},
			"force_delete": &schema.BoolAttribute{
				Optional:    true,
				Description: "Set to true when you plan to delete a protected BlockIndexer service. You must apply this value before running terraform destroy.",
			},
		},
	}
}

func (data *BlockIndexerServiceResourceModel) toBlockIndexerServiceAPI(ctx context.Context, api *ServiceAPIModel, diagnostics *diag.Diagnostics) {
	api.Type = "BlockIndexerService"
	api.Name = data.Name.ValueString()
	api.StackID = data.StackID.ValueString()
	api.Runtime.ID = data.Runtime.ValueString()
	api.Config = make(map[string]interface{})

	if !data.Ui.IsNull() && data.Ui.ValueString() != "" {
		api.Config["ui"] = data.Ui.ValueString()
	}
	if !data.Rpcclient.IsNull() && data.Rpcclient.ValueString() != "" {
		api.Config["rpcClient"] = data.Rpcclient.ValueString()
	}
	api.Config["node"] = map[string]interface{}{
		"id": data.Node.ValueString(),
	}
	api.Config["enableTraceTransactions"] = data.Enabletracetransactions.ValueBool()
	if !data.Maxworkqueuesize.IsNull() {
		api.Config["maxWorkQueueSize"] = data.Maxworkqueuesize.ValueInt64()
	}
	if !data.Requiredconfirmations.IsNull() {
		api.Config["requiredConfirmations"] = data.Requiredconfirmations.ValueInt64()
	}
	api.Config["evmGateway"] = map[string]interface{}{
		"id": data.Evmgateway.ValueString(),
	}
	api.Config["contractManager"] = map[string]interface{}{
		"id": data.Contractmanager.ValueString(),
	}
	if !data.Startingblock.IsNull() && data.Startingblock.ValueString() != "" {
		api.Config["startingBlock"] = data.Startingblock.ValueString()
	}
}

func (api *ServiceAPIModel) toBlockIndexerServiceData(data *BlockIndexerServiceResourceModel, diagnostics *diag.Diagnostics) {
	data.ID = types.StringValue(api.ID)
	data.EnvironmentMemberID = types.StringValue(api.EnvironmentMemberID)
	data.Runtime = types.StringValue(api.Runtime.ID)
	data.Name = types.StringValue(api.Name)
	data.StackID = types.StringValue(api.StackID)

	if v, ok := api.Config["evmGateway"].(map[string]interface{}); ok {
		if id, ok := v["id"].(string); ok {
			data.Evmgateway = types.StringValue(id)
		}
	}
	if v, ok := api.Config["contractManager"].(map[string]interface{}); ok {
		if id, ok := v["id"].(string); ok {
			data.Contractmanager = types.StringValue(id)
		}
	}
	if v, ok := api.Config["startingBlock"].(string); ok {
		data.Startingblock = types.StringValue(v)
	} else {
		data.Startingblock = types.StringValue("0")
	}
	if v, ok := api.Config["ui"].(string); ok {
		data.Ui = types.StringValue(v)
	} else {
		data.Ui = types.StringNull()
	}
	if v, ok := api.Config["rpcClient"].(string); ok {
		data.Rpcclient = types.StringValue(v)
	} else {
		data.Rpcclient = types.StringNull()
	}
	if v, ok := api.Config["node"].(map[string]interface{}); ok {
		if id, ok := v["id"].(string); ok {
			data.Node = types.StringValue(id)
		}
	}
	if v, ok := api.Config["enableTraceTransactions"].(bool); ok {
		data.Enabletracetransactions = types.BoolValue(v)
	} else {
		data.Enabletracetransactions = types.BoolValue(false)
	}
	if v, ok := api.Config["maxWorkQueueSize"].(float64); ok {
		data.Maxworkqueuesize = types.Int64Value(int64(v))
	} else {
		data.Maxworkqueuesize = types.Int64Value(5)
	}
	if v, ok := api.Config["requiredConfirmations"].(float64); ok {
		data.Requiredconfirmations = types.Int64Value(int64(v))
	} else {
		data.Requiredconfirmations = types.Int64Value(0)
	}
}

func (r *blockindexerserviceResource) apiPath(data *BlockIndexerServiceResourceModel) string {
	path := fmt.Sprintf("/api/v1/environments/%s/services", data.Environment.ValueString())
	if data.ID.ValueString() != "" {
		path = path + "/" + data.ID.ValueString()
	}
	if !data.ForceDelete.IsNull() && data.ForceDelete.ValueBool() {
		path = path + "?force=true"
	}
	return path
}

func (r *blockindexerserviceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data BlockIndexerServiceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	var api ServiceAPIModel
	data.toBlockIndexerServiceAPI(ctx, &api, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	ok, _ := r.apiRequest(ctx, http.MethodPost, r.apiPath(&data), api, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	api.toBlockIndexerServiceData(&data, &resp.Diagnostics)
	r.waitForReadyStatus(ctx, r.apiPath(&data), &resp.Diagnostics)

	api.ID = data.ID.ValueString()
	ok, _ = r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	api.toBlockIndexerServiceData(&data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *blockindexerserviceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data BlockIndexerServiceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)

	var api ServiceAPIModel
	if ok, _ := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics); !ok {
		return
	}

	data.toBlockIndexerServiceAPI(ctx, &api, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if ok, _ := r.apiRequest(ctx, http.MethodPut, r.apiPath(&data), api, &api, &resp.Diagnostics); !ok {
		return
	}

	api.toBlockIndexerServiceData(&data, &resp.Diagnostics)
	r.waitForReadyStatus(ctx, r.apiPath(&data), &resp.Diagnostics)

	if ok, _ := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics); !ok {
		return
	}
	api.toBlockIndexerServiceData(&data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *blockindexerserviceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data BlockIndexerServiceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	var api ServiceAPIModel
	api.ID = data.ID.ValueString()
	ok, status := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics, Allow404())
	if !ok {
		return
	}
	if status == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	api.toBlockIndexerServiceData(&data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *blockindexerserviceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data BlockIndexerServiceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data), nil, nil, &resp.Diagnostics, Allow404())
	r.waitForRemoval(ctx, r.apiPath(&data), &resp.Diagnostics)
}
