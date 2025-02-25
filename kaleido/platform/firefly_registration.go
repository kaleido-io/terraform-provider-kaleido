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

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kaleido-io/terraform-provider-kaleido/kaleido/kaleidobase"
)

type FireFlyRegistrationResourceModel struct {
	Environment  types.String `tfsdk:"environment"`
	Service      types.String `tfsdk:"service"`
	OrgID        types.String `tfsdk:"org_id"`
	OrgDID       types.String `tfsdk:"org_did"`
	OrgVerifiers types.Set    `tfsdk:"org_verifiers"`
	NodeID       types.String `tfsdk:"node_id"`
}

type FireFlyStatusAPIModel struct {
	Node FireFlyStatusNodeAPIModel `json:"node"`
	Org  FireFlyStatusOrgAPIModel  `json:"org"`
}

type FireFlyStatusOrgAPIModel struct {
	Name       string                          `json:"name"`
	Registered bool                            `json:"registered"`
	DID        string                          `json:"did"`
	ID         string                          `json:"id"`
	Verifiers  []FireFlyStatusVerifierAPIModel `json:"verifiers"`
}

type FireFlyStatusNodeAPIModel struct {
	Name       string `json:"name"`
	Registered bool   `json:"registered"`
	ID         string `json:"id"`
}

type FireFlyStatusVerifierAPIModel struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

func FireFlyRegistrationResourceFactory() resource.Resource {
	return &firefly_registrationResource{}
}

type firefly_registrationResource struct {
	commonResource
}

func (r *firefly_registrationResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_firefly_registration"
}

func (r *firefly_registrationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Once you have created your Firefly multi-party network members, the final step is to instruct the FireFly namespaces to register their nodes and organizations in the network. This resource is only supported for Firefly services with multiparty enabled.",
		Attributes: map[string]schema.Attribute{
			"environment": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Environment ID",
			},
			"service": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Firefly Service ID to register in the multiparty network",
			},
			"org_id": &schema.StringAttribute{
				Computed: true,
			},
			"org_did": &schema.StringAttribute{
				Computed: true,
			},
			"org_verifiers": &schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
			"node_id": &schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (api *FireFlyStatusAPIModel) toData(data *FireFlyRegistrationResourceModel, diagnostics *diag.Diagnostics) bool {
	if api.Node.Registered && api.Org.Registered {
		data.OrgDID = types.StringValue(api.Org.DID)
		data.OrgID = types.StringValue(api.Org.ID)
		verifiers := make([]attr.Value, len(api.Org.Verifiers))
		for i, v := range api.Org.Verifiers {
			verifiers[i] = types.StringValue(v.Value)
		}
		verifiersSet, diag := types.SetValue(types.StringType, verifiers)
		diagnostics.Append(diag...)
		data.OrgVerifiers = verifiersSet
		data.NodeID = types.StringValue(api.Node.ID)
		return true
	}
	return false
}

func (r *firefly_registrationResource) apiPath(data *FireFlyRegistrationResourceModel, fireflyPath string) string {
	return fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/%s", data.Environment.ValueString(), data.Service.ValueString(), fireflyPath)
}

func (r *firefly_registrationResource) ensureRegistered(ctx context.Context, data *FireFlyRegistrationResourceModel, diagnostics *diag.Diagnostics) {
	var status FireFlyStatusAPIModel
	nodeSubmitted := false
	orgSubmitted := false
	cancelInfo := APICancelInfo()
	_ = kaleidobase.Retry.Do(ctx, "register", func(attempt int) (retry bool, err error) {
		ok, _ := r.apiRequest(ctx, http.MethodGet, r.apiPath(data, "status"), nil, &status, diagnostics, cancelInfo)
		if !ok {
			return false, fmt.Errorf("status-check failed") // already set in diag
		}
		if registered := status.toData(data, diagnostics); registered {
			return false, nil
		}
		if !status.Org.Registered {
			if !orgSubmitted {
				ok, _ := r.apiRequest(ctx, http.MethodPost, r.apiPath(data, "network/organizations/self"), struct{}{}, nil, diagnostics, cancelInfo)
				if !ok {
					return false, fmt.Errorf("org-register failed") // already set in diag
				}
				orgSubmitted = true
			}
		} else if !status.Node.Registered {
			if !nodeSubmitted {
				ok, _ := r.apiRequest(ctx, http.MethodPost, r.apiPath(data, "network/nodes/self"), struct{}{}, nil, diagnostics, cancelInfo)
				if !ok {
					return false, fmt.Errorf("node-register failed") // already set in diag
				}
				nodeSubmitted = true
			}
		}
		return true, fmt.Errorf("waiting for registration to complete")
	})
}

func (r *firefly_registrationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var data FireFlyRegistrationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	r.ensureRegistered(ctx, &data, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)

}

func (r *firefly_registrationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

	var data FireFlyRegistrationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	r.ensureRegistered(ctx, &data, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *firefly_registrationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data FireFlyRegistrationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	var status FireFlyStatusAPIModel
	ok, _ := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data, "status"), nil, &status, &resp.Diagnostics)
	if !ok {
		return
	}
	if registered := status.toData(&data, &resp.Diagnostics); !registered {
		// We're not registered, remove the registration resource
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *firefly_registrationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// no-op
}
