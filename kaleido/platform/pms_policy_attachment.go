// Copyright © Kaleido, Inc. 2024

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

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type PMSPolicyAttachmentResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	PolicyDeploymentID types.String `tfsdk:"policy_deployment_id"`
	Type               types.String `tfsdk:"type"`
	AttachmentPoint    types.String `tfsdk:"attachment_point"`
	Environment        types.String `tfsdk:"environment"`
	Service            types.String `tfsdk:"service"`
}

type PMSPolicyAttachmentAPIModel struct {
	AttachmentPointType string `json:"attachmentPointType"`
	AttachmentPointName string `json:"attachmentPoint"`
}

func PMSPolicyAttachmentResourceFactory() resource.Resource {
	return &pms_policy_attachmentResource{}
}

type pms_policy_attachmentResource struct {
	commonResource
}

func (r *pms_policy_attachmentResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_pms_policy_attachment"
}

func (r *pms_policy_attachmentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The Policy Manager Attachment Point resource allows you to manage attachment points for policy deployments.",
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				Description:   "Unique ID of the attachment point (computed from policy_deployment_id, type, and name)",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"policy_deployment_id": &schema.StringAttribute{
				Required:      true,
				Description:   "The ID of the policy deployment to attach to",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"type": &schema.StringAttribute{
				Required:      true,
				Description:   "The type of attachment point (e.g., 'wallet_id', 'asset_id')",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"attachment_point": &schema.StringAttribute{
				Required:      true,
				Description:   "The name/identifier of the attachment point (e.g., 'wal:1234567890', 'was:1234567890')",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"environment": &schema.StringAttribute{
				Required:      true,
				Description:   "The environment ID",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"service": &schema.StringAttribute{
				Required:      true,
				Description:   "The service ID",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},
	}
}

func (r *pms_policy_attachmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.commonResource.Configure(ctx, req, resp)
}

func (r *pms_policy_attachmentResource) generateID(data *PMSPolicyAttachmentResourceModel) string {
	return fmt.Sprintf("%s:%s:%s", data.PolicyDeploymentID.ValueString(), data.Type.ValueString(), data.AttachmentPoint.ValueString())
}

func (r *pms_policy_attachmentResource) apiAddAttachmentPath(data *PMSPolicyAttachmentResourceModel) string {
	env := data.Environment.ValueString()
	service := data.Service.ValueString()
	policyDeploymentID := data.PolicyDeploymentID.ValueString()
	return fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/policy-deployments/%s/add-attachment", env, service, policyDeploymentID)
}

func (r *pms_policy_attachmentResource) apiRemoveAttachmentPath(data *PMSPolicyAttachmentResourceModel) string {
	env := data.Environment.ValueString()
	service := data.Service.ValueString()
	policyDeploymentID := data.PolicyDeploymentID.ValueString()
	return fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/policy-deployments/%s/remove-attachment", env, service, policyDeploymentID)
}

func (r *pms_policy_attachmentResource) toAPI(data *PMSPolicyAttachmentResourceModel) *PMSPolicyAttachmentAPIModel {
	return &PMSPolicyAttachmentAPIModel{
		AttachmentPointType: data.Type.ValueString(),
		AttachmentPointName: data.AttachmentPoint.ValueString(),
	}
}

func (r *pms_policy_attachmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PMSPolicyAttachmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate the ID
	data.ID = types.StringValue(r.generateID(&data))

	// Convert to API model
	api := r.toAPI(&data)

	// Call the add-attachment API
	ok, _ := r.apiRequest(ctx, http.MethodPost, r.apiAddAttachmentPath(&data), api, nil, &resp.Diagnostics)
	if !ok {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *pms_policy_attachmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PMSPolicyAttachmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// For policy attachments, we don't have a direct read API
	// The state is managed by the Create/Delete operations
	// We just return the current state as-is
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *pms_policy_attachmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Updates are not supported for policy attachments
	// All fields have RequiresReplace() plan modifiers
	resp.Diagnostics.AddError("Update not supported", "Policy attachments cannot be updated. Use delete and recreate to change policy attachments.")
}

func (r *pms_policy_attachmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PMSPolicyAttachmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert to API model
	api := r.toAPI(&data)

	// Call the remove-attachment API
	_, _ = r.apiRequest(ctx, http.MethodPost, r.apiRemoveAttachmentPath(&data), api, nil, &resp.Diagnostics, Allow404())
}
