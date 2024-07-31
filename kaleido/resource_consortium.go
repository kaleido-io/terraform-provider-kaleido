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
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	kaleido "github.com/kaleido-io/kaleido-sdk-go/kaleido"
)

type resourceConsortium struct {
	baasBaseResource
}

func ResourceConsortiumFactory() resource.Resource {
	return &resourceConsortium{}
}

type ConsortiumResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	SharedDeployment types.Bool   `tfsdk:"shared_deployment"`
}

func (r *resourceConsortium) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_consortium"
}

func (r *resourceConsortium) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": &schema.StringAttribute{
				Required: true,
			},
			"description": &schema.StringAttribute{
				Optional: true,
			},
			"shared_deployment": &schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "The decentralized nature of Kaleido means a consortium might be shared with other accounts. When true only create if name does not exist, and delete becomes a no-op.",
			},
		},
	}
}

func (r *resourceConsortium) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var data ConsortiumResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	apiModel := kaleido.Consortium{}
	apiModel.Name = data.Name.ValueString()
	apiModel.Description = data.Description.ValueString()

	sharedExisting := false
	if data.SharedDeployment.ValueBool() {
		var consortia []kaleido.Consortium
		res, err := r.BaaS.ListConsortium(&consortia)
		if err != nil {
			resp.Diagnostics.AddError("failed to list consortia", err.Error())
			return
		}
		if res.StatusCode() != 200 {
			resp.Diagnostics.AddError("failed to list consortia", fmt.Sprintf("Failed to list existing consortia with status %d: %s", res.StatusCode(), res.String()))
			return
		}
		for _, c := range consortia {
			if c.Name == data.Name.ValueString() && !strings.Contains(c.State, "delete") {
				// Already exists, just re-use
				sharedExisting = true
				apiModel = c
			}
		}
	}

	if !sharedExisting {
		res, err := r.BaaS.CreateConsortium(&apiModel)
		if err != nil {
			resp.Diagnostics.AddError("failed to create consortium", err.Error())
			return
		}
		status := res.StatusCode()
		if status != 201 {
			resp.Diagnostics.AddError("failed to create consortium", fmt.Sprintf("Failed to create consortium with status %d", status))
			return
		}
	}

	data.ID = types.StringValue(apiModel.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *resourceConsortium) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ConsortiumResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	apiModel := kaleido.Consortium{}
	apiModel.Name = data.Name.ValueString()
	apiModel.Description = data.Description.ValueString()
	var consortiumID types.String
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("id"), &consortiumID)...)

	res, err := r.BaaS.UpdateConsortium(consortiumID.ValueString(), &apiModel)
	if err != nil {
		resp.Diagnostics.AddError("failed to update consortium", err.Error())
		return
	}
	status := res.StatusCode()
	if status != 200 {
		resp.Diagnostics.AddError("failed to update consortium", fmt.Sprintf("Failed to update consortium %s with status: %d", consortiumID, status))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *resourceConsortium) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ConsortiumResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	var apiModel kaleido.Consortium
	consortiumID := data.ID.ValueString()
	res, err := r.BaaS.GetConsortium(consortiumID, &apiModel)
	if err != nil {
		resp.Diagnostics.AddError("failed to query consortium", err.Error())
		return
	}

	status := res.StatusCode()
	if status == 404 {
		resp.State.RemoveResource(ctx)
		return
	}
	if status != 200 {
		resp.Diagnostics.AddError("failed to query consortium", fmt.Sprintf("Failed to read consortium with id %s status was: %d, error: %s", consortiumID, status, res.String()))
		return
	}

	data.Name = types.StringValue(apiModel.Name)
	data.Description = types.StringValue(apiModel.Description)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *resourceConsortium) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ConsortiumResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if data.SharedDeployment.ValueBool() {
		// Cannot safely delete if this is shared with other terraform deployments
		// Pretend we deleted it
		return
	}

	consortiumID := data.ID.ValueString()
	res, err := r.BaaS.DeleteConsortium(consortiumID)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete consortium", err.Error())
		return
	}

	status := res.StatusCode()
	if status != 202 {
		resp.Diagnostics.AddError("failed to delete consortium", fmt.Sprintf("failed to delete consortium with id %s status was %d, error: %s", consortiumID, status, res.String()))
		return
	}
}
