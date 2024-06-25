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

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type AMSPolicyResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	Environment    types.String `tfsdk:"environment"`
	Service        types.String `tfsdk:"service"`
	Document       types.String `tfsdk:"document"`      // this is propagated to a policy version
	ExampleInput   types.String `tfsdk:"example_input"` // this is propagated to a policy version
	Hash           types.String `tfsdk:"hash"`
	AppliedVersion types.String `tfsdk:"applied_version"`
}

type AMSPolicyAPIModel struct {
	ID             string `json:"id,omitempty"`
	Name           string `json:"name,omitempty"`
	Description    string `json:"description,omitempty"`
	Created        string `json:"created,omitempty"`
	Updated        string `json:"updated,omitempty"`
	CurrentVersion string `json:"currentVersion,omitempty"`
}

type AMSPolicyVersionAPIModel struct {
	ID           string `json:"id,omitempty"`
	Description  string `json:"description,omitempty"`
	Document     string `json:"document,omitempty"`
	ExampleInput string `json:"exampleInput,omitempty"`
	Hash         string `json:"hash,omitempty"`
	Created      string `json:"created,omitempty"`
	Updated      string `json:"updated,omitempty"`
}

func AMSPolicyResourceFactory() resource.Resource {
	return &ams_policyResource{}
}

type ams_policyResource struct {
	commonResource
}

func (r *ams_policyResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_ams_policy"
}

func (r *ams_policyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"environment": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": &schema.StringAttribute{
				Required: true,
			},
			"description": &schema.StringAttribute{
				Optional: true,
			},
			"service": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"document": &schema.StringAttribute{
				Required:    true,
				Description: "This is the definition of the policy - which will be put into a new version each time the policy is updated",
			},
			"example_input": &schema.StringAttribute{
				Optional: true,
			},
			"hash": &schema.StringAttribute{
				Computed: true,
			},
			"applied_version": &schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (api *AMSPolicyAPIModel) toData(data *AMSPolicyResourceModel) {
	data.ID = types.StringValue(api.ID)
	data.AppliedVersion = types.StringValue(api.CurrentVersion)
}

func (api *AMSPolicyVersionAPIModel) toData(data *AMSPolicyResourceModel) {
	data.Hash = types.StringValue(api.Hash)
}

func (data *AMSPolicyResourceModel) toAPI(api *AMSPolicyAPIModel, apiV *AMSPolicyVersionAPIModel) {
	api.Name = data.Name.ValueString()
	api.Description = data.Description.ValueString()

	apiV.Description = data.Description.ValueString()
	apiV.Document = data.Document.ValueString()
	if data.ExampleInput.ValueString() != "" {
		apiV.ExampleInput = data.ExampleInput.ValueString()
	}
}

func (r *ams_policyResource) apiPath(data *AMSPolicyResourceModel, idOrName string) string {
	return fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/policies/%s", data.Environment.ValueString(), data.Service.ValueString(), idOrName)
}

func (r *ams_policyResource) apiPolicyVersionPath(data *AMSPolicyResourceModel, idOrName string) string {
	return fmt.Sprintf("%s/versions", r.apiPath(data, idOrName))
}

func (r *ams_policyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var data AMSPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	var api AMSPolicyAPIModel
	var apiV AMSPolicyVersionAPIModel
	data.toAPI(&api, &apiV)
	// Policy PUT
	ok, _ := r.apiRequest(ctx, http.MethodPut, r.apiPath(&data, api.Name), &api, &api, &resp.Diagnostics)
	if ok {
		// Policy version POST
		ok, _ = r.apiRequest(ctx, http.MethodPost, r.apiPolicyVersionPath(&data, api.Name), &apiV, &apiV, &resp.Diagnostics)
	}
	if !ok {
		return
	}

	api.toData(&data)
	apiV.toData(&data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)

}

func (r *ams_policyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

	var data AMSPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)

	var api AMSPolicyAPIModel
	var apiV AMSPolicyVersionAPIModel
	data.toAPI(&api, &apiV)
	policyID := data.ID.ValueString()
	// Policy PATCH
	ok, _ := r.apiRequest(ctx, http.MethodPatch, r.apiPath(&data, policyID), &api, &api, &resp.Diagnostics)
	if ok {
		// Policy version POST
		ok, _ = r.apiRequest(ctx, http.MethodPost, r.apiPolicyVersionPath(&data, api.Name), &apiV, &apiV, &resp.Diagnostics)
	}
	if !ok {
		return
	}

	api.toData(&data)
	apiV.toData(&data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *ams_policyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AMSPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	var api AMSPolicyAPIModel
	api.ID = data.ID.ValueString()
	ok, status := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data, data.ID.ValueString()), nil, &api, &resp.Diagnostics, Allow404())
	if !ok {
		return
	}
	if status == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	api.toData(&data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *ams_policyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AMSPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data, data.ID.ValueString()), nil, nil, &resp.Diagnostics, Allow404())

	r.waitForRemoval(ctx, r.apiPath(&data, data.ID.ValueString()), &resp.Diagnostics)
}
