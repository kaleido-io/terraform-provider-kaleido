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
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type PolicyIdentityResourceModel struct {
	ID                       types.String `tfsdk:"id"`
	Environment              types.String `tfsdk:"environment"`
	Service                  types.String `tfsdk:"service"`
	Name                     types.String `tfsdk:"name"`
	Description              types.String `tfsdk:"description"`
	Owner                    types.String `tfsdk:"owner"`
	PreferredAssertionMethod types.String `tfsdk:"preferred_assertion_method"`
	AssertionMethod          types.List   `tfsdk:"assertion_method"`
	NotificationMethod       types.List   `tfsdk:"notification_method"`
}

type PolicyIdentityAPIModel struct {
	ID                       string               `json:"id,omitempty"`
	Created                  *time.Time           `json:"created,omitempty"`
	Updated                  *time.Time           `json:"updated,omitempty"`
	Name                     string               `json:"name,omitempty"`
	Description              string               `json:"description,omitempty"`
	Owner                    string               `json:"owner,omitempty"`
	PreferredAssertionMethod string               `json:"preferredAssertionMethod,omitempty"`
	AssertionMethod          []AssertionMethod    `json:"assertionMethod,omitempty"`
	NotificationMethod       []NotificationMethod `json:"notificationMethod,omitempty"`
}

type AssertionMethod struct {
	ID                   string     `json:"id,omitempty"`
	IdentityID           string     `json:"identityId,omitempty"`
	Name                 string     `json:"name,omitempty"`
	Type                 string     `json:"type,omitempty"`
	SigningMethod        string     `json:"signingMethod,omitempty"`
	VerificationMaterial string     `json:"verificationMaterial,omitempty"`
	Created              *time.Time `json:"created,omitempty"`
	Updated              *time.Time `json:"updated,omitempty"`
	Expires              *time.Time `json:"expires,omitempty"`
	Revoked              *time.Time `json:"revoked,omitempty"`
}

type NotificationMethod struct {
	Name  string `json:"name,omitempty"`
	Type  string `json:"type,omitempty"`
	Value string `json:"value,omitempty"`
}

func PMSIdentityResourceFactory() resource.Resource {
	return &policyIdentityResource{}
}

type policyIdentityResource struct {
	commonResource
}

func (r *policyIdentityResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_pms_identity"
}

func (r *policyIdentityResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The Policy Identity resource allows you to manage identities in the Policy Manager.",
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the identity",
			},
			"environment": &schema.StringAttribute{
				Required:    true,
				Description: "The environment ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"service": &schema.StringAttribute{
				Required:    true,
				Description: "The service ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": &schema.StringAttribute{
				Required:    true,
				Description: "The name of the identity",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": &schema.StringAttribute{
				Optional:    true,
				Description: "A description of the identity",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"owner": &schema.StringAttribute{
				Optional:    true,
				Description: "Optional owner (KID) of the identity, e.g. a user or application",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"preferred_assertion_method": &schema.StringAttribute{
				Optional:    true,
				Description: "The preferred assertion method for the identity",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"assertion_method": &schema.ListNestedAttribute{
				Optional:    true,
				Description: "Array of verification methods (cryptographic keys) that can be used to prove statements made by this identity",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": &schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "Unique ID of the assertion method",
						},
						"identity_id": &schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "ID of the identity this assertion method belongs to",
						},
						"name": &schema.StringAttribute{
							Optional:    true,
							Description: "Name of the assertion method",
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
							},
						},
						"type": &schema.StringAttribute{
							Optional:    true,
							Description: "Type of the assertion method",
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
							},
						},
						"signing_method": &schema.StringAttribute{
							Optional:    true,
							Description: "Signing method for the assertion method",
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
							},
						},
						"verification_material": &schema.StringAttribute{
							Optional:    true,
							Description: "Verification material for the assertion method",
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
							},
						},
						"created": &schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "Creation timestamp",
						},
						"expires": &schema.StringAttribute{
							Optional:    true,
							Description: "Expiration timestamp",
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
							},
						},
						"revoked": &schema.StringAttribute{
							Optional:    true,
							Description: "Revocation timestamp",
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
							},
						},
					},
				},
			},
			"notification_method": &schema.ListNestedAttribute{
				Optional:    true,
				Description: "Array of notification methods (e.g. email, phone) associated with this identity",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": &schema.StringAttribute{
							Optional:    true,
							Description: "Name of the notification method",
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
							},
						},
						"type": &schema.StringAttribute{
							Optional:    true,
							Description: "Type of the notification method",
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
							},
						},
						"value": &schema.StringAttribute{
							Optional:    true,
							Description: "Value of the notification method as JSON string",
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
							},
						},
					},
				},
			},
		},
	}
}

func (r *policyIdentityResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.commonResource.Configure(ctx, req, resp)
}

func (r *policyIdentityResource) apiPath(data *PolicyIdentityResourceModel) string {
	env := data.Environment.ValueString()
	service := data.Service.ValueString()

	if data.ID.IsNull() || data.ID.IsUnknown() {
		return fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/identities", env, service)
	}
	return fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/identities/%s?fetchDetails=true", env, service, data.ID.ValueString())
}

func (r *policyIdentityResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PolicyIdentityResourceModel
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var api PolicyIdentityAPIModel
	r.toAPI(&data, &api)
	ok, _ := r.apiRequest(ctx, "POST", r.apiPath(&data), api, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	r.toData(&api, &data)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *policyIdentityResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PolicyIdentityResourceModel
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var api PolicyIdentityAPIModel
	ok, _ := r.apiRequest(ctx, "GET", r.apiPath(&data), nil, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	r.toData(&api, &data)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *policyIdentityResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	//updates are not supported - requires replacement
	resp.Diagnostics.AddError("Update not supported", "Policy identities cannot be updated. Use replace instead.")
}

func (r *policyIdentityResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PolicyIdentityResourceModel
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, _ = r.apiRequest(ctx, "DELETE", r.apiPath(&data), nil, nil, &resp.Diagnostics)
}

func (r *policyIdentityResource) toAPI(data *PolicyIdentityResourceModel, api *PolicyIdentityAPIModel) {
	api.Name = data.Name.ValueString()
	api.Description = data.Description.ValueString()
	api.Owner = data.Owner.ValueString()
	api.PreferredAssertionMethod = data.PreferredAssertionMethod.ValueString()

	// Convert assertion methods
	if !data.AssertionMethod.IsNull() && !data.AssertionMethod.IsUnknown() {
		var assertionMethods []AssertionMethod
		for _, item := range data.AssertionMethod.Elements() {
			if obj, ok := item.(types.Object); ok {
				am := AssertionMethod{}
				attrs := obj.Attributes()

				if val, ok := attrs["id"]; ok && !val.IsNull() {
					am.ID = val.(types.String).ValueString()
				}
				if val, ok := attrs["identity_id"]; ok && !val.IsNull() {
					am.IdentityID = val.(types.String).ValueString()
				}
				if val, ok := attrs["name"]; ok && !val.IsNull() {
					am.Name = val.(types.String).ValueString()
				}
				if val, ok := attrs["type"]; ok && !val.IsNull() {
					am.Type = val.(types.String).ValueString()
				}
				if val, ok := attrs["signing_method"]; ok && !val.IsNull() {
					am.SigningMethod = val.(types.String).ValueString()
				}
				if val, ok := attrs["verification_material"]; ok && !val.IsNull() {
					am.VerificationMaterial = val.(types.String).ValueString()
				}
				assertionMethods = append(assertionMethods, am)
			}
		}
		api.AssertionMethod = assertionMethods
	}

	// Convert notification methods
	if !data.NotificationMethod.IsNull() && !data.NotificationMethod.IsUnknown() {
		var notificationMethods []NotificationMethod
		for _, item := range data.NotificationMethod.Elements() {
			if obj, ok := item.(types.Object); ok {
				nm := NotificationMethod{}
				attrs := obj.Attributes()

				if val, ok := attrs["name"]; ok && !val.IsNull() {
					nm.Name = val.(types.String).ValueString()
				}
				if val, ok := attrs["type"]; ok && !val.IsNull() {
					nm.Type = val.(types.String).ValueString()
				}
				if val, ok := attrs["value"]; ok && !val.IsNull() {
					nm.Value = val.(types.String).ValueString()
				}
				notificationMethods = append(notificationMethods, nm)
			}
		}
		api.NotificationMethod = notificationMethods
	}
}

func (r *policyIdentityResource) toData(api *PolicyIdentityAPIModel, data *PolicyIdentityResourceModel) {
	data.ID = types.StringValue(api.ID)
	data.Name = types.StringValue(api.Name)
	// Note: environment and service are not returned by the API, they remain as set in the resource

	if api.Description != "" {
		data.Description = types.StringValue(api.Description)
	} else {
		data.Description = types.StringNull()
	}

	if api.Owner != "" {
		data.Owner = types.StringValue(api.Owner)
	} else {
		data.Owner = types.StringNull()
	}

	if api.PreferredAssertionMethod != "" {
		data.PreferredAssertionMethod = types.StringValue(api.PreferredAssertionMethod)
	} else {
		data.PreferredAssertionMethod = types.StringNull()
	}

	// Convert assertion methods
	if len(api.AssertionMethod) > 0 {
		var assertionMethods []attr.Value
		for _, am := range api.AssertionMethod {
			attrs := map[string]attr.Value{
				"id":                    types.StringValue(am.ID),
				"identity_id":           types.StringValue(am.IdentityID),
				"name":                  types.StringValue(am.Name),
				"type":                  types.StringValue(am.Type),
				"signing_method":        types.StringValue(am.SigningMethod),
				"verification_material": types.StringValue(am.VerificationMaterial),
			}

			// Handle time fields
			if am.Created != nil {
				attrs["created"] = types.StringValue(am.Created.Format(time.RFC3339))
			} else {
				attrs["created"] = types.StringNull()
			}

			if am.Expires != nil {
				attrs["expires"] = types.StringValue(am.Expires.Format(time.RFC3339))
			} else {
				attrs["expires"] = types.StringNull()
			}

			if am.Revoked != nil {
				attrs["revoked"] = types.StringValue(am.Revoked.Format(time.RFC3339))
			} else {
				attrs["revoked"] = types.StringNull()
			}

			obj, _ := types.ObjectValue(map[string]attr.Type{
				"id":                    types.StringType,
				"identity_id":           types.StringType,
				"name":                  types.StringType,
				"type":                  types.StringType,
				"signing_method":        types.StringType,
				"verification_material": types.StringType,
				"created":               types.StringType,
				"expires":               types.StringType,
				"revoked":               types.StringType,
			}, attrs)

			assertionMethods = append(assertionMethods, obj)
		}
		data.AssertionMethod = types.ListValueMust(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"id":                    types.StringType,
				"identity_id":           types.StringType,
				"name":                  types.StringType,
				"type":                  types.StringType,
				"signing_method":        types.StringType,
				"verification_material": types.StringType,
				"created":               types.StringType,
				"expires":               types.StringType,
				"revoked":               types.StringType,
			},
		}, assertionMethods)
	} else {
		data.AssertionMethod = types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"id":                    types.StringType,
				"identity_id":           types.StringType,
				"name":                  types.StringType,
				"type":                  types.StringType,
				"signing_method":        types.StringType,
				"verification_material": types.StringType,
				"created":               types.StringType,
				"expires":               types.StringType,
				"revoked":               types.StringType,
			},
		})
	}

	// Convert notification methods
	if len(api.NotificationMethod) > 0 {
		var notificationMethods []attr.Value
		for _, nm := range api.NotificationMethod {
			attrs := map[string]attr.Value{
				"name":  types.StringValue(nm.Name),
				"type":  types.StringValue(nm.Type),
				"value": types.StringValue(nm.Value),
			}

			obj, _ := types.ObjectValue(map[string]attr.Type{
				"name":  types.StringType,
				"type":  types.StringType,
				"value": types.StringType,
			}, attrs)

			notificationMethods = append(notificationMethods, obj)
		}
		data.NotificationMethod = types.ListValueMust(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"name":  types.StringType,
				"type":  types.StringType,
				"value": types.StringType,
			},
		}, notificationMethods)
	} else {
		data.NotificationMethod = types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"name":  types.StringType,
				"type":  types.StringType,
				"value": types.StringType,
			},
		})
	}
}
