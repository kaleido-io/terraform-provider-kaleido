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
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type TransactionManagerServiceResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Environment         types.String `tfsdk:"environment"`
	EnvironmentMemberID types.String `tfsdk:"environment_member_id"`
	Runtime             types.String `tfsdk:"runtime"`
	Name                types.String `tfsdk:"name"`
	StackID             types.String `tfsdk:"stack_id"`
	Type                types.String `tfsdk:"type"`
	KeyManager          types.String `tfsdk:"key_manager"`

	// EVM configuration - flattened key fields
	EVMConnectorType     types.String `tfsdk:"evm_connector_type"` // "url" or "evmGateway"
	EVMConnectorURL      types.String `tfsdk:"evm_connector_url"`
	EVMConnectorGateway  types.String `tfsdk:"evm_connector_gateway"`
	EVMConnectorUsername types.String `tfsdk:"evm_connector_username"`
	EVMConnectorPassword types.String `tfsdk:"evm_connector_password"`
	// TODO flatten these
	EVMConnectorConfigJSON types.String `tfsdk:"evm_connector_config_json"`
	EVMTransactionsJSON    types.String `tfsdk:"evm_transactions_json"`
	EVMEventstreamsJSON    types.String `tfsdk:"evm_eventstreams_json"`
	EVMConfirmationsJSON   types.String `tfsdk:"evm_confirmations_json"`
	EVMTxwriterJSON        types.String `tfsdk:"evm_txwriter_json"`

	// TODO remove / ignore
	// Fabric configuration
	FabricSigner            types.String `tfsdk:"fabric_signer"`
	FabricCCPConfig         types.String `tfsdk:"fabric_ccp_config"`  // JSON file content
	FabricCCPFiles          types.String `tfsdk:"fabric_ccp_files"`   // PEM files as JSON object
	FabricMSPArchive        types.String `tfsdk:"fabric_msp_archive"` // tar.gz content
	FabricConfigurationJSON types.String `tfsdk:"fabric_configuration_json"`

	ForceDelete types.Bool `tfsdk:"force_delete"`
}

func TransactionManagerServiceResourceFactory() resource.Resource {
	return &transactionManagerServiceResource{}
}

type transactionManagerServiceResource struct {
	commonResource
}

func (r *transactionManagerServiceResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_transactionmanager_service"
}

func (r *transactionManagerServiceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A Transaction Manager service that handles blockchain transaction submission and management.",
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"environment": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Environment ID where the Transaction Manager service will be deployed",
			},
			"environment_member_id": &schema.StringAttribute{
				Computed: true,
			},
			"runtime": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Runtime ID where the Transaction Manager service will be deployed",
			},
			"name": &schema.StringAttribute{
				Required:    true,
				Description: "Display name for the Transaction Manager service",
			},
			"stack_id": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Stack ID where the Transaction Manager service belongs (must be a FireflyStack)",
			},
			"type": &schema.StringAttribute{
				Required:      true,
				Description:   "The type of blockchain to connect to. Options: evm, fabric",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"key_manager": &schema.StringAttribute{
				Optional:    true,
				Description: "Key Manager service ID (required for EVM type)",
			},

			// EVM configuration
			"evm_connector_type": &schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("url"),
				Description: "EVM connector type. Options: url, evmGateway",
			},
			"evm_connector_url": &schema.StringAttribute{
				Optional:    true,
				Description: "The JSON/RPC endpoint URL (used when connector_type is 'url')",
			},
			"evm_connector_gateway": &schema.StringAttribute{
				Optional:    true,
				Description: "EVM Gateway service ID (used when connector_type is 'evmGateway')",
			},
			"evm_connector_username": &schema.StringAttribute{
				Optional:    true,
				Description: "Username for JSON/RPC authentication",
			},
			"evm_connector_password": &schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Password for JSON/RPC authentication",
			},
			"evm_connector_config_json": &schema.StringAttribute{
				Optional:    true,
				Description: "Additional connector configuration as JSON (throttle, retry, timeouts, etc.)",
			},
			"evm_transactions_json": &schema.StringAttribute{
				Optional:    true,
				Description: "Transaction handling configuration as JSON",
			},
			"evm_eventstreams_json": &schema.StringAttribute{
				Optional:    true,
				Description: "Event streams configuration as JSON",
			},
			"evm_confirmations_json": &schema.StringAttribute{
				Optional:    true,
				Description: "Confirmations configuration as JSON",
			},
			"evm_txwriter_json": &schema.StringAttribute{
				Optional:    true,
				Description: "Transaction writer configuration as JSON",
			},

			// Fabric configuration
			"fabric_signer": &schema.StringAttribute{
				Optional:    true,
				Description: "Default signer for consuming events (required for Fabric type)",
			},
			"fabric_ccp_config": &schema.StringAttribute{
				Optional:    true,
				Description: "Connection profile JSON configuration",
			},
			"fabric_ccp_files": &schema.StringAttribute{
				Optional:    true,
				Description: "Additional PEM files as JSON object (file name -> content)",
			},
			"fabric_msp_archive": &schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "MSP archive containing keys and certificates (base64 encoded tar.gz)",
			},
			"fabric_configuration_json": &schema.StringAttribute{
				Optional:    true,
				Description: "FabConnect configuration as JSON",
			},

			"force_delete": &schema.BoolAttribute{
				Optional:    true,
				Description: "Set to true when you plan to delete a protected Transaction Manager service. You must apply this value before running terraform destroy.",
			},
		},
	}
}

func (data *TransactionManagerServiceResourceModel) toServiceAPI(ctx context.Context, api *ServiceAPIModel, diagnostics *diag.Diagnostics) {
	api.Type = "TransactionManagerService"
	api.Name = data.Name.ValueString()
	api.StackID = data.StackID.ValueString()
	api.Runtime.ID = data.Runtime.ValueString()
	api.Config = make(map[string]interface{})

	// Type
	if !data.Type.IsNull() {
		api.Config["type"] = data.Type.ValueString()
	}

	// Key Manager reference (required for EVM)
	if !data.KeyManager.IsNull() && data.KeyManager.ValueString() != "" {
		api.Config["keyManager"] = map[string]interface{}{
			"id": data.KeyManager.ValueString(),
		}
	}

	// EVM configuration
	if data.Type.ValueString() == "evm" {
		evmConfig := make(map[string]interface{})

		// Connector configuration
		connectorConfig := make(map[string]interface{})
		connectorType := data.EVMConnectorType.ValueString()

		if connectorType == "evmGateway" && !data.EVMConnectorGateway.IsNull() {
			connectorConfig["evmGateway"] = map[string]interface{}{
				"id": data.EVMConnectorGateway.ValueString(),
			}
		} else if !data.EVMConnectorURL.IsNull() && data.EVMConnectorURL.ValueString() != "" {
			connectorConfig["url"] = data.EVMConnectorURL.ValueString()
		}

		// Handle connector authentication
		if (!data.EVMConnectorUsername.IsNull() && data.EVMConnectorUsername.ValueString() != "") ||
			(!data.EVMConnectorPassword.IsNull() && data.EVMConnectorPassword.ValueString() != "") {
			credSetName := "jsonRpcAuth"
			if api.Credsets == nil {
				api.Credsets = make(map[string]*CredSetAPI)
			}
			api.Credsets[credSetName] = &CredSetAPI{
				Name: credSetName,
				Type: "basic_auth",
				BasicAuth: &CredSetBasicAuthAPI{
					Username: data.EVMConnectorUsername.ValueString(),
					Password: data.EVMConnectorPassword.ValueString(),
				},
			}
			connectorConfig["auth"] = map[string]interface{}{
				"credSetRef": credSetName,
			}
		}

		// Additional connector config from JSON
		if !data.EVMConnectorConfigJSON.IsNull() && data.EVMConnectorConfigJSON.ValueString() != "" {
			var additionalConfig interface{}
			err := json.Unmarshal([]byte(data.EVMConnectorConfigJSON.ValueString()), &additionalConfig)
			if err != nil {
				diagnostics.AddAttributeError(
					path.Root("evm_connector_config_json"),
					"Failed to parse evm_connector_config_json",
					err.Error(),
				)
			} else {
				if configMap, ok := additionalConfig.(map[string]interface{}); ok {
					for k, v := range configMap {
						connectorConfig[k] = v
					}
				}
			}
		}

		evmConfig["connector"] = connectorConfig

		// Transactions config
		if !data.EVMTransactionsJSON.IsNull() && data.EVMTransactionsJSON.ValueString() != "" {
			var transactionsConfig interface{}
			err := json.Unmarshal([]byte(data.EVMTransactionsJSON.ValueString()), &transactionsConfig)
			if err != nil {
				diagnostics.AddAttributeError(
					path.Root("evm_transactions_json"),
					"Failed to parse evm_transactions_json",
					err.Error(),
				)
			} else {
				evmConfig["transactions"] = transactionsConfig
			}
		}

		// Event streams config
		if !data.EVMEventstreamsJSON.IsNull() && data.EVMEventstreamsJSON.ValueString() != "" {
			var eventstreamsConfig interface{}
			err := json.Unmarshal([]byte(data.EVMEventstreamsJSON.ValueString()), &eventstreamsConfig)
			if err != nil {
				diagnostics.AddAttributeError(
					path.Root("evm_eventstreams_json"),
					"Failed to parse evm_eventstreams_json",
					err.Error(),
				)
			} else {
				evmConfig["eventstreams"] = eventstreamsConfig
			}
		}

		// Confirmations config
		if !data.EVMConfirmationsJSON.IsNull() && data.EVMConfirmationsJSON.ValueString() != "" {
			var confirmationsConfig interface{}
			err := json.Unmarshal([]byte(data.EVMConfirmationsJSON.ValueString()), &confirmationsConfig)
			if err != nil {
				diagnostics.AddAttributeError(
					path.Root("evm_confirmations_json"),
					"Failed to parse evm_confirmations_json",
					err.Error(),
				)
			} else {
				evmConfig["confirmations"] = confirmationsConfig
			}
		}

		// TxWriter config
		if !data.EVMTxwriterJSON.IsNull() && data.EVMTxwriterJSON.ValueString() != "" {
			var txwriterConfig interface{}
			err := json.Unmarshal([]byte(data.EVMTxwriterJSON.ValueString()), &txwriterConfig)
			if err != nil {
				diagnostics.AddAttributeError(
					path.Root("evm_txwriter_json"),
					"Failed to parse evm_txwriter_json",
					err.Error(),
				)
			} else {
				evmConfig["txwriter"] = txwriterConfig
			}
		}

		api.Config["evm"] = evmConfig
	}

	// Fabric configuration
	if data.Type.ValueString() == "fabric" {
		fabricConfig := make(map[string]interface{})

		// Signer
		if !data.FabricSigner.IsNull() {
			fabricConfig["signer"] = data.FabricSigner.ValueString()
		}

		// CCP configuration
		if !data.FabricCCPConfig.IsNull() && data.FabricCCPConfig.ValueString() != "" {
			ccpFileSetName := "ccpFiles"
			if api.Filesets == nil {
				api.Filesets = make(map[string]*FileSetAPI)
			}
			api.Filesets[ccpFileSetName] = &FileSetAPI{
				Files: map[string]*FileAPI{
					"config.json": {
						Data: FileDataAPI{
							Text: data.FabricCCPConfig.ValueString(),
						},
					},
				},
			}

			// Add additional PEM files if provided
			if !data.FabricCCPFiles.IsNull() && data.FabricCCPFiles.ValueString() != "" {
				var pemFiles map[string]string
				err := json.Unmarshal([]byte(data.FabricCCPFiles.ValueString()), &pemFiles)
				if err != nil {
					diagnostics.AddAttributeError(
						path.Root("fabric_ccp_files"),
						"Failed to parse fabric_ccp_files JSON",
						err.Error(),
					)
				} else {
					for filename, content := range pemFiles {
						api.Filesets[ccpFileSetName].Files[filename] = &FileAPI{
							Data: FileDataAPI{
								Text: content,
							},
						}
					}
				}
			}

			fabricConfig["ccp"] = map[string]interface{}{
				"config": map[string]interface{}{
					"fileRef": "config.json",
				},
				"files": map[string]interface{}{
					"fileSetRef": ccpFileSetName,
				},
			}
		}

		// MSP configuration
		if !data.FabricMSPArchive.IsNull() && data.FabricMSPArchive.ValueString() != "" {
			mspFileSetName := "mspFiles"
			if api.Filesets == nil {
				api.Filesets = make(map[string]*FileSetAPI)
			}
			api.Filesets[mspFileSetName] = &FileSetAPI{
				Files: map[string]*FileAPI{
					"msp.tar.gz": {
						Data: FileDataAPI{
							Base64: data.FabricMSPArchive.ValueString(),
						},
					},
				},
			}

			fabricConfig["msp"] = map[string]interface{}{
				"mspArchive": map[string]interface{}{
					"fileRef": "msp.tar.gz",
				},
			}
		}

		// FabConnect configuration
		if !data.FabricConfigurationJSON.IsNull() && data.FabricConfigurationJSON.ValueString() != "" {
			var configurationConfig interface{}
			err := json.Unmarshal([]byte(data.FabricConfigurationJSON.ValueString()), &configurationConfig)
			if err != nil {
				diagnostics.AddAttributeError(
					path.Root("fabric_configuration_json"),
					"Failed to parse fabric_configuration_json",
					err.Error(),
				)
			} else {
				fabricConfig["configuration"] = configurationConfig
			}
		}

		api.Config["fabric"] = fabricConfig
	}
}

func (api *ServiceAPIModel) toTransactionManagerServiceData(data *TransactionManagerServiceResourceModel, diagnostics *diag.Diagnostics) {
	data.ID = types.StringValue(api.ID)
	data.EnvironmentMemberID = types.StringValue(api.EnvironmentMemberID)
	data.Runtime = types.StringValue(api.Runtime.ID)
	data.Name = types.StringValue(api.Name)
	data.StackID = types.StringValue(api.StackID)

	// Type
	if v, ok := api.Config["type"].(string); ok {
		data.Type = types.StringValue(v)
	} else {
		data.Type = types.StringNull()
	}

	// Key Manager
	if v, ok := api.Config["keyManager"].(map[string]interface{}); ok {
		if id, ok := v["id"].(string); ok {
			data.KeyManager = types.StringValue(id)
		}
	} else {
		data.KeyManager = types.StringNull()
	}

	// EVM configuration
	if v, ok := api.Config["evm"].(map[string]interface{}); ok {
		// Connector configuration
		if connector, ok := v["connector"].(map[string]interface{}); ok {
			if evmGateway, exists := connector["evmGateway"]; exists {
				data.EVMConnectorType = types.StringValue("evmGateway")
				if gw, ok := evmGateway.(map[string]interface{}); ok {
					if id, ok := gw["id"].(string); ok {
						data.EVMConnectorGateway = types.StringValue(id)
					}
				}
			} else if url, ok := connector["url"].(string); ok {
				data.EVMConnectorType = types.StringValue("url")
				data.EVMConnectorURL = types.StringValue(url)
			}
		}

		// Handle connector authentication from credsets
		data.EVMConnectorUsername = types.StringNull()
		data.EVMConnectorPassword = types.StringNull()
		if api.Credsets != nil {
			if credSet, ok := api.Credsets["jsonRpcAuth"]; ok && credSet != nil && credSet.BasicAuth != nil {
				data.EVMConnectorUsername = types.StringValue(credSet.BasicAuth.Username)
				data.EVMConnectorPassword = types.StringValue(credSet.BasicAuth.Password)
			}
		}

		// Other EVM configs as JSON
		if transactions := v["transactions"]; transactions != nil {
			if transactionsJSON, err := json.Marshal(transactions); err == nil {
				data.EVMTransactionsJSON = types.StringValue(string(transactionsJSON))
			}
		} else {
			data.EVMTransactionsJSON = types.StringNull()
		}

		if eventstreams := v["eventstreams"]; eventstreams != nil {
			if eventstreamsJSON, err := json.Marshal(eventstreams); err == nil {
				data.EVMEventstreamsJSON = types.StringValue(string(eventstreamsJSON))
			}
		} else {
			data.EVMEventstreamsJSON = types.StringNull()
		}

		if confirmations := v["confirmations"]; confirmations != nil {
			if confirmationsJSON, err := json.Marshal(confirmations); err == nil {
				data.EVMConfirmationsJSON = types.StringValue(string(confirmationsJSON))
			}
		} else {
			data.EVMConfirmationsJSON = types.StringNull()
		}

		if txwriter := v["txwriter"]; txwriter != nil {
			if txwriterJSON, err := json.Marshal(txwriter); err == nil {
				data.EVMTxwriterJSON = types.StringValue(string(txwriterJSON))
			}
		} else {
			data.EVMTxwriterJSON = types.StringNull()
		}
	} else {
		// Clear EVM fields
		data.EVMConnectorType = types.StringNull()
		data.EVMConnectorURL = types.StringNull()
		data.EVMConnectorGateway = types.StringNull()
		data.EVMConnectorUsername = types.StringNull()
		data.EVMConnectorPassword = types.StringNull()
		data.EVMConnectorConfigJSON = types.StringNull()
		data.EVMTransactionsJSON = types.StringNull()
		data.EVMEventstreamsJSON = types.StringNull()
		data.EVMConfirmationsJSON = types.StringNull()
		data.EVMTxwriterJSON = types.StringNull()
	}

	// Fabric configuration
	if v, ok := api.Config["fabric"].(map[string]interface{}); ok {
		if signer, ok := v["signer"].(string); ok {
			data.FabricSigner = types.StringValue(signer)
		} else {
			data.FabricSigner = types.StringNull()
		}

		// Handle file references from filesets
		data.FabricCCPConfig = types.StringNull()
		data.FabricCCPFiles = types.StringNull()
		data.FabricMSPArchive = types.StringNull()

		if api.Filesets != nil {
			if ccpFileSet, ok := api.Filesets["ccpFiles"]; ok && ccpFileSet != nil {
				if configFile, ok := ccpFileSet.Files["config.json"]; ok && configFile != nil {
					data.FabricCCPConfig = types.StringValue(configFile.Data.Text)
				}

				// Collect additional PEM files
				pemFiles := make(map[string]string)
				for filename, file := range ccpFileSet.Files {
					if filename != "config.json" && file != nil {
						pemFiles[filename] = file.Data.Text
					}
				}
				if len(pemFiles) > 0 {
					if pemFilesJSON, err := json.Marshal(pemFiles); err == nil {
						data.FabricCCPFiles = types.StringValue(string(pemFilesJSON))
					}
				}
			}

			if mspFileSet, ok := api.Filesets["mspFiles"]; ok && mspFileSet != nil {
				if mspFile, ok := mspFileSet.Files["msp.tar.gz"]; ok && mspFile != nil {
					data.FabricMSPArchive = types.StringValue(mspFile.Data.Base64)
				}
			}
		}

		if configuration := v["configuration"]; configuration != nil {
			if configurationJSON, err := json.Marshal(configuration); err == nil {
				data.FabricConfigurationJSON = types.StringValue(string(configurationJSON))
			}
		} else {
			data.FabricConfigurationJSON = types.StringNull()
		}
	} else {
		// Clear Fabric fields
		data.FabricSigner = types.StringNull()
		data.FabricCCPConfig = types.StringNull()
		data.FabricCCPFiles = types.StringNull()
		data.FabricMSPArchive = types.StringNull()
		data.FabricConfigurationJSON = types.StringNull()
	}
}

func (r *transactionManagerServiceResource) apiPath(data *TransactionManagerServiceResourceModel) string {
	path := fmt.Sprintf("/api/v1/environments/%s/services", data.Environment.ValueString())
	if data.ID.ValueString() != "" {
		path = path + "/" + data.ID.ValueString()
	}
	if !data.ForceDelete.IsNull() && data.ForceDelete.ValueBool() {
		path = path + "?force=true"
	}
	return path
}

func (r *transactionManagerServiceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data TransactionManagerServiceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	var api ServiceAPIModel
	data.toServiceAPI(ctx, &api, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	ok, _ := r.apiRequest(ctx, http.MethodPost, r.apiPath(&data), api, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	api.toTransactionManagerServiceData(&data, &resp.Diagnostics)
	r.waitForReadyStatus(ctx, r.apiPath(&data), &resp.Diagnostics)

	api.ID = data.ID.ValueString()
	ok, _ = r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	api.toTransactionManagerServiceData(&data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *transactionManagerServiceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data TransactionManagerServiceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)

	var api ServiceAPIModel
	if ok, _ := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics); !ok {
		return
	}

	data.toServiceAPI(ctx, &api, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if ok, _ := r.apiRequest(ctx, http.MethodPut, r.apiPath(&data), api, &api, &resp.Diagnostics); !ok {
		return
	}

	api.toTransactionManagerServiceData(&data, &resp.Diagnostics)
	r.waitForReadyStatus(ctx, r.apiPath(&data), &resp.Diagnostics)

	if ok, _ := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics); !ok {
		return
	}
	api.toTransactionManagerServiceData(&data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *transactionManagerServiceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data TransactionManagerServiceResourceModel
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

	api.toTransactionManagerServiceData(&data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *transactionManagerServiceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data TransactionManagerServiceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data), nil, nil, &resp.Diagnostics, Allow404())
	r.waitForRemoval(ctx, r.apiPath(&data), &resp.Diagnostics)
}
