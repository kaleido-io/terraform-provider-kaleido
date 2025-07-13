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

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type BesuNodeServiceResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Environment         types.String `tfsdk:"environment"`
	EnvironmentMemberID types.String `tfsdk:"environment_member_id"`
	Runtime             types.String `tfsdk:"runtime"`
	Name                types.String `tfsdk:"name"`
	StackID             types.String `tfsdk:"stack_id"`
	Network             types.String `tfsdk:"network"`
	Mode                types.String `tfsdk:"mode"`
	Signer              types.Bool   `tfsdk:"signer"`
	LogLevel            types.String `tfsdk:"log_level"`
	SyncMode            types.String `tfsdk:"sync_mode"`
	DataStorageFormat   types.String `tfsdk:"data_storage_format"`
	ApisEnabled         types.List   `tfsdk:"apis_enabled"`
	TargetGasLimit      types.Int64  `tfsdk:"target_gas_limit"`
	GasPrice            types.String `tfsdk:"gas_price"`
	CustomBesuArgs      types.List   `tfsdk:"custom_besu_args"`
	NodeKey             types.String `tfsdk:"node_key"`
	// Endpoints         types.Map    `tfsdk:"endpoints"`
	// ConnectivityJSON  types.String `tfsdk:"connectivity_json"`
	ForceDelete types.Bool `tfsdk:"force_delete"`
}

func BesuNodeServiceResourceFactory() resource.Resource {
	return &besuNodeServiceResource{}
}

type besuNodeServiceResource struct {
	commonResource
}

func (r *besuNodeServiceResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_besunode_service"
}

func (r *besuNodeServiceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A Besu blockchain node service that connects to a Besu network and processes blockchain transactions.",
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"environment": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Environment ID where the BesuNode service will be deployed",
			},
			"environment_member_id": &schema.StringAttribute{
				Computed: true,
			},
			"runtime": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Runtime ID where the BesuNode service will be deployed",
			},
			"name": &schema.StringAttribute{
				Required:    true,
				Description: "Display name for the BesuNode service",
			},
			"stack_id": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Stack ID where the BesuNode service belongs (must be a BesuStack)",
			},
			"network": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "The network ID where the BesuNode service belongs (must be a BesuNetwork)",
			},
			"mode": &schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("active"),
				Description: "Node mode - determines if the node can receive RPC requests. Options: active, standby",
			},
			"signer": &schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether the node should be added as a signer to the network",
			},
			"log_level": &schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("INFO"),
				Description: "Node runtime log level. Options: INFO, DEBUG, TRACE",
			},
			"sync_mode": &schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("FULL"),
				Description: "Blockchain synchronization mode. Options: FAST, FULL, SNAP",
			},
			"data_storage_format": &schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("BONSAI"),
				Description: "Database storage format. Options: FOREST, BONSAI",
			},
			"apis_enabled": &schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, nil)),
				Description: "Additional API methods to enable. ETH, QBFT/IBFT, ADMIN, NET are always enabled. Options: DEBUG, MINER, PERM, PLUGINS, TRACE, TXPOOL, WEB3",
			},
			"target_gas_limit": &schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
				Description: "Target gas limit for blocks produced by this node",
			},
			"gas_price": &schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("0"),
				Description: "Gas price for transactions",
			},
			"custom_besu_args": &schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, nil)),
				Description: "Additional arguments to pass to the Besu command line",
			},
			"node_key": &schema.StringAttribute{
				Optional:      true,
				Sensitive:     true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "The node key for the BesuNode service",
			},
			"force_delete": &schema.BoolAttribute{
				Optional:    true,
				Description: "Set to true when you plan to delete a protected BesuNode service. You must apply this value before running terraform destroy.",
			},
			// "endpoints": &schema.MapAttribute{
			// 	ElementType: types.StringType,
			// 	Computed:    true,
			// 	Description: "Endpoints for the BesuNode service",
			// },
			// "connectivity_json": &schema.StringAttribute{
			// 	Computed:    true,
			// 	Description: "Connectivity JSON for the BesuNode service",
			// },
		},
	}
}

func (data *BesuNodeServiceResourceModel) toServiceAPI(ctx context.Context, api *ServiceAPIModel, diagnostics *diag.Diagnostics) {
	api.Type = "BesuNodeService"
	api.Name = data.Name.ValueString()
	api.StackID = data.StackID.ValueString()
	api.Runtime.ID = data.Runtime.ValueString()
	api.Config = make(map[string]interface{})

	if !data.Network.IsNull() {
		api.Config["network"] = map[string]interface{}{
			"id": data.Network.ValueString(),
		}
	}
	if !data.Mode.IsNull() {
		api.Config["mode"] = data.Mode.ValueString()
	}
	if !data.Signer.IsNull() {
		api.Config["signer"] = data.Signer.ValueBool()
	}
	if !data.LogLevel.IsNull() {
		api.Config["logLevel"] = data.LogLevel.ValueString()
	}
	if !data.SyncMode.IsNull() {
		api.Config["syncMode"] = data.SyncMode.ValueString()
	}
	if !data.DataStorageFormat.IsNull() {
		api.Config["dataStorageFormat"] = data.DataStorageFormat.ValueString()
	}
	if !data.GasPrice.IsNull() {
		api.Config["gasPrice"] = data.GasPrice.ValueString()
	}
	if !data.TargetGasLimit.IsNull() && data.TargetGasLimit.ValueInt64() > 0 {
		api.Config["targetGasLimit"] = data.TargetGasLimit.ValueInt64()
	}

	if !data.ApisEnabled.IsNull() {
		var apisEnabled []string
		diagnostics.Append(data.ApisEnabled.ElementsAs(ctx, &apisEnabled, false)...)
		if len(apisEnabled) > 0 {
			api.Config["apisEnabled"] = apisEnabled
		}
	}

	if !data.CustomBesuArgs.IsNull() {
		var customArgs []string
		diagnostics.Append(data.CustomBesuArgs.ElementsAs(ctx, &customArgs, false)...)
		if len(customArgs) > 0 {
			api.Config["customBesuArgs"] = customArgs
		}
	}

	if !data.NodeKey.IsNull() && data.NodeKey.ValueString() != "" {
		credSetName := "nodeKey"
		if api.Credsets == nil {
			api.Credsets = make(map[string]*CredSetAPI)
		}
		api.Credsets[credSetName] = &CredSetAPI{
			Name: credSetName,
			Type: "key",
			Key: &CredSetKeyAPI{
				Value: data.NodeKey.ValueString(),
			},
		}
		api.Config["nodeKey"] = map[string]interface{}{
			"credSetRef": credSetName,
		}
	}
}

func (api *ServiceAPIModel) toBesuNodeServiceData(data *BesuNodeServiceResourceModel, diagnostics *diag.Diagnostics) {
	data.ID = types.StringValue(api.ID)
	// var d diag.Diagnostics
	// data.Endpoints, d = types.MapValue(types.ObjectType{
	// 	AttrTypes: map[string]attr.Type{
	// 		"type": types.StringType,
	// 		"urls": types.ListType{types.ElementType: types.StringType},
	// 	},
	// }, map[string]attr.Value(api.Endpoints))
	// diagnostics.Append(d...)
	// data.ConnectivityJSON = types.StringValue(api.StatusDetails.Connectivity.Identity)
	// Set basic fields
	data.ID = types.StringValue(api.ID)
	data.EnvironmentMemberID = types.StringValue(api.EnvironmentMemberID)
	data.Runtime = types.StringValue(api.Runtime.ID)
	data.Name = types.StringValue(api.Name)
	data.StackID = types.StringValue(api.StackID)
	data.Mode = types.StringValue("")
	if v, ok := api.Config["mode"].(string); ok {
		data.Mode = types.StringValue(v)
	}
	data.Signer = types.BoolValue(false)
	if v, ok := api.Config["signer"].(bool); ok {
		data.Signer = types.BoolValue(v)
	}
	data.LogLevel = types.StringValue("")
	if v, ok := api.Config["logLevel"].(string); ok {
		data.LogLevel = types.StringValue(v)
	}
	data.SyncMode = types.StringValue("")
	if v, ok := api.Config["syncMode"].(string); ok {
		data.SyncMode = types.StringValue(v)
	}
	data.DataStorageFormat = types.StringValue("")
	if v, ok := api.Config["dataStorageFormat"].(string); ok {
		data.DataStorageFormat = types.StringValue(v)
	}
	// apis_enabled
	if v, ok := api.Config["apisEnabled"].([]interface{}); ok {
		apis := make([]attr.Value, len(v))
		for i, apiVal := range v {
			if s, ok := apiVal.(string); ok {
				apis[i] = types.StringValue(s)
			}
		}
		data.ApisEnabled, _ = types.ListValue(types.StringType, apis)
	} else {
		data.ApisEnabled = types.ListNull(types.StringType)
	}
	// target_gas_limit
	data.TargetGasLimit = types.Int64Null()
	if v, ok := api.Config["targetGasLimit"].(int64); ok {
		data.TargetGasLimit = types.Int64Value(v)
	} else if v, ok := api.Config["targetGasLimit"].(float64); ok {
		data.TargetGasLimit = types.Int64Value(int64(v))
	}
	// gas_price
	data.GasPrice = types.StringNull()
	if v, ok := api.Config["gasPrice"].(string); ok {
		data.GasPrice = types.StringValue(v)
	}
	// custom_besu_args
	if v, ok := api.Config["customBesuArgs"].([]interface{}); ok {
		args := make([]attr.Value, len(v))
		for i, argVal := range v {
			if s, ok := argVal.(string); ok {
				args[i] = types.StringValue(s)
			}
		}
		data.CustomBesuArgs, _ = types.ListValue(types.StringType, args)
	} else {
		data.CustomBesuArgs = types.ListNull(types.StringType)
	}
	// node_key
	data.NodeKey = types.StringNull()
	if api.Credsets != nil {
		if credSet, ok := api.Credsets["nodeKey"]; ok && credSet != nil && credSet.Key != nil {
			data.NodeKey = types.StringValue(credSet.Key.Value)
		}
	}
	// endpoints
	// if api.Endpoints != nil {
	// 	endpointMap := make(map[string]attr.Value)
	// 	for k, v := range api.Endpoints {
	// 		switch val := v.(type) {
	// 		case string:
	// 			endpointMap[k] = types.StringValue(val)
	// 		case []interface{}:
	// 			// Convert []interface{} to []attr.Value
	// 			strList := make([]attr.Value, len(val))
	// 			for i, item := range val {
	// 				if s, ok := item.(string); ok {
	// 					strList[i] = types.StringValue(s)
	// 				}
	// 			}
	// 			endpointMap[k], _ = types.ListValue(types.StringType, strList)
	// 		}
	// 	}
	// 	data.Endpoints, _ = types.MapValue(types.StringType, endpointMap)
	// } else {
	// 	data.Endpoints = types.MapNull(types.StringType)
	// }
	// connectivity_json
	// data.ConnectivityJSON = types.StringNull()
	// if api.StatusDetails != nil && api.StatusDetails.Connectivity != nil {
	// 	if api.StatusDetails.Connectivity.Identity != "" {
	// 		data.ConnectivityJSON = types.StringValue(api.StatusDetails.Connectivity.Identity)
	// 	}
	// }
	// force_delete
	data.ForceDelete = types.BoolNull()
	if v, ok := api.Config["forceDelete"].(bool); ok {
		data.ForceDelete = types.BoolValue(v)
	}

}

func (r *besuNodeServiceResource) apiPath(data *BesuNodeServiceResourceModel) string {
	path := fmt.Sprintf("/api/v1/environments/%s/services", data.Environment.ValueString())
	if data.ID.ValueString() != "" {
		path = path + "/" + data.ID.ValueString()
	}
	if !data.ForceDelete.IsNull() && data.ForceDelete.ValueBool() {
		path = path + "?force=true"
	}
	return path
}

func (r *besuNodeServiceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data BesuNodeServiceResourceModel
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

	api.toBesuNodeServiceData(&data, &resp.Diagnostics)
	r.waitForReadyStatus(ctx, r.apiPath(&data), &resp.Diagnostics)

	api.ID = data.ID.ValueString()
	ok, _ = r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	api.toBesuNodeServiceData(&data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *besuNodeServiceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data BesuNodeServiceResourceModel
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

	api.toBesuNodeServiceData(&data, &resp.Diagnostics)
	r.waitForReadyStatus(ctx, r.apiPath(&data), &resp.Diagnostics)

	if ok, _ := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics); !ok {
		return
	}
	api.toBesuNodeServiceData(&data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *besuNodeServiceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data BesuNodeServiceResourceModel
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

	api.toBesuNodeServiceData(&data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *besuNodeServiceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data BesuNodeServiceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data), nil, nil, &resp.Diagnostics, Allow404())
	r.waitForRemoval(ctx, r.apiPath(&data), &resp.Diagnostics)
}
