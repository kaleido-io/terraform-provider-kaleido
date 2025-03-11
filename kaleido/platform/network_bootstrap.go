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
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type NetworkBootstrapDatasourceModel struct {
	Network        types.String    `tfsdk:"network"`
	Environment    types.String    `tfsdk:"environment"`
	BootstrapFiles *BootstrapFiles `tfsdk:"bootstrap_files"`
}

type BootstrapFiles struct {
	Name  types.String `tfsdk:"name"`
	Files types.Map    `tfsdk:"files"`
}

type NetworkInitData struct {
	Name  string              `json:"name"`
	Files map[string]*FileAPI `json:"files"`
}

func NetworkBootstrapDatasourceModelFactory() datasource.DataSource {
	return &networkBootstrapDatasource{}
}

type networkBootstrapDatasource struct {
	commonDataSource
}

func (r *networkBootstrapDatasource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_network_bootstrap_data"
}

func (r *networkBootstrapDatasource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"network": &schema.StringAttribute{
				Required: true,
			},
			"environment": &schema.StringAttribute{
				Required:    true,
				Description: "Environment ID",
			},
			"bootstrap_files": &schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"files": &schema.MapNestedAttribute{
						Required: true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"type": &schema.StringAttribute{
									Optional: true,
								},
								"data": &schema.SingleNestedAttribute{
									Optional: true,
									Attributes: map[string]schema.Attribute{
										/*"base64": &schema.StringAttribute{
											Optional: true,
										},*/
										"text": &schema.StringAttribute{
											Optional: true,
										},
										/*"hex": &schema.StringAttribute{
											Optional: true,
										},*/
									},
								},
							},
						},
					},
					"name": &schema.StringAttribute{
						Required: true,
					},
				},
			},
		},
	}
}

func (r *networkBootstrapDatasource) apiPath(data *NetworkBootstrapDatasourceModel) string {
	path := fmt.Sprintf("/api/v1/environments/%s/networks/%s/initdata", data.Environment.ValueString(), data.Network.ValueString())
	return path
}

func (r *networkBootstrapDatasource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {

	var data NetworkBootstrapDatasourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	var api NetworkInitData

	ok, status := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics, Allow404())
	if !ok {
		return
	}
	if status == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	api.toBootstrapData(ctx, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (api *NetworkInitData) toBootstrapData(ctx context.Context, data *NetworkBootstrapDatasourceModel, diagnostics *diag.Diagnostics) {

	/*initDataAttrs := map[string]attr.Type{
		"name": types.StringType,
		"files": types.MapType{
			ElemType: types.ObjectType{
				AttrTypes: fileAttrs,
			},
		},
	}*/
	if api != nil {
		var d diag.Diagnostics

		data.BootstrapFiles = &BootstrapFiles{}

		//copy api.Name to data.BoostrapFiles.Name
		nameValue := types.StringValue(api.Name)
		data.BootstrapFiles.Name = nameValue

		//iterate api.Files
		dataAttrs := map[string]attr.Type{
			//"base64": types.StringType,
			"text": types.StringType,
			//"hex":    types.StringType,
		}
		fileAttrs := map[string]attr.Type{
			"type": types.StringType,
			"data": types.ObjectType{
				AttrTypes: dataAttrs,
			},
		}
		files := map[string]attr.Value{}

		for k, f := range api.Files {
			file := map[string]attr.Value{}
			file["type"] = types.StringValue(f.Type)

			data := map[string]attr.Value{}

			if len(f.Data.Text) > 0 {
				data["text"] = types.StringValue(f.Data.Text)
			}

			/*if len(f.Data.Base64) > 0 {
				data["base64"] = types.StringValue(f.Data.Base64)
			}

			if len(f.Data.Hex) > 0 {
				data["hex"] = types.StringValue(f.Data.Hex)
			}*/

			tfData, d := types.ObjectValue(dataAttrs, data)
			diagnostics.Append(d...)
			file["data"] = tfData

			tfFile, d := types.ObjectValue(fileAttrs, file)
			diagnostics.Append(d...)
			files[k] = tfFile
		}
		data.BootstrapFiles.Files, d = types.MapValue(types.ObjectType{
			AttrTypes: fileAttrs,
		}, files)
		diagnostics.Append(d...)

		/*for name, data := range api.Files {
			fileData, diag := types.ObjectValueFrom(ctx, fileAttrs, data)
			diagnostics.Append(diag...)

			filesValue[name] = fileData
		}*/

		//files
		/*filesValue := make(map[string]attr.Value)
		for name, data := range api.Files {
			fileData, _ := types.ObjectValueFrom(ctx, fileAttrs, data)
			filesValue[name] = fileData
		}*/

		//data.StatusInitFiles, _ = types.MapValue(types.StringType, statusInitFiles)
		//diagnostics.Append(d...)

		//var bootstrapFiles *BootstrapFiles
		//diag := api.Files.As(ctx, &bootstrapFiles, basetypes.ObjectAsOptions{})

		//filesMapValue, diag := types.MapValueFrom(ctx, types.Map{}, filesValue)
		//filesMapValue, diag := types.MapValueFrom(ctx, types.ObjectType{AttrTypes: fileAttrs}, filesValue)
		//diagnostics.Append(diag...)
		//nameValue := types.StringValue(api.Name)

		//nameStringValue := types.StringValue(api.Name)

	} else {
		diagnostics.AddError(
			"API failed to get network initdata",
			"",
		)
	}

}
