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
	"net/http"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type ConnectorCustomAPIResourceModel struct {
	ID               types.String `tfsdk:"id"`
	ServiceID        types.String `tfsdk:"service_id"`
	Environment      types.String `tfsdk:"environment"`
	Name             types.String `tfsdk:"name"`
	ABI              types.String `tfsdk:"abi"`
	Bytecode         types.String `tfsdk:"bytecode"`
	DevDoc           types.String `tfsdk:"devdoc"`
	FlowTypeBindings types.Map    `tfsdk:"flow_type_bindings"`
}

type CustomAPIDeploy struct {
	CustomAPIDeployBase
	GeneratorInput ABIGeneratorInput `json:"generatorInput"`
}

type CustomAPIDeployBase struct {
	Name             string            `json:"name"`
	FlowTypeBindings map[string]string `json:"flowTypeBindings"`
}

type ABIGeneratorInput struct {
	Bin    string          `json:"bin"`    // bytecode hex string
	ABI    json.RawMessage `json:"abi"`    // ABI JSON
	DevDoc json.RawMessage `json:"devdoc"` // devdoc JSON (optional)
}

func ConnectorCustomAPIResourceFactory() resource.Resource {
	return &connectorCustomAPIResource{}
}

type connectorCustomAPIResource struct {
	commonResource
}

func (r *connectorCustomAPIResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_connector_custom_api"
}

func (r *connectorCustomAPIResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Deploys a custom API to a connector service using ABI, bytecode, and optional devdoc. This resource generates API endpoints from smart contract interfaces.",
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"service_id": &schema.StringAttribute{
				Required:    true,
				Description: "The ID of the connector service where the custom API will be deployed",
			},
			"environment": &schema.StringAttribute{
				Required:    true,
				Description: "The environment ID where the connector service is located",
			},
			"name": &schema.StringAttribute{
				Required:    true,
				Description: "The name of the custom API",
			},
			"abi": &schema.StringAttribute{
				Required:    true,
				Description: "The ABI JSON as a string or file path. If a file path is provided, the file will be read.",
			},
			"bytecode": &schema.StringAttribute{
				Required:    true,
				Description: "The contract bytecode as a hex string (with or without 0x prefix) or file path. If a file path is provided, the file will be read.",
			},
			"devdoc": &schema.StringAttribute{
				Optional:    true,
				Description: "(Optional) The devdoc JSON as a string or file path. If a file path is provided, the file will be read.",
			},
			"flow_type_bindings": &schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Optional map of flow type to flow name. If not provided, will be auto-discovered from connector flows.",
			},
		},
	}
}

func (r *connectorCustomAPIResource) readFileOrString(value string) (string, error) {
	if _, err := os.Stat(value); err == nil {
		content, err := os.ReadFile(value)
		if err != nil {
			return "", fmt.Errorf("failed to read file %s: %w", value, err)
		}
		return string(content), nil
	}
	// if cannot be read as a file, treat as inline string
	return value, nil
}

func (r *connectorCustomAPIResource) apiPath(data *ConnectorCustomAPIResourceModel, path string) string {
	return fmt.Sprintf("/endpoint/%s/%s/rest%s", data.Environment.ValueString(), data.ServiceID.ValueString(), path)
}

func (r *connectorCustomAPIResource) discoverFlowTypeBindings(ctx context.Context, data *ConnectorCustomAPIResourceModel, diagnostics *diag.Diagnostics) map[string]string {
	// get setup info to discover connector flows and their flow types
	var setupInfo struct {
		ConnectorFlows []ConnectorFlowInfo `json:"connectorFlows"`
	}
	ok, _ := r.apiRequest(ctx, http.MethodGet, r.apiPath(data, "/api/v1/metadata/setup-info"), nil, &setupInfo, diagnostics)
	if !ok {
		return nil
	}

	// query deployed connector flows to get their actual names
	var flowsResult struct {
		Items []struct {
			Labels map[string]string `json:"labels"`
		} `json:"items"`
	}
	ok, _ = r.apiRequest(ctx, http.MethodGet, r.apiPath(data, "/api/v1/connector-flows"), nil, &flowsResult, diagnostics)
	if !ok {
		return nil
	}

	// build flowType -> flowName mapping from deployed flows
	flowTypeBindings := make(map[string]string)
	for _, flow := range flowsResult.Items {
		if flow.Labels["connector_flow"] != "" && flow.Labels["connector_flow_type"] != "" {
			flowTypeBindings[flow.Labels["connector_flow_type"]] = flow.Labels["connector_flow"]
		}
	}

	return flowTypeBindings
}

// isNotFound checks if a status code and diagnostics indicate a "not found" condition
// Returns true if statusCode is 404, or if statusCode is 500 and the error message contains "not found"
func (r *connectorCustomAPIResource) isNotFound(statusCode int, diagnostics *diag.Diagnostics) bool {
	if statusCode == 404 {
		return true
	}
	if statusCode == 500 {
		if diagnostics.HasError() {
			errs := diagnostics.Errors()
			for _, err := range errs {
				errorText := strings.ToLower(err.Summary() + " " + err.Detail())
				if strings.Contains(errorText, "not found") {
					return true
				}
			}
		}
	}
	return false
}

// deployCustomAPI handles the common logic for deploying/updating a custom API
func (r *connectorCustomAPIResource) deployCustomAPI(ctx context.Context, data *ConnectorCustomAPIResourceModel, actionContext string, diagnostics *diag.Diagnostics) bool {
	// verify service exists and is ready
	var api ServiceAPIModel
	api.ID = data.ServiceID.ValueString()
	path := fmt.Sprintf("/api/v1/environments/%s/services/%s", data.Environment.ValueString(), data.ServiceID.ValueString())
	ok, _ := r.apiRequest(ctx, http.MethodGet, path, nil, &api, diagnostics)
	if !ok {
		return false
	}
	if api.Status != "ready" {
		diagnostics.AddError("Service not ready", fmt.Sprintf("Service %s is not ready (status: %s). Please wait for the service to be ready before %s.", data.ServiceID.ValueString(), api.Status, actionContext))
		return false
	}
	if restEndpoint, ok := api.Endpoints["rest"]; !ok || len(restEndpoint.URLS) == 0 {
		diagnostics.AddError("No REST endpoint", fmt.Sprintf("Service %s does not have a REST endpoint", data.ServiceID.ValueString()))
		return false
	}

	// read ABI (file or string)
	abiContent, err := r.readFileOrString(data.ABI.ValueString())
	if err != nil {
		diagnostics.AddError("Failed to read ABI", err.Error())
		return false
	}

	var abiJSON json.RawMessage
	if err := json.Unmarshal([]byte(abiContent), &abiJSON); err != nil {
		diagnostics.AddError("Invalid ABI", fmt.Sprintf("ABI must be valid JSON: %v", err))
		return false
	}

	// read bytecode (file or string)
	bytecodeContent, err := r.readFileOrString(data.Bytecode.ValueString())
	if err != nil {
		diagnostics.AddError("Failed to read bytecode", err.Error())
		return false
	}

	// remove 0x prefix if present
	bytecode := bytecodeContent
	if len(bytecode) > 2 && bytecode[0:2] == "0x" {
		bytecode = bytecode[2:]
	}

	// read devdoc if provided (file or string)
	var devdocJSON json.RawMessage
	if !data.DevDoc.IsNull() && !data.DevDoc.IsUnknown() {
		devdocContent, err := r.readFileOrString(data.DevDoc.ValueString())
		if err != nil {
			diagnostics.AddError("Failed to read devdoc", err.Error())
			return false
		}
		if err := json.Unmarshal([]byte(devdocContent), &devdocJSON); err != nil {
			diagnostics.AddError("Invalid devdoc", fmt.Sprintf("DevDoc must be valid JSON: %v", err))
			return false
		}
	}

	// get flowTypeBindings
	flowTypeBindings := make(map[string]string)
	if !data.FlowTypeBindings.IsNull() && !data.FlowTypeBindings.IsUnknown() {
		elements := data.FlowTypeBindings.Elements()
		for k, v := range elements {
			flowTypeBindings[k] = v.(types.String).ValueString()
		}
	} else {
		discovered := r.discoverFlowTypeBindings(ctx, data, diagnostics)
		if diagnostics.HasError() {
			return false
		}
		flowTypeBindings = discovered
	}

	// delete existing API if it exists (idempotent)
	deleteDiags := &diag.Diagnostics{}
	_, statusCode := r.apiRequest(ctx, http.MethodDelete, r.apiPath(data, fmt.Sprintf("/api/v1/apis/%s", data.Name.ValueString())), nil, nil, deleteDiags, Allow404())
	// Ignore "not found" errors (API doesn't exist yet, which is fine for idempotent delete)
	if r.isNotFound(statusCode, deleteDiags) {
		// API doesn't exist, which is fine - continue with deploy
		// Clear any errors from deleteDiags since we're treating "not found" as success
		*deleteDiags = diag.Diagnostics{}
	}

	// deploy the custom API
	deployBody := CustomAPIDeploy{
		CustomAPIDeployBase: CustomAPIDeployBase{
			Name:             data.Name.ValueString(),
			FlowTypeBindings: flowTypeBindings,
		},
		GeneratorInput: ABIGeneratorInput{
			Bin:    bytecode,
			ABI:    abiJSON,
			DevDoc: devdocJSON,
		},
	}

	ok, _ = r.apiRequest(ctx, http.MethodPost, r.apiPath(data, "/api/v1/metadata/custom-api"), deployBody, nil, diagnostics)
	return ok
}

func (r *connectorCustomAPIResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ConnectorCustomAPIResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if !r.deployCustomAPI(ctx, &data, "deploying custom API", &resp.Diagnostics) {
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s:%s", data.ServiceID.ValueString(), data.Name.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *connectorCustomAPIResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ConnectorCustomAPIResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	// verify service exists and is ready
	var api ServiceAPIModel
	api.ID = data.ServiceID.ValueString()
	path := fmt.Sprintf("/api/v1/environments/%s/services/%s", data.Environment.ValueString(), data.ServiceID.ValueString())
	ok, _ := r.apiRequest(ctx, http.MethodGet, path, nil, &api, &resp.Diagnostics)
	if !ok {
		// If service doesn't exist, remove from state
		resp.State.RemoveResource(ctx)
		return
	}
	if api.Status != "ready" {
		resp.State.RemoveResource(ctx)
		return
	}
	if restEndpoint, ok := api.Endpoints["rest"]; !ok || len(restEndpoint.URLS) == 0 {
		resp.State.RemoveResource(ctx)
		return
	}

	// verify the custom API exists by querying it by name
	var customAPI interface{}
	// Use a local diagnostics to capture the error message without adding it to the response
	localDiags := &diag.Diagnostics{}
	ok, statusCode := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data, fmt.Sprintf("/api/v1/apis/%s", data.Name.ValueString())), nil, &customAPI, localDiags, Allow404())
	if r.isNotFound(statusCode, localDiags) {
		// Custom API doesn't exist, remove from state
		resp.State.RemoveResource(ctx)
		return
	}
	if statusCode == 500 {
		// 500 error but not "not found", don't remove from state
		return
	}
	if !ok {
		// Error occurred, but don't remove from state on error
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *connectorCustomAPIResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ConnectorCustomAPIResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)

	if !r.deployCustomAPI(ctx, &data, "updating custom API", &resp.Diagnostics) {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *connectorCustomAPIResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ConnectorCustomAPIResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	// verify service exists (optional check for delete)
	var api ServiceAPIModel
	api.ID = data.ServiceID.ValueString()
	path := fmt.Sprintf("/api/v1/environments/%s/services/%s", data.Environment.ValueString(), data.ServiceID.ValueString())
	ok, _ := r.apiRequest(ctx, http.MethodGet, path, nil, &api, &resp.Diagnostics)
	if !ok {
		// if service doesn't exist, consider it already deleted
		return
	}
	if api.Status != "ready" {
		return
	}
	if restEndpoint, ok := api.Endpoints["rest"]; !ok || len(restEndpoint.URLS) == 0 {
		return
	}

	// delete the custom API
	// use a local diagnostics since we don't want to fail on delete errors
	deleteDiags := &diag.Diagnostics{}
	ok, statusCode := r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data, fmt.Sprintf("/api/v1/apis/%s", data.Name.ValueString())), nil, nil, deleteDiags, Allow404())
	if r.isNotFound(statusCode, deleteDiags) {
		return
	}
	if !ok {
		// log error but don't fail - API might already be deleted
		if deleteDiags.HasError() {
			errs := deleteDiags.Errors()
			if len(errs) > 0 {
				tflog.Warn(ctx, fmt.Sprintf("Failed to delete custom API: %s: %s", errs[0].Summary(), errs[0].Detail()))
			}
		}
	}
}
