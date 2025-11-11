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
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type ConnectorSetupResourceModel struct {
	ID             types.String `tfsdk:"id"`
	ServiceID      types.String `tfsdk:"service_id"`
	Environment    types.String `tfsdk:"environment"`
	ConfigProfiles types.Map    `tfsdk:"config_profiles"`
}

type SetupInfo struct {
	RequiredConfigTypes      []ConfigTypeInfo     `json:"requiredConfigTypes"`
	ConnectorFlows           []ConnectorFlowInfo  `json:"connectorFlows"`
	ConnectorStreamFactories []StreamFactoryInfo  `json:"connectorStreamFactories"`
	StandardAPIs             []StandardAPIInfo    `json:"standardAPIs"`
	StandardStreams          []StandardStreamInfo `json:"standardStreams"`
}

type ConfigTypeInfo struct {
	Name string `json:"name"`
}

type ConnectorFlowInfo struct {
	Name     string `json:"name"`
	FlowType string `json:"flowType"`
}

type StreamFactoryInfo struct {
	Name string `json:"name"`
}

type StandardAPIInfo struct {
	Name string `json:"name"`
}

type StandardStreamInfo struct {
	Name string `json:"name"`
}

type ConfigProfile struct {
	ID         string          `json:"id,omitempty"`
	ConfigType string          `json:"configType"`
	Value      json.RawMessage `json:"value"`
}

type ConnectorFlow struct {
	Name               string                 `json:"name"`
	ConfigTypeBindings map[string]interface{} `json:"configTypeBindings"`
}

type StandardAPI struct {
	Name             string            `json:"name"`
	FlowTypeBindings map[string]string `json:"flowTypeBindings"`
}

type StandardStream struct {
	Name               string                 `json:"name"`
	ConfigTypeBindings map[string]interface{} `json:"configTypeBindings"`
}

func ConnectorSetupResourceFactory() resource.Resource {
	return &connectorSetupResource{}
}

type connectorSetupResource struct {
	commonResource
}

func (r *connectorSetupResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_connector_setup"
}

func (r *connectorSetupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Configures a connector by setting up config types, config profiles, connector flows, stream factories, standard APIs, and standard streams. This resource is generic and works with different connector types (EVM, Solana, etc.).",
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"service_id": &schema.StringAttribute{
				Required:    true,
				Description: "The ID of the connector service to configure",
			},
			"environment": &schema.StringAttribute{
				Required:    true,
				Description: "The environment ID where the connector service is located",
			},
			"config_profiles": &schema.MapAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "A map of config profile names to JSON-encoded values for ALL required config types. Keys should match config type names. Values should be JSON-encoded strings.",
			},
		},
	}
}

func (r *connectorSetupResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// validate the plan during plan time
	if req.Plan.Raw.IsNull() {
		return
	}

	var data ConnectorSetupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// if environment is unknown, skip validation
	if data.Environment.IsNull() || data.Environment.IsUnknown() {
		return
	}

	// otherwise, verify a flow engine is present in the environment
	diags := r.verifyWorkflowEngine(ctx, data.Environment.ValueString())
	if diags.HasError() {
		resp.Diagnostics.Append(*diags...)
		return
	}
}

func (r *connectorSetupResource) verifyWorkflowEngine(ctx context.Context, environment string) *diag.Diagnostics {
	diagnostics := &diag.Diagnostics{}

	// fetch and validate workflow engine service is present in the environment
	path := fmt.Sprintf("/api/v1/environments/%s/services?type=WorkflowEngineService", environment)
	var servicesResult struct {
		Items []ServiceAPIModel `json:"items"`
	}
	ok, _ := r.apiRequest(ctx, http.MethodGet, path, nil, &servicesResult, diagnostics)
	if !ok {
		diagnostics.AddError("Failed to check for WorkflowEngine", "Could not query for WorkflowEngine service. Please ensure a WorkflowEngine service exists in the environment.")
		return diagnostics
	}

	if len(servicesResult.Items) == 0 {
		diagnostics.AddError("WorkflowEngine service required", "A WorkflowEngine service must exist before setting up a connector. Please create a WorkflowEngine service first.")
		return diagnostics
	}

	// check if the workflow engine service is ready
	service := servicesResult.Items[0]
	if service.Status != "ready" {
		diagnostics.AddError("WorkflowEngine service not ready", fmt.Sprintf("WorkflowEngine service %s is not ready (status: %s). Please wait for the service to be ready before setting up the connector.", service.ID, service.Status))
		return diagnostics
	}

	tflog.Info(ctx, "WorkflowEngine service verified and ready")
	return diagnostics
}

func (r *connectorSetupResource) getServiceEndpoint(ctx context.Context, serviceID, environment string) (string, *diag.Diagnostics) {
	diagnostics := &diag.Diagnostics{}

	var api ServiceAPIModel
	api.ID = serviceID
	path := fmt.Sprintf("/api/v1/environments/%s/services/%s", environment, serviceID)
	ok, _ := r.apiRequest(ctx, http.MethodGet, path, nil, &api, diagnostics)
	if !ok {
		diagnostics.AddError("Failed to get service", fmt.Sprintf("Could not retrieve service %s", serviceID))
		return "", diagnostics
	}

	// check if the service is ready
	if api.Status != "ready" {
		diagnostics.AddError("Service not ready", fmt.Sprintf("Service %s is not ready (status: %s). Please wait for the service to be ready before configuring it.", serviceID, api.Status))
		return "", diagnostics
	}

	// get the rest endpoint URL
	if restEndpoint, ok := api.Endpoints["rest"]; ok && len(restEndpoint.URLS) > 0 {
		return restEndpoint.URLS[0], diagnostics
	}

	diagnostics.AddError("No REST endpoint", fmt.Sprintf("Service %s does not have a REST endpoint", serviceID))
	return "", diagnostics
}

func (r *connectorSetupResource) connectorAPIRequest(ctx context.Context, baseURL, method, path string, body, result interface{}) error {
	fullURL := baseURL + path

	req := r.Platform.R().
		SetContext(ctx).
		SetDoNotParseResponse(true).
		SetHeader("Content-Type", "application/json")

	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		req = req.SetBody(bodyBytes)
		tflog.Debug(ctx, fmt.Sprintf("Connector API Request Body: %s", string(bodyBytes)))
	}

	res, err := req.Execute(method, fullURL)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}

	statusCode := res.StatusCode()
	tflog.Debug(ctx, fmt.Sprintf("Connector API Response: %d %s", statusCode, res.Status()))

	if !res.IsSuccess() {
		var rawBytes []byte
		if res.RawResponse != nil && res.RawResponse.Body != nil {
			rawBytes, _ = io.ReadAll(res.RawBody())
			res.RawResponse.Body.Close()
		}
		return fmt.Errorf("request failed with status %d: %s", statusCode, string(rawBytes))
	}

	if result != nil {
		var rawBytes []byte
		if res.RawResponse != nil && res.RawResponse.Body != nil {
			rawBytes, _ = io.ReadAll(res.RawBody())
			res.RawResponse.Body.Close()
		}
		if len(rawBytes) > 0 {
			if err := json.Unmarshal(rawBytes, result); err != nil {
				return fmt.Errorf("failed to unmarshal response: %w", err)
			}
		}
	}

	return nil
}

// setup the connector by creating config types, config profiles, connector flows, stream factories, standard APIs, and standard streams
func (r *connectorSetupResource) setupConnector(ctx context.Context, baseURL string, configProfiles map[string]string) error {

	var setupInfo SetupInfo
	if err := r.connectorAPIRequest(ctx, baseURL, http.MethodGet, "/api/v1/metadata/setup-info", nil, &setupInfo); err != nil {
		return fmt.Errorf("failed to get setup info: %w", err)
	}

	// create the required config types, and the config profiles for each config type
	profileBindings := make(map[string]interface{})
	for _, ct := range setupInfo.RequiredConfigTypes {
		ctName := ct.Name

		if err := r.connectorAPIRequest(ctx, baseURL, http.MethodPut, fmt.Sprintf("/api/v1/metadata/config-types/%s", ctName), map[string]interface{}{}, nil); err != nil {
			return fmt.Errorf("failed to establish config type %s: %w", ctName, err)
		}

		// create the config profile if provided
		// NOTE: if the config profile is not provided, the logic won't create a default empty config profile and will cause error during flow creation
		// this is intentional to encourage all config profiles are stored in the terraform state for full traceability
		if profileValue, ok := configProfiles[ctName]; ok {
			var profileValueJSON json.RawMessage
			if err := json.Unmarshal([]byte(profileValue), &profileValueJSON); err != nil {
				return fmt.Errorf("failed to parse config profile value for %s: %w", ctName, err)
			}

			var cp ConfigProfile
			cpBody := ConfigProfile{
				ConfigType: ctName,
				Value:      profileValueJSON,
			}
			if err := r.connectorAPIRequest(ctx, baseURL, http.MethodPut, fmt.Sprintf("/api/v1/config-profiles/%s", ctName), cpBody, &cp); err != nil {
				return fmt.Errorf("failed to create config profile %s: %w", ctName, err)
			}

			profileBindings[ctName] = map[string]interface{}{
				"configProfileID": cp.ID,
			}
		}
	}

	// deploy the connector flows
	flowTypeBindings := make(map[string]string)
	for _, cf := range setupInfo.ConnectorFlows {
		cfName := cf.Name

		// delete the existing connector flow if it exists (idempotent)
		_ = r.connectorAPIRequest(ctx, baseURL, http.MethodDelete, fmt.Sprintf("/api/v1/connector-flows/%s", cfName), nil, nil)

		var cfd ConnectorFlow
		cfBody := ConnectorFlow{
			Name:               cfName,
			ConfigTypeBindings: profileBindings,
		}
		if err := r.connectorAPIRequest(ctx, baseURL, http.MethodPost, fmt.Sprintf("/api/v1/metadata/connector-flows/%s", cfName), cfBody, &cfd); err != nil {
			return fmt.Errorf("failed to deploy connector flow %s: %w", cfName, err)
		}

		flowTypeBindings[cf.FlowType] = cfd.Name
	}

	// create the connector stream factories
	for _, sf := range setupInfo.ConnectorStreamFactories {
		sfName := sf.Name
		if err := r.connectorAPIRequest(ctx, baseURL, http.MethodPut, fmt.Sprintf("/api/v1/metadata/connector-stream-factories/%s", sfName), map[string]interface{}{}, nil); err != nil {
			return fmt.Errorf("failed to deploy connector stream factory %s: %w", sfName, err)
		}
	}

	// deploy the standard APIs
	for _, sa := range setupInfo.StandardAPIs {
		saName := sa.Name

		// Delete existing API if it exists (idempotent)
		_ = r.connectorAPIRequest(ctx, baseURL, http.MethodDelete, fmt.Sprintf("/api/v1/apis/%s", saName), nil, nil)

		saBody := StandardAPI{
			Name:             saName,
			FlowTypeBindings: flowTypeBindings,
		}
		if err := r.connectorAPIRequest(ctx, baseURL, http.MethodPost, fmt.Sprintf("/api/v1/metadata/standard-apis/%s", saName), saBody, nil); err != nil {
			return fmt.Errorf("failed to deploy standard API %s: %w", saName, err)
		}
	}

	// deploy the standard streams
	for _, ss := range setupInfo.StandardStreams {
		ssName := ss.Name

		// delete the existing standard stream if it exists (idempotent)
		_ = r.connectorAPIRequest(ctx, baseURL, http.MethodDelete, fmt.Sprintf("/api/v1/streams/%s", ssName), nil, nil)

		ssBody := StandardStream{
			Name:               ssName,
			ConfigTypeBindings: profileBindings,
		}
		if err := r.connectorAPIRequest(ctx, baseURL, http.MethodPost, fmt.Sprintf("/api/v1/metadata/standard-streams/%s", ssName), ssBody, nil); err != nil {
			return fmt.Errorf("failed to deploy standard stream %s: %w", ssName, err)
		}
	}

	return nil
}

func (r *connectorSetupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ConnectorSetupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	// verify the workflow engine service is ready
	diags := r.verifyWorkflowEngine(ctx, data.Environment.ValueString())
	if diags.HasError() {
		resp.Diagnostics.Append(*diags...)
		return
	}

	endpoint, diags := r.getServiceEndpoint(ctx, data.ServiceID.ValueString(), data.Environment.ValueString())
	if diags.HasError() {
		resp.Diagnostics.Append(*diags...)
		return
	}

	// parse the config profiles
	configProfiles := make(map[string]string)
	if !data.ConfigProfiles.IsNull() {
		elements := data.ConfigProfiles.Elements()
		for k, v := range elements {
			configProfiles[k] = v.(types.String).ValueString()
		}
	}

	// setup the connector
	if err := r.setupConnector(ctx, endpoint, configProfiles); err != nil {
		resp.Diagnostics.AddError("Failed to setup connector", err.Error())
		return
	}

	// set the ID to the service_id
	data.ID = data.ServiceID
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *connectorSetupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ConnectorSetupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	// verify the service still exists and is ready
	_, diags := r.getServiceEndpoint(ctx, data.ServiceID.ValueString(), data.Environment.ValueString())
	if diags.HasError() {
		// if the service doesn't exist or isn't ready, remove from state
		resp.State.RemoveResource(ctx)
		return
	}

	// NOTE: there is no need to validate the existing setup state here, because the setup is idempotent
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *connectorSetupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ConnectorSetupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)

	endpoint, diags := r.getServiceEndpoint(ctx, data.ServiceID.ValueString(), data.Environment.ValueString())
	if diags.HasError() {
		resp.Diagnostics.Append(*diags...)
		return
	}

	configProfiles := make(map[string]string)
	if !data.ConfigProfiles.IsNull() {
		elements := data.ConfigProfiles.Elements()
		for k, v := range elements {
			configProfiles[k] = v.(types.String).ValueString()
		}
	}

	// re-setup the connector (idempotent operation)
	if err := r.setupConnector(ctx, endpoint, configProfiles); err != nil {
		resp.Diagnostics.AddError("Failed to update connector setup", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *connectorSetupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ConnectorSetupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	// TODO: sort out the deletion of the connector setup to remove the workflow engine resources created by the setup
	tflog.Info(ctx, "Connector setup deletion is a no-op - configurations remain on the service")
}
