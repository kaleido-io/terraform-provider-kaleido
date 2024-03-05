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

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	kaleido "github.com/kaleido-io/kaleido-sdk-go/kaleido"
)

type resourceCZone struct {
	baasBaseResource
}

func ResourceCZoneFactory() resource.Resource {
	return &resourceCZone{}
}

type CZoneResourceModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	ConsortiumID types.String `tfsdk:"consortium_id"`
	Region       types.String `tfsdk:"region"`
	Cloud        types.String `tfsdk:"cloud"`
}

func (r *resourceCZone) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_czone"
}

func (r *resourceCZone) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"region": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"cloud": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},
	}
}

func (r *resourceCZone) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var data CZoneResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	apiModel := kaleido.CZone{}
	consortiumID := data.ConsortiumID.ValueString()
	apiModel.Name = data.Name.ValueString()
	apiModel.Region = data.Region.ValueString()
	apiModel.Cloud = data.Cloud.ValueString()

	var existing []kaleido.CZone
	res, err := r.BaaS.ListCZones(consortiumID, &existing)
	if err != nil {
		resp.Diagnostics.AddError("failed to list consortium zones", err.Error())
		return
	}
	if res.StatusCode() != 200 {
		resp.Diagnostics.AddError("failed to list consortium zones", fmt.Sprintf("Failed to list existing consortia zones with status %d: %s", res.StatusCode(), res.String()))
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

		res, err = r.BaaS.CreateCZone(consortiumID, &apiModel)
		if err != nil {
			resp.Diagnostics.AddError("failed to create consortium zone", err.Error())
			return
		}

		status := res.StatusCode()
		if status != 201 {
			msg := "Could not create czone in consortium %s with status %d: %s"
			resp.Diagnostics.AddError("failed to create consortium zone", fmt.Sprintf(msg, consortiumID, status, res.String()))
			return
		}
	}

	data.ID = types.StringValue(apiModel.ID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *resourceCZone) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data CZoneResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	apiModel := kaleido.CZone{}
	consortiumID := data.ConsortiumID.ValueString()
	czoneID := data.ID.ValueString()
	apiModel.Name = data.Name.ValueString()

	res, err := r.BaaS.UpdateCZone(consortiumID, czoneID, &apiModel)
	if err != nil {
		resp.Diagnostics.AddError("failed to update consortium zone", err.Error())
		return
	}

	status := res.StatusCode()
	if status != 200 {
		msg := "Could not update czone %s in consortium %s with status %d: %s"
		resp.Diagnostics.AddError("failed to update consortium zone", fmt.Sprintf(msg, czoneID, consortiumID, status, res.String()))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *resourceCZone) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CZoneResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	var apiModel kaleido.CZone
	consortiumID := data.ConsortiumID.ValueString()
	czoneID := data.ID.ValueString()

	res, err := r.BaaS.GetCZone(consortiumID, czoneID, &apiModel)
	if err != nil {
		resp.Diagnostics.AddError("failed to query consortium zone", err.Error())
		return
	}

	status := res.StatusCode()
	if status == 404 {
		resp.State.RemoveResource(ctx)
		return
	}
	if status != 200 {
		msg := "Could not find czone %s in consortium %s with status %d: %s"
		resp.Diagnostics.AddError("failed to query consortium zone", fmt.Sprintf(msg, czoneID, consortiumID, status, res.String()))
	}

	data.Name = types.StringValue(apiModel.Name)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *resourceCZone) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Treated as a no-op
}
