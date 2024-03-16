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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	kaleido "github.com/kaleido-io/kaleido-sdk-go/kaleido"
)

type resourceDestination struct {
	baasBaseResource
}

func ResourceDestinationFactory() resource.Resource {
	return &resourceDestination{}
}

type DestinationResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	ServiceType          types.String `tfsdk:"service_type"`
	ServiceID            types.String `tfsdk:"service_id"`
	Name                 types.String `tfsdk:"name"`
	KaleidoManaged       types.Bool   `tfsdk:"kaleido_managed"`
	ConsortiumID         types.String `tfsdk:"consortium_id"`
	MembershipID         types.String `tfsdk:"membership_id"`
	IDRegistryID         types.String `tfsdk:"idregistry_id"`
	AutoVerifyMembership types.Bool   `tfsdk:"auto_verify_membership"`
}

func (r *resourceDestination) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_destination"
}

func (r *resourceDestination) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"service_type": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"service_id": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"kaleido_managed": &schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
			"consortium_id": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"membership_id": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"idregistry_id": &schema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"auto_verify_membership": &schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Default:       booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
			},
		},
	}
}

func (r *resourceDestination) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var data DestinationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	apiModel := kaleido.Destination{}
	apiModel.Name = data.Name.ValueString()
	apiModel.KaleidoManaged = data.KaleidoManaged.ValueBool()
	consortiumID := data.ConsortiumID.ValueString()
	membershipID := data.MembershipID.ValueString()
	serviceType := data.ServiceType.ValueString()
	serviceID := data.ServiceID.ValueString()
	idregistryID := data.IDRegistryID.ValueString()

	var membership kaleido.Membership
	res, err := r.BaaS.GetMembership(consortiumID, membershipID, &membership)
	if err != nil {
		resp.Diagnostics.AddError("failed to query membership", err.Error())
		return
	}

	status := res.StatusCode()
	if status != 200 {
		msg := "Failed to get membership %s in consortium %s with status %d: %s"
		resp.Diagnostics.AddError("failed to query membership", fmt.Sprintf(msg, membershipID, consortiumID, status, res.String()))
	}

	if data.AutoVerifyMembership.ValueBool() {
		if membership.VerificationProof == "" {
			res, err = r.BaaS.CreateMembershipVerification(consortiumID, membershipID, &kaleido.MembershipVerification{
				TestCertificate: true,
			})
			if err != nil {
				resp.Diagnostics.AddError("failed to create membership verification", err.Error())
				return
			}
			status = res.StatusCode()
			if status != 200 {
				msg := "Failed to auto create self-signed membership verification proof for membership %s in consortium %s with status %d: %s"
				resp.Diagnostics.AddError("failed to create membership verification", fmt.Sprintf(msg, membershipID, consortiumID, status, res.String()))
				return
			}
		}

		res, err = r.BaaS.RegisterMembershipIdentity(idregistryID, membershipID)
		if err != nil {
			resp.Diagnostics.AddError("failed to auto register membership verification", err.Error())
			return
		}
		status = res.StatusCode()
		if status != 200 && status != 409 /* already registered */ {
			msg := "Failed to auto register membership verification for membership %s in consortium %s using idregistry %s with status %d: %s"
			resp.Diagnostics.AddError("failed to auto register membership verification", fmt.Sprintf(msg, membershipID, consortiumID, idregistryID, status, res.String()))
			return
		}
	}

	res, err = r.BaaS.CreateDestination(serviceType, serviceID, &apiModel)
	if err != nil {
		resp.Diagnostics.AddError("failed to create destination", err.Error())
		return
	}

	status = res.StatusCode()
	if status != 200 {
		msg := "Failed to create destination %s in %s service %s for membership %s with status %d: %s"
		resp.Diagnostics.AddError("failed to create destination", fmt.Sprintf(msg, apiModel.Name, serviceType, serviceID, membershipID, status, res.String()))
		return
	}

	var destinations []kaleido.Destination
	res, err = r.BaaS.ListDestinations(serviceType, serviceID, &destinations)

	if err != nil {
		resp.Diagnostics.AddError("failed to list destinations", err.Error())
		return
	}

	status = res.StatusCode()
	if status != 200 {
		msg := "Failed to query newly created destination %s in %s service %s for membership %s with status %d: %s"
		resp.Diagnostics.AddError("failed to list destinations", fmt.Sprintf(msg, apiModel.Name, serviceType, serviceID, membershipID, status, res.String()))
		return
	}

	var createdDest *kaleido.Destination
	for _, d := range destinations {
		if d.Name == apiModel.Name {
			createdDest = &d
			break
		}
	}

	if createdDest == nil {
		msg := "Failed to find newly created destination %s in %s service %s for membership %s"
		resp.Diagnostics.AddError("failed to query destination after creation", fmt.Sprintf(msg, apiModel.Name, serviceType, serviceID, membershipID))
		return
	}

	data.ID = types.StringValue(createdDest.URI)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *resourceDestination) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// No-op - nothing about a destination is editable currently
}

func (r *resourceDestination) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DestinationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	serviceType := data.ServiceType.ValueString()
	serviceID := data.ServiceID.ValueString()
	membershipID := data.MembershipID.ValueString()
	destName := data.Name.ValueString()
	destURI := data.ID.ValueString()

	var destinations []kaleido.Destination
	res, err := r.BaaS.ListDestinations(serviceType, serviceID, &destinations)

	if err != nil {
		resp.Diagnostics.AddError("failed to list destinations", err.Error())
		return
	}

	status := res.StatusCode()
	if status != 200 {
		msg := "Failed to query newly created destination %s in %s service %s for membership %s with status %d: %s"
		resp.Diagnostics.AddError("failed to list destinations", fmt.Sprintf(msg, destName, serviceType, serviceID, membershipID, status, res.String()))
		return
	}

	var destination *kaleido.Destination
	for _, d := range destinations {
		if d.URI == destURI {
			destination = &d
			break
		}
	}

	if destination == nil {
		msg := "Failed to find destination %s in %s service %s"
		resp.Diagnostics.AddError("failed to find destination", fmt.Sprintf(msg, destURI, serviceType, serviceID))
		return
	}

}

func (r *resourceDestination) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DestinationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	serviceType := data.ServiceType.ValueString()
	serviceID := data.ServiceID.ValueString()
	destName := data.Name.ValueString()

	res, err := r.BaaS.DeleteDestination(serviceType, serviceID, destName)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete destination", err.Error())
		return
	}

	status := res.StatusCode()
	if status != 204 && status != 404 {
		msg := "Failed to delete destination %s in %s service %s with status %d: %s"
		resp.Diagnostics.AddError("failed to delete destination", fmt.Sprintf(msg, destName, serviceType, serviceID, status, res.String()))
		return
	}
}
