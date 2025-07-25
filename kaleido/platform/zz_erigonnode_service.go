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

type ErigonNodeServiceResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Environment         types.String `tfsdk:"environment"`
	EnvironmentMemberID types.String `tfsdk:"environment_member_id"`
	Runtime             types.String `tfsdk:"runtime"`
	Name                types.String `tfsdk:"name"`
	StackID             types.String `tfsdk:"stack_id"`
	Apisenabled types.String `tfsdk:"apisenabled"`
	Blobarchive types.Bool `tfsdk:"blobarchive"`
	Loglevel    types.String `tfsdk:"loglevel"`
	Network     types.String `tfsdk:"network"`
	Nodekey     types.String `tfsdk:"nodekey"`
	ForceDelete         types.Bool   `tfsdk:"force_delete"`
}

func ErigonNodeServiceResourceFactory() resource.Resource {
	return &erigonnodeserviceResource{}
}

type erigonnodeserviceResource struct {
	commonResource
}

func (r *erigonnodeserviceResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_erigonnode"
}

func (r *erigonnodeserviceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A ErigonNode service that provides blockchain functionality.",
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"environment": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Environment ID where the ErigonNode service will be deployed",
			},
			"environment_member_id": &schema.StringAttribute{
				Computed: true,
			},
			"runtime": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Runtime ID where the ErigonNode service will be deployed",
			},
			"name": &schema.StringAttribute{
				Required:    true,
				Description: "Display name for the ErigonNode service",
			},
			"stack_id": &schema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Stack ID where the ErigonNode service belongs (optional)",
			},
			"apisenabled": &schema.StringAttribute{
				Optional:    true,
				Description: "",
			},
			"blobarchive": &schema.BoolAttribute{
				Optional:    true,
				Description: "",
			},
			"loglevel": &schema.StringAttribute{
				Optional:    true,
				Description: "",
			},
			"network": &schema.StringAttribute{
				Optional:    true,
				Description: "ethereumPOSNetwork",
			},
			"nodekey": &schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "",
			},
			"force_delete": &schema.BoolAttribute{
				Optional:    true,
				Description: "Set to true when you plan to delete a protected ErigonNode service. You must apply this value before running terraform destroy.",
			},
		},
	}
}

func (data *ErigonNodeServiceResourceModel) toErigonNodeServiceAPI(ctx context.Context, api *ServiceAPIModel, diagnostics *diag.Diagnostics) {
	api.Type = "ErigonNodeService"
	api.Name = data.Name.ValueString()
	api.StackID = data.StackID.ValueString()
	api.Runtime.ID = data.Runtime.ValueString()
	api.Config = make(map[string]interface{})

	api.Config["blobArchive"] = data.Blobarchive.ValueBool()
	// Handle Apisenabled as JSON
	if !data.Apisenabled.IsNull() && data.Apisenabled.ValueString() != "" {
		var apisenabledData interface{}
		err := json.Unmarshal([]byte(data.Apisenabled.ValueString()), &apisenabledData)
		if err != nil {
			diagnostics.AddAttributeError(
				path.Root("apisenabled"),
				"Failed to parse Apisenabled",
				err.Error(),
			)
		} else {
			api.Config["apisEnabled"] = apisenabledData
		}
	}
	if !data.Network.IsNull() && data.Network.ValueString() != "" {
		api.Config["network"] = data.Network.ValueString()
	}
	if !data.Loglevel.IsNull() && data.Loglevel.ValueString() != "" {
		api.Config["logLevel"] = data.Loglevel.ValueString()
	}
	// Handle Nodekey credentials
	if !data.Nodekey.IsNull() && data.Nodekey.ValueString() != "" {
		if api.Credsets == nil {
			api.Credsets = make(map[string]*CredSetAPI)
		}
		api.Credsets["nodeKey"] = &CredSetAPI{
			Name: "nodeKey",
			Type: "key",
			Key: &CredSetKeyAPI{
				Value: data.Nodekey.ValueString(),
			},
		}
		api.Config["nodeKey"] = map[string]interface{}{
			"credSetRef": "nodeKey",
		}
	}
}

func (api *ServiceAPIModel) toErigonNodeServiceData(data *ErigonNodeServiceResourceModel, diagnostics *diag.Diagnostics) {
	data.ID = types.StringValue(api.ID)
	data.EnvironmentMemberID = types.StringValue(api.EnvironmentMemberID)
	data.Runtime = types.StringValue(api.Runtime.ID)
	data.Name = types.StringValue(api.Name)
	data.StackID = types.StringValue(api.StackID)

	if apisenabledData := api.Config["apisEnabled"]; apisenabledData != nil {
		if apisenabledJSON, err := json.Marshal(apisenabledData); err == nil {
			data.Apisenabled = types.StringValue(string(apisenabledJSON))
		} else {
			data.Apisenabled = types.StringNull()
		}
	} else {
		data.Apisenabled = types.StringNull()
	}
	if v, ok := api.Config["network"].(string); ok {
		data.Network = types.StringValue(v)
	} else {
		data.Network = types.StringNull()
	}
	if v, ok := api.Config["logLevel"].(string); ok {
		data.Loglevel = types.StringValue(v)
	} else {
		data.Loglevel = types.StringNull()
	}
	if credset, ok := api.Credsets["nodeKey"]; ok && credset.Key != nil {
		data.Nodekey = types.StringValue(credset.Key.Value)
	} else {
		data.Nodekey = types.StringNull()
	}
	if v, ok := api.Config["blobArchive"].(bool); ok {
		data.Blobarchive = types.BoolValue(v)
	} else {
		data.Blobarchive = types.BoolValue(false)
	}
}

func (r *erigonnodeserviceResource) apiPath(data *ErigonNodeServiceResourceModel) string {
	path := fmt.Sprintf("/api/v1/environments/%s/services", data.Environment.ValueString())
	if data.ID.ValueString() != "" {
		path = path + "/" + data.ID.ValueString()
	}
	if !data.ForceDelete.IsNull() && data.ForceDelete.ValueBool() {
		path = path + "?force=true"
	}
	return path
}

func (r *erigonnodeserviceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ErigonNodeServiceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	var api ServiceAPIModel
	data.toErigonNodeServiceAPI(ctx, &api, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	ok, _ := r.apiRequest(ctx, http.MethodPost, r.apiPath(&data), api, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	api.toErigonNodeServiceData(&data, &resp.Diagnostics)
	r.waitForReadyStatus(ctx, r.apiPath(&data), &resp.Diagnostics)

	api.ID = data.ID.ValueString()
	ok, _ = r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	api.toErigonNodeServiceData(&data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *erigonnodeserviceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ErigonNodeServiceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)

	var api ServiceAPIModel
	if ok, _ := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics); !ok {
		return
	}

	data.toErigonNodeServiceAPI(ctx, &api, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if ok, _ := r.apiRequest(ctx, http.MethodPut, r.apiPath(&data), api, &api, &resp.Diagnostics); !ok {
		return
	}

	api.toErigonNodeServiceData(&data, &resp.Diagnostics)
	r.waitForReadyStatus(ctx, r.apiPath(&data), &resp.Diagnostics)

	if ok, _ := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics); !ok {
		return
	}
	api.toErigonNodeServiceData(&data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *erigonnodeserviceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ErigonNodeServiceResourceModel
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

	api.toErigonNodeServiceData(&data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *erigonnodeserviceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ErigonNodeServiceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data), nil, nil, &resp.Diagnostics, Allow404())
	r.waitForRemoval(ctx, r.apiPath(&data), &resp.Diagnostics)
}
