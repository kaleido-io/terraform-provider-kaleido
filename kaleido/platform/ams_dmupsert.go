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

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"gopkg.in/yaml.v3"
)

type AMSDMUpsertResourceModel struct {
	Environment    types.String `tfsdk:"environment"`
	Service        types.String `tfsdk:"service"`
	BulkUpsertYAML types.String `tfsdk:"bulk_upsert_yaml"`
}

func AMSDMUpsertResourceFactory() resource.Resource {
	return &ams_dmupsertResource{}
}

type ams_dmupsertResource struct {
	commonResource
}

func (r *ams_dmupsertResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_ams_dmupsert"
}

func (r *ams_dmupsertResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"environment": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"service": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"bulk_upsert_yaml": &schema.StringAttribute{
				Required:    true,
				Description: "This is a bulk upsert input payload in YAML/JSON",
			},
		},
	}
}

func (data *AMSDMUpsertResourceModel) toAPI(diagnostics *diag.Diagnostics) map[string]interface{} {
	var parsedYAML map[string]interface{}
	err := yaml.Unmarshal([]byte(data.BulkUpsertYAML.ValueString()), &parsedYAML)
	if err != nil {
		diagnostics.AddError("invalid task YAML", err.Error())
		return nil
	}
	return parsedYAML
}

func (r *ams_dmupsertResource) apiPath(data *AMSDMUpsertResourceModel) string {
	return fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/bulk/datamodel", data.Environment.ValueString(), data.Service.ValueString())
}

func (r *ams_dmupsertResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var data AMSDMUpsertResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	parsedYAML := data.toAPI(&resp.Diagnostics)
	if parsedYAML != nil {
		_, _ = r.apiRequest(ctx, http.MethodPut, r.apiPath(&data), parsedYAML, nil, &resp.Diagnostics)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)

}

func (r *ams_dmupsertResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

	var data AMSDMUpsertResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	parsedYAML := data.toAPI(&resp.Diagnostics)
	if parsedYAML != nil {
		_, _ = r.apiRequest(ctx, http.MethodPut, r.apiPath(&data), parsedYAML, nil, &resp.Diagnostics)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *ams_dmupsertResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// no-op
}

func (r *ams_dmupsertResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// no-op
}
