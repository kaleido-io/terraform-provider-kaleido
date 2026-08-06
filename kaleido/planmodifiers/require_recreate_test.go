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

func TestRequireRecreateMap_ErrorsOnChange(t *testing.T) {
	var resp planmodifier.MapResponse
	prior, diags := types.MapValueFrom(context.Background(), types.StringType, map[string]string{"a": "1"})
	require.False(t, diags.HasError())
	proposed, diags := types.MapValueFrom(context.Background(), types.StringType, map[string]string{"a": "2"})
	require.False(t, diags.HasError())

	RequireRecreateMap("kaleido_platform_kms_key").PlanModifyMap(context.Background(), planmodifier.MapRequest{
		Path: path.Root("attributes"),
		State: tfsdk.State{Raw: tftypes.NewValue(tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"attributes": tftypes.Map{ElementType: tftypes.String},
			},
		}, map[string]tftypes.Value{
			"attributes": tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, map[string]tftypes.Value{
				"attribute": tftypes.NewValue(tftypes.String, "1"),
			}),
		})},
		StateValue: prior,
		PlanValue:  proposed,
	}, &resp)

	require.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "attributes")
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "kaleido_platform_kms_key")
}

func TestRequireRecreateList_ErrorsOnChange(t *testing.T) {
	var resp planmodifier.ListResponse
	prior, diags := types.ListValueFrom(context.Background(), types.StringType, []string{"address_ethereum"})
	require.False(t, diags.HasError())
	proposed, diags := types.ListValueFrom(context.Background(), types.StringType, []string{"address_ethereum", "address_ethereum_checksum"})
	require.False(t, diags.HasError())

	RequireRecreateList("kaleido_platform_kms_key").PlanModifyList(context.Background(), planmodifier.ListRequest{
		Path: path.Root("public_identifier_types"),
		State: tfsdk.State{Raw: tftypes.NewValue(tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"public_identifier_types": tftypes.List{ElementType: tftypes.String},
			},
		}, map[string]tftypes.Value{
			"public_identifier_types": tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
				tftypes.NewValue(tftypes.String, "address_ethereum"),
			}),
		})},
		StateValue: prior,
		PlanValue:  proposed,
	}, &resp)

	require.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "public_identifier_types")
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "kaleido_platform_kms_key")
}
