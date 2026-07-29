// Copyright © Kaleido, Inc. 2024-2026

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package planmodifiers

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequireRecreate_AllowsCreate(t *testing.T) {
	var resp planmodifier.StringResponse
	RequireRecreate("kaleido_platform_kms_key").PlanModifyString(context.Background(), planmodifier.StringRequest{
		Path:       path.Root("environment"),
		State:      tfsdk.State{Raw: tftypes.NewValue(tftypes.Object{}, nil)},
		StateValue: types.StringNull(),
		PlanValue:  types.StringValue("env1"),
	}, &resp)

	assert.False(t, resp.Diagnostics.HasError())
}

func TestRequireRecreate_AllowsUnchanged(t *testing.T) {
	var resp planmodifier.StringResponse
	RequireRecreate("kaleido_platform_kms_key").PlanModifyString(context.Background(), planmodifier.StringRequest{
		Path: path.Root("environment"),
		State: tfsdk.State{Raw: tftypes.NewValue(tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{"environment": tftypes.String},
		}, map[string]tftypes.Value{
			"environment": tftypes.NewValue(tftypes.String, "env1"),
		})},
		StateValue: types.StringValue("env1"),
		PlanValue:  types.StringValue("env1"),
	}, &resp)

	assert.False(t, resp.Diagnostics.HasError())
}

func TestRequireRecreate_ErrorsOnChange(t *testing.T) {
	var resp planmodifier.StringResponse
	RequireRecreate("kaleido_platform_kms_folder").PlanModifyString(context.Background(), planmodifier.StringRequest{
		Path: path.Root("name"),
		State: tfsdk.State{Raw: tftypes.NewValue(tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{"name": tftypes.String},
		}, map[string]tftypes.Value{
			"name": tftypes.NewValue(tftypes.String, "name1"),
		})},
		StateValue: types.StringValue("name1"),
		PlanValue:  types.StringValue("name2"),
	}, &resp)

	require.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Summary(), "Immutable attribute cannot be updated")
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "name")
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "kaleido_platform_kms_folder")
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "create a new, separate")
}

func TestRequireRecreate_ErrorsOnNullToValue(t *testing.T) {
	var resp planmodifier.StringResponse
	RequireRecreate("kaleido_platform_kms_folder").PlanModifyString(context.Background(), planmodifier.StringRequest{
		Path: path.Root("parent_folder_id"),
		State: tfsdk.State{Raw: tftypes.NewValue(tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{"parent_folder_id": tftypes.String},
		}, map[string]tftypes.Value{
			"parent_folder_id": tftypes.NewValue(tftypes.String, nil),
		})},
		StateValue: types.StringNull(),
		PlanValue:  types.StringValue("kmf:abc"),
	}, &resp)

	require.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "(null)")
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "kaleido_platform_kms_folder")
}
