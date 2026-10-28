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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const evmConnectorInvokeDefaultWaitTimeout = 10 * time.Minute

type EVMConnectorContractInvokeResourceModel struct {
	ID              types.String `tfsdk:"id"`
	Environment     types.String `tfsdk:"environment"`
	Service         types.String `tfsdk:"service"`
	API             types.String `tfsdk:"api"`
	Key             types.String `tfsdk:"key"`
	To              types.String `tfsdk:"to"`
	ABI             types.String `tfsdk:"abi"`
	Method          types.String `tfsdk:"method"`
	ParamsJSON      types.String `tfsdk:"params_json"`
	Value           types.String `tfsdk:"value"`
	Gas             types.String `tfsdk:"gas"`
	Nonce           types.String `tfsdk:"nonce"`
	OptionsJSON     types.String `tfsdk:"options_json"`
	IdempotencyKey  types.String `tfsdk:"idempotency_key"`
	WaitTimeout     types.String `tfsdk:"wait_timeout"`
	IgnoreDestroy   types.Bool   `tfsdk:"ignore_destroy"`
	TransactionHash types.String `tfsdk:"transaction_hash"`
	BlockNumber     types.String `tfsdk:"block_number"`
}

// EVMConnectorInvokeInputAPIModel is the standard EVM API "contract/invoke" operation input
type EVMConnectorInvokeInputAPIModel struct {
	Key     string `json:"key"`
	To      string `json:"to"`
	ABI     any    `json:"abi"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
	Value   string `json:"value,omitempty"`
	Gas     string `json:"gas,omitempty"`
	Nonce   string `json:"nonce,omitempty"`
	Options any    `json:"options,omitempty"`
}

type EVMConnectorInvokeSubmitAPIModel struct {
	IdempotencyKey string                          `json:"idempotencyKey"`
	Input          EVMConnectorInvokeInputAPIModel `json:"input"`
}

func EVMConnectorContractInvokeResourceFactory() resource.Resource {
	return &evmConnectorContractInvokeResource{}
}

type evmConnectorContractInvokeResource struct {
	commonResource
}

func (r *evmConnectorContractInvokeResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_evm_connector_contract_invoke"
}

func (r *evmConnectorContractInvokeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Invokes a function on a deployed smart contract through the 'contract/invoke' operation of a deployed EVM standard API on a connector service (see kaleido_platform_connector_standard_api), " +
			"for one-off state-changing transactions such as initializing or registering configuration on a contract deployed with kaleido_platform_evm_connector_contract_deploy. " +
			"The invocation is submitted as a workflow-engine transaction with a deterministic idempotency key, so each resource instance only ever submits one unique transaction. " +
			"If the idempotency key is already in use by an existing transaction the apply fails, for the user to inspect that transaction and resolve the conflict. " +
			"Any change to the invocation inputs replaces the resource, which submits a new transaction - the effect of the previous transaction remains on-chain.",
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				Description:   "The workflow-engine transaction ID of the invocation",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"environment": &schema.StringAttribute{
				Required:      true,
				Description:   "Environment ID",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"service": &schema.StringAttribute{
				Required:      true,
				Description:   "Connector service ID",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"api": &schema.StringAttribute{
				Required:      true,
				Description:   "Name (or ID) of the deployed EVM standard API instance on the connector (e.g. the name of a kaleido_platform_connector_standard_api resource)",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"key": &schema.StringAttribute{
				Required:      true,
				Description:   "Lookup string used to resolve the signing key (e.g. a KMS key URI or address)",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"to": &schema.StringAttribute{
				Required:      true,
				Description:   "Address of the contract to invoke (e.g. the `contract_address` output of kaleido_platform_evm_connector_contract_deploy)",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"abi": &schema.StringAttribute{
				Required:      true,
				Description:   "The contract ABI as a JSON string, containing at least the function to invoke (pipes directly from the `abi` output of kaleido_platform_cms_build)",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"method": &schema.StringAttribute{
				Required: true,
				Description: "The function to invoke - either its name, or its full signature (e.g. `transfer(address,uint256)`) to select between overloaded functions. " +
					"An exact signature match takes precedence over a name match",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"params_json": &schema.StringAttribute{
				Optional:      true,
				Description:   "Function parameters as a JSON array (in input order) or object (keyed by input name) string. Tuple (struct) inputs are nested arrays or objects",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"value": &schema.StringAttribute{
				Optional:      true,
				Description:   "Optional value in wei to send with the invocation",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"gas": &schema.StringAttribute{
				Optional:      true,
				Description:   "Optional gas limit. When set, gas estimation is skipped",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"nonce": &schema.StringAttribute{
				Optional:      true,
				Description:   "Optional nonce override for gap recovery",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"options_json": &schema.StringAttribute{
				Optional:      true,
				Description:   "Additional options for the invocation, as a JSON object string (EVM connector semantics)",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"idempotency_key": &schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "Idempotency key for the workflow-engine transaction. When unset, a deterministic key is derived from the invocation inputs so this resource only ever submits one " +
					"unique transaction. Set explicitly to force a distinct invocation with otherwise identical inputs (e.g. to call the same function again with the same parameters).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"wait_timeout": &schema.StringAttribute{
				Optional:    true,
				Description: "Maximum time to wait for the invocation transaction to complete (Go duration string, default 10m)",
			},
			"ignore_destroy": &schema.BoolAttribute{
				Optional:    true,
				Description: "When true, destroy leaves the workflow-engine transaction record in place (the effect of the transaction always remains on-chain)",
			},
			"transaction_hash": &schema.StringAttribute{
				Computed:    true,
				Description: "Hash of the invocation transaction, from the transaction receipt",
			},
			"block_number": &schema.StringAttribute{
				Computed:    true,
				Description: "Block number the invocation transaction was mined in, from the transaction receipt",
			},
		},
	}
}

func (r *evmConnectorContractInvokeResource) invokePath(data *EVMConnectorContractInvokeResourceModel) string {
	return fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/apis/%s/api/contract/invoke",
		data.Environment.ValueString(), data.Service.ValueString(), data.API.ValueString())
}

func (r *evmConnectorContractInvokeResource) transactionPath(data *EVMConnectorContractInvokeResourceModel, idOrIdempotencyKey, suffix string) string {
	return fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/transactions/%s%s",
		data.Environment.ValueString(), data.Service.ValueString(), idOrIdempotencyKey, suffix)
}

// deriveIdempotencyKey deterministically derives an idempotency key from all the invocation
// inputs, so the same resource configuration always maps to the same workflow-engine
// transaction - a re-submission of the same invocation (e.g. after an apply that submitted
// the transaction but failed to record it in state) is rejected with a 409 rather than
// submitting a second transaction.
func (data *EVMConnectorContractInvokeResourceModel) deriveIdempotencyKey() string {
	hashInputs := []string{
		data.Environment.ValueString(),
		data.Service.ValueString(),
		data.API.ValueString(),
		data.Key.ValueString(),
		data.To.ValueString(),
		data.ABI.ValueString(),
		data.Method.ValueString(),
		data.ParamsJSON.ValueString(),
		data.Value.ValueString(),
		data.Gas.ValueString(),
		data.Nonce.ValueString(),
		data.OptionsJSON.ValueString(),
	}
	hashJSON, _ := json.Marshal(hashInputs)
	hash := sha256.Sum256(hashJSON)
	return "tfinvoke-" + hex.EncodeToString(hash[:16])
}

func (data *EVMConnectorContractInvokeResourceModel) toAPI(api *EVMConnectorInvokeSubmitAPIModel, diagnostics *diag.Diagnostics) bool {
	api.IdempotencyKey = data.IdempotencyKey.ValueString()
	api.Input = EVMConnectorInvokeInputAPIModel{
		Key:    data.Key.ValueString(),
		To:     data.To.ValueString(),
		Method: data.Method.ValueString(),
		Value:  data.Value.ValueString(),
		Gas:    data.Gas.ValueString(),
		Nonce:  data.Nonce.ValueString(),
	}
	if err := json.Unmarshal([]byte(data.ABI.ValueString()), &api.Input.ABI); err != nil {
		diagnostics.AddError("invalid ABI", fmt.Sprintf("abi must be valid JSON: %s", err))
		return false
	}
	if data.ParamsJSON.ValueString() != "" {
		if err := json.Unmarshal([]byte(data.ParamsJSON.ValueString()), &api.Input.Params); err != nil {
			diagnostics.AddError("invalid params JSON", fmt.Sprintf("params_json must be valid JSON: %s", err))
			return false
		}
	}
	if data.OptionsJSON.ValueString() != "" {
		if err := json.Unmarshal([]byte(data.OptionsJSON.ValueString()), &api.Input.Options); err != nil {
			diagnostics.AddError("invalid options JSON", fmt.Sprintf("options_json must be valid JSON: %s", err))
			return false
		}
	}
	return true
}

func (data *EVMConnectorContractInvokeResourceModel) fromAPI(api *EVMConnectorTransactionAPIModel) {
	data.ID = types.StringValue(api.ID)
	data.IdempotencyKey = types.StringValue(api.IdempotencyKey)
	transactionHash := ""
	blockNumber := ""
	if api.Output != nil && api.Output.Receipt != nil {
		transactionHash = api.Output.Receipt.TransactionHash
		blockNumber = rawJSONToString(api.Output.Receipt.BlockNumber)
	}
	data.TransactionHash = types.StringValue(transactionHash)
	data.BlockNumber = types.StringValue(blockNumber)
}

func (data *EVMConnectorContractInvokeResourceModel) waitTimeout(diagnostics *diag.Diagnostics) time.Duration {
	if data.WaitTimeout.ValueString() == "" {
		return evmConnectorInvokeDefaultWaitTimeout
	}
	timeout, err := time.ParseDuration(data.WaitTimeout.ValueString())
	if err != nil {
		diagnostics.AddError("invalid wait_timeout", fmt.Sprintf("wait_timeout must be a valid duration string: %s", err))
		return 0
	}
	return timeout
}

func (r *evmConnectorContractInvokeResource) failedTransactionError(api *EVMConnectorTransactionAPIModel) string {
	errorInfo := api.Error
	if errorInfo == "" {
		errorInfo = api.OutputError
	}
	return fmt.Sprintf("invoke transaction %s (idempotencyKey=%s) is in stage '%s': %s\n"+
		"The contract invocation did not complete successfully. Replace this resource (e.g. terraform taint / terraform apply -replace) to delete the failed transaction and submit a new invocation.",
		api.ID, api.IdempotencyKey, api.Stage, errorInfo)
}

func (r *evmConnectorContractInvokeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var data EVMConnectorContractInvokeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.IdempotencyKey.IsNull() || data.IdempotencyKey.IsUnknown() || data.IdempotencyKey.ValueString() == "" {
		data.IdempotencyKey = types.StringValue(data.deriveIdempotencyKey())
	}

	var submit EVMConnectorInvokeSubmitAPIModel
	if !data.toAPI(&submit, &resp.Diagnostics) {
		return
	}
	timeout := data.waitTimeout(&resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	txID := ""
	var result EVMConnectorSubmitResultAPIModel
	submitDiags := diag.Diagnostics{}
	ok, statusCode := r.apiRequest(ctx, http.MethodPost, r.invokePath(&data), submit, &result, &submitDiags)
	switch {
	case ok:
		txID = result.ID
	case statusCode == http.StatusConflict:
		// Another transaction - possibly one that is not this contract invocation at all -
		// already holds this idempotency key. Never adopt it automatically: fail the apply
		// so the user can inspect the existing transaction and decide what to do.
		resp.Diagnostics.AddError("invoke idempotency key conflict", fmt.Sprintf(
			"idempotency key '%s' is already in use by an existing transaction on this connector, which cannot be assumed to be this contract invocation. "+
				"Inspect it via the connector's transactions API (GET /api/v1/transactions/%s), then either delete that transaction if it is unwanted, "+
				"or set a distinct explicit idempotency_key on this resource.",
			submit.IdempotencyKey, submit.IdempotencyKey))
		return
	default:
		resp.Diagnostics.Append(submitDiags...)
		return
	}
	if txID == "" {
		resp.Diagnostics.AddError("invoke submission failed", "no transaction ID returned from invoke submission")
		return
	}

	var api EVMConnectorTransactionAPIModel
	if !waitForEVMConnectorTransaction(ctx, &r.commonResource, "invoke", r.transactionPath(&data, txID, "/wait"), txID, timeout, &api, r.failedTransactionError, &resp.Diagnostics) {
		return
	}

	data.fromAPI(&api)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *evmConnectorContractInvokeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data EVMConnectorContractInvokeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var api EVMConnectorTransactionAPIModel
	ok, status := r.apiRequest(ctx, http.MethodGet, r.transactionPath(&data, data.ID.ValueString(), ""), nil, &api, &resp.Diagnostics, Allow404())
	if !ok {
		return
	}
	if status == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	data.fromAPI(&api)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *evmConnectorContractInvokeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// All invocation inputs require replacement, so an in-place update only ever changes
	// non-functional attributes (wait_timeout / ignore_destroy) - just refresh the
	// computed attributes from the existing transaction.
	var data EVMConnectorContractInvokeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var api EVMConnectorTransactionAPIModel
	if ok, _ := r.apiRequest(ctx, http.MethodGet, r.transactionPath(&data, data.ID.ValueString(), ""), nil, &api, &resp.Diagnostics); !ok {
		return
	}

	data.fromAPI(&api)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *evmConnectorContractInvokeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data EVMConnectorContractInvokeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !data.IgnoreDestroy.IsNull() && data.IgnoreDestroy.ValueBool() {
		return
	}

	// The effect of the transaction cannot be undone on-chain - deleting the workflow-engine
	// transaction record frees the idempotency key so a recreate submits a fresh invocation.
	_, _ = r.apiRequest(ctx, http.MethodDelete, r.transactionPath(&data, data.ID.ValueString(), ""), nil, nil, &resp.Diagnostics, Allow404())
}
