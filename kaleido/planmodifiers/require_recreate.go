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
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// RequireRecreate returns a string plan modifier that rejects changes to an
// attribute after the resource has been created. Unlike RequiresReplace, it does
// not plan a destroy/create cycle — the plan fails and the user must destroy and
// recreate the resource themselves.
//
// resourceType should be the Terraform type name (e.g. "kaleido_platform_kms_key")
// used in diagnostic messages.
func RequireRecreate(resourceType string) planmodifier.String {
	return requireRecreateStringModifier{resourceType: resourceType}
}

// RequireRecreateMap is the map-attribute equivalent of RequireRecreate.
func RequireRecreateMap(resourceType string) planmodifier.Map {
	return requireRecreateMapModifier{resourceType: resourceType}
}

type requireRecreateStringModifier struct {
	resourceType string
}

func (m requireRecreateStringModifier) Description(_ context.Context) string {
	return requireRecreateDescription
}

func (m requireRecreateStringModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m requireRecreateStringModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	addRequireRecreateErrorIfChanged(
		req.State.Raw.IsNull(),
		req.PlanValue.IsUnknown() || req.StateValue.IsUnknown(),
		req.PlanValue.Equal(req.StateValue),
		req.Path,
		m.resourceType,
		stringValueForDiag(req.StateValue.IsNull(), req.StateValue.ValueString()),
		stringValueForDiag(req.PlanValue.IsNull(), req.PlanValue.ValueString()),
		&resp.Diagnostics,
	)
}

type requireRecreateMapModifier struct {
	resourceType string
}

func (m requireRecreateMapModifier) Description(_ context.Context) string {
	return requireRecreateDescription
}

func (m requireRecreateMapModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m requireRecreateMapModifier) PlanModifyMap(ctx context.Context, req planmodifier.MapRequest, resp *planmodifier.MapResponse) {
	addRequireRecreateErrorIfChanged(
		req.State.Raw.IsNull(),
		req.PlanValue.IsUnknown() || req.StateValue.IsUnknown(),
		req.PlanValue.Equal(req.StateValue),
		req.Path,
		m.resourceType,
		mapValueForDiag(req.StateValue),
		mapValueForDiag(req.PlanValue),
		&resp.Diagnostics,
	)
}

const requireRecreateDescription = "If the value of this attribute changes after create, planning fails because replace is not supported; create a new resource instead."

func addRequireRecreateErrorIfChanged(
	creating bool,
	unknown bool,
	equal bool,
	attrPath path.Path,
	resourceType string,
	priorValue string,
	proposedValue string,
	diagnostics *diag.Diagnostics,
) {
	if creating || unknown || equal {
		return
	}

	resourceLabel := resourceType
	if resourceLabel == "" {
		resourceLabel = "resource"
	}

	attr := attrPath.String()
	diagnostics.AddAttributeError(
		attrPath,
		"Immutable attribute cannot be updated",
		fmt.Sprintf(
			"Changing %q on an existing %s is not supported, and automatic resource replace is disabled for this attribute.\n\n"+
				"To use a different value, create a new, separate %s instead.\n\n"+
				"Prior value: %s\nProposed value: %s",
			attr,
			resourceLabel,
			resourceLabel,
			priorValue,
			proposedValue,
		),
	)
}

func stringValueForDiag(isNull bool, value string) string {
	if isNull {
		return "(null)"
	}
	return value
}

func mapValueForDiag(value types.Map) string {
	if value.IsNull() {
		return "(null)"
	}
	return value.String()
}
