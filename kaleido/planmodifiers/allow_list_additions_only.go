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

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// AllowListAdditionsOnly returns a list plan modifier that permits ADDING
// elements to an existing resource's list attribute, but REJECTS removing
// elements (or replacing existing ones with different values). It is meant for
// list attributes whose backing API supports additive-only updates — e.g.
// kaleido_platform_kms_key.public_identifier_types, where the Kaleido KMS API
// can create new identifier types on an existing key but has no way to remove
// them without destroying the key mapping.
//
// resourceType should be the Terraform type name (e.g. "kaleido_platform_kms_key")
// used in diagnostic messages.
func AllowListAdditionsOnly(resourceType string) planmodifier.List {
	return allowListAdditionsOnlyModifier{resourceType: resourceType}
}

type allowListAdditionsOnlyModifier struct {
	resourceType string
}

func (m allowListAdditionsOnlyModifier) Description(_ context.Context) string {
	return "New elements may be added to this list after create. Removing existing elements is not supported; planning fails if any element is removed."
}

func (m allowListAdditionsOnlyModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m allowListAdditionsOnlyModifier) PlanModifyList(ctx context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	if req.State.Raw.IsNull() {
		return
	}
	if req.PlanValue.IsUnknown() || req.StateValue.IsUnknown() {
		return
	}
	if req.PlanValue.Equal(req.StateValue) {
		return
	}

	state := extractListStrings(ctx, req.StateValue, &resp.Diagnostics)
	plan := extractListStrings(ctx, req.PlanValue, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	planSet := make(map[string]struct{}, len(plan))
	for _, v := range plan {
		planSet[v] = struct{}{}
	}

	removed := make([]string, 0)
	for _, v := range state {
		if _, ok := planSet[v]; !ok {
			removed = append(removed, v)
		}
	}
	if len(removed) == 0 {
		// Pure additions (or a reordering) — allowed. Terraform plans the
		// change and Update() applies it.
		return
	}

	resourceLabel := m.resourceType
	if resourceLabel == "" {
		resourceLabel = "resource"
	}
	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Removing elements from this list is not supported",
		fmt.Sprintf(
			"The Kaleido API does not support removing entries from %q on an existing %s. "+
				"You can add new entries, but to remove one you must destroy and recreate the resource.\n\n"+
				"Removed entries: %v\nPrior value: %s\nProposed value: %s",
			req.Path.String(),
			resourceLabel,
			removed,
			listValueForDiag(req.StateValue),
			listValueForDiag(req.PlanValue),
		),
	)
}

func extractListStrings(ctx context.Context, v types.List, diagnostics *diag.Diagnostics) []string {
	out := make([]string, 0, len(v.Elements()))
	if v.IsNull() || v.IsUnknown() {
		return out
	}
	diagnostics.Append(v.ElementsAs(ctx, &out, false)...)
	return out
}
