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

type NetworkResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Environment         types.String `tfsdk:"environment"`
	Type                types.String `tfsdk:"type"`
	Name                types.String `tfsdk:"name"`
	ConfigJSON          types.String `tfsdk:"config_json"`
	Info                types.Map    `tfsdk:"info"`
	EnvironmentMemberID types.String `tfsdk:"environment_member_id"`
	InitFiles           types.String `tfsdk:"init_files"`
	InitMode            types.String `tfsdk:"init_mode"`
	Initialized         types.Bool   `tfsdk:"initialized"`
	Filesets            types.Map    `tfsdk:"file_sets"`
	Credsets            types.Map    `tfsdk:"cred_sets"`
	StatusInitFiles     types.Map    `tfsdk:"status_init_files"`
	ForceDelete         types.Bool   `tfsdk:"force_delete"`
}

type NetworkAPIModel struct {
	ID                  string                 `json:"id,omitempty"`
	Created             *time.Time             `json:"created,omitempty"`
	Updated             *time.Time             `json:"updated,omitempty"`
	Type                string                 `json:"type"`
	Name                string                 `json:"name"`
	Config              map[string]interface{} `json:"config"`
	EnvironmentMemberID string                 `json:"environmentMemberId,omitempty"`
	Status              string                 `json:"status,omitempty"`
	Deleted             bool                   `json:"deleted,omitempty"`
	InitFiles           string                 `json:"initFiles,omitempty"`
	InitMode            string                 `json:"initMode,omitempty"`
	Initialized         bool                   `json:"initialized,omitempty"`
	Filesets            map[string]*FileSetAPI `json:"fileSets,omitempty"`
	Credsets            map[string]*CredSetAPI `json:"credSets,omitempty"`
	StatusDetails       NetworkStatusDetails   `json:"statusDetails,omitempty"`
}

type NetworkStatusDetails map[string]interface{}

func NetworkResourceFactory() resource.Resource {
	return &networkResource{}
}

type networkResource struct {
	commonResource
}

func (r *networkResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_network"
}

func (r *networkResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Networks provide an anchor object for multiple services that need to communicate together, and allow services to discover other services they need communicate with.",
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"type": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Network Type. Options are `Besu` and `IPFS`",
			},
			"name": &schema.StringAttribute{
				Required:    true,
				Description: "Network Display Name",
			},
			"environment": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Environment ID",
			},
			"environment_member_id": &schema.StringAttribute{
				Computed: true,
			},
			"config_json": &schema.StringAttribute{
				Required: true,
			},
			"info": &schema.MapAttribute{
				Description: "Top-level config captured from the network after creation, including generated values like the chain id",
				Computed:    true,
				ElementType: types.StringType,
			},
			"init_files": &schema.StringAttribute{
				Optional:    true,
				Description: "",
			},
			"init_mode": &schema.StringAttribute{
				Optional:    true,
				Description: "Options are `automated`(default) or `manual`.",
			},
			"initialized": &schema.BoolAttribute{
				Computed: true,
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
			"status_init_files": &schema.MapAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
			"force_delete": &schema.BoolAttribute{
				Optional:    true,
				Description: "Set to `true` when you plan to delete a protected network. You must apply the value before you can successfully `terraform destroy` the protected network.",
			},
		},
	}
}

func (data *NetworkResourceModel) toAPI(ctx context.Context, api *NetworkAPIModel, diagnostics *diag.Diagnostics) {
	// required fields
	api.Type = data.Type.ValueString()
	api.Name = data.Name.ValueString()
	// optional fields
	api.Config = map[string]interface{}{}
	if !data.ConfigJSON.IsNull() {
		_ = json.Unmarshal([]byte(data.ConfigJSON.ValueString()), &api.Config)
	}

	if !data.InitMode.IsNull() {
		api.InitMode = data.InitMode.ValueString()
	}

	if !data.InitFiles.IsNull() {
		api.InitFiles = data.InitFiles.ValueString()
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

}

func (api *NetworkAPIModel) toData(data *NetworkResourceModel, diagnostics *diag.Diagnostics) {
	data.ID = types.StringValue(api.ID)
	data.Initialized = types.BoolValue(api.Initialized)
	data.EnvironmentMemberID = types.StringValue(api.EnvironmentMemberID)
	info := make(map[string]attr.Value)
	for k, v := range api.Config {
		v, isString := v.(string)
		if isString && v != "" {
			info[k] = types.StringValue(v)
		}
	}

	var d diag.Diagnostics
	data.Info, d = types.MapValue(types.StringType, info)
	diagnostics.Append(d...)

	//status init files
	statusInitFiles := make(map[string]attr.Value)

	_, initFilesExist := api.StatusDetails["initFiles"]
	if initFilesExist {
		initFiles, ok := api.StatusDetails["initFiles"].(map[string]interface{})
		if !ok {
			diagnostics.AddError("invalid initFiles", "initFiles must be a map of strings")
			return
		}
		for name, data := range initFiles {
			strData, ok := data.(string)
			if !ok {
				diagnostics.AddError("invalid initFiles", "initFiles must be a map of strings")
				return
			}

			initFileData := types.StringValue(strData)
			statusInitFiles[name] = initFileData
		}
	}
	data.StatusInitFiles, d = types.MapValue(types.StringType, statusInitFiles)
	diagnostics.Append(d...)
}

func (r *networkResource) apiPath(data *NetworkResourceModel) string {
	path := fmt.Sprintf("/api/v1/environments/%s/networks", data.Environment.ValueString())
	if data.ID.ValueString() != "" {
		path = path + "/" + data.ID.ValueString()
	}
	if data.ForceDelete.ValueBool() {
		path = path + "?force=true"
	}
	return path
}

func (r *networkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var data NetworkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	var api NetworkAPIModel
	data.toAPI(ctx, &api, &resp.Diagnostics)
	ok, _ := r.apiRequest(ctx, http.MethodPost, r.apiPath(&data), api, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	api.toData(&data, &resp.Diagnostics) // need the ID copied over
	r.waitForReadyStatus(ctx, r.apiPath(&data), &resp.Diagnostics)
	api.toData(&data, &resp.Diagnostics) // need the latest status after the readiness check completes, to extract generated values
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)

}

func (r *networkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

	var data NetworkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)

	// Read full current object
	var api NetworkAPIModel
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
	api.toData(&data, &resp.Diagnostics) // need the latest status after the readiness check completes, to extract generated values
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *networkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data NetworkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	var api NetworkAPIModel
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

func (r *networkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data NetworkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data), nil, nil, &resp.Diagnostics, Allow404())

	r.waitForRemoval(ctx, r.apiPath(&data), &resp.Diagnostics)
}
