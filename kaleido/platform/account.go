package platform

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"net/http"
)

type AccountDatasourceModel struct {
	AccountID types.String `tfsdk:"account_id"`
}

type SelfIdentityAPIModel struct {
	AccountID string `json:"accountId,omitempty"`
	IsAdmin   bool   `json:"isAdmin,omitempty"`
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
