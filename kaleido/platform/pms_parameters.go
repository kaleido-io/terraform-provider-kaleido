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
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// PMSParameterAPIModel is a parameter declared by an evidence source or an output
// formatter: a named, typed input that a policy bound to it supplies a value expression
// for. Enum and Default are free-form JSON values.
type PMSParameterAPIModel struct {
	Name        string        `json:"name"`
	Type        string        `json:"type,omitempty"`
	DisplayName string        `json:"displayName,omitempty"`
	Description string        `json:"description,omitempty"`
	Enum        []interface{} `json:"enum,omitempty"`
	Default     interface{}   `json:"default,omitempty"`
}

var pmsParameterAttrTypes = map[string]attr.Type{
	"name":         types.StringType,
	"type":         types.StringType,
	"display_name": types.StringType,
	"description":  types.StringType,
	"enum_json":    types.StringType,
	"default_json": types.StringType,
}

var pmsParameterListType = types.ListType{ElemType: types.ObjectType{AttrTypes: pmsParameterAttrTypes}}

// pmsParameterNestedSchema is the schema of the 'parameter' blocks shared by
// kaleido_platform_pms_evidence_source and kaleido_platform_pms_output_formatter.
func pmsParameterNestedSchema(description string) *schema.ListNestedAttribute {
	return &schema.ListNestedAttribute{
		Optional:    true,
		Description: description,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"name": &schema.StringAttribute{
					Required:    true,
					Description: "The name of the parameter, as expressions refer to it",
				},
				"type": &schema.StringAttribute{
					Optional:    true,
					Description: "The type of the parameter, e.g. 'string', 'number', 'boolean', 'object'",
				},
				"display_name": &schema.StringAttribute{
					Optional:    true,
					Description: "The display name of the parameter, used for user interfaces",
				},
				"description": &schema.StringAttribute{
					Optional:    true,
					Description: "A description of the parameter",
				},
				"enum_json": &schema.StringAttribute{
					Optional:    true,
					Description: "The permitted values of the parameter, as a JSON array (use jsonencode)",
				},
				"default_json": &schema.StringAttribute{
					Optional:    true,
					Description: "The default value of the parameter, as JSON (use jsonencode). A parameter with a default may be left out by a policy bound to the source or formatter.",
				},
			},
		},
	}
}

// jsonAnyAttr decodes a *_json attribute that may hold any JSON value, not only an object.
func jsonAnyAttr(attrs map[string]attr.Value, name string, diagnostics *diag.Diagnostics) interface{} {
	raw := stringAttr(attrs, name)
	if raw == "" {
		return nil
	}
	var out interface{}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		diagnostics.AddError("Invalid JSON", fmt.Sprintf("%s is not valid JSON: %s", name, err))
		return nil
	}
	return out
}

// jsonAnyToAttr renders a decoded JSON value for a *_json attribute. It keeps the
// configured string when it means the same thing, so the server's canonical serialisation
// does not fight the configured jsonencode() output. A nil value, including a nil map or
// slice, is absent.
func jsonAnyToAttr(configured types.String, value interface{}) types.String {
	if value == nil {
		return types.StringNull()
	}
	if rv := reflect.ValueOf(value); (rv.Kind() == reflect.Map || rv.Kind() == reflect.Slice) && rv.IsNil() {
		return types.StringNull()
	}
	b, err := json.Marshal(value)
	if err != nil {
		return types.StringNull()
	}
	if !configured.IsNull() && !configured.IsUnknown() && jsonSemanticallyEqual([]byte(configured.ValueString()), b) {
		return configured
	}
	return types.StringValue(string(b))
}

// pmsParametersToAPI reads the 'parameter' blocks into the wire form, rejecting duplicate
// names up front as the server would.
func pmsParametersToAPI(list types.List, diagnostics *diag.Diagnostics) []PMSParameterAPIModel {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}
	seen := map[string]bool{}
	params := []PMSParameterAPIModel{}
	for _, item := range list.Elements() {
		obj, ok := item.(types.Object)
		if !ok {
			continue
		}
		attrs := obj.Attributes()
		param := PMSParameterAPIModel{
			Name:        stringAttr(attrs, "name"),
			Type:        stringAttr(attrs, "type"),
			DisplayName: stringAttr(attrs, "display_name"),
			Description: stringAttr(attrs, "description"),
			Default:     jsonAnyAttr(attrs, "default_json", diagnostics),
		}
		if seen[param.Name] {
			diagnostics.AddError("Duplicate parameter", fmt.Sprintf("parameter %q is declared more than once", param.Name))
			return nil
		}
		seen[param.Name] = true
		if enum := jsonAnyAttr(attrs, "enum_json", diagnostics); enum != nil {
			values, isArray := enum.([]interface{})
			if !isArray {
				diagnostics.AddError("Invalid JSON", fmt.Sprintf("enum_json of parameter %q is not a JSON array", param.Name))
				return nil
			}
			param.Enum = values
		}
		params = append(params, param)
	}
	return params
}

// pmsParametersToData renders the server's parameters as 'parameter' blocks. The JSON
// attributes keep their configured spelling, matched by parameter name, when unchanged.
func pmsParametersToData(params []PMSParameterAPIModel, configured types.List, diagnostics *diag.Diagnostics) types.List {
	if len(params) == 0 {
		return types.ListNull(pmsParameterListType.ElemType)
	}
	configuredByName := map[string]map[string]attr.Value{}
	if !configured.IsNull() && !configured.IsUnknown() {
		for _, item := range configured.Elements() {
			if obj, ok := item.(types.Object); ok {
				configuredByName[stringAttr(obj.Attributes(), "name")] = obj.Attributes()
			}
		}
	}
	elements := make([]attr.Value, 0, len(params))
	for _, param := range params {
		prior := configuredByName[param.Name]
		obj, diags := types.ObjectValue(pmsParameterAttrTypes, map[string]attr.Value{
			"name":         types.StringValue(param.Name),
			"type":         optionalString(param.Type),
			"display_name": optionalString(param.DisplayName),
			"description":  optionalString(param.Description),
			"enum_json":    jsonAnyToAttr(priorStringAttr(prior, "enum_json"), param.Enum),
			"default_json": jsonAnyToAttr(priorStringAttr(prior, "default_json"), param.Default),
		})
		diagnostics.Append(diags...)
		elements = append(elements, obj)
	}
	list, diags := types.ListValue(pmsParameterListType.ElemType, elements)
	diagnostics.Append(diags...)
	return list
}

func priorStringAttr(attrs map[string]attr.Value, name string) types.String {
	if attrs == nil {
		return types.StringNull()
	}
	if s, ok := attrs[name].(types.String); ok {
		return s
	}
	return types.StringNull()
}
