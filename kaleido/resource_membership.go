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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	kaleido "github.com/kaleido-io/kaleido-sdk-go/kaleido"
)

type resourceMembership struct {
	baasBaseResource
}

func ResourceMembershipFactory() resource.Resource {
	return &resourceMembership{}
}

type MembershipResourceModel struct {
	ID           types.String `tfsdk:"id"`
	ConsortiumID types.String `tfsdk:"consortium_id"`
	OrgName      types.String `tfsdk:"org_name"`
	PreExisting  types.Bool   `tfsdk:"pre_existing"`
}

func (r *resourceMembership) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_membership"
}

func (r *resourceMembership) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"pre_existing": &schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "In a decentalized consortium memberships are driven by invitation, and will be pre-existing at the point of deploying infrastructure.",
			},
		},
	}
}

func (r *resourceMembership) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var data MembershipResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	apiModel := kaleido.Membership{}
	consortiumID := data.ConsortiumID.ValueString()
	apiModel.OrgName = data.OrgName.ValueString()

	if data.PreExisting.ValueBool() {
		var memberships []kaleido.Membership
		res, err := r.baas.ListMemberships(consortiumID, &memberships)
		if err != nil {
			resp.Diagnostics.AddError("failed to list memberships", err.Error())
			return
		}
		if res.StatusCode() != 200 {
			resp.Diagnostics.AddError("failed to list memberships", fmt.Sprintf("Failed to list existing memberships with status %d: %s", res.StatusCode(), res.String()))
			return
		}
		found := false
		for _, e := range memberships {
			if e.OrgName == data.OrgName.ValueString() {
				apiModel = e
				found = true
				break
			}
		}
		if !found {
			msg := "pre_existing set and no existing membership found with org_name '%s'"
			resp.Diagnostics.AddError("membership not found", fmt.Sprintf(msg, data.OrgName.ValueString()))
			return
		}
	} else {
		res, err := r.baas.CreateMembership(consortiumID, &apiModel)
		if err != nil {
			resp.Diagnostics.AddError("failed to create membership", err.Error())
			return
		}

		status := res.StatusCode()
		if status != 201 {
			msg := "Failed to create membership %s in consortium %s with status %d: %s"
			resp.Diagnostics.AddError("failed to create service", fmt.Sprintf(msg, apiModel.OrgName, consortiumID, status, res.String()))
			return
		}
	}

	data.ID = types.StringValue(apiModel.ID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *resourceMembership) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data MembershipResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	apiModel := kaleido.Membership{}
	apiModel.OrgName = data.OrgName.ValueString()
	consortiumID := data.ConsortiumID.ValueString()
	membershipID := data.ID.ValueString()

	res, err := r.baas.UpdateMembership(consortiumID, membershipID, &apiModel)
	if err != nil {
		resp.Diagnostics.AddError("failed to update membership", err.Error())
		return
	}

	status := res.StatusCode()
	if status != 200 {
		msg := "Failed to update membership %s for %s in consortium %s with status %d: %s"
		resp.Diagnostics.AddError("failed to update service", fmt.Sprintf(msg, membershipID, apiModel.OrgName, consortiumID, status, res.String()))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *resourceMembership) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data MembershipResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	var apiModel kaleido.Membership
	consortiumID := data.ConsortiumID.ValueString()
	res, err := r.baas.GetMembership(consortiumID, data.ID.ValueString(), &apiModel)

	if err != nil {
		resp.Diagnostics.AddError("failed to query membership", err.Error())
		return
	}

	status := res.StatusCode()
	if status != 200 {
		msg := "Failed to find membership %s in consortium %s with status %d: %s"
		resp.Diagnostics.AddError("failed to query membership", fmt.Sprintf(msg, apiModel.OrgName, consortiumID, status, res.String()))
		return
	}

	data.OrgName = types.StringValue(apiModel.OrgName)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *resourceMembership) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data MembershipResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if data.PreExisting.ValueBool() {
		// Cannot safely delete if this is shared with other terraform deployments
		// Pretend we deleted it
		return
	}

	consortiumID := data.ConsortiumID.ValueString()
	membershipID := data.ID.ValueString()

	err := Retry.Do(ctx, "Delete", func(attempt int) (retry bool, err error) {
		res, deleteErr := r.baas.DeleteMembership(consortiumID, membershipID)
		if deleteErr != nil {
			return false, deleteErr
		}

		statusCode := res.StatusCode()
		if statusCode >= 500 {
			err := fmt.Errorf("deletion of membership %s failed: %d", membershipID, statusCode)
			return true, err
		} else if statusCode != 204 {
			msg := "failed to delete membership %s in consortium %s with status %d: %s"
			return true, fmt.Errorf(msg, membershipID, consortiumID, statusCode, res.String())
		}

		return false, nil
	})

	if err != nil {
		resp.Diagnostics.AddError("failed to delete membership", err.Error())
		return
	}
}
