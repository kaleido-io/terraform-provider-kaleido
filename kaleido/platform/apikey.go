// Copyright © Kaleido, Inc. 2025

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

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type APIKeyResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Name                types.String `tfsdk:"name"`
	ApplicationID       types.String `tfsdk:"application_id"`
	Secret              types.String `tfsdk:"secret"`
	NoExpiry            types.Bool   `tfsdk:"no_expiry"`
	ExpiryDate          types.String `tfsdk:"expiry_date"`
	FormattedExpiryDate types.String `tfsdk:"formatted_expiry_date"`
}

type APIKeyAPIModel struct {
	ID            string `json:"id,omitempty"`
	Name          string `json:"name"`
	ApplicationID string `json:"application,omitempty"`

	Secret     string     `json:"secret,omitempty"`
	Created    *time.Time `json:"created,omitempty"`
	NoExpiry   *bool      `json:"noExpiry,omitempty"`
	ExpiryDate string     `json:"expiryDate,omitempty"`
}

func APIKeyResourceFactory() resource.Resource {
	return &api_keyResource{}
}

type api_keyResource struct {
	commonResource
}

func (r *api_keyResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_api_key"
}

func (r *api_keyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "API keys are generated, strong static keys for authenticating to the platform as an application, with a configurable expiry.",
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "API Key Name",
			},
			"application_id": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "The ID of the application you wish to create the API key under. Note that the application's access, determines the capabilities of the API keys.",
			},
			"secret": &schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "API Key Value",
			},
			"expiry_date": &schema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Expiration date formatted in RFC3339, Unix, or UnixNano",
			},
			"formatted_expiry_date": &schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"no_expiry": &schema.BoolAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
				Description:   "Set to `true` for API keys that should never expire",
			},
		},
	}
}

func (data *APIKeyResourceModel) toAPI(api *APIKeyAPIModel) {
	api.Name = data.Name.ValueString()
	api.ApplicationID = data.ApplicationID.ValueString()
	if data.ExpiryDate.ValueString() != "" {
		api.ExpiryDate = data.ExpiryDate.ValueString()
	}
	if data.NoExpiry.ValueBoolPointer() != nil {
		api.NoExpiry = data.NoExpiry.ValueBoolPointer()
	}
}

func (api *APIKeyAPIModel) toData(data *APIKeyResourceModel) {
	data.ID = types.StringValue(api.ID)
	data.Name = types.StringValue(api.Name)
	data.ApplicationID = types.StringValue(api.ApplicationID)
	// Only set the secret value when the api key is first created
	if !types.StringValue(api.Secret).IsNull() {
		data.Secret = types.StringValue(api.Secret)
	}
	data.FormattedExpiryDate = types.StringValue(api.ExpiryDate)
	data.NoExpiry = types.BoolPointerValue(api.NoExpiry)
}

func (r *api_keyResource) apiPath(data *APIKeyResourceModel) string {
	path := fmt.Sprintf("/api/v1/applications/%s/api-keys", data.ApplicationID.ValueString())
	if data.ID.ValueString() != "" {
		path = path + "/" + data.ID.ValueString()
	}
	return path
}

func (r *api_keyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var data APIKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	var api APIKeyAPIModel
	data.toAPI(&api)
	ok, _ := r.apiRequest(ctx, http.MethodPost, r.apiPath(&data), api, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	api.toData(&data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)

}

func (r *api_keyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Update is not supported for API Keys - only creation and deletion
}

func (r *api_keyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data APIKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	var api APIKeyAPIModel
	api.ID = data.ID.ValueString()
	ok, status := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics, Allow404())
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

func (r *api_keyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data APIKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data), nil, nil, &resp.Diagnostics, Allow404())

	r.waitForRemoval(ctx, r.apiPath(&data), &resp.Diagnostics)
}
