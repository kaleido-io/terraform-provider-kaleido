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
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	kaleido "github.com/kaleido-io/kaleido-sdk-go/kaleido"
)

type resourceConfiguration struct {
	baasBaseResource
}

func ResourceConfigurationFactory() resource.Resource {
	return &resourceConfiguration{}
}

type ConfigurationResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Type          types.String `tfsdk:"type"`
	ConsortiumID  types.String `tfsdk:"consortium_id"`
	EnvironmentID types.String `tfsdk:"environment_id"`
	MembershipID  types.String `tfsdk:"membership_id"`
	Details       types.Map    `tfsdk:"details"`
	DetailsJSON   types.String `tfsdk:"details_json"`
	LastUpdated   types.Int64  `tfsdk:"last_updated"`
}

func (r *resourceConfiguration) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_environment"
}

func (r *resourceConfiguration) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": &schema.StringAttribute{
				Required: true,
			},
			"type": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
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
			"details": &schema.MapAttribute{
				Optional: true,
			},
			"details_json": &schema.StringAttribute{
				Optional: true,
			},
			"last_updated": &schema.Int64Attribute{
				Computed: true,
			},
		},
	}
}

func (r *resourceConfiguration) copyConfigurationData(ctx context.Context, apiModel *kaleido.Configuration, data *ConfigurationResourceModel, diagnostics diag.Diagnostics) {
	data.ID = types.StringValue(apiModel.ID)
	data.Name = types.StringValue(apiModel.Name)
	mapValue, diag := types.MapValueFrom(ctx, types.StringType, apiModel.Details)
	diagnostics.Append(diag...)
	data.Details = mapValue
	valueStr, _ := json.MarshalIndent(apiModel.Details, "", "  ")
	data.DetailsJSON = types.StringValue(string(valueStr))
	data.LastUpdated = types.Int64Value(time.Now().UnixNano())
}

func (r *resourceConfiguration) dataToAPIModel(ctx context.Context, data *ConfigurationResourceModel, apiModel *kaleido.Configuration, diagnostics diag.Diagnostics) {
	apiModel.MembershipID = data.MembershipID.ValueString()
	apiModel.Name = data.Name.ValueString()
	apiModel.Type = data.Type.ValueString()
	if !data.DetailsJSON.IsNull() {
		apiModel.Details = make(map[string]interface{})
		if err := json.Unmarshal([]byte(data.DetailsJSON.ValueString()), &apiModel.Details); err != nil {
			msg := "Could not parse details_json of %s %s in consortium %s in environment %s: %s"
			diagnostics.AddError("failed to parse details_json", fmt.Sprintf(msg, apiModel.Type, apiModel.Type, data.ConsortiumID.ValueString(), data.EnvironmentID.ValueString(), err))
			return
		}
	}
	if !data.DetailsJSON.IsNull() {
		diagnostics.Append(data.Details.ElementsAs(ctx, &apiModel.Details, true)...)
	}
}

func (r *resourceConfiguration) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var data ConfigurationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	apiModel := kaleido.Configuration{}
	consortiumID := data.ConsortiumID.ValueString()
	environmentID := data.EnvironmentID.ValueString()
	r.dataToAPIModel(ctx, &data, &apiModel, resp.Diagnostics)

	res, err := r.client.CreateConfiguration(consortiumID, environmentID, &apiModel)
	if err != nil {
		resp.Diagnostics.AddError("failed to create configuration", err.Error())
		return
	}

	status := res.StatusCode()
	if status != 201 {
		msg := "Could not create configuration %s in consortium %s in environment %s with status %d: %s"
		resp.Diagnostics.AddError("failed to create configuration", fmt.Sprintf(msg, apiModel.Type, consortiumID, environmentID, status, res.String()))
		return
	}

	r.copyConfigurationData(ctx, &apiModel, &data, resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *resourceConfiguration) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ConfigurationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	apiModel := kaleido.Configuration{}
	consortiumID := data.ConsortiumID.ValueString()
	environmentID := data.EnvironmentID.ValueString()
	r.dataToAPIModel(ctx, &data, &apiModel, resp.Diagnostics)
	configID := data.ID.String()

	res, err := r.client.UpdateConfiguration(consortiumID, environmentID, configID, &apiModel)
	if err != nil {
		resp.Diagnostics.AddError("failed to update configuration", err.Error())
		return
	}

	status := res.StatusCode()
	if status != 200 {
		msg := "Could not update configuration %s in consortium %s in environment %s with status %d: %s"
		resp.Diagnostics.AddError("failed to update configuration", fmt.Sprintf(msg, configID, consortiumID, environmentID, status, res.String()))
		return
	}

	r.copyConfigurationData(ctx, &apiModel, &data, resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *resourceConfiguration) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ConfigurationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	apiModel := kaleido.Configuration{}
	consortiumID := data.ConsortiumID.ValueString()
	environmentID := data.EnvironmentID.ValueString()
	configID := data.ID.String()

	res, err := r.client.GetConfiguration(consortiumID, environmentID, configID, &apiModel)
	if err != nil {
		resp.Diagnostics.AddError("failed to query configuration", err.Error())
		return
	}

	status := res.StatusCode()
	if status == 404 {
		resp.State.RemoveResource(ctx)
		return
	}
	if status != 200 {
		msg := "Could not find configuration %s in consortium %s in environment %s with status %d: %s"
		resp.Diagnostics.AddError("failed to query configuration", fmt.Sprintf(msg, configID, consortiumID, environmentID, status, res.String()))
		return
	}

	r.copyConfigurationData(ctx, &apiModel, &data, resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *resourceConfiguration) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ConfigurationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	consortiumID := data.ConsortiumID.ValueString()
	environmentID := data.EnvironmentID.ValueString()
	configID := data.ID.String()

	res, err := r.client.DeleteConfiguration(consortiumID, environmentID, configID)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete configuration", err.Error())
		return
	}

	statusCode := res.StatusCode()
	if statusCode != 202 && statusCode != 204 {
		msg := "Failed to delete configuration %s in consortium %s in environment %s with status %d: %s"
		resp.Diagnostics.AddError("failed to delete configuration", fmt.Sprintf(msg, configID, environmentID, consortiumID, statusCode, res.String()))
		return
	}

}
