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
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// stacks interface implementations for the resource and API
type StacksResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Environment         types.String `tfsdk:"environment"`
	EnvironmentMemberID types.String `tfsdk:"environment_member_id"`
	Name                types.String `tfsdk:"name"`
	Type                types.String `tfsdk:"type"`
	SubType             types.String `tfsdk:"sub_type"`
	NetworkId           types.String `tfsdk:"network_id"`
}

type StacksAPIModel struct {
	ID                  string     `json:"id,omitempty"`
	Created             *time.Time `json:"created,omitempty"`
	Updated             *time.Time `json:"updated,omitempty"`
	EnvironmentMemberID string     `json:"environmentMemberId,omitempty"`
	Name                string     `json:"name"`
	Type                string     `json:"type"`
	SubType             string     `json:"subType,omitempty"`
	NetworkId           string     `json:"networkId,omitempty"`
}

func StacksResourceFactory() resource.Resource {
	return &stacksResource{}
}

type stacksResource struct {
	commonResource
}

func (r *stacksResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_stack"
}

func (r *stacksResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A stack is a collection of services within Digital Assets, Web3 Middleware or Chain Infrastructure. \n Stacks provide guidance around the optimal relationships and architecture of services for specific use cases, business units or chain connections. \n Every resource created within a stack is created in the context of an environment.",
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": &schema.StringAttribute{
				Required:    true,
				Description: "Stack Display Name",
			},
			"type": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Stack Type. Options include: `chain_infrastructure`, `web3_middleware`, and `digital_assets`",
				Validators: []validator.String{
					stringvalidator.OneOf(
						"chain_infrastructure",
						"web3_middleware",
						"digital_assets",
					),
				},
			},
			"sub_type": &schema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Stack sub-type specific to each stack type. Options include: `TokenizationStack`,`CustodyStack` for `digital_assets`, `FireflyStack` for `web3_middleware` and `BesuStack`,`IPFSNetwork` for `chain_infrastructure`",
			},
			"environment": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Environment ID",
			},
			"environment_member_id": &schema.StringAttribute{
				Computed: true,
			},
			"network_id": &schema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Specify a network ID for `chain_infrastructure` stacks that contain a Besu or IPFS network.",
			},
		},
	}
}

func (data *StacksResourceModel) toAPI(api *StacksAPIModel) {
	api.Type = data.Type.ValueString()
	if !data.SubType.IsNull() {
		api.SubType = data.SubType.ValueString()
	}
	api.Name = data.Name.ValueString()
	if !data.NetworkId.IsNull() {
		api.NetworkId = data.NetworkId.ValueString()
	}
}

func (api *StacksAPIModel) toData(data *StacksResourceModel) {
	data.ID = types.StringValue(api.ID)
	data.EnvironmentMemberID = types.StringValue(api.EnvironmentMemberID)
	data.Name = types.StringValue(api.Name)
	data.Type = types.StringValue(api.Type)
	if api.SubType != "" {
		data.SubType = types.StringValue(api.SubType)
	}
	if api.NetworkId != "" && !data.NetworkId.IsNull() {
		data.NetworkId = types.StringValue(api.NetworkId)
	}
}

func (r *stacksResource) apiPath(data *StacksResourceModel) string {
	path := fmt.Sprintf("/api/v1/environments/%s/stacks", data.Environment.ValueString())
	if data.ID.ValueString() != "" {
		path = path + "/" + data.ID.ValueString()
	}
	return path
}

func (r *stacksResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var data StacksResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	var api StacksAPIModel
	data.toAPI(&api)
	ok, _ := r.apiRequest(ctx, http.MethodPost, r.apiPath(&data), api, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	api.toData(&data) // need the ID copied over
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *stacksResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

	var data StacksResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)

	// Read full current object
	var api StacksAPIModel
	if ok, _ := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics); !ok {
		return
	}

	// Update from plan
	data.toAPI(&api)
	if ok, _ := r.apiRequest(ctx, http.MethodPut, r.apiPath(&data), api, &api, &resp.Diagnostics); !ok {
		return
	}

	api.toData(&data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *stacksResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data StacksResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	var api StacksAPIModel
	api.ID = data.ID.ValueString()
	ok, status := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics, Allow404())
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

func (r *stacksResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data StacksResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data), nil, nil, &resp.Diagnostics, Allow404())

	r.waitForRemoval(ctx, r.apiPath(&data), &resp.Diagnostics)
}
