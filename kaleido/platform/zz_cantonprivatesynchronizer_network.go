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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type CantonPrivateSynchronizerNetworkResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Environment types.String `tfsdk:"environment"`
	Name        types.String `tfsdk:"name"`
	Sequencer types.String `tfsdk:"sequencer"`
	InitMode    types.String `tfsdk:"init_mode"`
	ForceDelete types.Bool   `tfsdk:"force_delete"`
}

func CantonPrivateSynchronizerNetworkResourceFactory() resource.Resource {
	return &cantonprivatesynchronizernetworkResource{}
}

type cantonprivatesynchronizernetworkResource struct {
	commonResource
}

func (r *cantonprivatesynchronizernetworkResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_cantonprivatesynchronizer_network"
}

func (r *cantonprivatesynchronizernetworkResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A CantonPrivateSynchronizer network that provides blockchain infrastructure.",
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"environment": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Environment ID where the CantonPrivateSynchronizer network will be deployed",
			},
			"name": &schema.StringAttribute{
				Required:    true,
				Description: "Display name for the CantonPrivateSynchronizer network",
			},
			"sequencer": &schema.StringAttribute{
				Optional:    true,
				Description: "Remote Sequencer Endpoint",
			},
			"init_mode": &schema.StringAttribute{
				Optional:    true,
				Default:     stringdefault.StaticString("automated"),
				Description: "Initialization mode for the network (automated or manual)",
			},
			"force_delete": &schema.BoolAttribute{
				Optional:    true,
				Description: "Set to true when you plan to delete a protected CantonPrivateSynchronizer network. You must apply this value before running terraform destroy.",
			},
		},
	}
}

func (data *CantonPrivateSynchronizerNetworkResourceModel) toNetworkAPI(ctx context.Context, api *NetworkAPIModel, diagnostics *diag.Diagnostics) {
	api.Type = "CantonPrivateSynchronizerNetwork"
	api.Name = data.Name.ValueString()
	api.InitMode = data.InitMode.ValueString()
	api.Config = make(map[string]interface{})

	if !data.Sequencer.IsNull() && data.Sequencer.ValueString() != "" {
		api.Config["sequencer"] = data.Sequencer.ValueString()
	}
}

func (api *NetworkAPIModel) toCantonPrivateSynchronizerNetworkData(data *CantonPrivateSynchronizerNetworkResourceModel, diagnostics *diag.Diagnostics) {
	data.ID = types.StringValue(api.ID)
	data.Name = types.StringValue(api.Name)
	data.InitMode = types.StringValue(api.InitMode)

	if v, ok := api.Config["sequencer"].(string); ok {
		data.Sequencer = types.StringValue(v)
	} else {
		data.Sequencer = types.StringNull()
	}
}

func (r *cantonprivatesynchronizernetworkResource) apiPath(data *CantonPrivateSynchronizerNetworkResourceModel) string {
	path := fmt.Sprintf("/api/v1/environments/%s/networks", data.Environment.ValueString())
	if data.ID.ValueString() != "" {
		path = path + "/" + data.ID.ValueString()
	}
	if !data.ForceDelete.IsNull() && data.ForceDelete.ValueBool() {
		path = path + "?force=true"
	}
	return path
}

func (r *cantonprivatesynchronizernetworkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data CantonPrivateSynchronizerNetworkResourceModel
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

	api.toCantonPrivateSynchronizerNetworkData(&data, &resp.Diagnostics)
	r.waitForReadyStatus(ctx, r.apiPath(&data), &resp.Diagnostics)

	api.ID = data.ID.ValueString()
	ok, _ = r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	api.toCantonPrivateSynchronizerNetworkData(&data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *cantonprivatesynchronizernetworkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data CantonPrivateSynchronizerNetworkResourceModel
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

	api.toCantonPrivateSynchronizerNetworkData(&data, &resp.Diagnostics)
	r.waitForReadyStatus(ctx, r.apiPath(&data), &resp.Diagnostics)

	if ok, _ := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics); !ok {
		return
	}
	api.toCantonPrivateSynchronizerNetworkData(&data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *cantonprivatesynchronizernetworkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CantonPrivateSynchronizerNetworkResourceModel
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

	api.toCantonPrivateSynchronizerNetworkData(&data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *cantonprivatesynchronizernetworkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CantonPrivateSynchronizerNetworkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data), nil, nil, &resp.Diagnostics, Allow404())
	r.waitForRemoval(ctx, r.apiPath(&data), &resp.Diagnostics)
}
