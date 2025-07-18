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
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type KeyManagerServiceResourceModel struct {
	ID                     types.String `tfsdk:"id"`
	Environment            types.String `tfsdk:"environment"`
	EnvironmentMemberID    types.String `tfsdk:"environment_member_id"`
	Runtime                types.String `tfsdk:"runtime"`
	Name                   types.String `tfsdk:"name"`
	StackID                types.String `tfsdk:"stack_id"`
	FineGrainedPermissions types.Bool   `tfsdk:"fine_grained_permissions"`
	ForceDelete            types.Bool   `tfsdk:"force_delete"`
}

func KeyManagerServiceResourceFactory() resource.Resource {
	return &keyManagerServiceResource{}
}

type keyManagerServiceResource struct {
	commonResource
}

func (r *keyManagerServiceResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_keymanager_service"
}

func (r *keyManagerServiceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A Key Manager service that provides cryptographic key management capabilities for blockchain applications.",
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"environment": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Environment ID where the Key Manager service will be deployed",
			},
			"environment_member_id": &schema.StringAttribute{
				Computed: true,
			},
			"runtime": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Runtime ID where the Key Manager service will be deployed",
			},
			"name": &schema.StringAttribute{
				Required:    true,
				Description: "Display name for the Key Manager service",
			},
			"stack_id": &schema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Stack ID where the Key Manager service belongs (optional)",
			},
			"fine_grained_permissions": &schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether to enable fine-grained permissions for key operations",
			},
			"force_delete": &schema.BoolAttribute{
				Optional:    true,
				Description: "Set to true when you plan to delete a protected Key Manager service. You must apply this value before running terraform destroy.",
			},
		},
	}
}

func (data *KeyManagerServiceResourceModel) toServiceAPI(ctx context.Context, api *ServiceAPIModel, diagnostics *diag.Diagnostics) {
	api.Type = "KeyManagerService"
	api.Name = data.Name.ValueString()
	api.StackID = data.StackID.ValueString()
	api.Runtime.ID = data.Runtime.ValueString()
	api.Config = make(map[string]interface{})

	// Fine-grained permissions
	api.Config["fineGrainedPermissions"] = data.FineGrainedPermissions.ValueBool()
}

func (api *ServiceAPIModel) toKeyManagerServiceData(data *KeyManagerServiceResourceModel, diagnostics *diag.Diagnostics) {
	data.ID = types.StringValue(api.ID)
	data.EnvironmentMemberID = types.StringValue(api.EnvironmentMemberID)
	data.Runtime = types.StringValue(api.Runtime.ID)
	data.Name = types.StringValue(api.Name)
	data.StackID = types.StringValue(api.StackID)

	// Fine-grained permissions
	if v, ok := api.Config["fineGrainedPermissions"].(bool); ok {
		data.FineGrainedPermissions = types.BoolValue(v)
	} else {
		data.FineGrainedPermissions = types.BoolValue(false)
	}
}

func (r *keyManagerServiceResource) apiPath(data *KeyManagerServiceResourceModel) string {
	path := fmt.Sprintf("/api/v1/environments/%s/services", data.Environment.ValueString())
	if data.ID.ValueString() != "" {
		path = path + "/" + data.ID.ValueString()
	}
	if !data.ForceDelete.IsNull() && data.ForceDelete.ValueBool() {
		path = path + "?force=true"
	}
	return path
}

func (r *keyManagerServiceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data KeyManagerServiceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	var api ServiceAPIModel
	data.toServiceAPI(ctx, &api, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	ok, _ := r.apiRequest(ctx, http.MethodPost, r.apiPath(&data), api, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	api.toKeyManagerServiceData(&data, &resp.Diagnostics)
	r.waitForReadyStatus(ctx, r.apiPath(&data), &resp.Diagnostics)

	api.ID = data.ID.ValueString()
	ok, _ = r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	api.toKeyManagerServiceData(&data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *keyManagerServiceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data KeyManagerServiceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)

	var api ServiceAPIModel
	if ok, _ := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics); !ok {
		return
	}

	data.toServiceAPI(ctx, &api, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if ok, _ := r.apiRequest(ctx, http.MethodPut, r.apiPath(&data), api, &api, &resp.Diagnostics); !ok {
		return
	}

	api.toKeyManagerServiceData(&data, &resp.Diagnostics)
	r.waitForReadyStatus(ctx, r.apiPath(&data), &resp.Diagnostics)

	if ok, _ := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics); !ok {
		return
	}
	api.toKeyManagerServiceData(&data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *keyManagerServiceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data KeyManagerServiceResourceModel
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

	api.toKeyManagerServiceData(&data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *keyManagerServiceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data KeyManagerServiceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data), nil, nil, &resp.Diagnostics, Allow404())
	r.waitForRemoval(ctx, r.apiPath(&data), &resp.Diagnostics)
}
