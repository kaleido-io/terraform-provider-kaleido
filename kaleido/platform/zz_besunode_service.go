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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type BesuNodeServiceResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Environment         types.String `tfsdk:"environment"`
	EnvironmentMemberID types.String `tfsdk:"environment_member_id"`
	Runtime             types.String `tfsdk:"runtime"`
	Name                types.String `tfsdk:"name"`
	StackID             types.String `tfsdk:"stack_id"`
	Apisenabled       types.String `tfsdk:"apis_enabled"`
	Custombesuargs    types.String `tfsdk:"custom_besu_args"`
	Datastorageformat types.String `tfsdk:"data_storage_format"`
	Gasprice          types.String `tfsdk:"gas_price"`
	Loglevel          types.String `tfsdk:"log_level"`
	Mode              types.String `tfsdk:"mode"`
	Network           types.String `tfsdk:"network"`
	Nodekey           types.String `tfsdk:"node_key"`
	Signer            types.Bool `tfsdk:"signer"`
	Syncmode          types.String `tfsdk:"sync_mode"`
	Targetgaslimit    types.Int64 `tfsdk:"target_gas_limit"`
	ForceDelete         types.Bool   `tfsdk:"force_delete"`
}

func BesuNodeServiceResourceFactory() resource.Resource {
	return &besunodeserviceResource{}
}

type besunodeserviceResource struct {
	commonResource
}

func (r *besunodeserviceResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_besunode"
}

func (r *besunodeserviceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A Besu Node service that provides Ethereum blockchain node capabilities.",
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
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Stack ID where the BesuNode service belongs (optional)",
			},
			"apis_enabled": &schema.StringAttribute{
				Optional:    true,
				Description: "List of enabled besu API methods. ETH, QBFT/IBFT, ADMIN, NET, DEBUG, TXPOOL, and WEB3 methods will always be enabled",
			},
			"custom_besu_args": &schema.StringAttribute{
				Optional:    true,
				Description: "A list of additional arguments to pass to the Besu command line. These are appended to the default arguments",
			},
			"data_storage_format": &schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("BONSAI"),
				Description: "The database storage format to use",
			},
			"gas_price": &schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("0"),
				Description: "The gas price for transactions",
			},
			"log_level": &schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("INFO"),
				Description: "Desired log level of the besu runtime",
			},
			"mode": &schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("active"),
				Description: "Determines if the node can receive RPC requests",
			},
			"network": &schema.StringAttribute{
				Optional:    true,
				Description: "besuNetwork",
			},
			"node_key": &schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "An secp256k1 private key for the node identity, please omit '0x' when providing. The key will be generated if not set.",
			},
			"signer": &schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Determines if the node should be added as a signer to the network",
			},
			"sync_mode": &schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("FULL"),
				Description: "The blockchain sync mode",
			},
			"target_gas_limit": &schema.Int64Attribute{
				Optional:    true,
				Description: "The maximum amount of gas that can be spent in a transaction",
			},
			"force_delete": &schema.BoolAttribute{
				Optional:    true,
				Description: "Set to true when you plan to delete a protected BesuNode service. You must apply this value before running terraform destroy.",
			},
		},
	}
}

func (data *BesuNodeServiceResourceModel) toBesuNodeServiceAPI(ctx context.Context, api *ServiceAPIModel, diagnostics *diag.Diagnostics) {
	api.Type = "BesuNodeService"
	api.Name = data.Name.ValueString()
	api.StackID = data.StackID.ValueString()
	api.Runtime.ID = data.Runtime.ValueString()
	api.Config = make(map[string]interface{})

	if !data.Targetgaslimit.IsNull() {
		api.Config["targetGasLimit"] = data.Targetgaslimit.ValueInt64()
	}
	// Handle Custombesuargs as JSON
	if !data.Custombesuargs.IsNull() && data.Custombesuargs.ValueString() != "" {
		var custombesuargsData interface{}
		err := json.Unmarshal([]byte(data.Custombesuargs.ValueString()), &custombesuargsData)
		if err != nil {
			diagnostics.AddAttributeError(
				path.Root("custombesuargs"),
				"Failed to parse Custombesuargs",
				err.Error(),
			)
		} else {
			api.Config["customBesuArgs"] = custombesuargsData
		}
	}
	// Handle Nodekey credentials
	if !data.Nodekey.IsNull() && data.Nodekey.ValueString() != "" {
		if api.Credsets == nil {
			api.Credsets = make(map[string]*CredSetAPI)
		}
		api.Credsets["nodeKey"] = &CredSetAPI{
			Name: "nodeKey",
			Type: "key",
			Key: &CredSetKeyAPI{
				Value: data.Nodekey.ValueString(),
			},
		}
		api.Config["nodeKey"] = map[string]interface{}{
			"credSetRef": "nodeKey",
		}
	}
	if !data.Network.IsNull() && data.Network.ValueString() != "" {
		api.Config["network"] = data.Network.ValueString()
	}
	if !data.Mode.IsNull() && data.Mode.ValueString() != "" {
		api.Config["mode"] = data.Mode.ValueString()
	}
	api.Config["signer"] = data.Signer.ValueBool()
	if !data.Loglevel.IsNull() && data.Loglevel.ValueString() != "" {
		api.Config["logLevel"] = data.Loglevel.ValueString()
	}
	if !data.Datastorageformat.IsNull() && data.Datastorageformat.ValueString() != "" {
		api.Config["dataStorageFormat"] = data.Datastorageformat.ValueString()
	}
	if !data.Gasprice.IsNull() && data.Gasprice.ValueString() != "" {
		api.Config["gasPrice"] = data.Gasprice.ValueString()
	}
	if !data.Syncmode.IsNull() && data.Syncmode.ValueString() != "" {
		api.Config["syncMode"] = data.Syncmode.ValueString()
	}
	// Handle Apisenabled as JSON
	if !data.Apisenabled.IsNull() && data.Apisenabled.ValueString() != "" {
		var apisenabledData interface{}
		err := json.Unmarshal([]byte(data.Apisenabled.ValueString()), &apisenabledData)
		if err != nil {
			diagnostics.AddAttributeError(
				path.Root("apisenabled"),
				"Failed to parse Apisenabled",
				err.Error(),
			)
		} else {
			api.Config["apisEnabled"] = apisenabledData
		}
	}
}

func (api *ServiceAPIModel) toBesuNodeServiceData(data *BesuNodeServiceResourceModel, diagnostics *diag.Diagnostics) {
	data.ID = types.StringValue(api.ID)
	data.EnvironmentMemberID = types.StringValue(api.EnvironmentMemberID)
	data.Runtime = types.StringValue(api.Runtime.ID)
	data.Name = types.StringValue(api.Name)
	data.StackID = types.StringValue(api.StackID)

	if v, ok := api.Config["mode"].(string); ok {
		data.Mode = types.StringValue(v)
	} else {
		data.Mode = types.StringValue("active")
	}
	if v, ok := api.Config["signer"].(bool); ok {
		data.Signer = types.BoolValue(v)
	} else {
		data.Signer = types.BoolValue(true)
	}
	if v, ok := api.Config["logLevel"].(string); ok {
		data.Loglevel = types.StringValue(v)
	} else {
		data.Loglevel = types.StringValue("INFO")
	}
	if v, ok := api.Config["dataStorageFormat"].(string); ok {
		data.Datastorageformat = types.StringValue(v)
	} else {
		data.Datastorageformat = types.StringValue("BONSAI")
	}
	if v, ok := api.Config["gasPrice"].(string); ok {
		data.Gasprice = types.StringValue(v)
	} else {
		data.Gasprice = types.StringValue("0")
	}
	if v, ok := api.Config["syncMode"].(string); ok {
		data.Syncmode = types.StringValue(v)
	} else {
		data.Syncmode = types.StringValue("FULL")
	}
	if apisenabledData := api.Config["apisEnabled"]; apisenabledData != nil {
		if apisenabledJSON, err := json.Marshal(apisenabledData); err == nil {
			data.Apisenabled = types.StringValue(string(apisenabledJSON))
		} else {
			data.Apisenabled = types.StringNull()
		}
	} else {
		data.Apisenabled = types.StringNull()
	}
	if v, ok := api.Config["targetGasLimit"].(float64); ok {
		data.Targetgaslimit = types.Int64Value(int64(v))
	} else {
		data.Targetgaslimit = types.Int64Null()
	}
	if custombesuargsData := api.Config["customBesuArgs"]; custombesuargsData != nil {
		if custombesuargsJSON, err := json.Marshal(custombesuargsData); err == nil {
			data.Custombesuargs = types.StringValue(string(custombesuargsJSON))
		} else {
			data.Custombesuargs = types.StringNull()
		}
	} else {
		data.Custombesuargs = types.StringNull()
	}
	if credset, ok := api.Credsets["nodeKey"]; ok && credset.Key != nil {
		data.Nodekey = types.StringValue(credset.Key.Value)
	} else {
		data.Nodekey = types.StringNull()
	}
	if v, ok := api.Config["network"].(string); ok {
		data.Network = types.StringValue(v)
	} else {
		data.Network = types.StringNull()
	}
}

func (r *besunodeserviceResource) apiPath(data *BesuNodeServiceResourceModel) string {
	path := fmt.Sprintf("/api/v1/environments/%s/services", data.Environment.ValueString())
	if data.ID.ValueString() != "" {
		path = path + "/" + data.ID.ValueString()
	}
	if !data.ForceDelete.IsNull() && data.ForceDelete.ValueBool() {
		path = path + "?force=true"
	}
	return path
}

func (r *besunodeserviceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data BesuNodeServiceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	var api ServiceAPIModel
	data.toBesuNodeServiceAPI(ctx, &api, &resp.Diagnostics)
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

func (r *besunodeserviceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data BesuNodeServiceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)

	var api ServiceAPIModel
	if ok, _ := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics); !ok {
		return
	}

	data.toBesuNodeServiceAPI(ctx, &api, &resp.Diagnostics)
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

func (r *besunodeserviceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
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

func (r *besunodeserviceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data BesuNodeServiceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data), nil, nil, &resp.Diagnostics, Allow404())
	r.waitForRemoval(ctx, r.apiPath(&data), &resp.Diagnostics)
}
