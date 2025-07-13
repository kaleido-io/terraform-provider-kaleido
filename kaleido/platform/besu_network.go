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

type BesuNetworkResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Environment      types.String `tfsdk:"environment"`
	Name             types.String `tfsdk:"name"`
	ChainID          types.Int64  `tfsdk:"chain_id"`
	BootstrapOptions types.String `tfsdk:"bootstrap_options"`
	InitMode         types.String `tfsdk:"init_mode"`
	Genesis          types.String `tfsdk:"genesis"`
	ForceDelete      types.Bool   `tfsdk:"force_delete"`
}

func BesuNetworkResourceFactory() resource.Resource {
	return &besuNetworkResource{}
}

type besuNetworkResource struct {
	commonResource
}

func (r *besuNetworkResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_besu_network"
}

func (r *besuNetworkResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A Besu blockchain network that provides consensus and networking capabilities for Besu nodes.",
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"environment": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Environment ID where the Besu network will be deployed",
			},
			"name": &schema.StringAttribute{
				Required:    true,
				Description: "Display name for the Besu network",
			},
			"chain_id": &schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Chain ID for the blockchain network (will be auto-generated if not provided)",
			},
			"bootstrap_options": &schema.StringAttribute{
				Optional:    true,
				Description: "A JSON string defining the bootstrap configuration options for the network (e.g. for QBFT or IBFT consensus).",
			},
			"init_mode": &schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("automated"),
				Description: "Initialization mode. Options: automated, manual",
			},
			"genesis": &schema.StringAttribute{
				Optional:    true,
				Description: "The content of the genesis JSON file for the network.",
			},
			"force_delete": &schema.BoolAttribute{
				Optional:    true,
				Description: "Set to true when you plan to delete a protected Besu network. You must apply this value before running terraform destroy.",
			},
		},
	}
}

func (data *BesuNetworkResourceModel) toNetworkAPI(ctx context.Context, api *NetworkAPIModel, diagnostics *diag.Diagnostics) {
	api.Type = "BesuNetwork"
	api.Name = data.Name.ValueString()
	api.Config = make(map[string]interface{})
	api.Filesets = make(map[string]*FileSetAPI)

	// Chain ID
	if !data.ChainID.IsNull() && data.ChainID.ValueInt64() > 0 {
		api.Config["chainID"] = data.ChainID.ValueInt64()
	}

	// Bootstrap options - passed through as raw JSON
	if !data.BootstrapOptions.IsNull() && data.BootstrapOptions.ValueString() != "" {
		var bootstrapOptions interface{}
		err := json.Unmarshal([]byte(data.BootstrapOptions.ValueString()), &bootstrapOptions)
		if err != nil {
			diagnostics.AddAttributeError(
				path.Root("bootstrap_options"),
				"Failed to parse bootstrap_options JSON",
				err.Error(),
			)
		} else {
			api.Config["bootstrapOptions"] = bootstrapOptions
		}
	}

	// Init mode
	if !data.InitMode.IsNull() {
		api.InitMode = data.InitMode.ValueString()
	}

	// Genesis file set is constructed dynamically
	if !data.Genesis.IsNull() && data.Genesis.ValueString() != "" {
		api.Filesets["init_files"] = &FileSetAPI{
			Files: map[string]*FileAPI{
				"genesis.json": {
					Data: FileDataAPI{
						Text: data.Genesis.ValueString(),
					},
				},
			},
		}
		api.Config["init_files"] = map[string]interface{}{
			"fileSetRef": "init_files",
		}
	}
}

func (api *NetworkAPIModel) toBesuNetworkData(data *BesuNetworkResourceModel, diagnostics *diag.Diagnostics) {
	data.ID = types.StringValue(api.ID)

	// TODO

	// We only need to read back computed values, as inputs are tracked by terraform.
	if val, ok := api.Config["chainID"]; ok {
		// Numbers from JSON unmarshalling are float64
		if vFloat, ok := val.(float64); ok {
			data.ChainID = types.Int64Value(int64(vFloat))
		}
	}
	data.InitMode = types.StringValue(api.InitMode)
}

func (r *besuNetworkResource) apiPath(data *BesuNetworkResourceModel) string {
	path := fmt.Sprintf("/api/v1/environments/%s/networks", data.Environment.ValueString())
	if data.ID.ValueString() != "" {
		path = path + "/" + data.ID.ValueString()
	}
	if data.ForceDelete.ValueBool() {
		path = path + "?force=true"
	}
	return path
}

func (r *besuNetworkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data BesuNetworkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	var api NetworkAPIModel
	data.toNetworkAPI(ctx, &api, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	ok, _ := r.apiRequest(ctx, http.MethodPost, r.apiPath(&data), api, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	api.toBesuNetworkData(&data, &resp.Diagnostics)
	r.waitForReadyStatus(ctx, r.apiPath(&data), &resp.Diagnostics)

	// Re-read from API after readiness check
	api.ID = data.ID.ValueString()
	ok, _ = r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	api.toBesuNetworkData(&data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *besuNetworkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data BesuNetworkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)

	// Read current object
	var api NetworkAPIModel
	if ok, _ := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics); !ok {
		return
	}

	// Update from plan
	data.toNetworkAPI(ctx, &api, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if ok, _ := r.apiRequest(ctx, http.MethodPut, r.apiPath(&data), api, &api, &resp.Diagnostics); !ok {
		return
	}

	api.toBesuNetworkData(&data, &resp.Diagnostics)
	r.waitForReadyStatus(ctx, r.apiPath(&data), &resp.Diagnostics)

	// Re-read from API
	if ok, _ := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics); !ok {
		return
	}
	api.toBesuNetworkData(&data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *besuNetworkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data BesuNetworkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	var api NetworkAPIModel
	api.ID = data.ID.ValueString()
	ok, status := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics, Allow404())
	if !ok {
		return
	}
	if status == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	api.toBesuNetworkData(&data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *besuNetworkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data BesuNetworkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data), nil, nil, &resp.Diagnostics, Allow404())
	r.waitForRemoval(ctx, r.apiPath(&data), &resp.Diagnostics)
}
