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
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type datasourcePrivateStackBridge struct {
	baasBaseDatasource
}

func DatasourcePrivateStackBridgeFactory() datasource.DataSource {
	return &datasourcePrivateStackBridge{}
}

type PrivateStackBridgeResourceModel struct {
	ID            types.String `tfsdk:"id"`
	ConsortiumID  types.String `tfsdk:"consortium_id"`
	EnvironmentID types.String `tfsdk:"environment_id"`
	ServiceID     types.String `tfsdk:"service_id"`
	AppCredID     types.String `tfsdk:"appcred_id"`
	AppCredSecret types.String `tfsdk:"appcred_secret"`
	ConfigJSON    types.String `tfsdk:"config_json"`
}

func (d *datasourcePrivateStackBridge) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "kaleido_privatestack_bridge"
}

func (d *datasourcePrivateStackBridge) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"consortium_id": &schema.StringAttribute{
				Required: true,
			},
			"environment_id": &schema.StringAttribute{
				Required: true,
			},
			"service_id": &schema.StringAttribute{
				Required: true,
			},
			"appcred_id": &schema.StringAttribute{
				Description: "Optionally provide an application credential to inject into the downloaded config, making it ready for use",
				Optional:    true,
			},
			"appcred_secret": &schema.StringAttribute{
				Description: "Optionally provide an application credential to inject into the downloaded config, making it ready for use",
				Optional:    true,
				Sensitive:   true,
			},
			"config_json": &schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (d *datasourcePrivateStackBridge) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {

	var data PrivateStackBridgeResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	var conf map[string]interface{}
	res, err := d.baas.GetPrivateStackBridgeConfig(data.ConsortiumID.ValueString(), data.EnvironmentID.ValueString(), data.ServiceID.ValueString(), &conf)

	if err != nil {
		resp.Diagnostics.AddError("failed to get config", err.Error())
		return
	}

	status := res.StatusCode()
	if status != 200 {
		resp.Diagnostics.AddError("failed to list services", fmt.Sprintf("Failed to read config with id %s status was: %d, error: %s", data.ServiceID.ValueString(), status, res.String()))
	}

	appcredID := data.AppCredID.ValueString()
	appcredSecret := data.AppCredSecret.ValueString()
	if appcredID != "" && appcredSecret != "" {
		if nodesEntry, ok := conf["nodes"]; ok {
			if nodesArray, ok := nodesEntry.([]interface{}); ok {
				for _, nodeInterface := range nodesArray {
					if node, ok := nodeInterface.(map[string]interface{}); ok {
						node["auth"] = map[string]string{
							"user":   appcredID,
							"secret": appcredSecret,
						}
					}
				}
			}
		}
	}

	data.ID = types.StringValue(data.ServiceID.ValueString())
	confstr, _ := json.MarshalIndent(conf, "", "  ")
	data.ConfigJSON = types.StringValue(string(confstr))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
