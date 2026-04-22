// Copyright © Kaleido, Inc. 2026

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

type WFEStreamTransformResourceModel struct {
	UniquenessPrefix types.String `tfsdk:"uniqueness_prefix"`
	FilterJSONata    types.String `tfsdk:"filter_jsonata"`
	MappingJSONata   types.String `tfsdk:"mapping_jsonata"`
}

type WFEStreamResourceModel struct {
	ID                 types.String                     `tfsdk:"id"`
	Name               types.String                     `tfsdk:"name"`
	Description        types.String                     `tfsdk:"description"`
	Environment        types.String                     `tfsdk:"environment"`
	Service            types.String                     `tfsdk:"service"`
	Transform          *WFEStreamTransformResourceModel `tfsdk:"transform"`
	EventSourceType    types.String                     `tfsdk:"event_source_type"`
	EventSourceJSON    types.String                     `tfsdk:"event_source_json"`
	EventProcessorType types.String                     `tfsdk:"event_processor_type"`
	EventProcessorJSON types.String                     `tfsdk:"event_processor_json"`
	Started            types.Bool                       `tfsdk:"started"`
	Created            types.String                     `tfsdk:"created"`
	Updated            types.String                     `tfsdk:"updated"`
}

type WFEStreamTransformFilterMappingAPI struct {
	JSONata string `json:"jsonata,omitempty"`
}

type WFEStreamTransformMappingAPI struct {
	JSONata string `json:"jsonata,omitempty"`
}

type WFEStreamTransformAPI struct {
	UniquenessPrefix string                              `json:"uniquenessPrefix,omitempty"`
	Filter           *WFEStreamTransformFilterMappingAPI `json:"filter,omitempty"`
	Mapping          *WFEStreamTransformMappingAPI       `json:"mapping,omitempty"`
}

// WFEStreamAPIModel mirrors the Stream JSON from workflow-engine engtypes (StreamInput/Stream).
type WFEStreamAPIModel struct {
	ID             string                 `json:"id,omitempty"`
	Name           string                 `json:"name,omitempty"`
	Description    string                 `json:"description,omitempty"`
	Transform      *WFEStreamTransformAPI `json:"transform,omitempty"`
	EventSource    map[string]interface{} `json:"eventSource,omitempty"`
	EventProcessor map[string]interface{} `json:"eventProcessor,omitempty"`
	Started        *bool                  `json:"started,omitempty"`
	Created        *time.Time             `json:"created,omitempty"`
	Updated        *time.Time             `json:"updated,omitempty"`
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
		Description: "Manages Workflow Engine streams for event streaming",
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"environment": &schema.StringAttribute{
				Required:      true,
				Description:   "Environment ID",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"service": &schema.StringAttribute{
				Required:      true,
				Description:   "Workflow Engine service ID",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": &schema.StringAttribute{
				Required:      true,
				Description:   "The name of the stream",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"transform": &schema.SingleNestedAttribute{
				Optional:    true,
				Description: "The transform configuration for this stream, including filter, mapping, and uniqueness prefix",
				Attributes: map[string]schema.Attribute{
					"uniqueness_prefix": &schema.StringAttribute{
						Optional:    true,
						Description: "A string prefix to prepend to all 'topic' and 'idempotencyKey' values generated by this stream",
					},
					"filter_jsonata": &schema.StringAttribute{
						Optional:    true,
						Description: "Optional JSONata condition to evaluate against events to filter them, before any evaluation (must return true/false)",
					},
					"mapping_jsonata": &schema.StringAttribute{
						Optional:    true,
						Description: "Optional JSONata mapping to transform events",
					},
				},
			},
			"event_source_type": &schema.StringAttribute{
				Required:      true,
				Description:   "The event source type for the stream (for example: handler, transaction).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"event_source_json": &schema.StringAttribute{
				Required:    true,
				Description: "The event source configuration as JSON (the type-specific content nested under the event source type key).",
			},
			"event_processor_type": &schema.StringAttribute{
				Required:      true,
				Description:   "The event processor type for the stream (for example: handler, correlation, transaction_dispatch).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"event_processor_json": &schema.StringAttribute{
				Required:    true,
				Description: "The event processor configuration as JSON (the type-specific content nested under the event processor type key).",
			},
			"description": &schema.StringAttribute{
				Optional:    true,
				Description: "Description of the stream",
			},
			"started": &schema.BoolAttribute{
				Optional:    true,
				Description: "Whether the stream should be started",
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
	if !data.Started.IsNull() {
		started := data.Started.ValueBool()
		api.Started = &started
	}

	if data.Transform != nil {
		transform := WFEStreamTransformAPI{}
		hasContent := false

		if !data.Transform.UniquenessPrefix.IsNull() && data.Transform.UniquenessPrefix.ValueString() != "" {
			transform.UniquenessPrefix = data.Transform.UniquenessPrefix.ValueString()
			hasContent = true
		}

		if !data.Transform.FilterJSONata.IsNull() && data.Transform.FilterJSONata.ValueString() != "" {
			transform.Filter = &WFEStreamTransformFilterMappingAPI{
				JSONata: data.Transform.FilterJSONata.ValueString(),
			}
			hasContent = true
		}

		if !data.Transform.MappingJSONata.IsNull() && data.Transform.MappingJSONata.ValueString() != "" {
			transform.Mapping = &WFEStreamTransformMappingAPI{
				JSONata: data.Transform.MappingJSONata.ValueString(),
			}
			hasContent = true
		}

		if hasContent {
			api.Transform = &transform
		}
	}

	// EventSource: nest type and content under a type-specific key
	eventSourceType := data.EventSourceType.ValueString()
	var eventSourceContent map[string]interface{}
	if err := json.Unmarshal([]byte(data.EventSourceJSON.ValueString()), &eventSourceContent); err != nil {
		diagnostics.AddError("Invalid JSON", fmt.Sprintf("Failed to parse stream event source JSON: %v", err))
		return
	}
	api.EventSource = map[string]interface{}{
		"type":          eventSourceType,
		eventSourceType: eventSourceContent,
	}

	// EventProcessor: nest type and content under a type-specific key
	eventProcessorType := data.EventProcessorType.ValueString()
	eventProcessorTypeCamelCase := eventProcessorType
	if eventProcessorType == "transaction_dispatch" {
		eventProcessorTypeCamelCase = "transactionDispatch"
	}
	var eventProcessorContent map[string]interface{}
	if err := json.Unmarshal([]byte(data.EventProcessorJSON.ValueString()), &eventProcessorContent); err != nil {
		diagnostics.AddError("Invalid JSON", fmt.Sprintf("Failed to parse stream event processor JSON: %v", err))
		return
	}
	api.EventProcessor = map[string]interface{}{
		"type":                      eventProcessorType,
		eventProcessorTypeCamelCase: eventProcessorContent,
	}
}

func (r *wfe_streamResource) toData(api *WFEStreamAPIModel, data *WFEStreamResourceModel, diagnostics *diag.Diagnostics) {
	data.ID = types.StringValue(api.ID)
	data.Name = types.StringValue(api.Name)

	if api.Description == "" {
		data.Description = types.StringNull()
	} else {
		data.Description = types.StringValue(api.Description)
	}

	// Extract transform fields
	if api.Transform != nil {
		if data.Transform == nil {
			data.Transform = &WFEStreamTransformResourceModel{}
		}

		if api.Transform.UniquenessPrefix == "" {
			data.Transform.UniquenessPrefix = types.StringNull()
		} else {
			data.Transform.UniquenessPrefix = types.StringValue(api.Transform.UniquenessPrefix)
		}

		if api.Transform.Filter != nil && api.Transform.Filter.JSONata != "" {
			data.Transform.FilterJSONata = types.StringValue(api.Transform.Filter.JSONata)
		} else {
			data.Transform.FilterJSONata = types.StringNull()
		}

		if api.Transform.Mapping != nil && api.Transform.Mapping.JSONata != "" {
			data.Transform.MappingJSONata = types.StringValue(api.Transform.Mapping.JSONata)
		} else {
			data.Transform.MappingJSONata = types.StringNull()
		}
	} else if data.Transform != nil {
		// API returned no transform, but we had one — clear it
		data.Transform = nil
	}

	// EventSource: extract type and type-specific content from nested structure
	if api.EventSource != nil {
		if t, ok := api.EventSource["type"].(string); ok && t != "" {
			data.EventSourceType = types.StringValue(t)
			if content, ok := api.EventSource[t]; ok {
				contentBytes, err := json.Marshal(content)
				if err != nil {
					diagnostics.AddError("JSON Marshal Error", fmt.Sprintf("Failed to marshal stream event source: %v", err))
					return
				}
				data.EventSourceJSON = types.StringValue(string(contentBytes))
			} else {
				data.EventSourceJSON = types.StringNull()
			}
		} else {
			data.EventSourceType = types.StringNull()
			data.EventSourceJSON = types.StringNull()
		}
	} else {
		data.EventSourceType = types.StringNull()
		data.EventSourceJSON = types.StringNull()
	}

	// EventProcessor: extract type and type-specific content from nested structure
	if api.EventProcessor != nil {
		if t, ok := api.EventProcessor["type"].(string); ok && t != "" {
			data.EventProcessorType = types.StringValue(t)
			eventProcessorTypeCamelCase := t
			if t == "transaction_dispatch" {
				eventProcessorTypeCamelCase = "transactionDispatch"
			}
			if content, ok := api.EventProcessor[eventProcessorTypeCamelCase]; ok {
				contentBytes, err := json.Marshal(content)
				if err != nil {
					diagnostics.AddError("JSON Marshal Error", fmt.Sprintf("Failed to marshal stream event processor: %v", err))
					return
				}
				data.EventProcessorJSON = types.StringValue(string(contentBytes))
			} else {
				data.EventProcessorJSON = types.StringNull()
			}
		} else {
			data.EventProcessorType = types.StringNull()
			data.EventProcessorJSON = types.StringNull()
		}
	} else {
		data.EventProcessorType = types.StringNull()
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

	// Use PUT (idempotent upsert) for updates. The full StreamInput is sent, which
	// includes both updatable fields (name, description, started, transform, eventSource)
	// and immutable fields (eventProcessor). Immutable fields are marked RequiresReplace
	// in the schema, so they cannot change without a destroy+create cycle.
	var api WFEStreamAPIModel
	r.toAPI(&data, &api, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	ok, _ := r.apiRequest(ctx, http.MethodPut, r.apiPath(&data, data.ID.ValueString()), &api, &api, &resp.Diagnostics)
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
