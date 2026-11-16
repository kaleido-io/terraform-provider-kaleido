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
	"net/http"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// PermissionSetDatasourceModel is a permission set read by name or id: a built-in set, such as
// admin or full-access, or one made elsewhere.
type PermissionSetDatasourceModel struct {
	ID              types.String    `tfsdk:"id"`
	Name            types.String    `tfsdk:"name"`
	Description     types.String    `tfsdk:"description"`
	Builtin         types.Bool      `tfsdk:"builtin"`
	PermissionsJSON jsonStringValue `tfsdk:"permissions_json"`
	ParamsJSON      jsonStringValue `tfsdk:"params_json"`
	Labels          types.Map       `tfsdk:"labels"`
}

func PermissionSetDatasourceFactory() datasource.DataSource {
	return &permissionSetDatasource{}
}

type permissionSetDatasource struct {
	commonDataSource
}

func (s *permissionSetDatasource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_permission_set"
}

func (s *permissionSetDatasource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Read a permission set by name or id: a built-in set (such as admin, full-access or platform-api-user) or one made elsewhere, to give it in a grant. Needs access grants on (the access_grants toggle).",
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Id of the set. Specify either id or name",
			},
			"name": &schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Name of the set. Specify either id or name",
			},
			"description": &schema.StringAttribute{
				Computed: true,
			},
			"builtin": &schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the set is built in, and so read-only",
			},
			"permissions_json": &schema.StringAttribute{
				CustomType:  jsonStringType{},
				Computed:    true,
				Description: "The set's permissions",
			},
			"params_json": &schema.StringAttribute{
				CustomType:  jsonStringType{},
				Computed:    true,
				Description: "The params the set declares, which a grant of it fills",
			},
			"labels": &schema.MapAttribute{
				ElementType: types.StringType,
				Computed:    true,
			},
		},
	}
}

func (s *permissionSetDatasource) ConfigValidators(_ context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(path.MatchRoot("id"), path.MatchRoot("name")),
	}
}

func (s *permissionSetDatasource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data PermissionSetDatasourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	nameOrID := data.ID.ValueString()
	if nameOrID == "" {
		nameOrID = data.Name.ValueString()
	}
	var api PermissionSetAPIModel
	if ok, _ := s.apiRequest(ctx, http.MethodGet, "/api/v1/permissionsets/"+url.PathEscape(nameOrID), nil, &api, &resp.Diagnostics); !ok {
		return
	}

	data.ID = types.StringValue(api.ID)
	if api.Name != nil {
		data.Name = types.StringValue(*api.Name)
	}
	data.Description = types.StringNull()
	if api.Description != nil {
		data.Description = types.StringValue(*api.Description)
	}
	data.Builtin = types.BoolValue(api.Builtin != nil && *api.Builtin)
	data.PermissionsJSON = jsonFromAPI(api.Permissions)
	data.ParamsJSON = jsonFromAPI(api.Params)
	data.Labels = labelsFromAPI(ctx, api.Labels, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}
