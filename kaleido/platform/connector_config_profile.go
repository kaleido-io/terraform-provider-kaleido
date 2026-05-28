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

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

type ConnectorConfigProfileResourceModel struct {
	ID           types.String              `tfsdk:"id"`
	Environment  types.String              `tfsdk:"environment"`
	Service      types.String              `tfsdk:"service"`
	Name         types.String              `tfsdk:"name"`
	ConfigType   types.String              `tfsdk:"config_type"`
	Description  types.String              `tfsdk:"description"`
	ValueJSON    jsonNullStrippedStringVal `tfsdk:"value_json"`
	ConfigTypeID types.String              `tfsdk:"config_type_id"`
}

type ConnectorConfigProfileAPIModel struct {
	ID           string         `json:"id,omitempty"`
	Name         string         `json:"name,omitempty"`
	ConfigType   string         `json:"configType,omitempty"`
	ConfigTypeID string         `json:"configTypeId,omitempty"`
	Description  string         `json:"description,omitempty"`
	Value        map[string]any `json:"value"`
}

func ConnectorConfigProfileResourceFactory() resource.Resource {
	return &connectorConfigProfileResource{}
}

type connectorConfigProfileResource struct {
	commonResource
}

func (r *connectorConfigProfileResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_connector_config_profile"
}

func (r *connectorConfigProfileResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A config profile on a connector service. Bound to a config type, validated against that type's JSON Schema by the server. The value is supplied as a JSON-encoded string.",
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
				Description:   "Connector service ID",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": &schema.StringAttribute{
				Required:      true,
				Description:   "Name of the config profile. By convention matches the config type name (e.g. evm.confirmations).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"config_type": &schema.StringAttribute{
				Required:    true,
				Description: "Name of the config type this profile binds to (e.g. evm.confirmations).",
			},
			"description": &schema.StringAttribute{
				Optional:    true,
				Description: "Optional description of the config profile.",
			},
			"value_json": &schema.StringAttribute{
				Required:    true,
				CustomType:  jsonNullStrippedStringType{},
				Description: "JSON-encoded profile value. Must validate against the bound config type's JSON Schema. Null fields are stripped before submission (upstream schemas treat absence as 'use default'; explicit null fails validation); a custom-type semantic equality compares values as null-stripped JSON so jsonencode() of typed objects doesn't show spurious drift.",
			},
			"config_type_id": &schema.StringAttribute{
				Computed:    true,
				Description: "Resolved config type ID after the profile is created.",
			},
		},
	}
}

func (r *connectorConfigProfileResource) apiPath(data *ConnectorConfigProfileResourceModel, idOrName string) string {
	return fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/config-profiles/%s",
		data.Environment.ValueString(), data.Service.ValueString(), idOrName)
}

func (r *connectorConfigProfileResource) toAPI(data *ConnectorConfigProfileResourceModel, api *ConnectorConfigProfileAPIModel, diagnostics *diag.Diagnostics) {
	api.Name = data.Name.ValueString()
	api.ConfigType = data.ConfigType.ValueString()
	if !data.Description.IsNull() {
		api.Description = data.Description.ValueString()
	}
	if !data.ValueJSON.IsNull() && data.ValueJSON.ValueString() != "" {
		var value map[string]any
		if err := json.Unmarshal([]byte(data.ValueJSON.ValueString()), &value); err != nil {
			diagnostics.AddError("Invalid value_json", fmt.Sprintf("config profile value_json must be a JSON object: %v", err))
			return
		}
		// Upstream JSON Schemas treat key absence as "use default" but reject
		// explicit nulls for non-nullable fields. Terraform's jsonencode() of a
		// typed object emits null for every unset optional() leaf, so strip
		// nulls recursively before sending to the API.
		stripped, _ := stripJSONNulls(value).(map[string]any)
		if stripped == nil {
			stripped = map[string]any{}
		}
		api.Value = stripped
	} else {
		api.Value = map[string]any{}
	}
}

func (r *connectorConfigProfileResource) toData(api *ConnectorConfigProfileAPIModel, data *ConnectorConfigProfileResourceModel, diagnostics *diag.Diagnostics) {
	if api.ID != "" {
		data.ID = types.StringValue(api.ID)
	}
	if api.ConfigTypeID != "" {
		data.ConfigTypeID = types.StringValue(api.ConfigTypeID)
	}
	if api.Value != nil {
		valueBytes, err := json.Marshal(api.Value)
		if err != nil {
			diagnostics.AddError("JSON Marshal Error", fmt.Sprintf("failed to marshal config profile value: %v", err))
			return
		}
		data.ValueJSON = newJSONNullStrippedString(string(valueBytes))
	}
}

func (r *connectorConfigProfileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ConnectorConfigProfileResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var api ConnectorConfigProfileAPIModel
	r.toAPI(&data, &api, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	ok, _ := r.apiRequest(ctx, http.MethodPut, r.apiPath(&data, data.Name.ValueString()), &api, &api, &resp.Diagnostics)
	if !ok {
		return
	}
	r.toData(&api, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *connectorConfigProfileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ConnectorConfigProfileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var api ConnectorConfigProfileAPIModel
	ok, status := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data, data.Name.ValueString()), nil, &api, &resp.Diagnostics, Allow404())
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

func (r *connectorConfigProfileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ConnectorConfigProfileResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var api ConnectorConfigProfileAPIModel
	r.toAPI(&data, &api, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	ok, _ := r.apiRequest(ctx, http.MethodPut, r.apiPath(&data, data.Name.ValueString()), &api, &api, &resp.Diagnostics)
	if !ok {
		return
	}
	r.toData(&api, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *connectorConfigProfileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ConnectorConfigProfileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data, data.Name.ValueString()), nil, nil, &resp.Diagnostics, Allow404())
}

// stripJSONNulls recursively removes keys whose value is nil from maps,
// and nil entries from slices. Returned slices/maps are new copies; scalar
// values are returned unchanged.
func stripJSONNulls(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			if val == nil {
				continue
			}
			out[k] = stripJSONNulls(val)
		}
		return out
	case []any:
		out := make([]any, 0, len(t))
		for _, val := range t {
			if val == nil {
				continue
			}
			out = append(out, stripJSONNulls(val))
		}
		return out
	default:
		return v
	}
}

func canonicalizeStrippedJSON(s string) (string, error) {
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return "", err
	}
	out, err := json.Marshal(stripJSONNulls(v))
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// jsonNullStrippedStringType is a Terraform Plugin Framework custom string
// type whose semantic equality compares two JSON-encoded values as equal when
// they differ only by null fields. This lets the schema attribute remain
// Required while accepting a planned value (e.g. jsonencode of a typed object
// with unset optional() leaves) that the provider stores in a null-stripped
// canonical form — without triggering Terraform's "planned value does not
// match config value" or "inconsistent result after apply" guards.
type jsonNullStrippedStringType struct {
	basetypes.StringType
}

func (t jsonNullStrippedStringType) Equal(o attr.Type) bool {
	other, ok := o.(jsonNullStrippedStringType)
	if !ok {
		return false
	}
	return t.StringType.Equal(other.StringType)
}

func (t jsonNullStrippedStringType) String() string {
	return "jsonNullStrippedStringType"
}

func (t jsonNullStrippedStringType) ValueType(_ context.Context) attr.Value {
	return jsonNullStrippedStringVal{}
}

func (t jsonNullStrippedStringType) ValueFromString(_ context.Context, in basetypes.StringValue) (basetypes.StringValuable, diag.Diagnostics) {
	return jsonNullStrippedStringVal{StringValue: in}, nil
}

func (t jsonNullStrippedStringType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	attrValue, err := t.StringType.ValueFromTerraform(ctx, in)
	if err != nil {
		return nil, err
	}
	stringValue, ok := attrValue.(basetypes.StringValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type %T returned by StringType.ValueFromTerraform", attrValue)
	}
	stringValuable, diags := t.ValueFromString(ctx, stringValue)
	if diags.HasError() {
		return nil, fmt.Errorf("unexpected error converting StringValue to StringValuable: %v", diags)
	}
	return stringValuable, nil
}

type jsonNullStrippedStringVal struct {
	basetypes.StringValue
}

func newJSONNullStrippedString(s string) jsonNullStrippedStringVal {
	return jsonNullStrippedStringVal{StringValue: types.StringValue(s)}
}

func (v jsonNullStrippedStringVal) Type(_ context.Context) attr.Type {
	return jsonNullStrippedStringType{}
}

func (v jsonNullStrippedStringVal) Equal(o attr.Value) bool {
	other, ok := o.(jsonNullStrippedStringVal)
	if !ok {
		return false
	}
	return v.StringValue.Equal(other.StringValue)
}

func (v jsonNullStrippedStringVal) StringSemanticEquals(_ context.Context, newValuable basetypes.StringValuable) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	newValue, ok := newValuable.(jsonNullStrippedStringVal)
	if !ok {
		diags.AddError(
			"Semantic Equality Check Error",
			fmt.Sprintf("expected value type %T, got %T", v, newValuable),
		)
		return false, diags
	}
	if v.IsNull() != newValue.IsNull() || v.IsUnknown() != newValue.IsUnknown() {
		return false, diags
	}
	if v.IsNull() || v.IsUnknown() {
		return true, diags
	}
	a, errA := canonicalizeStrippedJSON(v.ValueString())
	b, errB := canonicalizeStrippedJSON(newValue.ValueString())
	if errA != nil || errB != nil {
		return v.ValueString() == newValue.ValueString(), diags
	}
	return a == b, diags
}
