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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type BesuNetworkResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Environment types.String `tfsdk:"environment"`
	Name        types.String `tfsdk:"name"`
	Blockperiodseconds    types.Int64 `tfsdk:"blockperiodseconds"`
	Blockreward           types.String `tfsdk:"blockreward"`
	Epochlength           types.Int64 `tfsdk:"epochlength"`
	Miningbeneficiary     types.String `tfsdk:"miningbeneficiary"`
	Requesttimeoutseconds types.Int64 `tfsdk:"requesttimeoutseconds"`
	InitMode    types.String `tfsdk:"init_mode"`
	ForceDelete types.Bool   `tfsdk:"force_delete"`
}

func BesuNetworkResourceFactory() resource.Resource {
	return &besunetworkResource{}
}

type besunetworkResource struct {
	commonResource
}

func (r *besunetworkResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_besu_network"
}

func (r *besunetworkResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A Besu network that provides Ethereum blockchain infrastructure with customizable consensus mechanisms.",
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
			"blockperiodseconds": &schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(5),
				Description: "Minimum block period internal in seconds",
			},
			"blockreward": &schema.StringAttribute{
				Optional:    true,
				Description: "The reward to give block producers (or the designated mining beneficiary) in Wei",
			},
			"epochlength": &schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(30000),
				Description: "Number of blocks that should pass before pending validator votes are reset",
			},
			"miningbeneficiary": &schema.StringAttribute{
				Optional:    true,
				Description: "The ethereum address to give block rewards to. If not set",
			},
			"requesttimeoutseconds": &schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(10),
				Description: "Timeout for each consensus round before a round change in seconds",
			},
			"init_mode": &schema.StringAttribute{
				Optional:    true,
				Default:     stringdefault.StaticString("automated"),
				Description: "Initialization mode for the network (automated or manual)",
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
	api.InitMode = data.InitMode.ValueString()
	api.Config = make(map[string]interface{})

	if !data.Blockperiodseconds.IsNull() {
		api.Config["blockperiodseconds"] = data.Blockperiodseconds.ValueInt64()
	}
	if !data.Epochlength.IsNull() {
		api.Config["epochlength"] = data.Epochlength.ValueInt64()
	}
	if !data.Requesttimeoutseconds.IsNull() {
		api.Config["requesttimeoutseconds"] = data.Requesttimeoutseconds.ValueInt64()
	}
	if !data.Blockreward.IsNull() && data.Blockreward.ValueString() != "" {
		api.Config["blockreward"] = data.Blockreward.ValueString()
	}
	if !data.Miningbeneficiary.IsNull() && data.Miningbeneficiary.ValueString() != "" {
		api.Config["miningbeneficiary"] = data.Miningbeneficiary.ValueString()
	}
}

func (api *NetworkAPIModel) toBesuNetworkData(data *BesuNetworkResourceModel, diagnostics *diag.Diagnostics) {
	data.ID = types.StringValue(api.ID)
	data.Name = types.StringValue(api.Name)
	data.InitMode = types.StringValue(api.InitMode)

	if v, ok := api.Config["blockreward"].(string); ok {
		data.Blockreward = types.StringValue(v)
	} else {
		data.Blockreward = types.StringNull()
	}
	if v, ok := api.Config["miningbeneficiary"].(string); ok {
		data.Miningbeneficiary = types.StringValue(v)
	} else {
		data.Miningbeneficiary = types.StringNull()
	}
	if v, ok := api.Config["blockperiodseconds"].(float64); ok {
		data.Blockperiodseconds = types.Int64Value(int64(v))
	} else {
		data.Blockperiodseconds = types.Int64Value(5)
	}
	if v, ok := api.Config["epochlength"].(float64); ok {
		data.Epochlength = types.Int64Value(int64(v))
	} else {
		data.Epochlength = types.Int64Value(30000)
	}
	if v, ok := api.Config["requesttimeoutseconds"].(float64); ok {
		data.Requesttimeoutseconds = types.Int64Value(int64(v))
	} else {
		data.Requesttimeoutseconds = types.Int64Value(10)
	}
}

func (r *besunetworkResource) apiPath(data *BesuNetworkResourceModel) string {
	path := fmt.Sprintf("/api/v1/environments/%s/networks", data.Environment.ValueString())
	if data.ID.ValueString() != "" {
		path = path + "/" + data.ID.ValueString()
	}
	if !data.ForceDelete.IsNull() && data.ForceDelete.ValueBool() {
		path = path + "?force=true"
	}
	return path
}

func (r *besunetworkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
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

	api.ID = data.ID.ValueString()
	ok, _ = r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	api.toBesuNetworkData(&data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *besunetworkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data BesuNetworkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)

	var api NetworkAPIModel
	if ok, _ := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics); !ok {
		return
	}

	data.toNetworkAPI(ctx, &api, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if ok, _ := r.apiRequest(ctx, http.MethodPut, r.apiPath(&data), api, &api, &resp.Diagnostics); !ok {
		return
	}

	api.toBesuNetworkData(&data, &resp.Diagnostics)
	r.waitForReadyStatus(ctx, r.apiPath(&data), &resp.Diagnostics)

	if ok, _ := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics); !ok {
		return
	}
	api.toBesuNetworkData(&data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *besunetworkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
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

func (r *besunetworkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data BesuNetworkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data), nil, nil, &resp.Diagnostics, Allow404())
	r.waitForRemoval(ctx, r.apiPath(&data), &resp.Diagnostics)
}
