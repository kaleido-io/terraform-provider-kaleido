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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	kaleido "github.com/kaleido-io/kaleido-sdk-go/kaleido"
)

type resourceEnvironment struct {
	baasBaseResource
}

func ResourceEnvironmentFactory() resource.Resource {
	return &resourceEnvironment{}
}

type EnvironmentResourceModel struct {
	ID                types.String `tfsdk:"id"`
	ConsortiumID      types.String `tfsdk:"consortium_id"`
	Name              types.String `tfsdk:"name"`
	Description       types.String `tfsdk:"description"`
	SharedDeployment  types.Bool   `tfsdk:"shared_deployment"`
	EnvType           types.String `tfsdk:"env_type"`
	ConsensusType     types.String `tfsdk:"consensus_type"`
	ReleaseID         types.String `tfsdk:"release_id"`
	MultiRegion       types.Bool   `tfsdk:"multi_region"`
	BlockPeriod       types.Int64  `tfsdk:"block_period"`
	PrefundedAccounts types.Map    `tfsdk:"prefunded_accounts"`
	TestFeaturesJSON  types.String `tfsdk:"test_features_json"`
	ChainID           types.Int64  `tfsdk:"chain_id"`
}

func (r *resourceEnvironment) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_environment"
}

func (r *resourceEnvironment) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"consortium_id": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": &schema.StringAttribute{
				Required: true,
			},
			"description": &schema.StringAttribute{
				Required: false,
				Optional: true,
			},
			"shared_deployment": &schema.BoolAttribute{
				Optional:    true,
				Default:     booldefault.StaticBool(false),
				Description: "The decentralized nature of Kaleido means an environment might be shared with other accounts. When true only create if name does not exist, and delete becomes a no-op.",
			},
			"env_type": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"consensus_type": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"release_id": &schema.StringAttribute{
				Computed: true,
				Optional: true,
			},
			"multi_region": &schema.BoolAttribute{
				Optional: true,
			},
			"block_period": &schema.Int64Attribute{
				Optional: true,
			},
			"prefunded_accounts": &schema.MapAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
			},
			"test_features_json": &schema.StringAttribute{
				Optional: true,
			},
			"chain_id": &schema.Int64Attribute{
				Computed: true,
				Optional: true,
			},
		},
	}
}

func (r *resourceEnvironment) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var data EnvironmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	apiModel := kaleido.Environment{}
	consortiumID := data.ConsortiumID.ValueString()
	if !data.PrefundedAccounts.IsNull() {
		apiModel.PrefundedAccounts = make(map[string]interface{})
		resp.Diagnostics.Append(data.PrefundedAccounts.ElementsAs(ctx, &apiModel.PrefundedAccounts, true)...)
	}
	apiModel.ChainID = uint(data.ChainID.ValueInt64())
	apiModel.Name = data.Name.ValueString()
	apiModel.Description = data.Description.ValueString()
	apiModel.Provider = data.EnvType.ValueString()
	apiModel.ConsensusType = data.ConsensusType.ValueString()
	apiModel.TestFeatures = map[string]interface{}{}
	if !data.TestFeaturesJSON.IsNull() {
		_ = json.Unmarshal([]byte(data.TestFeaturesJSON.ValueString()), &apiModel.TestFeatures)
	}
	if data.MultiRegion.ValueBool() {
		apiModel.TestFeatures["multi_region"] = true
	}
	apiModel.BlockPeriod = int(data.BlockPeriod.ValueInt64())
	apiModel.ChainID = uint(data.ChainID.ValueInt64())
	apiModel.ReleaseID = data.ReleaseID.ValueString()

	sharedExisting := false
	if data.SharedDeployment.ValueBool() {
		var environments []kaleido.Environment
		res, err := r.client.ListEnvironments(consortiumID, &environments)
		if err != nil {
			resp.Diagnostics.AddError("failed to list environments", err.Error())
			return
		}
		if res.StatusCode() != 200 {
			resp.Diagnostics.AddError("failed to list environments", fmt.Sprintf("Failed to list existing environments with status %d: %s", res.StatusCode(), res.String()))
			return
		}
		for _, e := range environments {
			if e.Name == data.Name.ValueString() && !strings.Contains(e.State, "delete") {
				// Already exists, just re-use
				sharedExisting = true
				apiModel = e
			}
		}
	}
	if !sharedExisting {
		res, err := r.client.CreateEnvironment(consortiumID, &apiModel)
		if err != nil {
			resp.Diagnostics.AddError("failed to create environment", err.Error())
			return
		}

		if res.StatusCode() != 201 {
			msg := "Could not create environment %s for consortia %s with status %d: %s"
			resp.Diagnostics.AddError("failed to create service", fmt.Sprintf(msg, apiModel.Name, consortiumID, res.StatusCode(), res.String()))
			return
		}
	}

	data.ID = types.StringValue(apiModel.ID)
	data.ReleaseID = types.StringValue(apiModel.ReleaseID)
	data.ChainID = types.Int64Value(int64(apiModel.ChainID))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *resourceEnvironment) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data EnvironmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	apiModel := kaleido.Environment{}
	consortiumID := data.ConsortiumID.ValueString()
	environmentID := data.ID.ValueString()
	apiModel.Name = data.Name.ValueString()
	apiModel.Description = data.Description.ValueString()
	if !data.TestFeaturesJSON.IsNull() {
		apiModel.TestFeatures = map[string]interface{}{}
		_ = json.Unmarshal([]byte(data.TestFeaturesJSON.ValueString()), &apiModel.TestFeatures)
	}

	res, err := r.client.UpdateEnvironment(consortiumID, environmentID, &apiModel)
	if err != nil {
		resp.Diagnostics.AddError("failed to update environment", err.Error())
		return
	}

	statusCode := res.StatusCode()
	if statusCode != 200 {
		msg := "Failed to update environment %s, in consortium %s with status %d: %s"
		resp.Diagnostics.AddError("failed to update environment", fmt.Sprintf(msg, environmentID, consortiumID, statusCode, res.String()))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *resourceEnvironment) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data EnvironmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	consortiumID := data.ConsortiumID.ValueString()
	environmentID := data.ID.ValueString()

	var apiModel kaleido.Environment
	res, err := r.client.GetEnvironment(consortiumID, environmentID, &apiModel)

	if err != nil {
		resp.Diagnostics.AddError("failed to query environment", err.Error())
		return
	}

	if res.StatusCode() == 404 {
		resp.State.RemoveResource(ctx)
		return
	}
	if res.StatusCode() != 200 {
		msg := "Failed to get environment %s, from consortium %s with status %d: %s"
		resp.Diagnostics.AddError("failed to query environment", fmt.Sprintf(msg, environmentID, consortiumID, res.StatusCode(), res.String()))
		return
	}

	data.Name = types.StringValue(apiModel.Name)
	data.Description = types.StringValue(apiModel.Description)
	data.EnvType = types.StringValue(apiModel.Provider)
	data.ConsensusType = types.StringValue(apiModel.ConsensusType)
	data.ReleaseID = types.StringValue(apiModel.ReleaseID)
	mapValue, diag := types.MapValueFrom(ctx, types.StringType, apiModel.PrefundedAccounts)
	resp.Diagnostics.Append(diag...)
	data.PrefundedAccounts = mapValue
	data.ChainID = types.Int64Value(int64(apiModel.ChainID))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *resourceEnvironment) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data EnvironmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if data.SharedDeployment.ValueBool() {
		// Cannot safely delete if this is shared with other terraform deployments
		// Pretend we deleted it
		return
	}

	consortiumID := data.ConsortiumID.ValueString()
	environmentID := data.ID.ValueString()

	res, err := r.client.DeleteEnvironment(consortiumID, environmentID)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete environment", err.Error())
		return
	}

	statusCode := res.StatusCode()
	if statusCode != 202 && statusCode != 204 {
		msg := "Failed to delete environment %s, in consortium %s with status %d: %s"
		resp.Diagnostics.AddError("failed to delete environment", fmt.Sprintf(msg, environmentID, consortiumID, statusCode, res.String()))
		return
	}
}
