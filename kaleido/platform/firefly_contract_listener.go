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

type FireFlyContractListenerResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Environment types.String `tfsdk:"environment"`
	Service     types.String `tfsdk:"service"`
	Namespace   types.String `tfsdk:"namespace"`
	Name        types.String `tfsdk:"name"`
	ConfigJSON  types.String `tfsdk:"config_json"`
}

type FireFlyContractListenerAPIModel struct {
	ID        string      `json:"id,omitempty"`
	Name      string      `json:"name,omitempty"`
	Created   string      `json:"created,omitempty"`
	Updated   string      `json:"updated,omitempty"`
	Location  interface{} `json:"location,omitempty"`
	Event     interface{} `json:"event,omitempty"`
	Options   interface{} `json:"options,omitempty"`
	Topic     string      `json:"topic,omitempty"`
	Signature string      `json:"signature,omitempty"`
}

func FireFlyContractListenerResourceFactory() resource.Resource {
	return &firefly_contract_listenerResource{}
}

type firefly_contract_listenerResource struct {
	commonResource
}

func (r *firefly_contract_listenerResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_firefly_contract_listener"
}

func (r *firefly_contract_listenerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A FireFly contract listener that listens for specific blockchain events from smart contracts.",
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
				Description:   "FireFly Service ID",
			},
			"namespace": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "FireFly namespace name, this should match name of the firefly service.",
			},
			"name": &schema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Listener name (optional, auto-generated if not provided). Note: FireFly contract listeners are immutable - changes require replacement.",
			},
			"config_json": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "JSON configuration for the listener (location, event, topic, options, etc.). Note: FireFly contract listeners are immutable - changes require replacement.",
			},
		},
	}
}

func (data *FireFlyContractListenerResourceModel) toAPI(api *FireFlyContractListenerAPIModel, diagnostics *diag.Diagnostics) bool {
	if !data.Name.IsNull() {
		api.Name = data.Name.ValueString()
	}
	err := json.Unmarshal([]byte(data.ConfigJSON.ValueString()), api)
	if err != nil {
		diagnostics.AddError("failed to serialize config JSON", err.Error())
		return false
	}
	return true
}

func (api *FireFlyContractListenerAPIModel) toData(data *FireFlyContractListenerResourceModel) {
	data.ID = types.StringValue(api.ID)
	if api.Name != "" {
		data.Name = types.StringValue(api.Name)
	}
}

func (r *firefly_contract_listenerResource) apiPath(data *FireFlyContractListenerResourceModel, idOrName string) string {
	// FireFly API through Kaleido endpoint proxy uses /rest/api/v1/contracts/listeners (no namespace in path)
	if idOrName != "" {
		return fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/contracts/listeners/%s",
			data.Environment.ValueString(),
			data.Service.ValueString(),
			idOrName)
	}
	return fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/contracts/listeners",
		data.Environment.ValueString(),
		data.Service.ValueString())
}

func (r *firefly_contract_listenerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data FireFlyContractListenerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	var api FireFlyContractListenerAPIModel
	ok := data.toAPI(&api, &resp.Diagnostics)
	if ok {
		ok, _ = r.apiRequest(ctx, http.MethodPost, r.apiPath(&data, ""), &api, &api, &resp.Diagnostics)
	}
	if !ok {
		return
	}

	api.toData(&data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *firefly_contract_listenerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// FireFly contract listeners are immutable - any change requires replacement
	// This method should not be called due to RequiresReplace plan modifiers,
	// but if it is, we'll treat it as a no-op and let Terraform handle replacement
	var data FireFlyContractListenerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)

	// Since listeners are immutable, we can't update them
	// Terraform should handle this via replacement, but if we get here,
	// we'll just return the current state
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *firefly_contract_listenerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data FireFlyContractListenerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	var api FireFlyContractListenerAPIModel
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
	// Reconstruct config_json from API response
	configMap := map[string]interface{}{
		"location":  api.Location,
		"event":     api.Event,
		"options":   api.Options,
		"signature": api.Signature,
	}
	if api.Topic != "" {
		configMap["topic"] = api.Topic
	}
	configJSON, err := json.Marshal(configMap)
	if err == nil {
		data.ConfigJSON = types.StringValue(string(configJSON))
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *firefly_contract_listenerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data FireFlyContractListenerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data, data.ID.ValueString()), nil, nil, &resp.Diagnostics, Allow404())

	r.waitForRemoval(ctx, r.apiPath(&data, data.ID.ValueString()), &resp.Diagnostics)
}
