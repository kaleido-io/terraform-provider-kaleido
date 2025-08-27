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
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type HostnameResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Environment types.String `tfsdk:"environment"`
	Service     types.String `tfsdk:"service"`
	Name        types.String `tfsdk:"name"`
	Hostname    types.String `tfsdk:"hostname"`
	Endpoints   types.List   `tfsdk:"endpoints"`
	MTLS        types.Bool   `tfsdk:"mtls"`
}

type HostnameAPIModel struct {
	ID        string     `json:"id,omitempty"`
	Created   *time.Time `json:"created,omitempty"`
	Updated   *time.Time `json:"updated,omitempty"`
	Endpoints []string   `json:"endpoints,omitempty"`
	MTLS      bool       `json:"mtls,omitempty"`
	Hostname  string     `json:"hostname,omitempty"`
}

type HostnameStatusDetails struct {
	Connectivity *Connectivity `json:"connectivity,omitempty"`
}

func HostnameResourceFactory() resource.Resource {
	return &hostnameResource{}
}

type hostnameResource struct {
	commonResource
}

func (r *hostnameResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_hostname"
}

func (r *hostnameResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Hostnames are used to create custom ingress routes to your Kaleido services.",
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"environment": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Environment ID",
			},
			"service": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Service ID",
			},
			"name": &schema.StringAttribute{
				Required:    true,
				Description: "Service Display Name",
			},
			"hostname": &schema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"endpoints": &schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
			},
			"mtls": &schema.BoolAttribute{
				Optional: true,
			},
		},
	}
}

func (data *HostnameResourceModel) toAPI(_ context.Context, api *HostnameAPIModel, diagnostics *diag.Diagnostics) {
	// required fields
	api.Hostname = data.Hostname.ValueString()
	api.MTLS = data.MTLS.ValueBool()
	endpoints := make([]string, len(data.Endpoints.Elements()))
	for i, e := range data.Endpoints.Elements() {
		endpoints[i] = e.(types.String).ValueString()
	}
	api.Endpoints = endpoints
}

func (api *HostnameAPIModel) toData(data *HostnameResourceModel, diagnostics *diag.Diagnostics) {
	data.ID = types.StringValue(api.ID)
	data.Hostname = types.StringValue(api.Hostname)
	data.MTLS = types.BoolValue(api.MTLS)
	endpoints := make([]attr.Value, len(api.Endpoints))
	for i, e := range api.Endpoints {
		endpoints[i] = types.StringValue(e)
	}
	tfEndpoints, d := types.ListValue(types.StringType, endpoints)
	diagnostics.Append(d...)
	data.Endpoints = tfEndpoints
}

func (r *hostnameResource) apiPath(data *HostnameResourceModel) string {
	path := fmt.Sprintf("/api/v1/environments/%s/services/%s/hostnames", data.Environment.ValueString(), data.Service.ValueString())
	if data.ID.ValueString() != "" {
		path = path + "/" + data.ID.ValueString()
	}

	return path
}

func (r *hostnameResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var data HostnameResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	var api HostnameAPIModel
	data.toAPI(ctx, &api, &resp.Diagnostics)
	ok, _ := r.apiRequest(ctx, http.MethodPost, r.apiPath(&data), api, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	api.toData(&data, &resp.Diagnostics) // need the ID copied over

	//re-read from api
	api.ID = data.ID.ValueString()
	ok, _ = r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics, &APIRequestOption{})
	if !ok {
		return
	}

	api.toData(&data, &resp.Diagnostics) // need the latest status after the readiness check completes, to extract generated values
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *hostnameResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

	var data HostnameResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)

	// Read full current object
	var api HostnameAPIModel
	if ok, _ := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics); !ok {
		return
	}

	// Update from plan
	data.toAPI(ctx, &api, &resp.Diagnostics)
	if ok, _ := r.apiRequest(ctx, http.MethodPut, r.apiPath(&data), api, &api, &resp.Diagnostics); !ok {
		return
	}

	api.toData(&data, &resp.Diagnostics) // need the ID copied over
	r.waitForReadyStatus(ctx, r.apiPath(&data), &resp.Diagnostics)
	//re-read from api
	if ok, _ := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics); !ok {
		return
	}
	api.toData(&data, &resp.Diagnostics) // need the latest status after the readiness check completes, to extract generated values
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *hostnameResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data HostnameResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	var api HostnameAPIModel
	api.ID = data.ID.ValueString()
	ok, status := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics, Allow404())
	if !ok {
		return
	}
	if status == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	api.toData(&data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *hostnameResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data HostnameResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data), nil, nil, &resp.Diagnostics, Allow404())

	r.waitForRemoval(ctx, r.apiPath(&data), &resp.Diagnostics)
}
