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
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type AMSVariableSetResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Environment    types.String `tfsdk:"environment"`
	Service        types.String `tfsdk:"service"`
	Name           types.String `tfsdk:"name"`
	Classification types.String `tfsdk:"classification"`
	Description    types.String `tfsdk:"description"`
	VariablesJSON  types.String `tfsdk:"variables_json"`
}

type AMSVariableSetAPIModel struct {
	ID             string      `json:"id,omitempty"`
	Name           string      `json:"name,omitempty"`
	Created        *time.Time  `json:"created,omitempty"`
	Updated        *time.Time  `json:"updated,omitempty"`
	Classification string      `json:"classification,omitempty"`
	Description    string      `json:"description,omitempty"`
	Variables      interface{} `json:"variables,omitempty"`
}

func AMSVariableSetResourceFactory() resource.Resource {
	return &ams_variablesetResource{}
}

type ams_variablesetResource struct {
	commonResource
}

func (r *ams_variablesetResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_ams_variableset"
}

func (r *ams_variablesetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"environment": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"service": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": &schema.StringAttribute{
				Required: true,
			},
			"classification": &schema.StringAttribute{
				Computed: true,
				Optional: true,
			},
			"description": &schema.StringAttribute{
				Optional: true,
			},
			"variables_json": &schema.StringAttribute{
				Required: true,
			},
		},
	}
}

func (data *AMSVariableSetResourceModel) toAPI(api *AMSVariableSetAPIModel, diagnostics *diag.Diagnostics) bool {
	api.Name = data.Name.ValueString()
	api.Description = data.Description.ValueString()
	api.Classification = data.Classification.ValueString()
	err := json.Unmarshal([]byte(data.VariablesJSON.ValueString()), &api.Variables)
	if err != nil {
		diagnostics.AddError("failed to serialize variables JSON", err.Error())
		return false
	}
	return true
}

func (api *AMSVariableSetAPIModel) toData(data *AMSVariableSetResourceModel) {
	data.ID = types.StringValue(api.ID)
	data.Name = types.StringValue(api.Name)
	if api.Description != "" {
		data.Description = types.StringValue(api.Description)
	}
	data.Classification = types.StringValue(api.Classification)
}

func (r *ams_variablesetResource) apiPath(data *AMSVariableSetResourceModel, idOrName string) string {
	return fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/variable-sets/%s", data.Environment.ValueString(), data.Service.ValueString(), idOrName)
}

func (r *ams_variablesetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var data AMSVariableSetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	var api AMSVariableSetAPIModel
	ok := data.toAPI(&api, &resp.Diagnostics)
	if ok {
		ok, _ = r.apiRequest(ctx, http.MethodPut, r.apiPath(&data, data.Name.ValueString()), &api, &api, &resp.Diagnostics)
	}
	if !ok {
		return
	}

	api.toData(&data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)

}

func (r *ams_variablesetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

	var data AMSVariableSetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)

	var api AMSVariableSetAPIModel
	ok := data.toAPI(&api, &resp.Diagnostics)
	if ok {
		ok, _ = r.apiRequest(ctx, http.MethodPut, r.apiPath(&data, data.ID.ValueString()), &api, &api, &resp.Diagnostics)
	}
	if !ok {
		return
	}

	api.toData(&data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *ams_variablesetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AMSVariableSetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	var api AMSVariableSetAPIModel
	api.ID = data.ID.ValueString()
	ok, status := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data, data.ID.ValueString()), nil, &api, &resp.Diagnostics, Allow404())
	if !ok {
		return
	}
	if status == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	api.toData(&data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *ams_variablesetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AMSVariableSetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data, data.ID.ValueString()), nil, nil, &resp.Diagnostics, Allow404())

	r.waitForRemoval(ctx, r.apiPath(&data, data.ID.ValueString()), &resp.Diagnostics)
}
