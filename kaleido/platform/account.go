// Copyright © Kaleido, Inc. 2025

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

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type AccountDatasourceModel struct {
	AccountID types.String `tfsdk:"account_id"`
}

type SelfIdentityAPIModel struct {
	AccountID string `json:"accountId,omitempty"`
}

func (m SelfIdentityAPIModel) toData(_ context.Context, s *AccountDatasourceModel, d *diag.Diagnostics) {
	if m.AccountID != "" {
		s.AccountID = types.StringValue(m.AccountID)
	} else {
		d.AddError("AccountDatasourceModel", "Account ID is not set")
	}
}

func AccountDatasourceModelFactory() datasource.DataSource {
	return &AccountDatasource{}
}

type AccountDatasource struct {
	commonDataSource
}

func (s AccountDatasource) Metadata(ctx context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_account"
}

func (s AccountDatasource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetch the account id of your Kaleido platform account.",
		Attributes: map[string]schema.Attribute{
			"account_id": &schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (r *AccountDatasource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data AccountDatasourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	var api SelfIdentityAPIModel

	ok, status := r.apiRequest(ctx, http.MethodGet, "/api/v1/self/identity", nil, &api, &resp.Diagnostics, Allow404())
	if !ok {
		return
	}
	if status == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	api.toData(ctx, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}
