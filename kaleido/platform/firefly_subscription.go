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

type FireFlySubscriptionResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Environment types.String `tfsdk:"environment"`
	Service     types.String `tfsdk:"service"`
	Namespace   types.String `tfsdk:"namespace"`
	Name        types.String `tfsdk:"name"`
	ConfigJSON  types.String `tfsdk:"config_json"`
}

type FireFlySubscriptionAPIModel struct {
	ID        string      `json:"id,omitempty"`
	Name      string      `json:"name,omitempty"`
	Created   string      `json:"created,omitempty"`
	Updated   string      `json:"updated,omitempty"`
	Transport string      `json:"transport,omitempty"`
	Filter    interface{} `json:"filter,omitempty"`
	Options   interface{} `json:"options,omitempty"`
	Webhook   interface{} `json:"webhook,omitempty"`
	Ephemeral bool        `json:"ephemeral,omitempty"`
}

// WebhookConfig represents the webhook configuration structure including TLSConfig
// This is used for documentation and type safety, but config_json allows full flexibility
type WebhookConfig struct {
	URL       string     `json:"url,omitempty"`
	TLSConfig *TLSConfig `json:"tlsConfig,omitempty"`
}

// TLSConfig represents TLS configuration for webhook connections
type TLSConfig struct {
	InsecureSkipVerify bool   `json:"insecureSkipVerify,omitempty"`
	TLSCAFile          string `json:"tlsCAFile,omitempty"`
	TLSCertFile        string `json:"tlsCertFile,omitempty"`
	TLSKeyFile         string `json:"tlsKeyFile,omitempty"`
}

func FireFlySubscriptionResourceFactory() resource.Resource {
	return &firefly_subscriptionResource{}
}

type firefly_subscriptionResource struct {
	commonResource
}

func (r *firefly_subscriptionResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_firefly_subscription"
}

func (r *firefly_subscriptionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A FireFly subscription that can forward events to webhooks or other transports.",
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
				Description:   "FireFly namespace name",
			},
			"name": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Subscription name",
			},
			"config_json": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "JSON configuration for the subscription (transport, filter, options). For webhook subscriptions, webhook-specific fields like 'url', 'method', 'headers', 'query', 'tlsConfigName', 'retry', and 'httpOptions' should be set directly in the 'options' object. See https://hyperledger.github.io/firefly/latest/reference/types/subscription/ for the full schema. Note: FireFly subscriptions are immutable - changes require replacement.",
			},
		},
	}
}

func (data *FireFlySubscriptionResourceModel) toAPI(api *FireFlySubscriptionAPIModel, diagnostics *diag.Diagnostics) bool {
	// Set the name (from the resource, not from config JSON)
	api.Name = data.Name.ValueString()

	// Unmarshal the config JSON directly into the API model
	// This preserves the structure of nested objects like webhook
	err := json.Unmarshal([]byte(data.ConfigJSON.ValueString()), api)
	if err != nil {
		diagnostics.AddError("failed to parse config JSON", err.Error())
		return false
	}

	// Ensure name is set (in case config JSON also had a name field)
	api.Name = data.Name.ValueString()

	return true
}

func (api *FireFlySubscriptionAPIModel) toData(data *FireFlySubscriptionResourceModel) {
	data.ID = types.StringValue(api.ID)
}

func (r *firefly_subscriptionResource) apiPath(data *FireFlySubscriptionResourceModel, idOrName string) string {
	// FireFly API through Kaleido endpoint proxy uses /rest/api/v1/subscriptions (no namespace in path)
	if idOrName != "" {
		return fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/subscriptions/%s",
			data.Environment.ValueString(),
			data.Service.ValueString(),
			idOrName)
	}
	return fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/subscriptions",
		data.Environment.ValueString(),
		data.Service.ValueString())
}

func (r *firefly_subscriptionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data FireFlySubscriptionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	var api FireFlySubscriptionAPIModel
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

func (r *firefly_subscriptionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// FireFly subscriptions are immutable - any change requires replacement
	// This method should not be called due to RequiresReplace plan modifiers,
	// but if it is, we'll treat it as a no-op and let Terraform handle replacement
	var data FireFlySubscriptionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)

	// Since subscriptions are immutable, we can't update them
	// Terraform should handle this via replacement, but if we get here,
	// we'll just return the current state
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *firefly_subscriptionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data FireFlySubscriptionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	var api FireFlySubscriptionAPIModel
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
		"transport": api.Transport,
		"filter":    api.Filter,
		"options":   api.Options,
		"webhook":   api.Webhook,
		"ephemeral": api.Ephemeral,
	}
	configJSON, err := json.Marshal(configMap)
	if err == nil {
		data.ConfigJSON = types.StringValue(string(configJSON))
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *firefly_subscriptionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data FireFlySubscriptionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data, data.ID.ValueString()), nil, nil, &resp.Diagnostics, Allow404())

	r.waitForRemoval(ctx, r.apiPath(&data, data.ID.ValueString()), &resp.Diagnostics)
}
