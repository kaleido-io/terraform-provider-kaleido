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

type AMSCollectionResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Environment types.String `tfsdk:"environment"`
	Service     types.String `tfsdk:"service"`
	Name        types.String `tfsdk:"name"`
	DisplayName types.String `tfsdk:"display_name"`
	Description types.String `tfsdk:"description"`
	InfoJSON    types.String `tfsdk:"info_json"`
	LabelsJSON  types.String `tfsdk:"labels_json"`
}

type AMSCollectionAPIModel struct {
	ID          string      `json:"id,omitempty"`
	Name        string      `json:"name,omitempty"`
	DisplayName string      `json:"displayName,omitempty"`
	Description string      `json:"description,omitempty"`
	Created     *time.Time  `json:"created,omitempty"`
	Updated     *time.Time  `json:"updated,omitempty"`
	Info        interface{} `json:"info,omitempty"`
	Labels      interface{} `json:"labels,omitempty"`
}

func AMSCollectionResourceFactory() resource.Resource {
	return &ams_collectionResource{}
}

type ams_collectionResource struct {
	commonResource
}

func (r *ams_collectionResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_ams_collection"
}

func (r *ams_collectionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"display_name": &schema.StringAttribute{
				Optional: true,
			},
			"description": &schema.StringAttribute{
				Optional: true,
			},
			"info_json": &schema.StringAttribute{
				Optional: true,
			},
			"labels_json": &schema.StringAttribute{
				Optional: true,
			},
		},
	}
}

func (data *AMSCollectionResourceModel) toAPI(api *AMSCollectionAPIModel, diagnostics *diag.Diagnostics) bool {
	api.Name = data.Name.ValueString()
	api.Description = data.Description.ValueString()
	api.DisplayName = data.DisplayName.ValueString()
	if !data.InfoJSON.IsNull() {
		err := json.Unmarshal([]byte(data.InfoJSON.ValueString()), &api.Info)
		if err != nil {
			diagnostics.AddError("failed to serialize info JSON", err.Error())
			return false
		}
	}
	if !data.LabelsJSON.IsNull() {
		err := json.Unmarshal([]byte(data.LabelsJSON.ValueString()), &api.Labels)
		if err != nil {
			diagnostics.AddError("failed to serialize labels JSON", err.Error())
			return false
		}
	}
	return true
}

func (api *AMSCollectionAPIModel) toData(data *AMSCollectionResourceModel) {
	data.ID = types.StringValue(api.ID)
	data.Name = types.StringValue(api.Name)
	if api.Description != "" {
		data.Description = types.StringValue(api.Description)
	}
	data.DisplayName = types.StringValue(api.DisplayName)
}

func (r *ams_collectionResource) apiPath(data *AMSCollectionResourceModel, idOrName string) string {
	return fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/collections/%s", data.Environment.ValueString(), data.Service.ValueString(), idOrName)
}

func (r *ams_collectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var data AMSCollectionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	var api AMSCollectionAPIModel
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

func (r *ams_collectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

	var data AMSCollectionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)

	var api AMSCollectionAPIModel
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

func (r *ams_collectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AMSCollectionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	var api AMSCollectionAPIModel
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

func (r *ams_collectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AMSCollectionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data, data.ID.ValueString()), nil, nil, &resp.Diagnostics, Allow404())

	r.waitForRemoval(ctx, r.apiPath(&data, data.ID.ValueString()), &resp.Diagnostics)
}
