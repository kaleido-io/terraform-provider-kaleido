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

type resourceInvitation struct {
	baasBaseResource
}

func ResourceInvitationFactory() resource.Resource {
	return &resourceInvitation{}
}

type InvitationResourceModel struct {
	ID           types.String `tfsdk:"id"`
	ConsortiumID types.String `tfsdk:"consortium_id"`
	OrgName      types.String `tfsdk:"org_name"`
	Email        types.String `tfsdk:"email"`
}

func (r *resourceInvitation) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_invitation"
}

func (r *resourceInvitation) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed: true,
			},
			"consortium_id": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"org_name": &schema.StringAttribute{
				Required: true,
			},
			"email": &schema.StringAttribute{
				Required: true,
			},
		},
	}
}

func (r *resourceInvitation) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var data InvitationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	apiModel := kaleido.Invitation{}
	consortiumID := data.ConsortiumID.ValueString()
	apiModel.OrgName = data.OrgName.ValueString()
	apiModel.Email = data.Email.ValueString()

	res, err := r.BaaS.CreateInvitation(consortiumID, &apiModel)

	if err != nil {
		resp.Diagnostics.AddError("failed to create invitation", err.Error())
		return
	}

	status := res.StatusCode()
	if status != 201 {
		msg := "Failed to create invitation %s in consortium %s with status %d: %s"
		resp.Diagnostics.AddError("failed to create invitation", fmt.Sprintf(msg, apiModel.OrgName, consortiumID, status, res.String()))
		return
	}

	data.ID = types.StringValue(apiModel.ID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

}

func (r *resourceInvitation) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data InvitationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	apiModel := kaleido.Invitation{}
	apiModel.OrgName = data.OrgName.ValueString()
	apiModel.Email = data.Email.ValueString()
	consortiumID := data.ConsortiumID.ValueString()
	inviteID := data.ID.ValueString()

	res, err := r.BaaS.UpdateInvitation(consortiumID, inviteID, &apiModel)

	if err != nil {
		resp.Diagnostics.AddError("failed to update invitation", err.Error())
		return
	}

	status := res.StatusCode()
	if status != 200 {
		msg := "Failed to update invitation %s for %s in consortium %s with status %d: %s"
		resp.Diagnostics.AddError("failed to update invitation", fmt.Sprintf(msg, inviteID, apiModel.OrgName, consortiumID, status, res.String()))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *resourceInvitation) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data InvitationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	consortiumID := data.ConsortiumID.ValueString()
	inviteID := data.ID.ValueString()

	var apiModel kaleido.Invitation
	res, err := r.BaaS.GetInvitation(consortiumID, inviteID, &apiModel)

	if err != nil {
		resp.Diagnostics.AddError("failed to query invitation", err.Error())
		return
	}

	status := res.StatusCode()
	if status == 404 {
		resp.State.RemoveResource(ctx)
		return
	}
	if status != 200 {
		msg := "Failed to find invitation %s in consortium %s with status %d: %s"
		resp.Diagnostics.AddError("failed to query invitation", fmt.Sprintf(msg, apiModel.OrgName, consortiumID, status, res.String()))
	}

	data.OrgName = types.StringValue(apiModel.OrgName)
	data.Email = types.StringValue(apiModel.Email)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

}

func (r *resourceInvitation) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data InvitationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	consortiumID := data.ConsortiumID.ValueString()
	invitationID := data.ID.ValueString()

	res, err := r.BaaS.DeleteInvitation(consortiumID, invitationID)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete invitation", err.Error())
		return
	}

	status := res.StatusCode()
	if status != 204 {
		msg := "Failed to delete invitation %s in consortium %s with status %d: %s"
		resp.Diagnostics.AddError("failed to delete invitation", fmt.Sprintf(msg, invitationID, consortiumID, status, res.String()))
		return
	}
}
