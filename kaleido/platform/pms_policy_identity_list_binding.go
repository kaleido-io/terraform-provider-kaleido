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
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type PMSIdentityListBindingResourceModel struct {
	ID                    types.String `tfsdk:"id"`
	Environment           types.String `tfsdk:"environment"`
	Service               types.String `tfsdk:"service"`
	Policy                types.String `tfsdk:"policy"`
	AttesterLabel         types.String `tfsdk:"attester_label"`
	IdentityListVersionID types.String `tfsdk:"identity_list_version_id"`
}

type PMSIdentityListBindingAPIModel struct {
	ID                    string     `json:"id,omitempty"`
	PolicyID              string     `json:"policyId,omitempty"`
	AttesterLabel         string     `json:"attesterLabel,omitempty"`
	IdentityListVersionID string     `json:"identityListVersionId,omitempty"`
	Created               *time.Time `json:"created,omitempty"`
	Updated               *time.Time `json:"updated,omitempty"`
}

// PMSIdentityListBindingPatchAPIModel is the PATCH body - the binding is immutable
// except for the identity list version it points at.
type PMSIdentityListBindingPatchAPIModel struct {
	IdentityListVersionID string `json:"identityListVersionId,omitempty"`
}

func PMSPolicyIdentityListBindingResourceFactory() resource.Resource {
	return &pms_identityListBindingResource{}
}

type pms_identityListBindingResource struct {
	commonResource
}

func (r *pms_identityListBindingResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_pms_policy_identity_list_binding"
}

func (r *pms_identityListBindingResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an identity list binding on a Policy Manager policy. The binding resolves an attester label declared in the policy definition to a specific version of an identity list, determining who is allowed to attest to the evidence.",
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				Description:   "The binding ID assigned by the server",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"environment": &schema.StringAttribute{
				Required:      true,
				Description:   "Environment ID",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"service": &schema.StringAttribute{
				Required:      true,
				Description:   "Policy Manager service ID",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"policy": &schema.StringAttribute{
				Required:      true,
				Description:   "Name or ID of the policy this binding belongs to",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"attester_label": &schema.StringAttribute{
				Required:      true,
				Description:   "The attester label declared in a policy evidence attestation slot (e.g. 'treasuryOperations'). Immutable after create.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"identity_list_version_id": &schema.StringAttribute{
				Required:    true,
				Description: "ID of the identity list version whose members may attest under this label. This is a version ID, not a version name - use the applied_version_id attribute of a kaleido_platform_pms_identity_list.",
			},
		},
	}
}

func (r *pms_identityListBindingResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.commonResource.Configure(ctx, req, resp)
}

func (r *pms_identityListBindingResource) listPath(data *PMSIdentityListBindingResourceModel) string {
	return fmt.Sprintf("/endpoint/%s/%s/rest/api/v2/policies/%s/identity-list-bindings",
		data.Environment.ValueString(), data.Service.ValueString(), url.PathEscape(data.Policy.ValueString()))
}

func (r *pms_identityListBindingResource) instancePath(data *PMSIdentityListBindingResourceModel) string {
	return fmt.Sprintf("%s/%s", r.listPath(data), data.ID.ValueString())
}

func (r *pms_identityListBindingResource) toAPI(data *PMSIdentityListBindingResourceModel, api *PMSIdentityListBindingAPIModel) {
	api.AttesterLabel = data.AttesterLabel.ValueString()
	api.IdentityListVersionID = data.IdentityListVersionID.ValueString()
}

func (r *pms_identityListBindingResource) toData(api *PMSIdentityListBindingAPIModel, data *PMSIdentityListBindingResourceModel) {
	data.ID = types.StringValue(api.ID)
	if api.AttesterLabel != "" {
		data.AttesterLabel = types.StringValue(api.AttesterLabel)
	}
	if api.IdentityListVersionID != "" {
		data.IdentityListVersionID = types.StringValue(api.IdentityListVersionID)
	}
}

func (r *pms_identityListBindingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PMSIdentityListBindingResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var api PMSIdentityListBindingAPIModel
	r.toAPI(&data, &api)

	ok, _ := r.apiRequest(ctx, http.MethodPost, r.listPath(&data), &api, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	r.toData(&api, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *pms_identityListBindingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PMSIdentityListBindingResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var api PMSIdentityListBindingAPIModel
	ok, status := r.apiRequest(ctx, http.MethodGet, r.instancePath(&data), nil, &api, &resp.Diagnostics, Allow404())
	if !ok {
		return
	}
	if status == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	r.toData(&api, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *pms_identityListBindingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data PMSIdentityListBindingResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	patch := PMSIdentityListBindingPatchAPIModel{IdentityListVersionID: data.IdentityListVersionID.ValueString()}
	var api PMSIdentityListBindingAPIModel
	ok, _ := r.apiRequest(ctx, http.MethodPatch, r.instancePath(&data), &patch, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	r.toData(&api, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *pms_identityListBindingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PMSIdentityListBindingResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.instancePath(&data), nil, nil, &resp.Diagnostics, Allow404())
}
