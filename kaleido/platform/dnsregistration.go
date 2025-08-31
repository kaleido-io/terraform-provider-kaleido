package platform

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type DNSRegistrationResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Environment types.String `tfsdk:"environment"`
	Subdomain   types.String `tfsdk:"subdomain"`
	Zone        types.String `tfsdk:"zone"`

	RegisteredHosts types.List   `tfsdk:"registered_hosts"`
	Runtime         types.String `tfsdk:"runtime"`
}

type DNSRegistrationAPIModel struct {
	ID      string     `json:"id"`
	Created *time.Time `json:"created,omitempty"`
	Updated *time.Time `json:"updated,omitempty"`

	Environment string `json:"environment"`
	Subdomain   string `json:"subdomain"`
	Zone        string `json:"zone"`

	// read-only, computed fields
	RegisteredHosts []string `json:"registeredHosts,omitempty"`
	Runtime         string   `json:"runtime,omitempty"`
}

func DNSRegistrationResourceFactory() resource.Resource {
	return &dnsRegistrationResource{}
}

type dnsRegistrationResource struct {
	commonResource
}

func (r *dnsRegistrationResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_dnsregistration"
}

func (r *dnsRegistrationResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "DNS registration resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the DNS registration",
				Computed:    true,
			},
			"environment": schema.StringAttribute{
				Description: "The environment of the DNS registration",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"subdomain": schema.StringAttribute{
				Description: "The subdomain of the DNS registration",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"zone": schema.StringAttribute{
				Description: "The zone of the DNS registration",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"registered_hosts": schema.ListAttribute{
				Description: "The registered hosts of the DNS registration",
				Computed:    true,
				ElementType: types.StringType,
			},
			"runtime": schema.StringAttribute{
				Description: "The runtime of the DNS registration",
				Computed:    true,
			},
		},
	}
}

func (data *DNSRegistrationResourceModel) toAPIModel(_ context.Context, api *DNSRegistrationAPIModel, diagnostics *diag.Diagnostics) {
	api.Subdomain = data.Subdomain.ValueString()
	api.Zone = data.Zone.ValueString()
	api.Environment = data.Environment.ValueString()
}

func (api *DNSRegistrationAPIModel) toData(ctx context.Context, data *DNSRegistrationResourceModel, diagnostics *diag.Diagnostics) {
	data.ID = types.StringValue(api.ID)
	data.Subdomain = types.StringValue(api.Subdomain)
	data.Zone = types.StringValue(api.Zone)
	data.Environment = types.StringValue(api.Environment)
	data.Runtime = types.StringValue(api.Runtime)

	if len(api.RegisteredHosts) > 0 {
		registeredHosts, diag := types.ListValueFrom(ctx, types.StringType, api.RegisteredHosts)
		if diag.HasError() {
			diagnostics.Append(diag...)
		}
		data.RegisteredHosts = registeredHosts
	}

	if api.Runtime != "" {
		data.Runtime = types.StringValue(api.Runtime)
	}
}

func (r *dnsRegistrationResource) apiPath(data *DNSRegistrationResourceModel) string {
	path := fmt.Sprintf("/api/v1/environments/%s/dnsregistrations", data.Environment.ValueString())

	if data.ID.ValueString() != "" {
		path = fmt.Sprintf("%s/%s", path, data.ID.ValueString())
	}

	return path
}

func (r *dnsRegistrationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DNSRegistrationResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var apiData DNSRegistrationAPIModel
	data.toAPIModel(ctx, &apiData, &resp.Diagnostics)

	ok, _ := r.apiRequest(ctx, http.MethodPost, r.apiPath(&data), apiData, &apiData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiData.toData(ctx, &data, &resp.Diagnostics)

	// TODO need to re-read data ?

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *dnsRegistrationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DNSRegistrationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data), nil, nil, &resp.Diagnostics, Allow404())

	r.waitForRemoval(ctx, r.apiPath(&data), &resp.Diagnostics)
}

func (r *dnsRegistrationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DNSRegistrationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	var api DNSRegistrationAPIModel
	api.ID = data.ID.ValueString()
	ok, status := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics, Allow404())
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

func (r *dnsRegistrationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data DNSRegistrationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)

	// immutable otherwise, so nothing to do
}

func (r *dnsRegistrationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// TODO
}
