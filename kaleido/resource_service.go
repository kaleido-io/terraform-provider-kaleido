// Copyright © Kaleido, Inc. 2018, 2021

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

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	kaleido "github.com/kaleido-io/kaleido-sdk-go/kaleido"
)

type resourceService struct {
	baasBaseResource
}

func ResourceServiceFactory(client *kaleido.KaleidoClient) func() resource.Resource {
	return func() resource.Resource {
		return &resourceService{}
	}
}

type ServiceResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	ServiceType          types.String `tfsdk:"service_type"`
	ConsortiumID         types.String `tfsdk:"consortium_id"`
	EnvironmentID        types.String `tfsdk:"environment_id"`
	MembershipID         types.String `tfsdk:"membership_id"`
	ZoneID               types.String `tfsdk:"zone_id"`
	SharedDeployment     types.Bool   `tfsdk:"shared_deployment"`
	Size                 types.String `tfsdk:"size"`
	Details              types.Map    `tfsdk:"details"`
	HttpsURL             types.String `tfsdk:"https_url"`
	WebSocketURL         types.String `tfsdk:"websocket_url"`
	WebUiURL             types.String `tfsdk:"webui_url"`
	URLs                 types.Map    `tfsdk:"urls"`
	HybridPortAllocation types.Int64  `tfsdk:"hybrid_port_allocation"`
	UpdateTrigger        types.String `tfsdk:"update_trigger"`
}

func (r *resourceService) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_service"
}

func (r *resourceService) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": &schema.StringAttribute{
				Required: true,
			},
			"service_type": &schema.StringAttribute{
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
			"zone_id": &schema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"shared_deployment": &schema.BoolAttribute{
				Optional:    true,
				Default:     booldefault.StaticBool(false),
				Description: "The decentralized nature of Kaleido means a utility service might be shared with other accounts. When true only create if service_type does not exist, and delete becomes a no-op.",
			},
			"size": &schema.StringAttribute{
				Optional: true,
			},
			"details": &schema.MapAttribute{
				Optional: true,
			},
			"https_url": &schema.StringAttribute{
				Computed: true,
			},
			"websocket_url": &schema.StringAttribute{
				Computed: true,
			},
			"webui_url": &schema.StringAttribute{
				Computed: true,
			},
			"urls": &schema.MapAttribute{
				Computed: true,
			},
			"hybrid_port_allocation": &schema.Int64Attribute{
				Computed: true,
			},
			"update_trigger": &schema.StringAttribute{
				Optional: true,
			},
		},
	}
}

func (r *resourceService) waitUntilServiceStarted(ctx context.Context, op, consortiumID, environmentID, serviceID string, apiModel *kaleido.Service, data *ServiceResourceModel, diagnostics diag.Diagnostics) error {
	return Retry.Do(ctx, op, func(attempt int) (retry bool, err error) {
		res, getErr := r.client.GetService(consortiumID, environmentID, apiModel.ID, apiModel)
		if getErr != nil {
			return false, getErr
		}

		statusCode := res.StatusCode()
		if statusCode != 200 {
			return false, fmt.Errorf("Fetching service %s state failed: %d", apiModel.ID, statusCode)
		}

		if apiModel.State != "started" {
			msg := "Service %s in environment %s in consortium %s" +
				"took too long to enter state 'started'. Final state was '%s'."
			return true, fmt.Errorf(msg, apiModel.ID, environmentID, consortiumID, apiModel.State)
		}
		r.copyServiceData(ctx, apiModel, data, diagnostics)
		return false, nil
	})
}

func (r *resourceService) copyServiceData(ctx context.Context, apiModel *kaleido.Service, data *ServiceResourceModel, diagnostics diag.Diagnostics) {
	data.ID = types.StringValue(apiModel.ID)
	mapValue, diag := types.MapValueFrom(ctx, types.StringType, data.URLs)
	diagnostics.Append(diag...)
	data.URLs = mapValue
	if httpURL, ok := apiModel.Urls["http"]; ok {
		data.HttpsURL = types.StringValue(httpURL.(string))
	}
	if wsURL, ok := apiModel.Urls["ws"]; ok {
		data.WebSocketURL = types.StringValue(wsURL.(string))
	}
	if webuiURL, ok := apiModel.Urls["webui"]; ok {
		data.WebSocketURL = types.StringValue(webuiURL.(string))
	}
	data.HybridPortAllocation = types.Int64Value(apiModel.HybridPortAllocation)
}

func (r *resourceService) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var data ServiceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	apiModel := kaleido.Service{}
	consortiumID := data.ConsortiumID.ValueString()
	environmentID := data.EnvironmentID.ValueString()
	apiModel.MembershipID = data.MembershipID.ValueString()
	apiModel.Service = data.ServiceType.ValueString()
	apiModel.ZoneID = data.ZoneID.ValueString()
	apiModel.Size = data.Size.ValueString()

	apiModel.Details = make(map[string]interface{})
	resp.Diagnostics.Append(data.Details.ElementsAs(ctx, &apiModel.Details, true)...)

	sharedExisting := false
	if data.SharedDeployment.ValueBool() {
		var existing []kaleido.Service
		res, err := r.client.ListServices(consortiumID, environmentID, &existing)
		if err != nil {
			resp.Diagnostics.AddError("failed to list services", err.Error())
			return
		}
		if res.StatusCode() != 200 {
			resp.Diagnostics.AddError("failed to list services", fmt.Sprintf("Failed to list existing services with status %d: %s", res.StatusCode(), res.String()))
			return
		}
		for _, e := range existing {
			if data.SharedDeployment.ValueBool() && e.Service == apiModel.Service && !strings.Contains(e.State, "delete") {
				if e.ServiceType != "utility" {
					resp.Diagnostics.AddError("shared_deployment not valid", fmt.Sprintf("The shared_deployment option only applies to utility services. %s service %s is a '%s' service", apiModel.Service, apiModel.ID, apiModel.ServiceType))
				}
				// Already exists, just re-use
				sharedExisting = true
				apiModel = e
			}
		}
	}
	if !sharedExisting {
		res, err := r.client.CreateService(consortiumID, environmentID, &apiModel)
		if err != nil {
			resp.Diagnostics.AddError("failed to create service", err.Error())
			return
		}

		status := res.StatusCode()
		if status != 201 {
			msg := "Could not create service %s in consortium %s in environment %s with status %d: %s"
			resp.Diagnostics.AddError("failed to create service", fmt.Sprintf(msg, apiModel.ID, consortiumID, environmentID, status, res.String()))
			return
		}

	}

	err := r.waitUntilServiceStarted(ctx, "Create", consortiumID, environmentID, apiModel.ID, &apiModel, &data, resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError("failed to query service status", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *resourceService) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ServiceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	apiModel := kaleido.Service{}
	consortiumID := data.ConsortiumID.ValueString()
	environmentID := data.EnvironmentID.ValueString()
	apiModel.Size = data.Size.ValueString()
	serviceID := data.ID.ValueString()

	apiModel.Details = make(map[string]interface{})
	resp.Diagnostics.Append(data.Details.ElementsAs(ctx, &apiModel.Details, true)...)

	res, err := r.client.UpdateService(consortiumID, environmentID, serviceID, &apiModel)
	if err != nil {
		resp.Diagnostics.AddError("failed to update service", err.Error())
		return
	}

	status := res.StatusCode()
	if status != 200 {
		msg := "Could not update service %s in consortium %s in environment %s with status %d: %s"
		resp.Diagnostics.AddError("failed to update service", fmt.Sprintf(msg, serviceID, consortiumID, environmentID, status, res.String()))
		return
	}

	res, err = r.client.ResetService(consortiumID, environmentID, apiModel.ID)
	if err != nil {
		resp.Diagnostics.AddError("failed to reset service", err.Error())
		return
	}
	if status != 200 {
		msg := "Could not reset service %s in consortium %s in environment %s with status %d: %s"
		resp.Diagnostics.AddError("failed to update service", fmt.Sprintf(msg, serviceID, consortiumID, environmentID, status, res.String()))
		return
	}

	err = r.waitUntilServiceStarted(ctx, "Update", consortiumID, environmentID, serviceID, &apiModel, &data, resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError("failed to query service status", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *resourceService) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ServiceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	consortiumID := data.ConsortiumID.ValueString()
	environmentID := data.EnvironmentID.ValueString()
	serviceID := data.ID.ValueString()

	var apiModel kaleido.Service
	res, err := r.client.GetService(consortiumID, environmentID, serviceID, &apiModel)
	if err != nil {
		resp.Diagnostics.AddError("failed to query service", err.Error())
		return
	}

	status := res.StatusCode()
	if status == 404 {
		resp.State.RemoveResource(ctx)
		return
	}
	if status != 200 {
		msg := "Could not find service %s in consortium %s in environment %s with status %d: %s"
		resp.Diagnostics.AddError("failed to query service", fmt.Sprintf(msg, serviceID, consortiumID, environmentID, status, res.String()))
		return
	}

	r.copyServiceData(ctx, &apiModel, &data, resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *resourceService) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ServiceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if data.SharedDeployment.ValueBool() {
		// Cannot safely delete if this is shared with other terraform deployments
		// Pretend we deleted it
		return
	}

	consortiumID := data.ConsortiumID.ValueString()
	environmentID := data.EnvironmentID.ValueString()
	serviceID := data.ID.ValueString()

	res, err := r.client.DeleteService(consortiumID, environmentID, serviceID)
	if err != nil {
		resp.Diagnostics.AddError("failed to query service", err.Error())
		return
	}

	statusCode := res.StatusCode()
	if res.IsError() && statusCode != 404 {
		msg := "Failed to delete service %s in environment %s in consortium %s with status %d: %s"
		resp.Diagnostics.AddError("failed to query service", fmt.Sprintf(msg, serviceID, environmentID, consortiumID, statusCode, res.String()))
		return
	}
}
