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
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type CantonParticipantNodeServiceResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Environment         types.String `tfsdk:"environment"`
	EnvironmentMemberID types.String `tfsdk:"environment_member_id"`
	Runtime             types.String `tfsdk:"runtime"`
	Name                types.String `tfsdk:"name"`
	StackID             types.String `tfsdk:"stack_id"`
	Defaultparty        types.String `tfsdk:"defaultparty"`
	Domainnetworks      types.String `tfsdk:"domainnetworks"`
	Globaldomainnetwork types.String `tfsdk:"globaldomainnetwork"`
	Kms                 types.String `tfsdk:"kms"`
	Onboardingsecret    types.String `tfsdk:"onboardingsecret"`
	Walletusers         types.String `tfsdk:"walletusers"`
	ForceDelete         types.Bool   `tfsdk:"force_delete"`
}

func CantonParticipantNodeServiceResourceFactory() resource.Resource {
	return &cantonparticipantnodeserviceResource{}
}

type cantonparticipantnodeserviceResource struct {
	commonResource
}

func (r *cantonparticipantnodeserviceResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_cantonparticipantnode"
}

func (r *cantonparticipantnodeserviceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A CantonParticipantNode service that provides blockchain functionality.",
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"environment": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Environment ID where the CantonParticipantNode service will be deployed",
			},
			"environment_member_id": &schema.StringAttribute{
				Computed: true,
			},
			"runtime": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Runtime ID where the CantonParticipantNode service will be deployed",
			},
			"name": &schema.StringAttribute{
				Required:    true,
				Description: "Display name for the CantonParticipantNode service",
			},
			"stack_id": &schema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Stack ID where the CantonParticipantNode service belongs (optional)",
			},
			"defaultparty": &schema.StringAttribute{
				Optional:    true,
				Description: "Party Hint for the default party",
			},
			"domainnetworks": &schema.StringAttribute{
				Optional:    true,
				Description: "CantonPrivateSynchronizerNetworks",
			},
			"globaldomainnetwork": &schema.StringAttribute{
				Optional:    true,
				Description: "CantonGlobalDomainNetwork",
			},
			"kms": &schema.StringAttribute{
				Optional:    true,
				Description: "Configuration KMS for this canton node",
			},
			"onboardingsecret": &schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Onboarding secret provided by the super validator",
			},
			"walletusers": &schema.StringAttribute{
				Optional:    true,
				Description: "A list of additional wallet users",
			},
			"force_delete": &schema.BoolAttribute{
				Optional:    true,
				Description: "Set to true when you plan to delete a protected CantonParticipantNode service. You must apply this value before running terraform destroy.",
			},
		},
	}
}

func (data *CantonParticipantNodeServiceResourceModel) toCantonParticipantNodeServiceAPI(ctx context.Context, api *ServiceAPIModel, diagnostics *diag.Diagnostics) {
	api.Type = "CantonParticipantNodeService"
	api.Name = data.Name.ValueString()
	api.StackID = data.StackID.ValueString()
	api.Runtime.ID = data.Runtime.ValueString()
	api.Config = make(map[string]interface{})

	if !data.Globaldomainnetwork.IsNull() && data.Globaldomainnetwork.ValueString() != "" {
		api.Config["globalDomainNetwork"] = data.Globaldomainnetwork.ValueString()
	}
	// Handle Domainnetworks as JSON
	if !data.Domainnetworks.IsNull() && data.Domainnetworks.ValueString() != "" {
		var domainnetworksData interface{}
		err := json.Unmarshal([]byte(data.Domainnetworks.ValueString()), &domainnetworksData)
		if err != nil {
			diagnostics.AddAttributeError(
				path.Root("domainnetworks"),
				"Failed to parse Domainnetworks",
				err.Error(),
			)
		} else {
			api.Config["domainNetworks"] = domainnetworksData
		}
	}
	if !data.Defaultparty.IsNull() && data.Defaultparty.ValueString() != "" {
		api.Config["defaultParty"] = data.Defaultparty.ValueString()
	}
	if !data.Onboardingsecret.IsNull() && data.Onboardingsecret.ValueString() != "" {
		api.Config["onBoardingSecret"] = data.Onboardingsecret.ValueString()
	}
	if !data.Kms.IsNull() && data.Kms.ValueString() != "" {
		api.Config["kms"] = data.Kms.ValueString()
	}
	// Handle Walletusers as JSON
	if !data.Walletusers.IsNull() && data.Walletusers.ValueString() != "" {
		var walletusersData interface{}
		err := json.Unmarshal([]byte(data.Walletusers.ValueString()), &walletusersData)
		if err != nil {
			diagnostics.AddAttributeError(
				path.Root("walletusers"),
				"Failed to parse Walletusers",
				err.Error(),
			)
		} else {
			api.Config["walletUsers"] = walletusersData
		}
	}
}

func (api *ServiceAPIModel) toCantonParticipantNodeServiceData(data *CantonParticipantNodeServiceResourceModel, diagnostics *diag.Diagnostics) {
	data.ID = types.StringValue(api.ID)
	data.EnvironmentMemberID = types.StringValue(api.EnvironmentMemberID)
	data.Runtime = types.StringValue(api.Runtime.ID)
	data.Name = types.StringValue(api.Name)
	data.StackID = types.StringValue(api.StackID)

	if v, ok := api.Config["globalDomainNetwork"].(string); ok {
		data.Globaldomainnetwork = types.StringValue(v)
	} else {
		data.Globaldomainnetwork = types.StringNull()
	}
	if domainnetworksData := api.Config["domainNetworks"]; domainnetworksData != nil {
		if domainnetworksJSON, err := json.Marshal(domainnetworksData); err == nil {
			data.Domainnetworks = types.StringValue(string(domainnetworksJSON))
		} else {
			data.Domainnetworks = types.StringNull()
		}
	} else {
		data.Domainnetworks = types.StringNull()
	}
	if v, ok := api.Config["defaultParty"].(string); ok {
		data.Defaultparty = types.StringValue(v)
	} else {
		data.Defaultparty = types.StringNull()
	}
	if v, ok := api.Config["onBoardingSecret"].(string); ok {
		data.Onboardingsecret = types.StringValue(v)
	} else {
		data.Onboardingsecret = types.StringNull()
	}
	if v, ok := api.Config["kms"].(string); ok {
		data.Kms = types.StringValue(v)
	} else {
		data.Kms = types.StringNull()
	}
	if walletusersData := api.Config["walletUsers"]; walletusersData != nil {
		if walletusersJSON, err := json.Marshal(walletusersData); err == nil {
			data.Walletusers = types.StringValue(string(walletusersJSON))
		} else {
			data.Walletusers = types.StringNull()
		}
	} else {
		data.Walletusers = types.StringNull()
	}
}

func (r *cantonparticipantnodeserviceResource) apiPath(data *CantonParticipantNodeServiceResourceModel) string {
	path := fmt.Sprintf("/api/v1/environments/%s/services", data.Environment.ValueString())
	if data.ID.ValueString() != "" {
		path = path + "/" + data.ID.ValueString()
	}
	if !data.ForceDelete.IsNull() && data.ForceDelete.ValueBool() {
		path = path + "?force=true"
	}
	return path
}

func (r *cantonparticipantnodeserviceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data CantonParticipantNodeServiceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	var api ServiceAPIModel
	data.toCantonParticipantNodeServiceAPI(ctx, &api, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	ok, _ := r.apiRequest(ctx, http.MethodPost, r.apiPath(&data), api, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	api.toCantonParticipantNodeServiceData(&data, &resp.Diagnostics)
	r.waitForReadyStatus(ctx, r.apiPath(&data), &resp.Diagnostics)

	api.ID = data.ID.ValueString()
	ok, _ = r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	api.toCantonParticipantNodeServiceData(&data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *cantonparticipantnodeserviceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data CantonParticipantNodeServiceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)

	var api ServiceAPIModel
	if ok, _ := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics); !ok {
		return
	}

	data.toCantonParticipantNodeServiceAPI(ctx, &api, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if ok, _ := r.apiRequest(ctx, http.MethodPut, r.apiPath(&data), api, &api, &resp.Diagnostics); !ok {
		return
	}

	api.toCantonParticipantNodeServiceData(&data, &resp.Diagnostics)
	r.waitForReadyStatus(ctx, r.apiPath(&data), &resp.Diagnostics)

	if ok, _ := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics); !ok {
		return
	}
	api.toCantonParticipantNodeServiceData(&data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *cantonparticipantnodeserviceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CantonParticipantNodeServiceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	var api ServiceAPIModel
	api.ID = data.ID.ValueString()
	ok, status := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics, Allow404())
	if !ok {
		return
	}
	if status == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	api.toCantonParticipantNodeServiceData(&data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *cantonparticipantnodeserviceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CantonParticipantNodeServiceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data), nil, nil, &resp.Diagnostics, Allow404())
	r.waitForRemoval(ctx, r.apiPath(&data), &resp.Diagnostics)
}
