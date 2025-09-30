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
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type AMSAddressResourceModel struct {
	Environment         types.String `tfsdk:"environment"`
	Service            types.String `tfsdk:"service"`
	Address            types.String `tfsdk:"address"`
	DisplayName        types.String `tfsdk:"display_name"`
	Description        types.String `tfsdk:"description"`
	InfoJSON               types.String `tfsdk:"info"`
	Contract           types.Bool   `tfsdk:"contract"`
	ContractManagerService types.String `tfsdk:"contract_manager_service"`
	ContractManagerBuild   types.String `tfsdk:"contract_manager_build"`
	FireflyNamespace       types.String `tfsdk:"firefly_namespace"`
	FireflyAPI             types.String `tfsdk:"firefly_api"`
	Created            types.String `tfsdk:"created"`
	Updated            types.String `tfsdk:"updated"`
}

func AMSAddressResourceFactory() resource.Resource {
	return &ams_addressResource{}
}

type ams_addressResource struct {
	commonResource
}

func (r *ams_addressResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_ams_address"
}

func (r *ams_addressResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an address in the Kaleido Asset Manager Service data model",
		Attributes: map[string]schema.Attribute{
			"environment": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "The environment ID",
			},
			"service": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "The AMS service ID",
			},
			"address": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "The blockchain address (will be converted to lowercase)",
			},
			"display_name": &schema.StringAttribute{
				Optional:    true,
				Description: "Human-readable display name for the address",
			},
			"description": &schema.StringAttribute{
				Optional:    true,
				Description: "Description of the address",
			},
			"info_json": &schema.StringAttribute{
				Optional:    true,
				Description: "Additional metadata as JSON string",
			},
			"contract": &schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Default:       booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
				Description:   "Whether this address represents a smart contract",
			},
			"contract_manager_service": &schema.StringAttribute{
				Optional:    true,
				Description: "Contract manager service (only valid if contract=true)",
			},
			"contract_manager_build": &schema.StringAttribute{
				Optional:    true,
				Description: "Contract manager build UUID (only valid if contract=true)",
			},
			"firefly_namespace": &schema.StringAttribute{
				Optional:    true,
				Description: "FireFly namespace (only valid if contract=true)",
			},
			"firefly_api": &schema.StringAttribute{
				Optional:    true,
				Description: "FireFly API (only valid if contract=true)",
			},
			"created": &schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp when the address was created",
			},
			"updated": &schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp when the address was last updated",
			},
		},
	}
}

func (data *AMSAddressResourceModel) toAPI(diagnostics *diag.Diagnostics) map[string]interface{} {
	payload := make(map[string]interface{})
	
	// Required field - normalize to lowercase
	if !data.Address.IsNull() && !data.Address.IsUnknown() {
		payload["address"] = strings.ToLower(data.Address.ValueString())
	}
	
	// Optional fields
	if !data.DisplayName.IsNull() && !data.DisplayName.IsUnknown() {
		payload["displayName"] = data.DisplayName.ValueString()
	}
	
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		payload["description"] = data.Description.ValueString()
	}
	
	if !data.InfoJSON.IsNull() && !data.InfoJSON.IsUnknown() {
		// Assume info is JSON - could parse it here if needed
		raw := data.InfoJSON.ValueString()
		obj := make(map[string]interface{})
		err := json.Unmarshal([]byte(raw), &obj)
		if err != nil {
			diagnostics.AddError("Error unmarshalling info", err.Error())
			return nil
		}
		json.Unmarshal([]byte(raw), &obj)
		payload["info"] = obj
	}
	
	if !data.Contract.IsNull() && !data.Contract.IsUnknown() {
		payload["contract"] = data.Contract.ValueBool()
		
		// Contract manager fields only valid if contract=true
		if data.Contract.ValueBool() {
			contractManager := make(map[string]interface{})
			if !data.ContractManagerService.IsNull() && !data.ContractManagerService.IsUnknown() {
				contractManager["service"] = data.ContractManagerService.ValueString()
			}
			if !data.ContractManagerBuild.IsNull() && !data.ContractManagerBuild.IsUnknown() {
				contractManager["build"] = data.ContractManagerBuild.ValueString()
			}
			if len(contractManager) > 0 {
				payload["contractManager"] = contractManager
			}
			
			firefly := make(map[string]interface{})
			if !data.FireflyNamespace.IsNull() && !data.FireflyNamespace.IsUnknown() {
				firefly["namespace"] = data.FireflyNamespace.ValueString()
			}
			if !data.FireflyAPI.IsNull() && !data.FireflyAPI.IsUnknown() {
				firefly["api"] = data.FireflyAPI.ValueString()
			}
			if len(firefly) > 0 {
				payload["firefly"] = firefly
			}
		}
	}
	
	return payload
}

func (data *AMSAddressResourceModel) fromAPI(apiResponse map[string]interface{}, diagnostics *diag.Diagnostics) {
	if val, ok := apiResponse["address"]; ok && val != nil {
		data.Address = types.StringValue(val.(string))
	}
	
	if val, ok := apiResponse["displayName"]; ok && val != nil {
		data.DisplayName = types.StringValue(val.(string))
	} else {
		data.DisplayName = types.StringNull()
	}
	
	if val, ok := apiResponse["description"]; ok && val != nil {
		data.Description = types.StringValue(val.(string))
	} else {
		data.Description = types.StringNull()
	}
	
	if val, ok := apiResponse["info"]; ok && val != nil {
		// Convert info back to JSON string if needed
		raw, err := json.Marshal(val)
		if err != nil {
			diagnostics.AddError("Error marshalling info", err.Error())
			return
		}
		data.InfoJSON = types.StringValue(string(raw))
	} else {
		data.InfoJSON = types.StringNull()
	}
	
	if val, ok := apiResponse["contract"]; ok && val != nil {
		data.Contract = types.BoolValue(val.(bool))
	} else {
		data.Contract = types.BoolValue(false)
	}
	
	// Handle contract manager
	if cm, ok := apiResponse["contractManager"].(map[string]interface{}); ok {
		if val, exists := cm["service"]; exists && val != nil {
			data.ContractManagerService = types.StringValue(val.(string))
		} else {
			data.ContractManagerService = types.StringNull()
		}
		if val, exists := cm["build"]; exists && val != nil {
			data.ContractManagerBuild = types.StringValue(val.(string))
		} else {
			data.ContractManagerBuild = types.StringNull()
		}
	} else {
		data.ContractManagerService = types.StringNull()
		data.ContractManagerBuild = types.StringNull()
	}
	
	// Handle firefly
	if ff, ok := apiResponse["firefly"].(map[string]interface{}); ok {
		if val, exists := ff["namespace"]; exists && val != nil {
			data.FireflyNamespace = types.StringValue(val.(string))
		} else {
			data.FireflyNamespace = types.StringNull()
		}
		if val, exists := ff["api"]; exists && val != nil {
			data.FireflyAPI = types.StringValue(val.(string))
		} else {
			data.FireflyAPI = types.StringNull()
		}
	} else {
		data.FireflyNamespace = types.StringNull()
		data.FireflyAPI = types.StringNull()
	}
	
	if val, ok := apiResponse["created"]; ok && val != nil {
		data.Created = types.StringValue(val.(string))
	}
	
	if val, ok := apiResponse["updated"]; ok && val != nil {
		data.Updated = types.StringValue(val.(string))
	}
}

func (r *ams_addressResource) apiPathCollection(data *AMSAddressResourceModel) string {
	return fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/addresses", data.Environment.ValueString(), data.Service.ValueString())
}

func (r *ams_addressResource) apiPathResource(data *AMSAddressResourceModel) string {
	address := strings.ToLower(data.Address.ValueString())
	return fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/addresses/%s", data.Environment.ValueString(), data.Service.ValueString(), address)
}

func (r *ams_addressResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AMSAddressResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload := data.toAPI(&resp.Diagnostics)
	
	// Create address via POST to collection endpoint
	var apiResponse map[string]interface{}
	_, _ = r.apiRequest(ctx, http.MethodPost, r.apiPathCollection(&data), payload, &apiResponse, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update data model from API response
	data.fromAPI(apiResponse, &resp.Diagnostics)
	
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *ams_addressResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AMSAddressResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read address via GET to resource endpoint
	var apiResponse map[string]interface{}
	ok, statusCode := r.apiRequest(ctx, http.MethodGet, r.apiPathResource(&data), nil, &apiResponse, &resp.Diagnostics)
	if !ok {
		if statusCode == 404 {
			// Address not found, remove from state
			resp.State.RemoveResource(ctx)
			return
		}
		return
	}

	// Update data model from API response
	data.fromAPI(apiResponse, &resp.Diagnostics)
	
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *ams_addressResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data AMSAddressResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload := data.toAPI(&resp.Diagnostics)
	
	// Update address via PUT to resource endpoint
	var apiResponse map[string]interface{}
	_, _ = r.apiRequest(ctx, http.MethodPut, r.apiPathResource(&data), payload, &apiResponse, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update data model from API response
	data.fromAPI(apiResponse, &resp.Diagnostics)
	
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *ams_addressResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AMSAddressResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete address via DELETE to resource endpoint
	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPathResource(&data), nil, nil, &resp.Diagnostics)
}
