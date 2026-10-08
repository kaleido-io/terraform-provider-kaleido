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
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr/xattr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/stretchr/testify/assert"
)

func TestJSONSemanticallyEqual(t *testing.T) {
	assert.True(t, jsonSemanticallyEqual([]byte(`{"a":1,"b":[1,2]}`), []byte("{ \"b\": [1, 2],\n \"a\": 1 }")), "key order and whitespace")
	assert.True(t, jsonSemanticallyEqual([]byte(`{"n":1e21}`), []byte(`{"n":1000000000000000000000}`)), "number formatting")
	assert.True(t, jsonSemanticallyEqual([]byte(`[1.0]`), []byte(`[1]`)), "trailing zeros")
	assert.False(t, jsonSemanticallyEqual([]byte(`{"n":1000000000000000000001}`), []byte(`{"n":1000000000000000000000}`)),
		"numbers beyond float64 precision are still compared exactly")
	assert.False(t, jsonSemanticallyEqual([]byte(`[1,2]`), []byte(`[2,1]`)), "array order matters")
	assert.False(t, jsonSemanticallyEqual([]byte(`{`), []byte(`{`)), "invalid JSON is never equal")
}

func TestJSONStringSemanticEquals(t *testing.T) {
	equal, diags := jsonStringOf(`{"a":1,"b":2}`).StringSemanticEquals(context.Background(), jsonStringOf(`{"b":2,"a":1}`))
	assert.False(t, diags.HasError())
	assert.True(t, equal)
}

func TestJSONStringValidateAttribute(t *testing.T) {
	var resp xattr.ValidateAttributeResponse
	jsonStringOf(`{"a":`).ValidateAttribute(context.Background(), xattr.ValidateAttributeRequest{Path: path.Root("options_json")}, &resp)
	assert.True(t, resp.Diagnostics.HasError())

	resp = xattr.ValidateAttributeResponse{}
	jsonStringOf(`{"a":1}`).ValidateAttribute(context.Background(), xattr.ValidateAttributeRequest{Path: path.Root("options_json")}, &resp)
	assert.False(t, resp.Diagnostics.HasError())
}

func TestJSONToAndFromAPI(t *testing.T) {
	assert.Nil(t, jsonToAPI(jsonStringNull()))
	assert.Equal(t, json.RawMessage(`{"n":1000000000000000000001}`), jsonToAPI(jsonStringOf(`{"n":1000000000000000000001}`)), "sent byte for byte")

	assert.True(t, jsonFromAPI(nil).IsNull())
	assert.True(t, jsonFromAPI(json.RawMessage(`null`)).IsNull())
	assert.Equal(t, `{}`, jsonFromAPI(json.RawMessage(`{}`)).ValueString())

	assert.True(t, jsonObjectFromAPI(json.RawMessage(`{}`)).IsNull())
	assert.True(t, jsonObjectFromAPI(json.RawMessage(` [ ] `)).IsNull())
	assert.Equal(t, `{"a":1}`, jsonObjectFromAPI(json.RawMessage(`{"a":1}`)).ValueString())
}
