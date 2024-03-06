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

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	kaleido "github.com/kaleido-io/kaleido-sdk-go/kaleido"
)

type resourceEZone struct {
	baasBaseResource
}

func ResourceEZoneFactory() resource.Resource {
	return &resourceEZone{}
}

type EZoneResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	ConsortiumID  types.String `tfsdk:"consortium_id"`
	EnvironmentID types.String `tfsdk:"environment_id"`
	Region        types.String `tfsdk:"region"`
	Cloud         types.String `tfsdk:"cloud"`
	BridgeID      types.String `tfsdk:"bridge_id"`
	Type          types.String `tfsdk:"type"`
}

func (r *resourceEZone) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_ezone"
}

func (r *resourceEZone) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed: true,
			},
			"name": &schema.StringAttribute{
				Optional: true,
			},
			"consortium_id": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"environment_id": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"region": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"cloud": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"bridge_id": &schema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"type": &schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("kaleido"),
			},
		},
	}
}

func (r *resourceEZone) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var data EZoneResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	apiModel := kaleido.EZone{}
	consortiumID := data.ConsortiumID.ValueString()
	environmentID := data.EnvironmentID.ValueString()
	apiModel.Name = data.Name.ValueString()
	apiModel.Region = data.Region.ValueString()
	apiModel.Cloud = data.Cloud.ValueString()
	apiModel.Type = data.Type.ValueString()
	apiModel.BridgeID = data.BridgeID.ValueString()

	var existing []kaleido.EZone
	res, err := r.BaaS.ListEZones(consortiumID, environmentID, &existing)
	if err != nil {
		resp.Diagnostics.AddError("failed to list environment zones", err.Error())
		return
	}
	if res.StatusCode() != 200 {
		resp.Diagnostics.AddError("failed to list environment zones", fmt.Sprintf("Failed to list existing environment zones with status %d: %s", res.StatusCode(), res.String()))
		return
	}
	exists := false
	for _, e := range existing {
		if e.Cloud == data.Cloud.ValueString() && e.Region == data.Region.ValueString() {
			// Already exists, just re-use
			apiModel = e
			exists = true
		}
	}
	if !exists {
		res, err = r.BaaS.CreateEZone(consortiumID, environmentID, &apiModel)
		if err != nil {
			resp.Diagnostics.AddError("failed to create environment zone", err.Error())
			return
		}

		status := res.StatusCode()
		if status != 201 {
			msg := "Could not create ezone in consortium %s in environment %s with status %d: %s"
			resp.Diagnostics.AddError("failed to create environment zone", fmt.Sprintf(msg, consortiumID, environmentID, status, res.String()))
			return
		}
	}

	data.ID = types.StringValue(apiModel.ID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *resourceEZone) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data EZoneResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	apiModel := kaleido.EZone{}
	consortiumID := data.ConsortiumID.ValueString()
	environmentID := data.EnvironmentID.ValueString()
	apiModel.Name = data.Name.ValueString()
	var ezoneID types.String
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("id"), &ezoneID)...)

	res, err := r.BaaS.UpdateEZone(consortiumID, environmentID, ezoneID.ValueString(), &apiModel)
	if err != nil {
		resp.Diagnostics.AddError("failed to update environment zone", err.Error())
		return
	}

	status := res.StatusCode()
	if status != 200 {
		msg := "Could not update ezone %s in consortium %s in environment %s with status %d: %s"
		resp.Diagnostics.AddError("failed to update environment zone", fmt.Sprintf(msg, ezoneID, consortiumID, environmentID, status, res.String()))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *resourceEZone) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data EZoneResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	var apiModel kaleido.EZone
	consortiumID := data.ConsortiumID.ValueString()
	environmentID := data.EnvironmentID.ValueString()
	ezoneID := data.ID.ValueString()

	res, err := r.BaaS.GetEZone(consortiumID, environmentID, ezoneID, &apiModel)
	if err != nil {
		resp.Diagnostics.AddError("failed to query environment zone", err.Error())
		return
	}

	status := res.StatusCode()
	if status == 404 {
		resp.State.RemoveResource(ctx)
		return
	}
	if status != 200 {
		msg := "Could not find ezone %s in consortium %s in environment %s with status %d: %s"
		resp.Diagnostics.AddError("failed to query environment zone", fmt.Sprintf(msg, ezoneID, consortiumID, environmentID, status, res.String()))
	}

	data.Name = types.StringValue(apiModel.Name)
	data.Type = types.StringValue(apiModel.Type)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *resourceEZone) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Treated as a no-op
}
