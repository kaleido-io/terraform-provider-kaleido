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

// WFEStreamFactoryResourceModel is the Terraform state model for a WFE stream factory.
type WFEStreamFactoryResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	Environment          types.String `tfsdk:"environment"`
	Service              types.String `tfsdk:"service"`
	Name                 types.String `tfsdk:"name"`
	Description          types.String `tfsdk:"description"`
	ConfigType           types.String `tfsdk:"config_type"`
	UniquenessPrefix     types.String `tfsdk:"uniqueness_prefix"`
	EventSourceType      types.String `tfsdk:"event_source_type"`
	EventSourceJSON      types.String `tfsdk:"event_source_json"`
	ConstantsJSON        types.String `tfsdk:"constants_json"`
	ParametersSchemaJSON types.String `tfsdk:"parameters_schema_json"`
	APIURL               types.String `tfsdk:"api_url"`
	OpenAPIURL           types.String `tfsdk:"openapi_url"`
	Created              types.String `tfsdk:"created"`
	Updated              types.String `tfsdk:"updated"`
}

// WFEStreamFactoryURLsAPI mirrors the APIURLs JSON from workflow-engine engtypes.
type WFEStreamFactoryURLsAPI struct {
	API     string `json:"api,omitempty"`
	OpenAPI string `json:"openapi,omitempty"`
}

// WFEStreamFactoryAPIModel mirrors the StreamFactory / StreamFactoryInput JSON from
// workflow-engine engtypes. The ConfigType field is input-only (not returned by GET).
type WFEStreamFactoryAPIModel struct {
	ID               string                   `json:"id,omitempty"`
	ConfigType       string                   `json:"configType,omitempty"`
	Name             string                   `json:"name,omitempty"`
	Description      string                   `json:"description,omitempty"`
	Constants        map[string]interface{}   `json:"constants,omitempty"`
	UniquenessPrefix string                   `json:"uniquenessPrefix,omitempty"`
	EventSource      map[string]interface{}   `json:"eventSource,omitempty"`
	ParametersSchema map[string]interface{}   `json:"parametersSchema,omitempty"`
	URLs             *WFEStreamFactoryURLsAPI `json:"urls,omitempty"`
	Created          *time.Time               `json:"created,omitempty"`
	Updated          *time.Time               `json:"updated,omitempty"`
}

func WFEStreamFactoryResourceFactory() resource.Resource {
	return &wfe_streamFactoryResource{}
}

type wfe_streamFactoryResource struct {
	commonResource
}

func (r *wfe_streamFactoryResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_wfe_stream_factory"
}

func (r *wfe_streamFactoryResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The Workflow Engine Stream Factory resource allows you to manage stream factories that create pre-configured streams in the Workflow Engine.",
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				Description:   "Unique ID of the stream factory",
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
				Description:   "The name of the stream factory",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"description": &schema.StringAttribute{
				Optional:    true,
				Description: "A description of the stream factory",
			},
			"config_type": &schema.StringAttribute{
				Required:      true,
				Description:   "The config type ID to associate with this stream factory",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"uniqueness_prefix": &schema.StringAttribute{
				Optional:    true,
				Description: "A string prefix to prepend to all 'topic' and 'idempotencyKey' values generated by streams created from this factory.",
			},
			"event_source_type": &schema.StringAttribute{
				Required:      true,
				Description:   "The event source type for the factory (for example: handler).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"event_source_json": &schema.StringAttribute{
				Required:    true,
				Description: "The event source configuration as JSON (the type-specific content nested under the event source type key, e.g. handler name, provider, configTypeId, configMapping).",
			},
			"constants_json": &schema.StringAttribute{
				Optional:    true,
				Description: "Constant values as JSON that are merged into every stream created by this factory.",
			},
			"parameters_schema_json": &schema.StringAttribute{
				Optional:    true,
				Description: "JSON Schema for the parameters that can be passed when creating streams from this factory.",
			},
			"api_url": &schema.StringAttribute{
				Computed:    true,
				Description: "The factory-scoped API URL for creating streams.",
			},
			"openapi_url": &schema.StringAttribute{
				Computed:    true,
				Description: "The factory-scoped OpenAPI specification URL.",
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

func (r *wfe_streamFactoryResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.commonResource.Configure(ctx, req, resp)
}

func (r *wfe_streamFactoryResource) apiPath(data *WFEStreamFactoryResourceModel, idOrName string) string {
	env := data.Environment.ValueString()
	service := data.Service.ValueString()
	return fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/stream-factories/%s", env, service, idOrName)
}

func (r *wfe_streamFactoryResource) toAPI(data *WFEStreamFactoryResourceModel, api *WFEStreamFactoryAPIModel, diagnostics *diag.Diagnostics) {
	api.Name = data.Name.ValueString()
	api.Description = data.Description.ValueString()
	api.ConfigType = data.ConfigType.ValueString()

	if !data.UniquenessPrefix.IsNull() && data.UniquenessPrefix.ValueString() != "" {
		api.UniquenessPrefix = data.UniquenessPrefix.ValueString()
	}

	// EventSource: nest type and content under a type-specific key
	eventSourceType := data.EventSourceType.ValueString()
	var eventSourceContent map[string]interface{}
	if err := json.Unmarshal([]byte(data.EventSourceJSON.ValueString()), &eventSourceContent); err != nil {
		diagnostics.AddError("Invalid JSON", fmt.Sprintf("Failed to parse stream factory event source JSON: %v", err))
		return
	}
	api.EventSource = map[string]interface{}{
		"type":          eventSourceType,
		eventSourceType: eventSourceContent,
	}

	// Constants: optional JSON object
	if !data.ConstantsJSON.IsNull() && data.ConstantsJSON.ValueString() != "" {
		var constants map[string]interface{}
		if err := json.Unmarshal([]byte(data.ConstantsJSON.ValueString()), &constants); err != nil {
			diagnostics.AddError("Invalid JSON", fmt.Sprintf("Failed to parse stream factory constants JSON: %v", err))
			return
		}
		api.Constants = constants
	}

	// ParametersSchema: optional JSON object
	if !data.ParametersSchemaJSON.IsNull() && data.ParametersSchemaJSON.ValueString() != "" {
		var parametersSchema map[string]interface{}
		if err := json.Unmarshal([]byte(data.ParametersSchemaJSON.ValueString()), &parametersSchema); err != nil {
			diagnostics.AddError("Invalid JSON", fmt.Sprintf("Failed to parse stream factory parameters schema JSON: %v", err))
			return
		}
		api.ParametersSchema = parametersSchema
	}
}

func (r *wfe_streamFactoryResource) toData(api *WFEStreamFactoryAPIModel, data *WFEStreamFactoryResourceModel, diagnostics *diag.Diagnostics) {
	data.ID = types.StringValue(api.ID)
	data.Name = types.StringValue(api.Name)

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

	// EventSource: extract type and type-specific content from nested structure
	if api.EventSource != nil {
		if t, ok := api.EventSource["type"].(string); ok && t != "" {
			data.EventSourceType = types.StringValue(t)
			if content, ok := api.EventSource[t]; ok {
				contentBytes, err := json.Marshal(content)
				if err != nil {
					diagnostics.AddError("JSON Marshal Error", fmt.Sprintf("Failed to marshal stream factory event source: %v", err))
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

	// Constants: marshal back to JSON string
	if api.Constants != nil {
		constantsBytes, err := json.Marshal(api.Constants)
		if err != nil {
			diagnostics.AddError("JSON Marshal Error", fmt.Sprintf("Failed to marshal stream factory constants: %v", err))
			return
		}
		data.ConstantsJSON = types.StringValue(string(constantsBytes))
	} else if !data.ConstantsJSON.IsNull() {
		// Preserve null if not set by API
		data.ConstantsJSON = types.StringNull()
	}

	// ParametersSchema: marshal back to JSON string
	if api.ParametersSchema != nil {
		schemaBytes, err := json.Marshal(api.ParametersSchema)
		if err != nil {
			diagnostics.AddError("JSON Marshal Error", fmt.Sprintf("Failed to marshal stream factory parameters schema: %v", err))
			return
		}
		data.ParametersSchemaJSON = types.StringValue(string(schemaBytes))
	} else if !data.ParametersSchemaJSON.IsNull() {
		data.ParametersSchemaJSON = types.StringNull()
	}

	// URLs: computed by the server
	if api.URLs != nil {
		if api.URLs.API != "" {
			data.APIURL = types.StringValue(api.URLs.API)
		} else {
			data.APIURL = types.StringNull()
		}
		if api.URLs.OpenAPI != "" {
			data.OpenAPIURL = types.StringValue(api.URLs.OpenAPI)
		} else {
			data.OpenAPIURL = types.StringNull()
		}
	} else {
		data.APIURL = types.StringNull()
		data.OpenAPIURL = types.StringNull()
	}

	// Note: ConfigType is input-only and not returned by GET. We preserve the
	// value already in TF state (set during Create or from the plan).

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

func (r *wfe_streamFactoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data WFEStreamFactoryResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var api WFEStreamFactoryAPIModel
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

func (r *wfe_streamFactoryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data WFEStreamFactoryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var api WFEStreamFactoryAPIModel
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

func (r *wfe_streamFactoryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data WFEStreamFactoryResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Use PUT (idempotent upsert) for updates. Immutable fields (name, config_type,
	// event_source_type) are marked RequiresReplace, so they cannot change without
	// a destroy+create cycle.
	var api WFEStreamFactoryAPIModel
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

func (r *wfe_streamFactoryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data WFEStreamFactoryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data, data.ID.ValueString()), nil, nil, &resp.Diagnostics, Allow404())
	r.waitForRemoval(ctx, r.apiPath(&data, data.ID.ValueString()), &resp.Diagnostics)
}
