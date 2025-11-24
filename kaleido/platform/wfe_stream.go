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

type WFEStreamResourceModel struct {
	ID                      types.String `tfsdk:"id"`
	Name                    types.String `tfsdk:"name"`
	Description             types.String `tfsdk:"description"`
	Environment             types.String `tfsdk:"environment"`
	Service                 types.String `tfsdk:"service"`
	UniquenessPrefix        types.String `tfsdk:"uniqueness_prefix"`
	ListenerHandler         types.String `tfsdk:"listener_handler"`
	ListenerHandlerProvider types.String `tfsdk:"listener_handler_provider"`
	TransactionTemplateJSON types.String `tfsdk:"transaction_template_json"` // valid when type=transaction_dispatch
	PostFilterJSON          types.String `tfsdk:"post_filter_json"`
	Type                    types.String `tfsdk:"type"`
	Config                  types.String `tfsdk:"config"` // JSON string containing stream configuration
	Created                 types.String `tfsdk:"created"`
	Updated                 types.String `tfsdk:"updated"`
}

type WFEStreamAPIModel struct {
	ID                      string                 `json:"id,omitempty"`
	Name                    string                 `json:"name,omitempty"`
	Description             string                 `json:"description,omitempty"`
	UniquenessPrefix        string                 `json:"uniquenessPrefix,omitempty"`
	ListenerHandler         string                 `json:"listenerHandler,omitempty"`
	ListenerHandlerProvider string                 `json:"listenerHandlerProvider,omitempty"`
	Type                    string                 `json:"type,omitempty"`
	TransactionTemplate     map[string]interface{} `json:"transactionTemplate,omitempty"` // valid when type=transaction_dispatch
	PostFilter              map[string]interface{} `json:"postFilter,omitempty"`
	Config                  map[string]interface{} `json:"config,omitempty"`
	Created                 *time.Time             `json:"created,omitempty"`
	Updated                 *time.Time             `json:"updated,omitempty"`
}

func WFEStreamResourceFactory() resource.Resource {
	return &wfe_streamResource{}
}

type wfe_streamResource struct {
	commonResource
}

func (r *wfe_streamResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_wfe_stream"
}

func (r *wfe_streamResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The Workflow Engine Stream resource allows you to manage streams in the Workflow Engine.",
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				Description:   "Unique ID of the stream",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"environment": &schema.StringAttribute{
				Required:      true,
				Description:   "The environment ID",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"service": &schema.StringAttribute{
				Required:      true,
				Description:   "The service ID",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": &schema.StringAttribute{
				Required:      true,
				Description:   "The name of the stream",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"uniqueness_prefix": &schema.StringAttribute{
				Optional:      true,
				Description:   "The uniqueness prefix for the stream",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"listener_handler": &schema.StringAttribute{
				Required:      true,
				Description:   "The listener handler for the stream",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"listener_handler_provider": &schema.StringAttribute{
				Required:      true,
				Description:   "The listener handler provider for the stream",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"type": &schema.StringAttribute{
				Required:      true,
				Description:   "The type of the stream",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"transaction_template_json": &schema.StringAttribute{
				Optional:      true,
				Description:   "The transaction template for the stream as JSON.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"post_filter_json": &schema.StringAttribute{
				Optional:      true,
				Description:   "The post filter for the stream as JSON.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"description": &schema.StringAttribute{
				Optional:    true,
				Description: "A description of the stream",
			},
			"config": &schema.StringAttribute{
				Required:    true,
				Description: "The stream configuration as JSON",
			},
			"created": &schema.StringAttribute{
				Computed:    true,
				Description: "Creation timestamp",
			},
			"updated": &schema.StringAttribute{
				Computed:    true,
				Description: "Last update timestamp",
			},
		},
	}
}

func (r *wfe_streamResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.commonResource.Configure(ctx, req, resp)
}

func (r *wfe_streamResource) apiPath(data *WFEStreamResourceModel, idOrName string) string {
	env := data.Environment.ValueString()
	service := data.Service.ValueString()
	return fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/streams/%s", env, service, idOrName)
}

func (r *wfe_streamResource) toAPI(data *WFEStreamResourceModel, api *WFEStreamAPIModel, diagnostics *diag.Diagnostics) {
	api.Name = data.Name.ValueString()
	api.Description = data.Description.ValueString()
	api.UniquenessPrefix = data.UniquenessPrefix.ValueString()
	api.ListenerHandler = data.ListenerHandler.ValueString()
	api.ListenerHandlerProvider = data.ListenerHandlerProvider.ValueString()
	api.Type = data.Type.ValueString()

	if data.PostFilterJSON.ValueString() != "" {
		// Parse the PostFilter JSON string
		var postFilter map[string]interface{}
		if err := json.Unmarshal([]byte(data.PostFilterJSON.ValueString()), &postFilter); err != nil {
			diagnostics.AddError("Invalid JSON", fmt.Sprintf("Failed to parse post filter JSON: %v", err))
			return
		}
		api.PostFilter = postFilter
	}

	if data.TransactionTemplateJSON.ValueString() != "" {
		// Parse the TransactionTemplate JSON string
		var transactionTemplate map[string]interface{}
		if err := json.Unmarshal([]byte(data.TransactionTemplateJSON.ValueString()), &transactionTemplate); err != nil {
			diagnostics.AddError("Invalid JSON", fmt.Sprintf("Failed to parse stream transaction template JSON: %v", err))
			return
		}
		api.TransactionTemplate = transactionTemplate
	}

	// Parse the config JSON string
	var config map[string]interface{}
	if err := json.Unmarshal([]byte(data.Config.ValueString()), &config); err != nil {
		diagnostics.AddError("Invalid JSON", fmt.Sprintf("Failed to parse stream config JSON: %v", err))
		return
	}
	api.Config = config
}

func (r *wfe_streamResource) toData(api *WFEStreamAPIModel, data *WFEStreamResourceModel, diagnostics *diag.Diagnostics) {
	data.ID = types.StringValue(api.ID)
	data.Name = types.StringValue(api.Name)

	// Note: environment and service are not returned by the API, they remain as set in the resource

	if api.Description == "" {
		data.Description = types.StringNull()
	} else {
		data.Description = types.StringValue(api.Description)
	}

	if api.UniquenessPrefix == "" {
		data.UniquenessPrefix = types.StringNull()
	} else {
		data.UniquenessPrefix = types.StringValue(api.UniquenessPrefix)
	}

	if api.ListenerHandler == "" {
		data.ListenerHandler = types.StringNull()
	} else {
		data.ListenerHandler = types.StringValue(api.ListenerHandler)
	}

	if api.ListenerHandlerProvider == "" {
		data.ListenerHandlerProvider = types.StringNull()
	} else {
		data.ListenerHandlerProvider = types.StringValue(api.ListenerHandlerProvider)
	}

	if api.Type == "" {
		data.Type = types.StringNull()
	} else {
		data.Type = types.StringValue(api.Type)
	}

	if api.TransactionTemplate != nil {
		transactionTemplateBytes, err := json.Marshal(api.TransactionTemplate)
		if err != nil {
			diagnostics.AddError("JSON Marshal Error", fmt.Sprintf("Failed to marshal stream transaction template: %v", err))
			return
		}
		data.TransactionTemplateJSON = types.StringValue(string(transactionTemplateBytes))
	} else {
		data.TransactionTemplateJSON = types.StringNull()
	}

	if api.PostFilter != nil {
		postFilterBytes, err := json.Marshal(api.PostFilter)
		if err != nil {
			diagnostics.AddError("JSON Marshal Error", fmt.Sprintf("Failed to marshal stream post filter: %v", err))
			return
		}
		data.PostFilterJSON = types.StringValue(string(postFilterBytes))
	} else {
		data.PostFilterJSON = types.StringNull()
	}

	// Handle config if present in API response
	if api.Config != nil {
		configBytes, err := json.Marshal(api.Config)
		if err != nil {
			diagnostics.AddError("JSON Marshal Error", fmt.Sprintf("Failed to marshal stream config: %v", err))
			return
		}
		data.Config = types.StringValue(string(configBytes))
	}

	if api.Created == nil {
		data.Created = types.StringNull()
	} else {
		data.Created = types.StringValue(api.Created.Format(time.RFC3339))
	}
	if api.Updated == nil {
		data.Updated = types.StringNull()
	} else {
		data.Updated = types.StringValue(api.Updated.Format(time.RFC3339))
	}
}

func (r *wfe_streamResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data WFEStreamResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var api WFEStreamAPIModel
	r.toAPI(&data, &api, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create the stream
	ok, _ := r.apiRequest(ctx, http.MethodPut, r.apiPath(&data, api.Name), &api, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	r.toData(&api, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *wfe_streamResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data WFEStreamResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var api WFEStreamAPIModel
	ok, status := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data, data.ID.ValueString()), nil, &api, &resp.Diagnostics, Allow404())
	if !ok {
		return
	}
	if status == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	r.toData(&api, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *wfe_streamResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data WFEStreamResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var api WFEStreamAPIModel
	r.toAPI(&data, &api, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	streamID := data.ID.ValueString()

	// Update the stream
	ok, _ := r.apiRequest(ctx, http.MethodPatch, r.apiPath(&data, streamID), &api, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	r.toData(&api, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *wfe_streamResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data WFEStreamResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data, data.ID.ValueString()), nil, nil, &resp.Diagnostics, Allow404())
	r.waitForRemoval(ctx, r.apiPath(&data, data.ID.ValueString()), &resp.Diagnostics)
}
