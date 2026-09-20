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
	"net/url"
	"reflect"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	evidenceSourceTypeApproval       = "approval"
	evidenceSourceTypeServiceRequest = "serviceRequest"
	evidenceSourceTypeWorkflow       = "workflow"
	evidenceSourceTypeAttachment     = "attachment"
)

type PMSEvidenceSourceResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Environment        types.String `tfsdk:"environment"`
	Service            types.String `tfsdk:"service"`
	Name               types.String `tfsdk:"name"`
	Description        types.String `tfsdk:"description"`
	SchemaJSON         types.String `tfsdk:"schema_json"`
	PayloadJSONata     types.String `tfsdk:"payload_jsonata"`
	AttestationJSONata types.String `tfsdk:"attestation_jsonata"`
	Parameters         types.List   `tfsdk:"parameter"`
	Type               types.String `tfsdk:"type"`
	Approval           types.Object `tfsdk:"approval"`
	ServiceRequest     types.Object `tfsdk:"service_request"`
	Workflow           types.Object `tfsdk:"workflow"`
}

// PMSTypedDataV4ResponseAPIModel describes the EIP-712 document an approver signs for a
// response: the struct types declared inline, the primary type, and the JSONata that builds
// the message from {request, decision}.
type PMSTypedDataV4ResponseAPIModel struct {
	Types          map[string][]PMSEIP712TypeMemberAPIModel `json:"types"`
	PrimaryType    string                                   `json:"primaryType"`
	MessageMapping *JSONataMappingAPI                       `json:"messageMapping,omitempty"`
}

type PMSEIP712TypeMemberAPIModel struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type PMSApprovalResponseAPIModel struct {
	Type        string                          `json:"type"`
	TypedDataV4 *PMSTypedDataV4ResponseAPIModel `json:"typedDataV4,omitempty"`
}

type PMSApprovalResponsesAPIModel struct {
	Approve *PMSApprovalResponseAPIModel `json:"approve,omitempty"`
	Reject  *PMSApprovalResponseAPIModel `json:"reject,omitempty"`
}

type PMSApprovalEvidenceSourceAPIModel struct {
	Responses *PMSApprovalResponsesAPIModel `json:"responses,omitempty"`
}

type PMSServiceRequestDynamicOptionsAPIModel struct {
	Path     *JSONataMappingRaw `json:"path,omitempty"`
	Method   *JSONataMappingRaw `json:"method,omitempty"`
	Endpoint *JSONataMappingRaw `json:"endpoint,omitempty"`
	Body     *JSONataMappingRaw `json:"body,omitempty"`
}

// JSONataMappingRaw is a bare JSONata expression string on the wire (the policy manager's
// ServiceRequestDynamicOptions fields), as opposed to the {jsonata: "..."} wrapper.
type JSONataMappingRaw string

type PMSServiceRequestEvidenceSourceAPIModel struct {
	Service        string                                   `json:"service,omitempty"`
	Type           string                                   `json:"type,omitempty"`
	Options        map[string]interface{}                   `json:"options,omitempty"`
	DynamicOptions *PMSServiceRequestDynamicOptionsAPIModel `json:"dynamicOptions,omitempty"`
}

type PMSWorkflowEvidenceSourceAPIModel struct {
	Workflow            string                 `json:"workflow,omitempty"`
	Operation           string                 `json:"operation,omitempty"`
	TransactionTemplate map[string]interface{} `json:"transactionTemplate,omitempty"`
}

// PMSEvidenceSourceAPIModel is an evidence source on the wire. Schema is the JSON Schema of
// one item of the evidence the source produces: authored for a serviceRequest, workflow or
// attachment source, derived by the server from the typed data of an approval source's
// responses. The two mappings are JSONata over {request, decision, body} that select the
// evidence payload and attestation out of what the source receives; an approval source
// has neither.
type PMSEvidenceSourceAPIModel struct {
	ID                 string                                   `json:"id,omitempty"`
	Name               string                                   `json:"name,omitempty"`
	Description        string                                   `json:"description,omitempty"`
	Schema             map[string]interface{}                   `json:"schema,omitempty"`
	PayloadMapping     *JSONataMappingAPI                       `json:"payloadMapping,omitempty"`
	AttestationMapping *JSONataMappingAPI                       `json:"attestationMapping,omitempty"`
	Parameters         []PMSParameterAPIModel                   `json:"parameters,omitempty"`
	Type               string                                   `json:"type,omitempty"`
	Approval           *PMSApprovalEvidenceSourceAPIModel       `json:"approval,omitempty"`
	ServiceRequest     *PMSServiceRequestEvidenceSourceAPIModel `json:"serviceRequest,omitempty"`
	Workflow           *PMSWorkflowEvidenceSourceAPIModel       `json:"workflow,omitempty"`
	Created            *time.Time                               `json:"created,omitempty"`
	Updated            *time.Time                               `json:"updated,omitempty"`
}

var esResponseAttrTypes = map[string]attr.Type{
	"types_json":      types.StringType,
	"primary_type":    types.StringType,
	"message_jsonata": types.StringType,
}

var esApprovalAttrTypes = map[string]attr.Type{
	"approve": types.ObjectType{AttrTypes: esResponseAttrTypes},
	"reject":  types.ObjectType{AttrTypes: esResponseAttrTypes},
}

var esDynamicOptionsAttrTypes = map[string]attr.Type{
	"path_jsonata":     types.StringType,
	"method_jsonata":   types.StringType,
	"endpoint_jsonata": types.StringType,
	"body_jsonata":     types.StringType,
}

var esServiceRequestAttrTypes = map[string]attr.Type{
	"service":         types.StringType,
	"type":            types.StringType,
	"options_json":    types.StringType,
	"dynamic_options": types.ObjectType{AttrTypes: esDynamicOptionsAttrTypes},
}

var esWorkflowAttrTypes = map[string]attr.Type{
	"workflow":                  types.StringType,
	"operation":                 types.StringType,
	"transaction_template_json": types.StringType,
}

func PMSEvidenceSourceResourceFactory() resource.Resource {
	return &pms_evidenceSourceResource{}
}

type pms_evidenceSourceResource struct {
	commonResource
}

func (r *pms_evidenceSourceResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_pms_evidence_source"
}

func approvalResponseSchema(description string) *schema.SingleNestedAttribute {
	return &schema.SingleNestedAttribute{
		Optional:    true,
		Description: description + " The approver signs an EIP-712 (TypedDataV4) document built from these attributes; a 'decisionId' string member is added to the primary type.",
		Attributes: map[string]schema.Attribute{
			"types_json": &schema.StringAttribute{
				Required:    true,
				Description: "The EIP-712 struct types (use jsonencode), keyed by type name, each a list of {name, type} members",
			},
			"primary_type": &schema.StringAttribute{
				Required:    true,
				Description: "The EIP-712 primary type of the message; must be declared in types_json",
			},
			"message_jsonata": &schema.StringAttribute{
				Optional:    true,
				Description: "JSONata building the message from the evaluation context {request, decision}",
			},
		},
	}
}

func (r *pms_evidenceSourceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Policy Manager evidence source: a reusable, policy-independent definition of where a piece of evidence comes from, how it is requested, and the schema of what it produces. A policy uses one through a kaleido_platform_pms_policy_evidence_source_binding. The source's JSONata evaluates against {request, decision}, where 'request' is the object the policy's evidence 'request' block produced for the slot and 'decision' carries {id, policy: {id, name, version}, evidence, idempotencyKey}.",
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				Description:   "The evidence source ID assigned by the server. This is the value to bind to: the evidence_source_id of a kaleido_platform_pms_policy_evidence_source_binding.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"environment": &schema.StringAttribute{
				Required:      true,
				Description:   "Environment ID",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"service": &schema.StringAttribute{
				Required:      true,
				Description:   "Policy Manager service ID",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": &schema.StringAttribute{
				Required:      true,
				Description:   "Unique name of the evidence source. Immutable after create.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"description": &schema.StringAttribute{
				Optional:    true,
				Description: "Description of the evidence source",
			},
			"schema_json": &schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "JSON Schema (use jsonencode) of one item of the evidence this source produces, read by the policy composer for the slots bound to it. Set it for a 'serviceRequest' or 'workflow' source, and always for an 'attachment' source. Leave it unset for an 'approval' source: the server derives it from the typed data of the responses, and it is reported here.",
			},
			"payload_jsonata": &schema.StringAttribute{
				Optional:    true,
				Description: "JSONata selecting the evidence payload out of what the source receives, evaluated against {request, decision, body}. Not permitted on an 'approval' source.",
			},
			"attestation_jsonata": &schema.StringAttribute{
				Optional:    true,
				Description: "JSONata selecting the attestation out of what the source receives, evaluated against {request, decision, body}. Not permitted on an 'approval' source.",
			},
			"parameter": pmsParameterNestedSchema("The parameters the source needs to gather evidence. A policy slot bound to the source supplies a value expression for each in its evidence 'request'; a parameter with a default may be left out. An 'attachment' source requests nothing and so declares no parameters."),
			"type": &schema.StringAttribute{
				Required:      true,
				Description:   "The type of evidence source: 'approval', 'serviceRequest', 'workflow' or 'attachment'. An attachment source requests nothing; it describes evidence that is seeded by a matcher, attached manually or supplied late-bound, and needs only schema_json. Immutable after create.",
				Validators:    []validator.String{stringvalidator.OneOf(evidenceSourceTypeApproval, evidenceSourceTypeServiceRequest, evidenceSourceTypeWorkflow, evidenceSourceTypeAttachment)},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"approval": &schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Configuration for a source of type 'approval': the responses an approver may give, and the document each one signs. Who is asked comes from the binding's 'attesters'.",
				Attributes: map[string]schema.Attribute{
					"approve": approvalResponseSchema("The response that approves the request."),
					"reject":  approvalResponseSchema("The response that rejects the request."),
				},
			},
			"service_request": &schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Configuration for a source of type 'serviceRequest': evidence is fetched by calling a platform service, acting as the binding's 'run_as' application.",
				Attributes: map[string]schema.Attribute{
					"service": &schema.StringAttribute{
						Required:    true,
						Description: "The name of the service to invoke to gather the evidence",
					},
					"type": &schema.StringAttribute{
						Optional:    true,
						Description: "The service type the named service is expected to be, e.g. 'WalletManagerService'",
					},
					"options_json": &schema.StringAttribute{
						Optional:    true,
						Description: "Static request options (use jsonencode): method, path, endpoint, headers, query, body, allowFailStatus",
					},
					"dynamic_options": &schema.SingleNestedAttribute{
						Optional:    true,
						Description: "JSONata expressions for request options, evaluated against {request, decision}. Take precedence over the static options.",
						Attributes: map[string]schema.Attribute{
							"path_jsonata":     &schema.StringAttribute{Optional: true, Description: "JSONata producing the request path"},
							"method_jsonata":   &schema.StringAttribute{Optional: true, Description: "JSONata producing the HTTP method"},
							"endpoint_jsonata": &schema.StringAttribute{Optional: true, Description: "JSONata producing the service endpoint"},
							"body_jsonata":     &schema.StringAttribute{Optional: true, Description: "JSONata producing the request body"},
						},
					},
				},
			},
			"workflow": &schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Configuration for a source of type 'workflow': evidence is gathered by submitting a workflow transaction, acting as the binding's 'run_as' application.",
				Attributes: map[string]schema.Attribute{
					"workflow": &schema.StringAttribute{
						Optional:    true,
						Description: "The workflow to dispatch when sourcing evidence",
					},
					"operation": &schema.StringAttribute{
						Optional:    true,
						Description: "The workflow operation to dispatch when sourcing evidence",
					},
					"transaction_template_json": &schema.StringAttribute{
						Optional:    true,
						Description: "Transaction template (use jsonencode) describing how to submit the transaction: workflow, operation and a 'jsonata' input mapping evaluated against {request, decision}",
					},
				},
			},
		},
	}
}

func (r *pms_evidenceSourceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.commonResource.Configure(ctx, req, resp)
}

func (r *pms_evidenceSourceResource) listPath(data *PMSEvidenceSourceResourceModel) string {
	return fmt.Sprintf("/endpoint/%s/%s/rest/api/v2/evidence-sources", data.Environment.ValueString(), data.Service.ValueString())
}

func (r *pms_evidenceSourceResource) instancePath(data *PMSEvidenceSourceResourceModel) string {
	return fmt.Sprintf("%s/%s", r.listPath(data), url.PathEscape(data.ID.ValueString()))
}

func isSet(v attr.Value) bool {
	return !v.IsNull() && !v.IsUnknown()
}

// validateEvidenceSourceBlocks checks that the configuration block matching type is the
// one - and the only one - that is set, and that the attributes whose meaning depends on
// the type are consistent with it.
func validateEvidenceSourceBlocks(data *PMSEvidenceSourceResourceModel, diagnostics *diag.Diagnostics) {
	sourceType := data.Type.ValueString()
	set := map[string]bool{
		evidenceSourceTypeApproval:       isSet(data.Approval),
		evidenceSourceTypeServiceRequest: isSet(data.ServiceRequest),
		evidenceSourceTypeWorkflow:       isSet(data.Workflow),
	}
	blockName := map[string]string{
		evidenceSourceTypeApproval:       "approval",
		evidenceSourceTypeServiceRequest: "service_request",
		evidenceSourceTypeWorkflow:       "workflow",
	}
	for t, isSet := range set {
		if isSet && t != sourceType {
			diagnostics.AddError("Invalid configuration", fmt.Sprintf("the %s block must not be set when type is %q; set only the block matching type", blockName[t], sourceType))
			return
		}
	}
	if sourceType != evidenceSourceTypeAttachment && !set[sourceType] {
		diagnostics.AddError("Invalid configuration", fmt.Sprintf("the %s block must be set when type is %q", blockName[sourceType], sourceType))
		return
	}
	switch sourceType {
	case evidenceSourceTypeApproval:
		if isSet(data.SchemaJSON) {
			diagnostics.AddError("Invalid configuration", "schema_json must not be set when type is \"approval\": the schema is derived from the typed data of the responses")
		}
		if isSet(data.PayloadJSONata) || isSet(data.AttestationJSONata) {
			diagnostics.AddError("Invalid configuration", "payload_jsonata and attestation_jsonata must not be set when type is \"approval\"")
		}
	case evidenceSourceTypeAttachment:
		if !isSet(data.SchemaJSON) {
			diagnostics.AddError("Invalid configuration", "schema_json must be set when type is \"attachment\"")
		}
		if isSet(data.Parameters) && len(data.Parameters.Elements()) > 0 {
			diagnostics.AddError("Invalid configuration", "parameter blocks must not be set when type is \"attachment\": an attachment source requests nothing")
		}
	}
}

// objectAttr reads an optional nested object out of a terraform object's attribute map.
func objectAttr(attrs map[string]attr.Value, name string) (types.Object, bool) {
	val, ok := attrs[name]
	if !ok || val.IsNull() || val.IsUnknown() {
		return types.Object{}, false
	}
	obj, ok := val.(types.Object)
	return obj, ok
}

func jsonAttr(attrs map[string]attr.Value, name string, diagnostics *diag.Diagnostics) map[string]interface{} {
	raw := stringAttr(attrs, name)
	if raw == "" {
		return nil
	}
	var out map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		diagnostics.AddError("Invalid JSON", fmt.Sprintf("%s is not a JSON object: %s", name, err))
		return nil
	}
	return out
}

// jsonToAttr renders a decoded JSON value for a *_json attribute. A nil or empty map or
// slice is absent, not the string "null": the server omits the field, so the attribute
// stays null and matches an operator who never set it.
func jsonToAttr(value interface{}) types.String {
	if value == nil {
		return types.StringNull()
	}
	if rv := reflect.ValueOf(value); (rv.Kind() == reflect.Map || rv.Kind() == reflect.Slice) && rv.Len() == 0 {
		return types.StringNull()
	}
	b, err := json.Marshal(value)
	if err != nil {
		return types.StringNull()
	}
	return types.StringValue(string(b))
}

func approvalResponseToAPI(val attr.Value, diagnostics *diag.Diagnostics) *PMSApprovalResponseAPIModel {
	if val == nil || val.IsNull() || val.IsUnknown() {
		return nil
	}
	obj, ok := val.(types.Object)
	if !ok {
		return nil
	}
	attrs := obj.Attributes()
	typedData := &PMSTypedDataV4ResponseAPIModel{PrimaryType: stringAttr(attrs, "primary_type")}
	if raw := stringAttr(attrs, "types_json"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &typedData.Types); err != nil {
			diagnostics.AddError("Invalid JSON", fmt.Sprintf("types_json is not a map of EIP-712 types: %s", err))
			return nil
		}
	}
	if jsonata := stringAttr(attrs, "message_jsonata"); jsonata != "" {
		typedData.MessageMapping = &JSONataMappingAPI{JSONata: jsonata}
	}
	return &PMSApprovalResponseAPIModel{Type: "TypedDataV4", TypedDataV4: typedData}
}

func approvalResponseToData(response *PMSApprovalResponseAPIModel, diagnostics *diag.Diagnostics) types.Object {
	if response == nil || response.TypedDataV4 == nil {
		return types.ObjectNull(esResponseAttrTypes)
	}
	obj, diags := types.ObjectValue(esResponseAttrTypes, map[string]attr.Value{
		"types_json":      jsonToAttr(response.TypedDataV4.Types),
		"primary_type":    optionalString(response.TypedDataV4.PrimaryType),
		"message_jsonata": jsonataAttr(response.TypedDataV4.MessageMapping),
	})
	diagnostics.Append(diags...)
	return obj
}

func esApprovalToAPI(approval types.Object, diagnostics *diag.Diagnostics) *PMSApprovalEvidenceSourceAPIModel {
	if approval.IsNull() || approval.IsUnknown() {
		return nil
	}
	attrs := approval.Attributes()
	return &PMSApprovalEvidenceSourceAPIModel{Responses: &PMSApprovalResponsesAPIModel{
		Approve: approvalResponseToAPI(attrs["approve"], diagnostics),
		Reject:  approvalResponseToAPI(attrs["reject"], diagnostics),
	}}
}

func esApprovalToData(approval *PMSApprovalEvidenceSourceAPIModel, diagnostics *diag.Diagnostics) types.Object {
	if approval == nil || approval.Responses == nil {
		return types.ObjectNull(esApprovalAttrTypes)
	}
	obj, diags := types.ObjectValue(esApprovalAttrTypes, map[string]attr.Value{
		"approve": approvalResponseToData(approval.Responses.Approve, diagnostics),
		"reject":  approvalResponseToData(approval.Responses.Reject, diagnostics),
	})
	diagnostics.Append(diags...)
	return obj
}

func rawJSONata(attrs map[string]attr.Value, name string) *JSONataMappingRaw {
	if v := stringAttr(attrs, name); v != "" {
		raw := JSONataMappingRaw(v)
		return &raw
	}
	return nil
}

func rawJSONataAttr(raw *JSONataMappingRaw) types.String {
	if raw == nil || *raw == "" {
		return types.StringNull()
	}
	return types.StringValue(string(*raw))
}

func esServiceRequestToAPI(serviceRequest types.Object, diagnostics *diag.Diagnostics) *PMSServiceRequestEvidenceSourceAPIModel {
	if serviceRequest.IsNull() || serviceRequest.IsUnknown() {
		return nil
	}
	attrs := serviceRequest.Attributes()
	result := &PMSServiceRequestEvidenceSourceAPIModel{
		Service: stringAttr(attrs, "service"),
		Type:    stringAttr(attrs, "type"),
		Options: jsonAttr(attrs, "options_json", diagnostics),
	}
	if dyn, ok := objectAttr(attrs, "dynamic_options"); ok {
		dynAttrs := dyn.Attributes()
		result.DynamicOptions = &PMSServiceRequestDynamicOptionsAPIModel{
			Path:     rawJSONata(dynAttrs, "path_jsonata"),
			Method:   rawJSONata(dynAttrs, "method_jsonata"),
			Endpoint: rawJSONata(dynAttrs, "endpoint_jsonata"),
			Body:     rawJSONata(dynAttrs, "body_jsonata"),
		}
	}
	return result
}

func esServiceRequestToData(sr *PMSServiceRequestEvidenceSourceAPIModel, diagnostics *diag.Diagnostics) types.Object {
	if sr == nil {
		return types.ObjectNull(esServiceRequestAttrTypes)
	}
	dynamic := types.ObjectNull(esDynamicOptionsAttrTypes)
	if sr.DynamicOptions != nil {
		obj, diags := types.ObjectValue(esDynamicOptionsAttrTypes, map[string]attr.Value{
			"path_jsonata":     rawJSONataAttr(sr.DynamicOptions.Path),
			"method_jsonata":   rawJSONataAttr(sr.DynamicOptions.Method),
			"endpoint_jsonata": rawJSONataAttr(sr.DynamicOptions.Endpoint),
			"body_jsonata":     rawJSONataAttr(sr.DynamicOptions.Body),
		})
		diagnostics.Append(diags...)
		dynamic = obj
	}
	obj, diags := types.ObjectValue(esServiceRequestAttrTypes, map[string]attr.Value{
		"service":         optionalString(sr.Service),
		"type":            optionalString(sr.Type),
		"options_json":    jsonToAttr(sr.Options),
		"dynamic_options": dynamic,
	})
	diagnostics.Append(diags...)
	return obj
}

func esWorkflowToAPI(workflow types.Object, diagnostics *diag.Diagnostics) *PMSWorkflowEvidenceSourceAPIModel {
	if workflow.IsNull() || workflow.IsUnknown() {
		return nil
	}
	attrs := workflow.Attributes()
	return &PMSWorkflowEvidenceSourceAPIModel{
		Workflow:            stringAttr(attrs, "workflow"),
		Operation:           stringAttr(attrs, "operation"),
		TransactionTemplate: jsonAttr(attrs, "transaction_template_json", diagnostics),
	}
}

func esWorkflowToData(wf *PMSWorkflowEvidenceSourceAPIModel, diagnostics *diag.Diagnostics) types.Object {
	if wf == nil {
		return types.ObjectNull(esWorkflowAttrTypes)
	}
	obj, diags := types.ObjectValue(esWorkflowAttrTypes, map[string]attr.Value{
		"workflow":                  optionalString(wf.Workflow),
		"operation":                 optionalString(wf.Operation),
		"transaction_template_json": jsonToAttr(wf.TransactionTemplate),
	})
	diagnostics.Append(diags...)
	return obj
}

func (r *pms_evidenceSourceResource) toAPI(data *PMSEvidenceSourceResourceModel, api *PMSEvidenceSourceAPIModel, diagnostics *diag.Diagnostics) {
	validateEvidenceSourceBlocks(data, diagnostics)
	if diagnostics.HasError() {
		return
	}
	api.Name = data.Name.ValueString()
	api.Description = data.Description.ValueString()
	if isSet(data.SchemaJSON) {
		api.Schema = jsonAttr(map[string]attr.Value{"schema_json": data.SchemaJSON}, "schema_json", diagnostics)
	}
	if jsonata := data.PayloadJSONata.ValueString(); jsonata != "" {
		api.PayloadMapping = &JSONataMappingAPI{JSONata: jsonata}
	}
	if jsonata := data.AttestationJSONata.ValueString(); jsonata != "" {
		api.AttestationMapping = &JSONataMappingAPI{JSONata: jsonata}
	}
	api.Parameters = pmsParametersToAPI(data.Parameters, diagnostics)
	api.Type = data.Type.ValueString()
	api.Approval = esApprovalToAPI(data.Approval, diagnostics)
	api.ServiceRequest = esServiceRequestToAPI(data.ServiceRequest, diagnostics)
	api.Workflow = esWorkflowToAPI(data.Workflow, diagnostics)
}

func (r *pms_evidenceSourceResource) toData(api *PMSEvidenceSourceAPIModel, data *PMSEvidenceSourceResourceModel, diagnostics *diag.Diagnostics) {
	data.ID = types.StringValue(api.ID)
	if api.Name != "" {
		data.Name = types.StringValue(api.Name)
	}
	data.Description = optionalString(api.Description)
	data.SchemaJSON = jsonAnyToAttr(data.SchemaJSON, api.Schema)
	data.PayloadJSONata = jsonataAttr(api.PayloadMapping)
	data.AttestationJSONata = jsonataAttr(api.AttestationMapping)
	data.Parameters = pmsParametersToData(api.Parameters, data.Parameters, diagnostics)
	// The wire value of an FFEnum is lowercased; keep the configured spelling.
	if data.Type.IsNull() || data.Type.IsUnknown() {
		data.Type = types.StringValue(api.Type)
	}
	data.Approval = types.ObjectNull(esApprovalAttrTypes)
	data.ServiceRequest = types.ObjectNull(esServiceRequestAttrTypes)
	data.Workflow = types.ObjectNull(esWorkflowAttrTypes)
	switch {
	case api.Approval != nil:
		data.Approval = esApprovalToData(api.Approval, diagnostics)
	case api.ServiceRequest != nil:
		data.ServiceRequest = esServiceRequestToData(api.ServiceRequest, diagnostics)
	case api.Workflow != nil:
		data.Workflow = esWorkflowToData(api.Workflow, diagnostics)
	}
}

func (r *pms_evidenceSourceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PMSEvidenceSourceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var api PMSEvidenceSourceAPIModel
	r.toAPI(&data, &api, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	ok, _ := r.apiRequest(ctx, http.MethodPost, r.listPath(&data), &api, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	r.toData(&api, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *pms_evidenceSourceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PMSEvidenceSourceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var api PMSEvidenceSourceAPIModel
	ok, status := r.apiRequest(ctx, http.MethodGet, r.instancePath(&data), nil, &api, &resp.Diagnostics, Allow404())
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

func (r *pms_evidenceSourceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data PMSEvidenceSourceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// A PATCH cannot clear a field, so the whole source is replaced with a PUT: an
	// attribute removed from the configuration is then removed from the server too.
	var desired PMSEvidenceSourceAPIModel
	r.toAPI(&data, &desired, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	var api PMSEvidenceSourceAPIModel
	ok, _ := r.apiRequest(ctx, http.MethodPut, r.instancePath(&data), &desired, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	r.toData(&api, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *pms_evidenceSourceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PMSEvidenceSourceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.instancePath(&data), nil, nil, &resp.Diagnostics, Allow404())
}
