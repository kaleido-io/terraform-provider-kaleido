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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type FireflyServiceResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Environment         types.String `tfsdk:"environment"`
	EnvironmentMemberID types.String `tfsdk:"environment_member_id"`
	Runtime             types.String `tfsdk:"runtime"`
	Name                types.String `tfsdk:"name"`
	StackID             types.String `tfsdk:"stack_id"`
	TransactionManager  types.String `tfsdk:"transaction_manager"`
	Key                 types.String `tfsdk:"key"`
	Description         types.String `tfsdk:"description"`

	// Multiparty configuration
	MultipartyEnabled          types.Bool   `tfsdk:"multiparty_enabled"`
	MultipartyOrgName          types.String `tfsdk:"multiparty_org_name"`
	MultipartyOrgKey           types.String `tfsdk:"multiparty_org_key"`
	MultipartyOrgDescription   types.String `tfsdk:"multiparty_org_description"`
	MultipartyNodeName         types.String `tfsdk:"multiparty_node_name"`
	MultipartyNetworkNamespace types.String `tfsdk:"multiparty_network_namespace"`
	MultipartyContractsJSON    types.String `tfsdk:"multiparty_contracts_json"`

	// IPFS configuration
	IPFSService    types.String `tfsdk:"ipfs_service"`
	IPFSAPIUrl     types.String `tfsdk:"ipfs_api_url"`
	IPFSGatewayUrl types.String `tfsdk:"ipfs_gateway_url"`
	IPFSUsername   types.String `tfsdk:"ipfs_username"`
	IPFSPassword   types.String `tfsdk:"ipfs_password"`
	IPFSProxyUrl   types.String `tfsdk:"ipfs_proxy_url"`

	// Other services
	PrivateDataManager types.String `tfsdk:"private_data_manager"`
	TokenIndexersJSON  types.String `tfsdk:"token_indexers_json"`

	// TLS configurations as JSON
	TLSConfigsJSON types.String `tfsdk:"tls_configs_json"`

	ForceDelete types.Bool `tfsdk:"force_delete"`
}

func FireflyServiceResourceFactory() resource.Resource {
	return &fireflyServiceResource{}
}

type fireflyServiceResource struct {
	commonResource
}

func (r *fireflyServiceResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_firefly_service"
}

func (r *fireflyServiceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A FireFly service that provides Web3 middleware capabilities for multi-party systems.",
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"environment": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Environment ID where the Firefly service will be deployed",
			},
			"environment_member_id": &schema.StringAttribute{
				Computed: true,
			},
			"runtime": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Runtime ID where the Firefly service will be deployed",
			},
			"name": &schema.StringAttribute{
				Required:    true,
				Description: "Display name for the Firefly service",
			},
			"stack_id": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Stack ID where the Firefly service belongs (must be a FireflyStack)",
			},
			"transaction_manager": &schema.StringAttribute{
				Required:    true,
				Description: "Transaction Manager service ID that this Firefly service will use",
			},
			"key": &schema.StringAttribute{
				Optional:    true,
				Description: "An optional default key to use when submitting blockchain transactions",
			},
			"description": &schema.StringAttribute{
				Optional:    true,
				Description: "Description for the Firefly service",
			},

			// Multiparty configuration
			"multiparty_enabled": &schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether to enable multiparty mode for this Firefly service",
			},
			"multiparty_org_name": &schema.StringAttribute{
				Optional:    true,
				Description: "Organization name for multiparty mode (required when multiparty_enabled is true)",
			},
			"multiparty_org_key": &schema.StringAttribute{
				Optional:    true,
				Description: "Organization key for multiparty mode. Falls back to the default signing key if not specified",
			},
			"multiparty_org_description": &schema.StringAttribute{
				Optional:    true,
				Description: "Organization description for multiparty mode",
			},
			"multiparty_node_name": &schema.StringAttribute{
				Optional:    true,
				Description: "Node name for multiparty mode",
			},
			"multiparty_network_namespace": &schema.StringAttribute{
				Optional:    true,
				Description: "Network namespace name for multiparty mode (must be same for all members, required when multiparty_enabled is true)",
			},
			"multiparty_contracts_json": &schema.StringAttribute{
				Optional:    true,
				Description: "JSON array of multiparty contracts configuration (required when multiparty_enabled is true)",
			},

			// IPFS configuration
			"ipfs_service": &schema.StringAttribute{
				Optional:    true,
				Description: "IPFS service ID to use for content storage (alternative to ipfs_api_url)",
			},
			"ipfs_api_url": &schema.StringAttribute{
				Optional:    true,
				Description: "IPFS API endpoint URL for upload (alternative to ipfs_service)",
			},
			"ipfs_gateway_url": &schema.StringAttribute{
				Optional:    true,
				Description: "IPFS gateway endpoint URL for download (defaults to API URL if not set)",
			},
			"ipfs_username": &schema.StringAttribute{
				Optional:    true,
				Description: "Username for IPFS authentication",
			},
			"ipfs_password": &schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Password for IPFS authentication",
			},
			"ipfs_proxy_url": &schema.StringAttribute{
				Optional:    true,
				Description: "HTTP proxy URL for IPFS requests",
			},

			// Other services
			"private_data_manager": &schema.StringAttribute{
				Optional:    true,
				Description: "Private Data Manager service ID (required when multiparty_enabled is true)",
			},

			// TODO remove / ignore
			"token_indexers_json": &schema.StringAttribute{
				Optional:    true,
				Description: "JSON array of token indexer service references",
			},

			// TODO this doesn't seem right ?
			// TLS configurations
			"tls_configs_json": &schema.StringAttribute{
				Optional:    true,
				Description: "JSON object containing TLS certificate configurations",
			},

			"force_delete": &schema.BoolAttribute{
				Optional:    true,
				Description: "Set to true when you plan to delete a protected Firefly service. You must apply this value before running terraform destroy.",
			},
		},
	}
}

func (data *FireflyServiceResourceModel) toServiceAPI(ctx context.Context, api *ServiceAPIModel, diagnostics *diag.Diagnostics) {
	api.Type = "FireflyService"
	api.Name = data.Name.ValueString()
	api.StackID = data.StackID.ValueString()
	api.Runtime.ID = data.Runtime.ValueString()
	api.Config = make(map[string]interface{})

	// Transaction Manager reference
	if !data.TransactionManager.IsNull() {
		api.Config["transactionManager"] = map[string]interface{}{
			"id": data.TransactionManager.ValueString(),
		}
	}

	// Optional fields
	if !data.Key.IsNull() && data.Key.ValueString() != "" {
		api.Config["key"] = data.Key.ValueString()
	}
	if !data.Description.IsNull() && data.Description.ValueString() != "" {
		api.Config["description"] = data.Description.ValueString()
	}

	// Multiparty configuration
	multipartyConfig := make(map[string]interface{})
	multipartyConfig["enabled"] = data.MultipartyEnabled.ValueBool()

	if data.MultipartyEnabled.ValueBool() {
		if !data.MultipartyOrgName.IsNull() {
			multipartyConfig["orgName"] = data.MultipartyOrgName.ValueString()
		}
		if !data.MultipartyOrgKey.IsNull() && data.MultipartyOrgKey.ValueString() != "" {
			multipartyConfig["orgKey"] = data.MultipartyOrgKey.ValueString()
		}
		if !data.MultipartyOrgDescription.IsNull() && data.MultipartyOrgDescription.ValueString() != "" {
			multipartyConfig["orgDescription"] = data.MultipartyOrgDescription.ValueString()
		}
		if !data.MultipartyNodeName.IsNull() && data.MultipartyNodeName.ValueString() != "" {
			multipartyConfig["nodeName"] = data.MultipartyNodeName.ValueString()
		}
		if !data.MultipartyNetworkNamespace.IsNull() {
			multipartyConfig["networkNamespace"] = data.MultipartyNetworkNamespace.ValueString()
		}
		if !data.MultipartyContractsJSON.IsNull() && data.MultipartyContractsJSON.ValueString() != "" {
			var contracts interface{}
			err := json.Unmarshal([]byte(data.MultipartyContractsJSON.ValueString()), &contracts)
			if err != nil {
				diagnostics.AddAttributeError(
					path.Root("multiparty_contracts_json"),
					"Failed to parse multiparty_contracts_json",
					err.Error(),
				)
			} else {
				multipartyConfig["contracts"] = contracts
			}
		}
	}
	api.Config["multiparty"] = multipartyConfig

	// IPFS configuration
	ipfsConfig := make(map[string]interface{})
	hasIPFSConfig := false

	if !data.IPFSService.IsNull() && data.IPFSService.ValueString() != "" {
		ipfsConfig["ipfsService"] = map[string]interface{}{
			"id": data.IPFSService.ValueString(),
		}
		hasIPFSConfig = true
	}
	if !data.IPFSAPIUrl.IsNull() && data.IPFSAPIUrl.ValueString() != "" {
		ipfsConfig["apiUrl"] = data.IPFSAPIUrl.ValueString()
		hasIPFSConfig = true
	}
	if !data.IPFSGatewayUrl.IsNull() && data.IPFSGatewayUrl.ValueString() != "" {
		ipfsConfig["gatewayUrl"] = data.IPFSGatewayUrl.ValueString()
		hasIPFSConfig = true
	}
	if !data.IPFSProxyUrl.IsNull() && data.IPFSProxyUrl.ValueString() != "" {
		ipfsConfig["proxy"] = map[string]interface{}{
			"url": data.IPFSProxyUrl.ValueString(),
		}
		hasIPFSConfig = true
	}

	// Handle IPFS credentials
	if (!data.IPFSUsername.IsNull() && data.IPFSUsername.ValueString() != "") ||
		(!data.IPFSPassword.IsNull() && data.IPFSPassword.ValueString() != "") {
		credSetName := "ipfsAuth"
		if api.Credsets == nil {
			api.Credsets = make(map[string]*CredSetAPI)
		}
		api.Credsets[credSetName] = &CredSetAPI{
			Name: credSetName,
			Type: "basic_auth",
			BasicAuth: &CredSetBasicAuthAPI{
				Username: data.IPFSUsername.ValueString(),
				Password: data.IPFSPassword.ValueString(),
			},
		}
		ipfsConfig["auth"] = map[string]interface{}{
			"credSetRef": credSetName,
		}
		hasIPFSConfig = true
	}

	if hasIPFSConfig {
		api.Config["ipfs"] = ipfsConfig
	}

	// Private Data Manager reference
	if !data.PrivateDataManager.IsNull() && data.PrivateDataManager.ValueString() != "" {
		api.Config["privatedatamanager"] = map[string]interface{}{
			"id": data.PrivateDataManager.ValueString(),
		}
	}

	// Token indexers
	if !data.TokenIndexersJSON.IsNull() && data.TokenIndexersJSON.ValueString() != "" {
		var tokenIndexers interface{}
		err := json.Unmarshal([]byte(data.TokenIndexersJSON.ValueString()), &tokenIndexers)
		if err != nil {
			diagnostics.AddAttributeError(
				path.Root("token_indexers_json"),
				"Failed to parse token_indexers_json",
				err.Error(),
			)
		} else {
			api.Config["tokenIndexers"] = tokenIndexers
		}
	}

	// TLS configurations
	if !data.TLSConfigsJSON.IsNull() && data.TLSConfigsJSON.ValueString() != "" {
		var tlsConfigs interface{}
		err := json.Unmarshal([]byte(data.TLSConfigsJSON.ValueString()), &tlsConfigs)
		if err != nil {
			diagnostics.AddAttributeError(
				path.Root("tls_configs_json"),
				"Failed to parse tls_configs_json",
				err.Error(),
			)
		} else {
			api.Config["tlsConfigs"] = tlsConfigs
		}
	}
}

func (api *ServiceAPIModel) toFireflyServiceData(data *FireflyServiceResourceModel, diagnostics *diag.Diagnostics) {
	data.ID = types.StringValue(api.ID)
	data.EnvironmentMemberID = types.StringValue(api.EnvironmentMemberID)
	data.Runtime = types.StringValue(api.Runtime.ID)
	data.Name = types.StringValue(api.Name)
	data.StackID = types.StringValue(api.StackID)

	// Transaction Manager
	if v, ok := api.Config["transactionManager"].(map[string]interface{}); ok {
		if id, ok := v["id"].(string); ok {
			data.TransactionManager = types.StringValue(id)
		}
	}

	// Optional fields
	if v, ok := api.Config["key"].(string); ok {
		data.Key = types.StringValue(v)
	} else {
		data.Key = types.StringNull()
	}

	if v, ok := api.Config["description"].(string); ok {
		data.Description = types.StringValue(v)
	} else {
		data.Description = types.StringNull()
	}

	// Multiparty configuration
	if v, ok := api.Config["multiparty"].(map[string]interface{}); ok {
		if enabled, ok := v["enabled"].(bool); ok {
			data.MultipartyEnabled = types.BoolValue(enabled)
		}
		if orgName, ok := v["orgName"].(string); ok {
			data.MultipartyOrgName = types.StringValue(orgName)
		} else {
			data.MultipartyOrgName = types.StringNull()
		}
		if orgKey, ok := v["orgKey"].(string); ok {
			data.MultipartyOrgKey = types.StringValue(orgKey)
		} else {
			data.MultipartyOrgKey = types.StringNull()
		}
		if orgDesc, ok := v["orgDescription"].(string); ok {
			data.MultipartyOrgDescription = types.StringValue(orgDesc)
		} else {
			data.MultipartyOrgDescription = types.StringNull()
		}
		if nodeName, ok := v["nodeName"].(string); ok {
			data.MultipartyNodeName = types.StringValue(nodeName)
		} else {
			data.MultipartyNodeName = types.StringNull()
		}
		if namespace, ok := v["networkNamespace"].(string); ok {
			data.MultipartyNetworkNamespace = types.StringValue(namespace)
		} else {
			data.MultipartyNetworkNamespace = types.StringNull()
		}
		if contracts := v["contracts"]; contracts != nil {
			if contractsJSON, err := json.Marshal(contracts); err == nil {
				data.MultipartyContractsJSON = types.StringValue(string(contractsJSON))
			} else {
				data.MultipartyContractsJSON = types.StringNull()
			}
		} else {
			data.MultipartyContractsJSON = types.StringNull()
		}
	} else {
		data.MultipartyEnabled = types.BoolValue(false)
		data.MultipartyOrgName = types.StringNull()
		data.MultipartyOrgKey = types.StringNull()
		data.MultipartyOrgDescription = types.StringNull()
		data.MultipartyNodeName = types.StringNull()
		data.MultipartyNetworkNamespace = types.StringNull()
		data.MultipartyContractsJSON = types.StringNull()
	}

	// IPFS configuration
	if v, ok := api.Config["ipfs"].(map[string]interface{}); ok {
		if service, ok := v["ipfsService"].(map[string]interface{}); ok {
			if id, ok := service["id"].(string); ok {
				data.IPFSService = types.StringValue(id)
			}
		} else {
			data.IPFSService = types.StringNull()
		}
		if apiUrl, ok := v["apiUrl"].(string); ok {
			data.IPFSAPIUrl = types.StringValue(apiUrl)
		} else {
			data.IPFSAPIUrl = types.StringNull()
		}
		if gatewayUrl, ok := v["gatewayUrl"].(string); ok {
			data.IPFSGatewayUrl = types.StringValue(gatewayUrl)
		} else {
			data.IPFSGatewayUrl = types.StringNull()
		}
		if proxy, ok := v["proxy"].(map[string]interface{}); ok {
			if url, ok := proxy["url"].(string); ok {
				data.IPFSProxyUrl = types.StringValue(url)
			}
		} else {
			data.IPFSProxyUrl = types.StringNull()
		}
	} else {
		data.IPFSService = types.StringNull()
		data.IPFSAPIUrl = types.StringNull()
		data.IPFSGatewayUrl = types.StringNull()
		data.IPFSProxyUrl = types.StringNull()
	}

	// Handle IPFS credentials from credsets
	data.IPFSUsername = types.StringNull()
	data.IPFSPassword = types.StringNull()
	if api.Credsets != nil {
		if credSet, ok := api.Credsets["ipfsAuth"]; ok && credSet != nil && credSet.BasicAuth != nil {
			data.IPFSUsername = types.StringValue(credSet.BasicAuth.Username)
			data.IPFSPassword = types.StringValue(credSet.BasicAuth.Password)
		}
	}

	// Private Data Manager
	if v, ok := api.Config["privatedatamanager"].(map[string]interface{}); ok {
		if id, ok := v["id"].(string); ok {
			data.PrivateDataManager = types.StringValue(id)
		}
	} else {
		data.PrivateDataManager = types.StringNull()
	}

	// Token indexers
	if tokenIndexers := api.Config["tokenIndexers"]; tokenIndexers != nil {
		if tokenIndexersJSON, err := json.Marshal(tokenIndexers); err == nil {
			data.TokenIndexersJSON = types.StringValue(string(tokenIndexersJSON))
		} else {
			data.TokenIndexersJSON = types.StringNull()
		}
	} else {
		data.TokenIndexersJSON = types.StringNull()
	}

	// TLS configurations
	if tlsConfigs := api.Config["tlsConfigs"]; tlsConfigs != nil {
		if tlsConfigsJSON, err := json.Marshal(tlsConfigs); err == nil {
			data.TLSConfigsJSON = types.StringValue(string(tlsConfigsJSON))
		} else {
			data.TLSConfigsJSON = types.StringNull()
		}
	} else {
		data.TLSConfigsJSON = types.StringNull()
	}
}

func (r *fireflyServiceResource) apiPath(data *FireflyServiceResourceModel) string {
	path := fmt.Sprintf("/api/v1/environments/%s/services", data.Environment.ValueString())
	if data.ID.ValueString() != "" {
		path = path + "/" + data.ID.ValueString()
	}
	if !data.ForceDelete.IsNull() && data.ForceDelete.ValueBool() {
		path = path + "?force=true"
	}
	return path
}

func (r *fireflyServiceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data FireflyServiceResourceModel
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

	api.toFireflyServiceData(&data, &resp.Diagnostics)
	r.waitForReadyStatus(ctx, r.apiPath(&data), &resp.Diagnostics)

	api.ID = data.ID.ValueString()
	ok, _ = r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	api.toFireflyServiceData(&data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *fireflyServiceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data FireflyServiceResourceModel
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

	api.toFireflyServiceData(&data, &resp.Diagnostics)
	r.waitForReadyStatus(ctx, r.apiPath(&data), &resp.Diagnostics)

	if ok, _ := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics); !ok {
		return
	}
	api.toFireflyServiceData(&data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *fireflyServiceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data FireflyServiceResourceModel
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

	api.toFireflyServiceData(&data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *fireflyServiceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data FireflyServiceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data), nil, nil, &resp.Diagnostics, Allow404())
	r.waitForRemoval(ctx, r.apiPath(&data), &resp.Diagnostics)
}
