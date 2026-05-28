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
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStripJSONNulls(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want any
	}{
		{
			name: "top-level null removed",
			in:   map[string]any{"a": "x", "b": nil},
			want: map[string]any{"a": "x"},
		},
		{
			name: "nested null removed",
			in: map[string]any{
				"source": map[string]any{
					"fixedGasPrice": map[string]any{"enabled": true, "gasPrice": nil},
					"gasOracleAPI":  nil,
				},
				"autoIncrement": nil,
			},
			want: map[string]any{
				"source": map[string]any{
					"fixedGasPrice": map[string]any{"enabled": true},
				},
			},
		},
		{
			name: "null entries in slice removed",
			in:   map[string]any{"items": []any{"a", nil, "b"}},
			want: map[string]any{"items": []any{"a", "b"}},
		},
		{
			name: "scalar value preserved",
			in:   "string",
			want: "string",
		},
		{
			name: "empty map preserved (not null)",
			in:   map[string]any{"x": map[string]any{}},
			want: map[string]any{"x": map[string]any{}},
		},
		{
			name: "false and zero preserved",
			in:   map[string]any{"enabled": false, "count": float64(0), "name": ""},
			want: map[string]any{"enabled": false, "count": float64(0), "name": ""},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, stripJSONNulls(tc.in))
		})
	}
}

func TestCanonicalizeStrippedJSON_Equal(t *testing.T) {
	// Planned (typed-object jsonencode output) vs stored (server round-trip)
	planned := `{"format":null,"source":{"fixedGasPrice":{"enabled":true,"maxFeePerGas":"0x0","maxPriorityFeePerGas":"0x0","gasPrice":null},"gasOracleAPI":null,"RPCEndpoint":null},"autoIncrement":null,"caps":null}`
	stored := `{"source":{"fixedGasPrice":{"enabled":true,"maxFeePerGas":"0x0","maxPriorityFeePerGas":"0x0"}}}`

	a, err := canonicalizeStrippedJSON(planned)
	require.NoError(t, err)
	b, err := canonicalizeStrippedJSON(stored)
	require.NoError(t, err)
	assert.Equal(t, a, b, "null-stripped canonical forms should match")
}

func TestCanonicalizeStrippedJSON_Differ(t *testing.T) {
	planned := `{"count":3}`
	stored := `{"count":2}`

	a, err := canonicalizeStrippedJSON(planned)
	require.NoError(t, err)
	b, err := canonicalizeStrippedJSON(stored)
	require.NoError(t, err)
	assert.NotEqual(t, a, b)
}

func TestCanonicalizeStrippedJSON_InvalidJSON(t *testing.T) {
	_, err := canonicalizeStrippedJSON("not json")
	assert.Error(t, err)
}

func TestJSONNullStrippedStringSemanticEquals(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name string
		a, b string
		want bool
	}{
		{
			name: "config has null, state stripped - equal",
			a:    `{"count":0,"resubmission":null}`,
			b:    `{"count":0}`,
			want: true,
		},
		{
			name: "both fully populated identical",
			a:    `{"count":3}`,
			b:    `{"count":3}`,
			want: true,
		},
		{
			name: "different counts not equal",
			a:    `{"count":2}`,
			b:    `{"count":3}`,
			want: false,
		},
		{
			name: "deeply nested null stripping",
			a:    `{"format":null,"source":{"fixedGasPrice":{"enabled":true,"gasPrice":null},"gasOracleAPI":null},"caps":null}`,
			b:    `{"source":{"fixedGasPrice":{"enabled":true}}}`,
			want: true,
		},
		{
			name: "both collapse to empty",
			a:    `{"previousTxnsCondition":null}`,
			b:    `{}`,
			want: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			va := jsonNullStrippedStringVal{StringValue: types.StringValue(tc.a)}
			vb := jsonNullStrippedStringVal{StringValue: types.StringValue(tc.b)}
			eq, diags := va.StringSemanticEquals(ctx, vb)
			require.False(t, diags.HasError(), "diagnostics: %+v", diags)
			assert.Equal(t, tc.want, eq)
		})
	}
}

func TestJSONNullStrippedStringSemanticEquals_NullVsValue(t *testing.T) {
	ctx := context.Background()
	null := jsonNullStrippedStringVal{StringValue: types.StringNull()}
	val := jsonNullStrippedStringVal{StringValue: types.StringValue(`{}`)}
	eq, diags := null.StringSemanticEquals(ctx, val)
	require.False(t, diags.HasError())
	assert.False(t, eq)

	// null == null
	null2 := jsonNullStrippedStringVal{StringValue: types.StringNull()}
	eq, diags = null.StringSemanticEquals(ctx, null2)
	require.False(t, diags.HasError())
	assert.True(t, eq)
}
