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

	"github.com/kaleido-io/terraform-provider-kaleido/kaleido/kaleidobase"
)

const evmConnectorDeployDefaultWaitTimeout = 10 * time.Minute

type EVMConnectorContractDeployResourceModel struct {
	ID              types.String `tfsdk:"id"`
	Environment     types.String `tfsdk:"environment"`
	Service         types.String `tfsdk:"service"`
	API             types.String `tfsdk:"api"`
	Key             types.String `tfsdk:"key"`
	ABI             types.String `tfsdk:"abi"`
	Bytecode        types.String `tfsdk:"bytecode"`
	ParamsJSON      types.String `tfsdk:"params_json"`
	Value           types.String `tfsdk:"value"`
	Gas             types.String `tfsdk:"gas"`
	Nonce           types.String `tfsdk:"nonce"`
	OptionsJSON     types.String `tfsdk:"options_json"`
	IdempotencyKey  types.String `tfsdk:"idempotency_key"`
	WaitTimeout     types.String `tfsdk:"wait_timeout"`
	IgnoreDestroy   types.Bool   `tfsdk:"ignore_destroy"`
	ContractAddress types.String `tfsdk:"contract_address"`
	TransactionHash types.String `tfsdk:"transaction_hash"`
	BlockNumber     types.String `tfsdk:"block_number"`
}

// EVMConnectorDeployInputAPIModel is the standard EVM API "contract/deploy" operation input
type EVMConnectorDeployInputAPIModel struct {
	Key      string `json:"key"`
	ABI      any    `json:"abi"`
	Bytecode string `json:"bytecode"`
	Params   any    `json:"params,omitempty"`
	Value    string `json:"value,omitempty"`
	Gas      string `json:"gas,omitempty"`
	Nonce    string `json:"nonce,omitempty"`
	Options  any    `json:"options,omitempty"`
}

type EVMConnectorDeploySubmitAPIModel struct {
	IdempotencyKey string                          `json:"idempotencyKey"`
	Input          EVMConnectorDeployInputAPIModel `json:"input"`
}

// EVMConnectorSubmitResultAPIModel is the 202 result of an asynchronous categorized operation
type EVMConnectorSubmitResultAPIModel struct {
	ID             string `json:"id,omitempty"`
	IdempotencyKey string `json:"idempotencyKey,omitempty"`
	Preexisting    bool   `json:"preexisting,omitempty"`
}

// EVMConnectorTransactionAPIModel is the subset of the workflow-engine TransactionWithState we consume
type EVMConnectorTransactionAPIModel struct {
	ID             string                            `json:"id,omitempty"`
	IdempotencyKey string                            `json:"idempotencyKey,omitempty"`
	Status         string                            `json:"status,omitempty"`
	Stage          string                            `json:"stage,omitempty"`
	Output         *EVMConnectorDeployOutputAPIModel `json:"output,omitempty"`
	OutputError    string                            `json:"outputError,omitempty"`
	Error          string                            `json:"error,omitempty"`
}

type EVMConnectorDeployOutputAPIModel struct {
	Receipt *EVMConnectorDeployReceiptAPIModel `json:"receipt,omitempty"`
}

type EVMConnectorDeployReceiptAPIModel struct {
	TransactionHash string          `json:"transactionHash,omitempty"`
	ContractAddress string          `json:"contractAddress,omitempty"`
	BlockNumber     json.RawMessage `json:"blockNumber,omitempty"`
}

func EVMConnectorContractDeployResourceFactory() resource.Resource {
	return &evmConnectorContractDeployResource{}
}

type evmConnectorContractDeployResource struct {
	commonResource
}

func (r *evmConnectorContractDeployResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_evm_connector_contract_deploy"
}

func (r *evmConnectorContractDeployResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Deploys a smart contract to an EVM chain through the 'contract/deploy' operation of a deployed EVM standard API on a connector service (see kaleido_platform_connector_standard_api). " +
			"The deployment is submitted as a workflow-engine transaction with a deterministic idempotency key, so each resource instance only ever submits one unique transaction. " +
			"If the idempotency key is already in use by an existing transaction the apply fails, for the user to inspect that transaction and resolve the conflict.",
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				Description:   "The workflow-engine transaction ID of the deployment",
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
			"abi": &schema.StringAttribute{
				Required:      true,
				Description:   "The contract ABI as a JSON string, including the constructor definition (pipes directly from the `abi` output of kaleido_platform_cms_build)",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"bytecode": &schema.StringAttribute{
				Required:      true,
				Description:   "The contract bytecode to deploy, as a hex string (pipes directly from the `bytecode` output of kaleido_platform_cms_build)",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"params_json": &schema.StringAttribute{
				Optional:      true,
				Description:   "Constructor parameters as a JSON array or object string",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"value": &schema.StringAttribute{
				Optional:      true,
				Description:   "Optional value in wei to send with the deployment",
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
				Description:   "Additional options for the deployment, as a JSON object string (EVM connector semantics)",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"idempotency_key": &schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "Idempotency key for the workflow-engine transaction. When unset, a deterministic key is derived from the deployment inputs so this resource only ever submits one " +
					"unique transaction. Set explicitly to force a distinct deployment with otherwise identical inputs.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"wait_timeout": &schema.StringAttribute{
				Optional:    true,
				Description: "Maximum time to wait for the deployment transaction to complete (Go duration string, default 10m)",
			},
			"ignore_destroy": &schema.BoolAttribute{
				Optional:    true,
				Description: "When true, destroy leaves the workflow-engine transaction record in place (the contract itself always remains on-chain)",
			},
			"contract_address": &schema.StringAttribute{
				Computed:    true,
				Description: "Address of the deployed contract, from the transaction receipt",
			},
			"transaction_hash": &schema.StringAttribute{
				Computed:    true,
				Description: "Hash of the deployment transaction, from the transaction receipt",
			},
			"block_number": &schema.StringAttribute{
				Computed:    true,
				Description: "Block number the deployment transaction was mined in, from the transaction receipt",
			},
		},
	}
}

func (r *evmConnectorContractDeployResource) deployPath(data *EVMConnectorContractDeployResourceModel) string {
	return fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/apis/%s/api/contract/deploy",
		data.Environment.ValueString(), data.Service.ValueString(), data.API.ValueString())
}

func (r *evmConnectorContractDeployResource) transactionPath(data *EVMConnectorContractDeployResourceModel, idOrIdempotencyKey, suffix string) string {
	return fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/transactions/%s%s",
		data.Environment.ValueString(), data.Service.ValueString(), idOrIdempotencyKey, suffix)
}

// deriveIdempotencyKey deterministically derives an idempotency key from all the deployment
// inputs, so the same resource configuration always maps to the same workflow-engine
// transaction - a re-submission of the same deployment (e.g. after an apply that submitted
// the transaction but failed to record it in state) is rejected with a 409 rather than
// deploying a second contract.
func (data *EVMConnectorContractDeployResourceModel) deriveIdempotencyKey() string {
	hashInputs := []string{
		data.Environment.ValueString(),
		data.Service.ValueString(),
		data.API.ValueString(),
		data.Key.ValueString(),
		data.ABI.ValueString(),
		data.Bytecode.ValueString(),
		data.ParamsJSON.ValueString(),
		data.Value.ValueString(),
		data.Gas.ValueString(),
		data.Nonce.ValueString(),
		data.OptionsJSON.ValueString(),
	}
	hashJSON, _ := json.Marshal(hashInputs)
	hash := sha256.Sum256(hashJSON)
	return "tfdeploy-" + hex.EncodeToString(hash[:16])
}

func (data *EVMConnectorContractDeployResourceModel) toAPI(api *EVMConnectorDeploySubmitAPIModel, diagnostics *diag.Diagnostics) bool {
	api.IdempotencyKey = data.IdempotencyKey.ValueString()
	api.Input = EVMConnectorDeployInputAPIModel{
		Key:      data.Key.ValueString(),
		Bytecode: data.Bytecode.ValueString(),
		Value:    data.Value.ValueString(),
		Gas:      data.Gas.ValueString(),
		Nonce:    data.Nonce.ValueString(),
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

func (api *EVMConnectorTransactionAPIModel) toData(data *EVMConnectorContractDeployResourceModel) {
	data.ID = types.StringValue(api.ID)
	data.IdempotencyKey = types.StringValue(api.IdempotencyKey)
	contractAddress := ""
	transactionHash := ""
	blockNumber := ""
	if api.Output != nil && api.Output.Receipt != nil {
		contractAddress = api.Output.Receipt.ContractAddress
		transactionHash = api.Output.Receipt.TransactionHash
		blockNumber = rawJSONToString(api.Output.Receipt.BlockNumber)
	}
	data.ContractAddress = types.StringValue(contractAddress)
	data.TransactionHash = types.StringValue(transactionHash)
	data.BlockNumber = types.StringValue(blockNumber)
}

// rawJSONToString renders a raw JSON scalar (string or number) as its plain string value
func rawJSONToString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	return string(raw)
}

func (data *EVMConnectorContractDeployResourceModel) waitTimeout(diagnostics *diag.Diagnostics) time.Duration {
	if data.WaitTimeout.ValueString() == "" {
		return evmConnectorDeployDefaultWaitTimeout
	}
	timeout, err := time.ParseDuration(data.WaitTimeout.ValueString())
	if err != nil {
		diagnostics.AddError("invalid wait_timeout", fmt.Sprintf("wait_timeout must be a valid duration string: %s", err))
		return 0
	}
	return timeout
}

func (r *evmConnectorContractDeployResource) failedTransactionError(api *EVMConnectorTransactionAPIModel) string {
	errorInfo := api.Error
	if errorInfo == "" {
		errorInfo = api.OutputError
	}
	return fmt.Sprintf("deploy transaction %s (idempotencyKey=%s) is in stage '%s': %s\n"+
		"The contract deployment did not complete successfully. Replace this resource (e.g. terraform taint / terraform apply -replace) to delete the failed transaction and submit a new deployment.",
		api.ID, api.IdempotencyKey, api.Stage, errorInfo)
}

// waitForTransaction uses the workflow-engine transaction wait API to block until the
// transaction completes (or the wait_timeout expires). Each individual wait call is subject
// to server-side and gateway timeouts, so it is retried until a terminal status is reached.
func (r *evmConnectorContractDeployResource) waitForTransaction(ctx context.Context, data *EVMConnectorContractDeployResourceModel, txID string, api *EVMConnectorTransactionAPIModel, diagnostics *diag.Diagnostics) bool {
	timeout := data.waitTimeout(diagnostics)
	if diagnostics.HasError() {
		return false
	}
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	waitPath := r.transactionPath(data, txID, "/wait")
	cancelInfo := APICancelInfo()
	cancelInfo.CancelInfo = "(waiting for deploy transaction submission)"
	err := kaleidobase.Retry.Do(waitCtx, fmt.Sprintf("deploy-wait %s", waitPath), func(attempt int) (retry bool, err error) {
		attemptDiags := diag.Diagnostics{}
		ok, statusCode := r.apiRequest(waitCtx, http.MethodGet, waitPath, nil, api, &attemptDiags, cancelInfo)
		if !ok {
			// The wait API returns an error when its server-side timeout expires before the
			// transaction completes, and intermediate gateways can time the request out too -
			// keep retrying those (and transport errors) until our own wait_timeout expires.
			if statusCode >= 400 && statusCode < 500 && statusCode != 429 {
				diagnostics.Append(attemptDiags...)
				return false, fmt.Errorf("deploy-wait failed") // already set in diag
			}
			return true, fmt.Errorf("transaction wait incomplete (status %d)", statusCode)
		}
		cancelInfo.CancelInfo = fmt.Sprintf("(waiting for completion - status: %s stage: %s)", api.Status, api.Stage)
		switch api.Status {
		case "success":
			return false, nil
		case "failure":
			diagnostics.AddError("deploy failed", r.failedTransactionError(api))
			return false, fmt.Errorf("deploy failed")
		default:
			return true, fmt.Errorf("transaction not complete yet (status: %s)", api.Status)
		}
	})
	if err != nil {
		if !diagnostics.HasError() {
			diagnostics.AddError("deploy wait failed", fmt.Sprintf("failed waiting for deploy transaction %s to complete: %s", txID, err))
		}
		return false
	}
	return true
}

func (r *evmConnectorContractDeployResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var data EVMConnectorContractDeployResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.IdempotencyKey.IsNull() || data.IdempotencyKey.IsUnknown() || data.IdempotencyKey.ValueString() == "" {
		data.IdempotencyKey = types.StringValue(data.deriveIdempotencyKey())
	}

	var submit EVMConnectorDeploySubmitAPIModel
	if !data.toAPI(&submit, &resp.Diagnostics) {
		return
	}

	txID := ""
	var result EVMConnectorSubmitResultAPIModel
	submitDiags := diag.Diagnostics{}
	ok, statusCode := r.apiRequest(ctx, http.MethodPost, r.deployPath(&data), submit, &result, &submitDiags)
	switch {
	case ok:
		txID = result.ID
	case statusCode == http.StatusConflict:
		// Another transaction - possibly one that is not this contract deployment at all -
		// already holds this idempotency key. Never adopt it automatically: fail the apply
		// so the user can inspect the existing transaction and decide what to do.
		resp.Diagnostics.AddError("deploy idempotency key conflict", fmt.Sprintf(
			"idempotency key '%s' is already in use by an existing transaction on this connector, which cannot be assumed to be this contract deployment. "+
				"Inspect it via the connector's transactions API (GET /api/v1/transactions/%s), then either delete that transaction if it is unwanted, "+
				"or set a distinct explicit idempotency_key on this resource.",
			submit.IdempotencyKey, submit.IdempotencyKey))
		return
	default:
		resp.Diagnostics.Append(submitDiags...)
		return
	}
	if txID == "" {
		resp.Diagnostics.AddError("deploy submission failed", "no transaction ID returned from deploy submission")
		return
	}

	var api EVMConnectorTransactionAPIModel
	if !r.waitForTransaction(ctx, &data, txID, &api, &resp.Diagnostics) {
		return
	}

	api.toData(&data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *evmConnectorContractDeployResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data EVMConnectorContractDeployResourceModel
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

	api.toData(&data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *evmConnectorContractDeployResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// All deployment inputs require replacement, so an in-place update only ever changes
	// non-functional attributes (wait_timeout / ignore_destroy) - just refresh the
	// computed attributes from the existing transaction.
	var data EVMConnectorContractDeployResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var api EVMConnectorTransactionAPIModel
	if ok, _ := r.apiRequest(ctx, http.MethodGet, r.transactionPath(&data, data.ID.ValueString(), ""), nil, &api, &resp.Diagnostics); !ok {
		return
	}

	api.toData(&data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *evmConnectorContractDeployResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data EVMConnectorContractDeployResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !data.IgnoreDestroy.IsNull() && data.IgnoreDestroy.ValueBool() {
		return
	}

	// The contract itself cannot be removed from the chain - deleting the workflow-engine
	// transaction record frees the idempotency key so a recreate submits a fresh deployment.
	_, _ = r.apiRequest(ctx, http.MethodDelete, r.transactionPath(&data, data.ID.ValueString(), ""), nil, nil, &resp.Diagnostics, Allow404())
}
