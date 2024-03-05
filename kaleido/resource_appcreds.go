// Copyright © Kaleido, Inc. 2018, 2024

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package kaleido

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	kaleido "github.com/kaleido-io/kaleido-sdk-go/kaleido"
)

type resourceAppCreds struct {
	baasBaseResource
}

func ResourceAppCredsFactory() resource.Resource {
	return &resourceAppCreds{}
}

type AppCredsResourceModel struct {
	ID            types.String `tfsdk:"id"`
	MembershipID  types.String `tfsdk:"membership_id"`
	ConsortiumID  types.String `tfsdk:"consortium_id"`
	EnvironmentID types.String `tfsdk:"environment_id"`
	Name          types.String `tfsdk:"name"`
	Username      types.String `tfsdk:"username"`
	Password      types.String `tfsdk:"password"`
	AuthType      types.String `tfsdk:"auth_type"`
}

func (r *resourceAppCreds) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed: true,
			},
			"membership_id": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"consortium_id": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"environment_id": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": &schema.StringAttribute{
				Required: true,
			},
			"username": &schema.StringAttribute{
				Computed: true,
			},
			"password": &schema.StringAttribute{
				Sensitive: true,
				Computed:  true,
			},
			"auth_type": &schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (r *resourceAppCreds) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_app_creds"
}

func (r *resourceAppCreds) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AppCredsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	apiModel := kaleido.AppCreds{}
	consortiumID := data.ConsortiumID.ValueString()
	environmentID := data.EnvironmentID.ValueString()
	apiModel.MembershipID = data.MembershipID.ValueString()
	apiModel.Name = data.Name.ValueString()

	res, err := r.BaaS.CreateAppCreds(consortiumID, environmentID, &apiModel)
	if err != nil {
		resp.Diagnostics.AddError("failed to create app credential", err.Error())
		return
	}

	if res.StatusCode() != 201 {
		msg := "Could not create AppKey in consortium %s, in environment %s, with membership %s with status %d: %s"
		resp.Diagnostics.AddError("failed to create app credential", fmt.Sprintf(msg, consortiumID, environmentID, apiModel.MembershipID, res.StatusCode(), res.String()))
		return
	}

	data.ID = types.StringValue(apiModel.ID)
	data.Username = types.StringValue(apiModel.Username)
	data.Password = types.StringValue(apiModel.Password)
	data.AuthType = types.StringValue(apiModel.AuthType)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *resourceAppCreds) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data AppCredsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	apiModel := kaleido.AppCreds{}
	consortiumID := data.ConsortiumID.ValueString()
	environmentID := data.EnvironmentID.ValueString()
	apiModel.MembershipID = data.MembershipID.ValueString()
	apiModel.Name = data.Name.ValueString()
	appKeyID := data.ID.ValueString()

	res, err := r.BaaS.UpdateAppCreds(consortiumID, environmentID, appKeyID, &apiModel)
	if err != nil {
		resp.Diagnostics.AddError("failed to update app credential", err.Error())
		return
	}

	if res.StatusCode() != 200 {
		msg := "Could not update AppKey %s in consortium %s, in environment %s, with membership %s with status %d: %s"
		resp.Diagnostics.AddError("failed to update app credential", fmt.Sprintf(msg, appKeyID, consortiumID, environmentID, apiModel.MembershipID, res.StatusCode(), res.String()))
		return
	}

	data.Username = types.StringValue(apiModel.Username)
	data.Password = types.StringValue(apiModel.Password)
	data.AuthType = types.StringValue(apiModel.AuthType)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *resourceAppCreds) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AppCredsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	apiModel := kaleido.AppCreds{}
	consortiumID := data.ConsortiumID.ValueString()
	environmentID := data.EnvironmentID.ValueString()
	appKeyID := data.ID.ValueString()

	res, err := r.BaaS.GetAppCreds(consortiumID, environmentID, appKeyID, &apiModel)
	if err != nil {
		resp.Diagnostics.AddError("failed to query app credential", err.Error())
		return
	}

	if res.StatusCode() != 200 {
		msg := "Could not fetch AppKey %s in consortium %s, in environment %s with status %d: %s"
		resp.Diagnostics.AddError("failed to app credential", fmt.Sprintf(msg, appKeyID, consortiumID, environmentID, res.StatusCode(), res.String()))
		return
	}

	data.Username = types.StringValue(apiModel.Username)
	data.Password = types.StringValue(apiModel.Password)
	data.AuthType = types.StringValue(apiModel.AuthType)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *resourceAppCreds) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AppCredsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	consortiumID := data.ConsortiumID.ValueString()
	environmentID := data.EnvironmentID.ValueString()
	appKeyID := data.ID.ValueString()

	res, err := r.BaaS.DeleteAppCreds(consortiumID, environmentID, appKeyID)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete app credential", err.Error())
		return
	}

	if res.StatusCode() != 204 {
		msg := "Could not delete AppKey %s in consortium %s, in environment %s with status %d: %s"
		resp.Diagnostics.AddError("failed to delete app credential", fmt.Sprintf(msg, appKeyID, consortiumID, environmentID, res.StatusCode(), res.String()))
		return
	}
}
