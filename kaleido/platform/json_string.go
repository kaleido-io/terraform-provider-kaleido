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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/attr/xattr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// jsonStringType is the type of a *_json attribute: a string holding a JSON document.
// The configured text is sent to the API byte for byte, so no value is re-encoded on the
// way out (large integers keep their precision), and a value read back that differs only
// in key order, whitespace or number formatting is semantically equal to the configured
// one, so the configured text is kept in state.
type jsonStringType struct {
	basetypes.StringType
}

var _ basetypes.StringTypable = jsonStringType{}

func (t jsonStringType) Equal(o attr.Type) bool {
	other, ok := o.(jsonStringType)
	return ok && t.StringType.Equal(other.StringType)
}

func (t jsonStringType) String() string {
	return "jsonStringType"
}

func (t jsonStringType) ValueType(_ context.Context) attr.Value {
	return jsonStringValue{}
}

func (t jsonStringType) ValueFromString(_ context.Context, in basetypes.StringValue) (basetypes.StringValuable, diag.Diagnostics) {
	return jsonStringValue{StringValue: in}, nil
}

func (t jsonStringType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	attrValue, err := t.StringType.ValueFromTerraform(ctx, in)
	if err != nil {
		return nil, err
	}
	stringValue, ok := attrValue.(basetypes.StringValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type %T returned by StringType.ValueFromTerraform", attrValue)
	}
	return jsonStringValue{StringValue: stringValue}, nil
}

type jsonStringValue struct {
	basetypes.StringValue
}

var (
	_ basetypes.StringValuableWithSemanticEquals = jsonStringValue{}
	_ xattr.ValidateableAttribute                = jsonStringValue{}
)

func jsonStringNull() jsonStringValue {
	return jsonStringValue{StringValue: basetypes.NewStringNull()}
}

func jsonStringOf(s string) jsonStringValue {
	return jsonStringValue{StringValue: basetypes.NewStringValue(s)}
}

func (v jsonStringValue) Type(_ context.Context) attr.Type {
	return jsonStringType{}
}

func (v jsonStringValue) Equal(o attr.Value) bool {
	other, ok := o.(jsonStringValue)
	return ok && v.StringValue.Equal(other.StringValue)
}

func (v jsonStringValue) StringSemanticEquals(_ context.Context, newValuable basetypes.StringValuable) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	newValue, ok := newValuable.(jsonStringValue)
	if !ok {
		diags.AddError("Semantic Equality Check Error", fmt.Sprintf("expected value type %T, got %T", v, newValuable))
		return false, diags
	}
	return jsonSemanticallyEqual([]byte(v.ValueString()), []byte(newValue.ValueString())), diags
}

func (v jsonStringValue) ValidateAttribute(_ context.Context, req xattr.ValidateAttributeRequest, resp *xattr.ValidateAttributeResponse) {
	if v.IsNull() || v.IsUnknown() {
		return
	}
	if !json.Valid([]byte(v.ValueString())) {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid JSON", fmt.Sprintf("%s is not valid JSON (use jsonencode)", req.Path))
	}
}

// jsonToAPI is the wire form of a *_json attribute: its text, unchanged, or nil when it
// is not set.
func jsonToAPI(v attr.Value) json.RawMessage {
	s, ok := v.(jsonStringValue)
	if !ok || s.IsNull() || s.IsUnknown() {
		return nil
	}
	return json.RawMessage(s.ValueString())
}

// jsonAttrToAPI is jsonToAPI for an attribute of a nested object, keyed as the schema
// names it.
func jsonAttrToAPI(attrs map[string]attr.Value, name string) json.RawMessage {
	return jsonToAPI(attrs[name])
}

// jsonFromAPI renders a JSON value read from the API as a *_json attribute. A value the
// API leaves out, or returns as null, is unset.
func jsonFromAPI(raw json.RawMessage) jsonStringValue {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return jsonStringNull()
	}
	return jsonStringOf(string(raw))
}

// jsonObjectFromAPI is jsonFromAPI for a value where an empty object or array means the
// same as no value: that is how such a value is cleared, and what the API may return
// once it has been.
func jsonObjectFromAPI(raw json.RawMessage) jsonStringValue {
	var decoded interface{}
	if json.Unmarshal(raw, &decoded) == nil {
		if rv := reflect.ValueOf(decoded); decoded != nil && (rv.Kind() == reflect.Map || rv.Kind() == reflect.Slice) && rv.Len() == 0 {
			return jsonStringNull()
		}
	}
	return jsonFromAPI(raw)
}

// jsonSemanticallyEqual reports whether two JSON documents hold the same value, whatever
// their key order, whitespace or number formatting. Numbers are compared exactly, not
// through float64.
func jsonSemanticallyEqual(a, b []byte) bool {
	aVal, aErr := decodeJSONExact(a)
	bVal, bErr := decodeJSONExact(b)
	if aErr != nil || bErr != nil {
		return false
	}
	return reflect.DeepEqual(aVal, bVal)
}

// decodeJSONExact decodes a JSON document with every number held as its canonical
// arbitrary-precision text, so that equal numbers written differently compare equal.
func decodeJSONExact(data []byte) (interface{}, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value interface{}
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	return canonicalJSONNumbers(value), nil
}

func canonicalJSONNumbers(value interface{}) interface{} {
	switch v := value.(type) {
	case map[string]interface{}:
		for key, item := range v {
			v[key] = canonicalJSONNumbers(item)
		}
		return v
	case []interface{}:
		for i, item := range v {
			v[i] = canonicalJSONNumbers(item)
		}
		return v
	case json.Number:
		if f, ok := new(big.Float).SetPrec(512).SetString(v.String()); ok {
			return f.Text('g', -1)
		}
		return v.String()
	default:
		return v
	}
}
