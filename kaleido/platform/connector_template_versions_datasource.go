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

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// connectorTemplateKindPaths maps each kind the data source accepts to its connector-manager
// metadata route segment.
var connectorTemplateKindPaths = map[string]string{
	"connector_flow": "connector-flows",
	"standard_api":   "standard-apis",
	"config_type":    "config-types",
}

type ConnectorTemplateVersionsDatasourceModel struct {
	Environment types.String                    `tfsdk:"environment"`
	Service     types.String                    `tfsdk:"service"`
	Kind        types.String                    `tfsdk:"kind"`
	Name        types.String                    `tfsdk:"name"`
	Latest      types.String                    `tfsdk:"latest"`
	Versions    []ConnectorTemplateVersionModel `tfsdk:"versions"`
}

type ConnectorTemplateVersionModel struct {
	Version            types.String `tfsdk:"version"`
	Latest             types.Bool   `tfsdk:"latest"`
	Supported          types.Bool   `tfsdk:"supported"`
	VersionDescription types.String `tfsdk:"version_description"`
}

type ConnectorTemplateVersionAPIModel struct {
	Version            string `json:"version"`
	Latest             bool   `json:"latest"`
	Supported          bool   `json:"supported"`
	VersionDescription string `json:"versionDescription,omitempty"`
}

func ConnectorTemplateVersionsDatasourceModelFactory() datasource.DataSource {
	return &connectorTemplateVersionsDatasource{}
}

type connectorTemplateVersionsDatasource struct {
	commonDataSource
}

func (s *connectorTemplateVersionsDatasource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_connector_template_versions"
}

func (s *connectorTemplateVersionsDatasource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The template versions a connector service stores for one connector flow, standard API or config type, newest first. " +
			"Set a connector flow's `version` to `latest` to have it upgrade whenever the connector offers a newer template - the upgrade " +
			"still appears in the plan, so it is always reviewable.",
		Attributes: map[string]schema.Attribute{
			"environment": &schema.StringAttribute{
				Required:    true,
				Description: "Environment ID",
			},
			"service": &schema.StringAttribute{
				Required:    true,
				Description: "Connector service ID",
			},
			"kind": &schema.StringAttribute{
				Required:    true,
				Description: "What the template is: connector_flow, standard_api or config_type.",
				Validators:  []validator.String{stringvalidator.OneOf("connector_flow", "standard_api", "config_type")},
			},
			"name": &schema.StringAttribute{
				Required:    true,
				Description: "Template name - for a connector flow or standard API its name (e.g. submission), for a config type its full name (e.g. evm.gasPricing).",
			},
			"latest": &schema.StringAttribute{
				Computed:    true,
				Description: "The newest version the connector service stores - the one deployed when a connector flow's version is omitted.",
			},
			"versions": &schema.ListNestedAttribute{
				Computed:    true,
				Description: "Every stored version, newest first.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"version": &schema.StringAttribute{
							Computed:    true,
							Description: "The version string.",
						},
						"latest": &schema.BoolAttribute{
							Computed:    true,
							Description: "True for the newest stored version.",
						},
						"supported": &schema.BoolAttribute{
							Computed: true,
							Description: "False when the version is below the protocol's minimum supported version: it can still be read, " +
								"but a connector flow cannot be deployed or upgraded to it.",
						},
						"version_description": &schema.StringAttribute{
							Computed:    true,
							Description: "What changed in this version, where the template describes it.",
						},
					},
				},
			},
		},
	}
}

func (s *connectorTemplateVersionsDatasource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ConnectorTemplateVersionsDatasourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	p := fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/metadata/%s/%s/versions",
		data.Environment.ValueString(), data.Service.ValueString(),
		connectorTemplateKindPaths[data.Kind.ValueString()], data.Name.ValueString())
	var api []ConnectorTemplateVersionAPIModel
	if ok, _ := s.apiRequest(ctx, http.MethodGet, p, nil, &api, &resp.Diagnostics); !ok {
		return
	}

	data.Latest = types.StringNull()
	data.Versions = make([]ConnectorTemplateVersionModel, 0, len(api))
	for _, v := range api {
		data.Versions = append(data.Versions, ConnectorTemplateVersionModel{
			Version:            types.StringValue(v.Version),
			Latest:             types.BoolValue(v.Latest),
			Supported:          types.BoolValue(v.Supported),
			VersionDescription: optionalString(v.VersionDescription),
		})
		if v.Latest {
			data.Latest = types.StringValue(v.Version)
		}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}
