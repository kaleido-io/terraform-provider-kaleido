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

	"github.com/hashicorp/terraform-plugin-framework/function"
)

var _ function.Function = &mergeJSONFunction{}

type mergeJSONFunction struct{}

func MergeJSONFunctionFactory() function.Function {
	return &mergeJSONFunction{}
}

func (f *mergeJSONFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "merge_json"
}

func (f *mergeJSONFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Deep merges one JSON document into another",
		MarkdownDescription: "Deep merges `overlay` into `base` and returns the result as JSON - for example to apply a " +
			"few settings on top of a set of defaults.\n\n" +
			"Objects are merged key by key, to any depth. A `null` in `overlay`, at any level, means *not set*: the " +
			"value in `base` is kept, so an object built from a Terraform type with optional attributes can be " +
			"passed straight through `jsonencode`. Any other value in `overlay` replaces the one in `base`, " +
			"including `false`, `0`, an empty string, and a list, which replaces the whole list in `base` rather " +
			"than being merged with it. Numbers are carried through exactly, without rounding.\n\n" +
			"Unlike a JSON Merge Patch (RFC 7396), a `null` never removes a value from `base`.",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:                "base",
				MarkdownDescription: "The JSON document to merge into.",
			},
			function.StringParameter{
				Name:                "overlay",
				MarkdownDescription: "The JSON document whose values are applied on top of `base`.",
			},
		},
		Return: function.StringReturn{},
	}
}

func (f *mergeJSONFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var base, overlay string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &base, &overlay))
	if resp.Error != nil {
		return
	}
	baseValue, err := decodeJSONExact(base)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("base is not valid JSON: %s", err))
		return
	}
	overlayValue, err := decodeJSONExact(overlay)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(1, fmt.Sprintf("overlay is not valid JSON: %s", err))
		return
	}
	merged, err := json.Marshal(deepMergeJSON(baseValue, overlayValue))
	if err != nil {
		resp.Error = function.NewFuncError(fmt.Sprintf("merged value could not be encoded: %s", err))
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, string(merged)))
}

// decodeJSONExact decodes a single JSON document, keeping numbers as json.Number so that large
// integers are not rounded through float64.
func decodeJSONExact(s string) (interface{}, error) {
	dec := json.NewDecoder(bytes.NewReader([]byte(s)))
	dec.UseNumber()
	var v interface{}
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	if dec.More() {
		return nil, fmt.Errorf("unexpected content after the JSON document")
	}
	return v, nil
}

// deepMergeJSON merges overlay into base: objects key by key, a nil in overlay leaving base's
// value as it is, and anything else in overlay replacing base's value.
func deepMergeJSON(base, overlay interface{}) interface{} {
	if overlay == nil {
		return base
	}
	overlayObj, overlayIsObj := overlay.(map[string]interface{})
	baseObj, baseIsObj := base.(map[string]interface{})
	if !overlayIsObj || !baseIsObj {
		if overlayIsObj {
			// Nothing to merge into, but the overlay's own nulls still mean "not set".
			return deepMergeJSON(map[string]interface{}{}, overlayObj)
		}
		return overlay
	}
	merged := make(map[string]interface{}, len(baseObj)+len(overlayObj))
	for k, v := range baseObj {
		merged[k] = v
	}
	for k, v := range overlayObj {
		if v == nil {
			continue
		}
		merged[k] = deepMergeJSON(baseObj[k], v)
	}
	return merged
}
