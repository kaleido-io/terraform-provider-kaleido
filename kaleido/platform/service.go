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
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ServiceResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Environment         types.String `tfsdk:"environment"`
	Runtime             types.String `tfsdk:"runtime"`
	Type                types.String `tfsdk:"type"`
	Name                types.String `tfsdk:"name"`
	StackID             types.String `tfsdk:"stack_id"`
	EnvironmentMemberID types.String `tfsdk:"environment_member_id"`
	ConfigJSON          types.String `tfsdk:"config_json"`
	Endpoints           types.Map    `tfsdk:"endpoints"`
	Hostnames           types.Map    `tfsdk:"hostnames"`
	Filesets            types.Map    `tfsdk:"file_sets"`
	Credsets            types.Map    `tfsdk:"cred_sets"`
	ConnectivityJSON    types.String `tfsdk:"connectivity_json"`
	ForceDelete         types.Bool   `tfsdk:"force_delete"`
	WaitForReady        types.Bool   `tfsdk:"wait_for_ready"`
}

type ServiceAPIModel struct {
	ID                  string                        `json:"id,omitempty"`
	Created             *time.Time                    `json:"created,omitempty"`
	Updated             *time.Time                    `json:"updated,omitempty"`
	Type                string                        `json:"type"`
	Name                string                        `json:"name"`
	StackID             string                        `json:"stackId,omitempty"`
	Runtime             ServiceAPIRuntimeRef          `json:"runtime,omitempty"`
	Account             string                        `json:"account,omitempty"`
	EnvironmentMemberID string                        `json:"environmentMemberId,omitempty"`
	Deleted             bool                          `json:"deleted,omitempty"`
	Config              map[string]interface{}        `json:"config"`
	Endpoints           map[string]ServiceAPIEndpoint `json:"endpoints,omitempty"`
	Hostnames           map[string][]string           `json:"hostnames,omitempty"`
	Filesets            map[string]*FileSetAPI        `json:"fileSets,omitempty"`
	Credsets            map[string]*CredSetAPI        `json:"credSets,omitempty"`
	Status              string                        `json:"status,omitempty"`
	StatusDetails       ServiceStatusDetails          `json:"statusDetails,omitempty"`
}

type ServiceAPIRuntimeRef struct {
	ID string `json:"id"`
}

type ServiceAPIEndpoint struct {
	Type string   `json:"type,omitempty"`
	URLS []string `json:"urls,omitempty"`
}

type ServiceStatusDetails struct {
	Connectivity *Connectivity `json:"connectivity,omitempty"`
}

type Connectivity struct {
	Identity  string     `json:"identity,omitempty"`
	Endpoints []Endpoint `json:"endpoints,omitempty"`
}

type Endpoint struct {
	Host     string `json:"host,omitempty"`
	NAT      string `json:"nat,omitempty"`
	Port     int64  `json:"port,omitempty"`
	Protocol string `json:"protocol,omitempty"`
}

func ServiceResourceFactory() resource.Resource {
	return &serviceResource{}
}

type serviceResource struct {
	commonResource
}

func (r *serviceResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_service"
}

func (r *serviceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Each capability of the Kaleido platform is made available as a service.",
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"environment": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Environment ID",
			},
			"runtime": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Runtime ID",
			},
			"type": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Service Type",
			},
			"name": &schema.StringAttribute{
				Required:    true,
				Description: "Service Display Name",
			},
			"stack_id": &schema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"environment_member_id": &schema.StringAttribute{
				Computed: true,
			},
			"config_json": &schema.StringAttribute{
				Required: true,
			},
			"endpoints": &schema.MapNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": &schema.StringAttribute{
							Computed: true,
						},
						"urls": &schema.ListAttribute{
							Computed:    true,
							ElementType: types.StringType,
						},
					},
				},
			},
			"hostnames": &schema.MapAttribute{
				Optional: true,
				ElementType: types.ListType{
					ElemType: types.StringType,
				},
			},
			"file_sets": &schema.MapNestedAttribute{
				Description: "Some services require binary files as part of their configuration, such as x509 certificates, or large JSON/YAML configuration files to be passed directly down to the service for verification. The files are individually encrypted.",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"files": &schema.MapNestedAttribute{
							Required: true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"type": &schema.StringAttribute{
										Required: true,
									},
									"data": &schema.SingleNestedAttribute{
										Required:  true,
										Sensitive: true,
										Attributes: map[string]schema.Attribute{
											"base64": &schema.StringAttribute{
												Optional: true,
											},
											"text": &schema.StringAttribute{
												Optional: true,
											},
											"hex": &schema.StringAttribute{
												Optional: true,
											},
										},
									},
								},
							},
						},
					},
				},
			},
			"cred_sets": &schema.MapNestedAttribute{
				Description: "Credentials such as usernames and passwords, or API Keys, required to integrate with external systems are also stored and encrypted separately to the main configuration of the service.",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": &schema.StringAttribute{
							Required: true,
						},
						"basic_auth": &schema.SingleNestedAttribute{
							Optional: true,
							Attributes: map[string]schema.Attribute{
								"username": &schema.StringAttribute{
									Required: true,
								},
								"password": &schema.StringAttribute{
									Required:  true,
									Sensitive: true,
								},
							},
						},
						"key": &schema.SingleNestedAttribute{
							Optional: true,
							Attributes: map[string]schema.Attribute{
								"value": &schema.StringAttribute{
									Required:  true,
									Sensitive: true,
								},
							},
						},
					},
				},
			},
			"connectivity_json": &schema.StringAttribute{
				Computed: true,
			},
			"force_delete": &schema.BoolAttribute{
				Optional:    true,
				Description: "Set to `true` when you plan to delete a protected service like a Besu validator node. You must apply the value before you can successfully `terraform destroy` the protected service.",
			},
			"wait_for_ready": &schema.BoolAttribute{
				Optional:    true,
				Description: "Set to `false` to ignore the service's readiness status before proceeding. Defaults to `true`.",
			},
		},
	}
}

func (data *ServiceResourceModel) toAPI(ctx context.Context, api *ServiceAPIModel, diagnostics *diag.Diagnostics) {
	// required fields
	api.Type = data.Type.ValueString()
	api.Name = data.Name.ValueString()
	api.StackID = data.StackID.ValueString()
	api.Runtime.ID = data.Runtime.ValueString()
	api.Config = map[string]interface{}{}
	if !data.ConfigJSON.IsNull() {
		_ = json.Unmarshal([]byte(data.ConfigJSON.ValueString()), &api.Config)
	}
	// hostnames
	if !data.Hostnames.IsNull() {
		d := data.Hostnames.ElementsAs(ctx, &api.Hostnames, false)
		diagnostics.Append(d...)
	}
	// filesets (complex nested structure)
	if !data.Filesets.IsNull() {
		dataAttrs := map[string]attr.Type{
			"base64": types.StringType,
			"text":   types.StringType,
			"hex":    types.StringType,
		}
		fileAttrs := map[string]attr.Type{
			"type": types.StringType,
			"data": types.ObjectType{
				AttrTypes: dataAttrs,
			},
		}
		fileSetAttrs := map[string]attr.Type{
			"files": types.MapType{
				ElemType: types.ObjectType{
					AttrTypes: fileAttrs,
				},
			},
		}
		api.Filesets = make(map[string]*FileSetAPI)
		tfFilesets := data.Filesets.Elements()
		for fileSetName, tfFileSet := range tfFilesets {
			tfFileSet, d := types.ObjectValueFrom(ctx, fileSetAttrs, tfFileSet)
			diagnostics.Append(d...)
			tfFiles, d := types.MapValueFrom(ctx, types.ObjectType{AttrTypes: fileAttrs}, tfFileSet.Attributes()["files"])
			diagnostics.Append(d...)
			fs := &FileSetAPI{
				Name:  fileSetName,
				Files: make(map[string]*FileAPI),
			}
			for filename, tfFileVal := range tfFiles.Elements() {
				tfFile, d := types.ObjectValueFrom(ctx, fileAttrs, tfFileVal)
				diagnostics.Append(d...)
				tfFileAttrs := tfFile.Attributes()
				tfFileData, d := types.ObjectValueFrom(ctx, dataAttrs, tfFileAttrs["data"])
				diagnostics.Append(d...)
				f := &FileAPI{
					Type: tfFileAttrs["type"].(types.String).ValueString(),
				}
				tfData := tfFileData.Attributes()
				if !tfData["base64"].IsNull() {
					f.Data.Base64 = tfData["base64"].(types.String).ValueString()
				} else if !tfData["text"].IsNull() {
					f.Data.Text = tfData["text"].(types.String).ValueString()
				} else if !tfData["hex"].IsNull() {
					f.Data.Hex = tfData["hex"].(types.String).ValueString()
				} else {
					diagnostics.AddError("missing data", fmt.Sprintf("must specify base64, text, or hexx data for file '%s'", filename))
					return
				}
				fs.Files[filename] = f
			}
			api.Filesets[fileSetName] = fs
		}
	}
	// credsets (complex nested structure)
	if !data.Credsets.IsNull() {
		basicAuthAttrs := map[string]attr.Type{
			"username": types.StringType,
			"password": types.StringType,
		}
		keyAttrs := map[string]attr.Type{
			"value": types.StringType,
		}
		api.Credsets = make(map[string]*CredSetAPI)
		tfCredsets := data.Credsets.Elements()
		for credSetName, tfCredSetAttr := range tfCredsets {
			tfCredSet := tfCredSetAttr.(types.Object)
			tfCredSetAttrs := tfCredSet.Attributes()
			crType := tfCredSetAttrs["type"].(types.String).ValueString()
			cr := &CredSetAPI{
				Name: credSetName,
				Type: crType,
			}
			if crType == "basic_auth" && !tfCredSetAttrs["basic_auth"].IsNull() {
				tfBasicAuth, d := types.ObjectValueFrom(ctx, basicAuthAttrs, tfCredSetAttrs["basic_auth"])
				diagnostics.Append(d...)
				tfBasicAuthAttrs := tfBasicAuth.Attributes()
				cr.BasicAuth = &CredSetBasicAuthAPI{
					Username: tfBasicAuthAttrs["username"].(types.String).ValueString(),
					Password: tfBasicAuthAttrs["password"].(types.String).ValueString(),
				}
			} else if crType == "key" && !tfCredSetAttrs["key"].IsNull() {
				tfKey, d := types.ObjectValueFrom(ctx, keyAttrs, tfCredSetAttrs["key"])
				diagnostics.Append(d...)
				tfKeyAttrs := tfKey.Attributes()
				cr.Key = &CredSetKeyAPI{
					Value: tfKeyAttrs["value"].(types.String).ValueString(),
				}
			} else {
				diagnostics.AddError("missing credential", fmt.Sprintf("must specify key/basic_auth as appropriate for type '%s'", crType))
				return
			}
			api.Credsets[credSetName] = cr
		}
	}

	//connectivity is computed. So only goes from api to data not via versa
}

func (api *ServiceAPIModel) toData(data *ServiceResourceModel, diagnostics *diag.Diagnostics) {
	data.ID = types.StringValue(api.ID)
	data.EnvironmentMemberID = types.StringValue(api.EnvironmentMemberID)
	endpoints := map[string]attr.Value{}
	endpointAttrTypes := map[string]attr.Type{
		"type": types.StringType,
		"urls": types.ListType{ElemType: types.StringType},
	}
	var d diag.Diagnostics
	for k, e := range api.Endpoints {
		endpoint := map[string]attr.Value{}
		endpoint["type"] = types.StringValue(e.Type)
		urls := make([]attr.Value, len(e.URLS))
		for i, u := range e.URLS {
			urls[i] = types.StringValue(u)
		}
		tfURLs, d := types.ListValue(types.StringType, urls)
		diagnostics.Append(d...)
		endpoint["urls"] = tfURLs
		tfEndpoint, d := types.ObjectValue(endpointAttrTypes, endpoint)
		diagnostics.Append(d...)
		endpoints[k] = tfEndpoint
	}
	data.Endpoints, d = types.MapValue(types.ObjectType{
		AttrTypes: endpointAttrTypes,
	}, endpoints)
	diagnostics.Append(d...)

	//connectivity
	if api.StatusDetails.Connectivity != nil {
		d, err := json.Marshal(api.StatusDetails.Connectivity)
		if err != nil {
			diagnostics.AddError("failed to marshal connectivity", err.Error())
			return
		}
		connectivityJSON := string(d)
		data.ConnectivityJSON = types.StringValue(connectivityJSON)
	} else {
		data.ConnectivityJSON = types.StringValue("")
	}
}

func (r *serviceResource) apiPath(data *ServiceResourceModel) string {
	path := fmt.Sprintf("/api/v1/environments/%s/services", data.Environment.ValueString())
	if data.ID.ValueString() != "" {
		path = path + "/" + data.ID.ValueString()
	}

	if data.ForceDelete.ValueBool() {
		path = path + "?force=true"
	}

	return path
}

func (r *serviceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var data ServiceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	var api ServiceAPIModel
	data.toAPI(ctx, &api, &resp.Diagnostics)
	ok, _ := r.apiRequest(ctx, http.MethodPost, r.apiPath(&data), api, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	api.toData(&data, &resp.Diagnostics) // need the ID copied over

	if data.WaitForReady.IsNull() || !data.WaitForReady.ValueBool() {
		r.waitForReadyStatus(ctx, r.apiPath(&data), &resp.Diagnostics)
	} else {
		// no need to re-read from api, so just set the state and return
		resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
		return
	}

	//re-read from api
	api.ID = data.ID.ValueString()
	ok, _ = r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics, &APIRequestOption{})
	if !ok {
		return
	}

	api.toData(&data, &resp.Diagnostics) // need the latest status after the readiness check completes, to extract generated values
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *serviceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

	var data ServiceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)

	// Read full current object
	var api ServiceAPIModel
	if ok, _ := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics); !ok {
		return
	}

	// Update from plan
	data.toAPI(ctx, &api, &resp.Diagnostics)
	if ok, _ := r.apiRequest(ctx, http.MethodPut, r.apiPath(&data), api, &api, &resp.Diagnostics); !ok {
		return
	}

	api.toData(&data, &resp.Diagnostics) // need the ID copied over
	r.waitForReadyStatus(ctx, r.apiPath(&data), &resp.Diagnostics)
	//re-read from api
	if ok, _ := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics); !ok {
		return
	}
	api.toData(&data, &resp.Diagnostics) // need the latest status after the readiness check completes, to extract generated values
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *serviceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ServiceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	var api ServiceAPIModel
	api.ID = data.ID.ValueString()
	ok, status := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics, Allow404())
	if !ok {
		return
	}
	if status == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	api.toData(&data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *serviceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ServiceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data), nil, nil, &resp.Diagnostics, Allow404())

	r.waitForRemoval(ctx, r.apiPath(&data), &resp.Diagnostics)
}
