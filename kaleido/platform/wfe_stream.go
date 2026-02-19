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
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	Description        types.String `tfsdk:"description"`
	Environment        types.String `tfsdk:"environment"`
	Service            types.String `tfsdk:"service"`
	UniquenessPrefix   types.String `tfsdk:"uniqueness_prefix"`
	PostFilterJSON     types.String `tfsdk:"post_filter_json"`
	EventSourceType    types.String `tfsdk:"event_source_type"`
	EventSourceJSON    types.String `tfsdk:"event_source_json"`
	EventProcessorType types.String `tfsdk:"event_processor_type"`
	EventProcessorJSON types.String `tfsdk:"event_processor_json"`
	Started            types.Bool   `tfsdk:"started"`
	Created            types.String `tfsdk:"created"`
	Updated            types.String `tfsdk:"updated"`
}

type WFEStreamAPIModel struct {
	ID                 string                 `json:"id,omitempty"`
	Name               string                 `json:"name,omitempty"`
	Description        string                 `json:"description,omitempty"`
	UniquenessPrefix   string                 `json:"uniquenessPrefix,omitempty"`
	PostFilter         map[string]interface{} `json:"postFilter,omitempty"`
	EventSourceType    string                 `json:"eventSourceType,omitempty"`
	EventSource        map[string]interface{} `json:"eventSource,omitempty"`
	EventProcessorType string                 `json:"eventProcessorType,omitempty"`
	EventProcessor     map[string]interface{} `json:"eventProcessor,omitempty"`
	Started            *bool                  `json:"started,omitempty"`
	Created            *time.Time             `json:"created,omitempty"`
	Updated            *time.Time             `json:"updated,omitempty"`
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
				Optional:    true,
				Description: "The uniqueness prefix for the stream",
			},
			"post_filter_json": &schema.StringAttribute{
				Optional:    true,
				Description: "The post filter for the stream as JSON.",
			},
			"event_source_type": &schema.StringAttribute{
				Required:      true,
				Description:   "The event source type for the stream (for example: handler, transaction).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"event_source_json": &schema.StringAttribute{
				Required:    true,
				Description: "The event source configuration as JSON.",
			},
			"event_processor_type": &schema.StringAttribute{
				Required:      true,
				Description:   "The event processor type for the stream (for example: handler, correlation, transaction_dispatch).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"event_processor_json": &schema.StringAttribute{
				Required:    true,
				Description: "The event processor configuration as JSON.",
			},
			"description": &schema.StringAttribute{
				Optional:    true,
				Description: "A description of the stream",
			},
			"started": &schema.BoolAttribute{
				Optional:    true,
				Description: "Whether the stream should be started.",
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
	api.EventSourceType = data.EventSourceType.ValueString()
	api.EventProcessorType = data.EventProcessorType.ValueString()
	if !data.Started.IsNull() {
		started := data.Started.ValueBool()
		api.Started = &started
	}

	if data.PostFilterJSON.ValueString() != "" {
		var postFilter map[string]interface{}
		if err := json.Unmarshal([]byte(data.PostFilterJSON.ValueString()), &postFilter); err != nil {
			diagnostics.AddError("Invalid JSON", fmt.Sprintf("Failed to parse post filter JSON: %v", err))
			return
		}
		api.PostFilter = postFilter
	}

	var eventSource map[string]interface{}
	if err := json.Unmarshal([]byte(data.EventSourceJSON.ValueString()), &eventSource); err != nil {
		diagnostics.AddError("Invalid JSON", fmt.Sprintf("Failed to parse stream event source JSON: %v", err))
		return
	}
	api.EventSource = eventSource

	var eventProcessor map[string]interface{}
	if err := json.Unmarshal([]byte(data.EventProcessorJSON.ValueString()), &eventProcessor); err != nil {
		diagnostics.AddError("Invalid JSON", fmt.Sprintf("Failed to parse stream event processor JSON: %v", err))
		return
	}
	api.EventProcessor = eventProcessor
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

	if api.EventSourceType == "" {
		data.EventSourceType = types.StringNull()
	} else {
		data.EventSourceType = types.StringValue(api.EventSourceType)
	}

	if api.EventSource != nil {
		eventSourceBytes, err := json.Marshal(api.EventSource)
		if err != nil {
			diagnostics.AddError("JSON Marshal Error", fmt.Sprintf("Failed to marshal stream event source: %v", err))
			return
		}
		data.EventSourceJSON = types.StringValue(string(eventSourceBytes))
	} else {
		data.EventSourceJSON = types.StringNull()
	}

	if api.EventProcessorType == "" {
		data.EventProcessorType = types.StringNull()
	} else {
		data.EventProcessorType = types.StringValue(api.EventProcessorType)
	}

	if api.EventProcessor != nil {
		eventProcessorBytes, err := json.Marshal(api.EventProcessor)
		if err != nil {
			diagnostics.AddError("JSON Marshal Error", fmt.Sprintf("Failed to marshal stream event processor: %v", err))
			return
		}
		data.EventProcessorJSON = types.StringValue(string(eventProcessorBytes))
	} else {
		data.EventProcessorJSON = types.StringNull()
	}

	if api.Started == nil {
		data.Started = types.BoolNull()
	} else {
		data.Started = types.BoolValue(*api.Started)
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
