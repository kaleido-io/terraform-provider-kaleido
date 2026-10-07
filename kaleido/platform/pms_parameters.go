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

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// PMSParameterAPIModel is a parameter declared by an evidence source or an output
// formatter: a named, typed input that a policy bound to it supplies a value expression
// for. Enum and Default are free-form JSON values, carried as the configured text.
type PMSParameterAPIModel struct {
	Name        string          `json:"name"`
	Type        string          `json:"type,omitempty"`
	DisplayName string          `json:"displayName,omitempty"`
	Description string          `json:"description,omitempty"`
	Enum        json.RawMessage `json:"enum,omitempty"`
	Default     json.RawMessage `json:"default,omitempty"`
}

var pmsParameterAttrTypes = map[string]attr.Type{
	"name":         types.StringType,
	"type":         types.StringType,
	"display_name": types.StringType,
	"description":  types.StringType,
	"enum_json":    jsonStringType{},
	"default_json": jsonStringType{},
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
					CustomType:  jsonStringType{},
					Optional:    true,
					Description: "The permitted values of the parameter, as a JSON array (use jsonencode)",
				},
				"default_json": &schema.StringAttribute{
					CustomType:  jsonStringType{},
					Optional:    true,
					Description: "The default value of the parameter, as JSON (use jsonencode). A parameter with a default may be left out by a policy bound to the source or formatter.",
				},
			},
		},
	}
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
			Enum:        jsonAttrToAPI(attrs, "enum_json"),
			Default:     jsonAttrToAPI(attrs, "default_json"),
		}
		if seen[param.Name] {
			diagnostics.AddError("Duplicate parameter", fmt.Sprintf("parameter %q is declared more than once", param.Name))
			return nil
		}
		seen[param.Name] = true
		params = append(params, param)
	}
	return params
}

// pmsParametersToData renders the server's parameters as 'parameter' blocks.
func pmsParametersToData(params []PMSParameterAPIModel, diagnostics *diag.Diagnostics) types.List {
	if len(params) == 0 {
		return types.ListNull(pmsParameterListType.ElemType)
	}
	elements := make([]attr.Value, 0, len(params))
	for _, param := range params {
		obj, diags := types.ObjectValue(pmsParameterAttrTypes, map[string]attr.Value{
			"name":         types.StringValue(param.Name),
			"type":         optionalString(param.Type),
			"display_name": optionalString(param.DisplayName),
			"description":  optionalString(param.Description),
			"enum_json":    jsonFromAPI(param.Enum),
			"default_json": jsonFromAPI(param.Default),
		})
		diagnostics.Append(diags...)
		elements = append(elements, obj)
	}
	list, diags := types.ListValue(pmsParameterListType.ElemType, elements)
	diagnostics.Append(diags...)
	return list
}
