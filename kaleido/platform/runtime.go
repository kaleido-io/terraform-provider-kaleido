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

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type RuntimeResourceModel struct {
	CommonResourceModel
	Type                types.String `tfsdk:"type"`
	Name                types.String `tfsdk:"name"`
	ConfigJSON          types.String `tfsdk:"config_json"`
	LogLevel            types.String `tfsdk:"log_level"`
	Size                types.String `tfsdk:"size"`
	EnvironmentMemberID types.String `tfsdk:"environment_member_id"`
	Status              types.String `tfsdk:"status"`
	Deleted             types.String `tfsdk:"deleted"`
	ExplicitStopped     types.String `tfsdk:"stopped"`
}

type RuntimeAPIModel struct {
	CommonAPIModel
	Type                string                 `json:"type"`
	Name                string                 `json:"name"`
	Config              map[string]interface{} `json:"config"`
	LogLevel            string                 `json:"loglevel"`
	Size                string                 `json:"size"`
	EnvironmentMemberID string                 `json:"environmentMemberId"`
	Status              string                 `json:"status"`
	Deleted             bool                   `json:"deleted"`
	ExplicitStopped     bool                   `json:"stopped"`
}

func RuntimeResourceFactory() resource.Resource {
	return &runtimeResource{}
}

type runtimeResource struct {
	commonResource
}

func (r *runtimeResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_runtime"
}

func (r *runtimeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: withCommon(map[string]schema.Attribute{
			"type": &schema.StringAttribute{
				Required: true,
			},
			"environment_member_id": &schema.StringAttribute{
				Required: true,
			},
			"name": &schema.StringAttribute{
				Required: true,
			},
			"config_json": &schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"log_level": &schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"size": &schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
		}),
	}
}

func (data *RuntimeResourceModel) toAPI() *RuntimeAPIModel {
	var config map[string]interface{}
	if !data.ConfigJSON.IsNull() {
		_ = json.Unmarshal([]byte(data.ConfigJSON.ValueString()), &config)
	}
	return &RuntimeAPIModel{
		CommonAPIModel:      data.CommonResourceModel.toAPI(),
		Type:                data.Type.ValueString(),
		EnvironmentMemberID: data.EnvironmentMemberID.ValueString(),
		Name:                data.Name.ValueString(),
		Config:              config,
		LogLevel:            data.LogLevel.ValueString(),
		Size:                data.Size.ValueString(),
	}
}

func (api *RuntimeAPIModel) toData() *RuntimeResourceModel {
	var config string
	if api.Config != nil {
		d, err := json.Marshal(api.Config)
		if err == nil {
			config = string(d)
		}
	}
	return &RuntimeResourceModel{
		CommonResourceModel: api.CommonAPIModel.toData(),
		Type:                types.StringValue(api.Type),
		EnvironmentMemberID: types.StringValue(api.EnvironmentMemberID),
		Name:                types.StringValue(api.Name),
		ConfigJSON:          types.StringValue(config),
		LogLevel:            types.StringValue(api.LogLevel),
		Size:                types.StringValue(api.Size),
	}
}

func (r *runtimeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var data RuntimeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	api := data.toAPI()
	ok, _ := r.apiRequest(ctx, http.MethodPost, "/api/v1/runtimes", api, &api, resp.Diagnostics)
	if !ok {
		return
	}

	r.waitForReadyStatus(ctx, fmt.Sprintf("/api/v1/runtimes/%s", api.ID), resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, api.toData())...)

}

func (r *runtimeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

	var data RuntimeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	api := data.toAPI()
	ok, _ := r.apiRequest(ctx, http.MethodPut, fmt.Sprintf("/api/v1/runtimes/%s", api.ID), api, &api, resp.Diagnostics)
	if !ok {
		return
	}

	r.waitForReadyStatus(ctx, fmt.Sprintf("/api/v1/runtimes/%s", api.ID), resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, api.toData())...)

}

func (r *runtimeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RuntimeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	api := data.toAPI()
	ok, status := r.apiRequest(ctx, http.MethodPut, fmt.Sprintf("/api/v1/runtimes/%s", api.ID), nil, &api, resp.Diagnostics, Allow404)
	if !ok {
		return
	}
	if status == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, api.toData())...)
}

func (r *runtimeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data RuntimeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	_, _ = r.apiRequest(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/runtimes/%s", data.ID.ValueString()), nil, nil, resp.Diagnostics, Allow404)

}
