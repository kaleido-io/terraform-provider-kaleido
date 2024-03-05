// Copyright © Kaleido, Inc. 2018, 2024

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package kaleido

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	kaleido "github.com/kaleido-io/kaleido-sdk-go/kaleido"
	"github.com/kaleido-io/terraform-provider-kaleido/kaleido/kaleidobase"
)

type resourceNode struct {
	baasBaseResource
}

func ResourceNodeFactory() resource.Resource {
	return &resourceNode{}
}

type NodeResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	ConsortiumID         types.String `tfsdk:"consortium_id"`
	EnvironmentID        types.String `tfsdk:"environment_id"`
	MembershipID         types.String `tfsdk:"membership_id"`
	Role                 types.String `tfsdk:"role"`
	WebSocketURL         types.String `tfsdk:"websocket_url"`
	HttpsURL             types.String `tfsdk:"https_url"`
	URLs                 types.Map    `tfsdk:"urls"`
	FirstUserAccount     types.String `tfsdk:"first_user_account"`
	ZoneID               types.String `tfsdk:"zone_id"`
	Size                 types.String `tfsdk:"size"`
	Remote               types.Bool   `tfsdk:"remote"`
	KmsID                types.String `tfsdk:"kms_id"`
	OpsmetricID          types.String `tfsdk:"opsmetric_id"`
	BackupID             types.String `tfsdk:"backup_id"`
	NetworkingID         types.String `tfsdk:"networking_id"`
	NodeConfigID         types.String `tfsdk:"node_config_id"`
	BafID                types.String `tfsdk:"baf_id"`
	HybridPortAllocation types.Int64  `tfsdk:"hybrid_port_allocation"`
	UpdateTrigger        types.String `tfsdk:"update_trigger"`
	DatabaseType         types.String `tfsdk:"database_type"`
}

func (r *resourceNode) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_node"
}

func (r *resourceNode) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed: true,
			},
			"name": &schema.StringAttribute{
				Required: true,
			},
			"consortium_id": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"environment_id": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"membership_id": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"role": &schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Default:       stringdefault.StaticString("validator"),
			},
			"websocket_url": &schema.StringAttribute{
				Computed: true,
			},
			"https_url": &schema.StringAttribute{
				Computed: true,
			},
			"urls": &schema.MapAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
			"first_user_account": &schema.StringAttribute{
				Computed: true,
			},
			"zone_id": &schema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"size": &schema.StringAttribute{
				Optional: true,
			},
			"remote": &schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"kms_id": &schema.StringAttribute{
				Optional: true,
			},
			"opsmetric_id": &schema.StringAttribute{
				Optional: true,
			},
			"backup_id": &schema.StringAttribute{
				Optional: true,
			},
			"networking_id": &schema.StringAttribute{
				Optional: true,
			},
			"node_config_id": &schema.StringAttribute{
				Optional: true,
			},
			"baf_id": &schema.StringAttribute{
				Optional: true,
			},
			"hybrid_port_allocation": &schema.Int64Attribute{
				Computed: true,
			},
			"update_trigger": &schema.StringAttribute{
				Optional: true,
			},
			"database_type": &schema.StringAttribute{
				Optional: true,
			},
		},
	}
}

func (r *resourceNode) waitUntilNodeStarted(ctx context.Context, op, consortiumID, environmentID, nodeID string, apiModel *kaleido.Node, data *NodeResourceModel, diagnostics diag.Diagnostics) error {
	return kaleidobase.Retry.Do(ctx, op, func(attempt int) (retry bool, err error) {
		res, getErr := r.BaaS.GetNode(consortiumID, environmentID, nodeID, apiModel)
		if getErr != nil {
			return false, getErr
		}

		statusCode := res.StatusCode()
		if statusCode != 200 {
			return false, fmt.Errorf("fetching node %s state failed: %d", apiModel.ID, statusCode)
		}

		if apiModel.State != "started" {
			msg := "node %s in environment %s in consortium %s" +
				"took too long to enter state 'started'. Final state was '%s'"
			return true, fmt.Errorf(msg, apiModel.ID, environmentID, consortiumID, apiModel.State)
		}
		r.copyNodeData(ctx, apiModel, data, diagnostics)
		return false, nil
	})
}

func (r *resourceNode) copyNodeData(ctx context.Context, apiModel *kaleido.Node, data *NodeResourceModel, diagnostics diag.Diagnostics) {
	data.ID = types.StringValue(apiModel.ID)
	data.Name = types.StringValue(apiModel.Name)
	data.Role = types.StringValue(apiModel.Role)
	mapValue, diag := types.MapValueFrom(ctx, types.StringType, apiModel.Urls)
	diagnostics.Append(diag...)
	data.URLs = mapValue
	if httpURL, ok := apiModel.Urls["rpc"]; ok {
		data.HttpsURL = types.StringValue(httpURL.(string))
	} else {
		data.HttpsURL = types.StringValue("")
	}
	if wsURL, ok := apiModel.Urls["wss"]; ok {
		data.WebSocketURL = types.StringValue(wsURL.(string))
	} else {
		data.WebSocketURL = types.StringValue("")
	}
	data.HybridPortAllocation = types.Int64Value(apiModel.HybridPortAllocation)
	data.FirstUserAccount = types.StringValue(apiModel.FirstUserAccount)

}

func (r *resourceNode) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data NodeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	apiModel := kaleido.Node{}
	consortiumID := data.ConsortiumID.ValueString()
	environmentID := data.EnvironmentID.ValueString()
	apiModel.Name = data.Name.ValueString()
	apiModel.MembershipID = data.MembershipID.ValueString()
	apiModel.ZoneID = data.ZoneID.ValueString()
	apiModel.Size = data.Size.ValueString()

	apiModel.KmsID = data.KmsID.ValueString()
	apiModel.OpsmetricID = data.OpsmetricID.ValueString()
	apiModel.BackupID = data.BackupID.ValueString()
	apiModel.NetworkingID = data.NetworkingID.ValueString()
	apiModel.NodeConfigID = data.NodeConfigID.ValueString()
	apiModel.BafID = data.BafID.ValueString()
	apiModel.Role = data.Role.ValueString()
	apiModel.DatabaseType = data.DatabaseType.ValueString()
	isRemote := data.Remote.ValueBool()

	res, err := r.BaaS.CreateNode(consortiumID, environmentID, &apiModel)
	if err != nil {
		resp.Diagnostics.AddError("failed to create node", err.Error())
		return
	}
	status := res.StatusCode()
	if status != 201 {
		msg := "Could not create node %s in consortium %s in environment %s with status %d: %s"
		resp.Diagnostics.AddError("failed to create node", fmt.Sprintf(msg, apiModel.Name, consortiumID, environmentID, status, res.String()))
		return
	}

	if !isRemote {
		// Do not wait for remote PrivateStack nodes to initialize
		err = r.waitUntilNodeStarted(ctx, "Create", consortiumID, environmentID, apiModel.ID, &apiModel, &data, resp.Diagnostics)
		if err != nil {
			resp.Diagnostics.AddError("failed to query node status", err.Error())
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *resourceNode) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data NodeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	apiModel := kaleido.Node{}
	consortiumID := data.ConsortiumID.ValueString()
	environmentID := data.EnvironmentID.ValueString()
	apiModel.Size = data.Size.ValueString()
	apiModel.KmsID = data.KmsID.ValueString()
	apiModel.OpsmetricID = data.OpsmetricID.ValueString()
	apiModel.BackupID = data.BackupID.ValueString()
	apiModel.NetworkingID = data.NetworkingID.ValueString()
	apiModel.NodeConfigID = data.NodeConfigID.ValueString()
	apiModel.BafID = data.BafID.ValueString()
	nodeID := data.ID.ValueString()
	isRemote := data.Remote.ValueBool()

	res, err := r.BaaS.UpdateNode(consortiumID, environmentID, nodeID, &apiModel)
	if err != nil {
		resp.Diagnostics.AddError("failed to update node", err.Error())
		return
	}
	status := res.StatusCode()
	if status != 200 {
		msg := "Could not update node %s in consortium %s in environment %s with status %d: %s"
		resp.Diagnostics.AddError("failed to update node", fmt.Sprintf(msg, nodeID, consortiumID, environmentID, status, res.String()))
		return
	}

	res, err = r.BaaS.ResetNode(consortiumID, environmentID, nodeID)
	if err != nil {
		resp.Diagnostics.AddError("failed to reset node", err.Error())
		return
	}
	if status != 200 {
		msg := "Could not reset node %s in consortium %s in environment %s with status %d: %s"
		resp.Diagnostics.AddError("failed to update service", fmt.Sprintf(msg, nodeID, consortiumID, environmentID, status, res.String()))
		return
	}

	if !isRemote {
		// Do not wait for remote PrivateStack nodes to initialize
		err = r.waitUntilNodeStarted(ctx, "Update", consortiumID, environmentID, nodeID, &apiModel, &data, resp.Diagnostics)
		if err != nil {
			resp.Diagnostics.AddError("failed to query node status", err.Error())
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *resourceNode) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data NodeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	consortiumID := data.ConsortiumID.ValueString()
	environmentID := data.EnvironmentID.ValueString()
	nodeID := data.ID.ValueString()

	var apiModel kaleido.Node
	res, err := r.BaaS.GetNode(consortiumID, environmentID, nodeID, &apiModel)
	if err != nil {
		resp.Diagnostics.AddError("failed to query node", err.Error())
		return
	}

	status := res.StatusCode()
	if status == 404 {
		resp.State.RemoveResource(ctx)
		return
	}
	if status != 200 {
		msg := "Could not find node %s in consortium %s in environment %s with status %d: %s"
		resp.Diagnostics.AddError("failed to query node", fmt.Sprintf(msg, nodeID, consortiumID, environmentID, status, res.String()))
		return
	}

	r.copyNodeData(ctx, &apiModel, &data, resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

}

func (r *resourceNode) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data NodeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	consortiumID := data.ConsortiumID.ValueString()
	environmentID := data.EnvironmentID.ValueString()
	nodeID := data.ID.ValueString()

	res, err := r.BaaS.DeleteNode(consortiumID, environmentID, nodeID)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete node", err.Error())
		return
	}

	statusCode := res.StatusCode()
	if res.IsError() && statusCode != 404 {
		msg := "Failed to delete node %s in environment %s in consortium %s with status %d: %s"
		resp.Diagnostics.AddError("failed to delete node", fmt.Sprintf(msg, nodeID, environmentID, consortiumID, statusCode, res.String()))
		return
	}

}
