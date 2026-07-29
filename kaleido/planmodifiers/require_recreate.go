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
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

// RequireRecreate returns a string plan modifier that rejects changes to an
// attribute after the resource has been created. Unlike RequiresReplace, it does
// not plan a destroy/create cycle — the plan fails and the user must destroy and
// recreate the resource themselves.
//
// resourceType should be the Terraform type name (e.g. "kaleido_platform_kms_key")
// used in diagnostic messages.
func RequireRecreate(resourceType string) planmodifier.String {
	return requireRecreateModifier{resourceType: resourceType}
}

type requireRecreateModifier struct {
	resourceType string
}

func (m requireRecreateModifier) Description(_ context.Context) string {
	return "If the value of this attribute changes after create, planning fails because replace is not supported; destroy and recreate the resource instead."
}

func (m requireRecreateModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m requireRecreateModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// Creating the resource — allow any value.
	if req.State.Raw.IsNull() {
		return
	}

	// Nothing to compare yet (e.g. unknown until apply).
	if req.PlanValue.IsUnknown() || req.StateValue.IsUnknown() {
		return
	}

	if req.PlanValue.Equal(req.StateValue) {
		return
	}

	resourceLabel := m.resourceType
	if resourceLabel == "" {
		resourceLabel = "resource"
	}

	attr := req.Path.String()
	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Immutable attribute cannot be updated",
		fmt.Sprintf(
			"Changing %q on an existing %s is not supported, and automatic resource replace is disabled for this attribute.\n\n"+
				"To use a different value, create a new, separate %s instead.\n\n"+
				"Prior value: %s\nProposed value: %s",
			attr,
			resourceLabel,
			resourceLabel,
			stringValueForDiag(req.StateValue.IsNull(), req.StateValue.ValueString()),
			stringValueForDiag(req.PlanValue.IsNull(), req.PlanValue.ValueString()),
		),
	)
}

func stringValueForDiag(isNull bool, value string) string {
	if isNull {
		return "(null)"
	}
	return value
}
