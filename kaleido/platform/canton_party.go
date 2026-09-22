package platform

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kaleido-io/terraform-provider-kaleido/kaleido/planmodifiers"
)

const (
	PartyTypeLocal    = "local"
	PartyTypeExternal = "external"
)

type CantonPartyResourceModel struct {
	Name          types.String `tfsdk:"name"`
	Party         types.String `tfsdk:"party"`
	Identifier    types.String `tfsdk:"identifier"`
	PartyType     types.String `tfsdk:"type"`
	Environment   types.String `tfsdk:"environment"`
	Service       types.String `tfsdk:"service"`
	Synchronizer  types.String `tfsdk:"synchronizer"`
	Namespace     types.String `tfsdk:"namespace"`
	SigningKeys   types.List   `tfsdk:"signing_keys"`
	Synchronizers types.List   `tfsdk:"synchronizers"`
}

type CantonPartyAPIModel struct {
	Name          string   `json:"name"`
	Party         string   `json:"party"`
	Identifier    string   `json:"identifier"`
	PartyType     string   `json:"type"`
	Synchronizer  string   `json:"synchronizer"`
	Keys          []Keys   `json:"keys,omitempty"`
	Synchronizers []string `json:"synchronizers,omitempty"`
}

type Keys struct {
	ID    string `json:"id"`
	Usage string `json:"usage"`
}

func CantonPartyResourceFactory() resource.Resource {
	return &cantonPartyResource{}
}

type cantonPartyResource struct {
	commonResource
}

func (r *cantonPartyResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_canton_party"
}

func (r *cantonPartyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	const typeName = "kaleido_platform_canton_party"
	resp.Schema = schema.Schema{
		Description: "A reference to a Canton party",
		Attributes: map[string]schema.Attribute{
			"name": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{planmodifiers.RequireRecreate(typeName)},
				Description:   "Human-readable name of the Canton party",
			},
			"party": &schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Description:   "Canton party ID. Immutable after create — changing this value is not supported; create a new, separate party instead.",
			},
			"identifier": &schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Description:   "Canton party identifier. Immutable after create — changing this value is not supported; create a new, separate party instead.",
			},
			"environment": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{planmodifiers.RequireRecreate(typeName)},
				Description:   "Environment ID. Immutable after create — changing this value is not supported; create a new, separate party instead.",
			},
			"service": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{planmodifiers.RequireRecreate(typeName)},
				Description:   "Canton Participant node Service ID for the party. Immutable after create — changing this value is not supported; create a new, separate party instead.",
			},
			"synchronizer": &schema.StringAttribute{
				Required:    true,
				Description: "Synchronizer ID. Immutable after create — changing this value is not supported; create a new, separate party instead.",
			},
			"namespace": &schema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{planmodifiers.RequireRecreate(typeName)},
				Description:   "Namespace key canton fingerprint for the party. Namespace key can be used for topology and protocol transactions. Immutable after create — changing this value is not supported; create a new, separate party instead.",
			},
			"type": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{planmodifiers.RequireRecreate(typeName)},
				Description:   "Canton party type Can be either 'local' or 'external'. Immutable after create — changing this value is not supported; create a new, separate party instead.",
			},
			"signing_keys": &schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Optional Canton fingerprints for additional signing keys for the party. These keys can be used for protocol transactions.",
			},
			"synchronizers": &schema.ListAttribute{
				Computed:      true,
				ElementType:   types.StringType,
				PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown()},
				Description:   "Synchronizers for the party. Immutable after create — changing this value is not supported; create a new, separate party instead.",
			},
		},
	}
}

func (data *CantonPartyResourceModel) toAPI(ctx context.Context, api *CantonPartyAPIModel, diagnostics *diag.Diagnostics) {
	api.Name = data.Name.ValueString()
	api.Synchronizer = data.Synchronizer.ValueString()

	if data.PartyType.ValueString() == PartyTypeLocal && (!data.Namespace.IsNull() || !data.SigningKeys.IsNull()) {
		diagnostics.AddError(
			"Invalid Canton party type",
			"Namespace and signing keys are only supported for external parties. Set the party type to 'external' to use them.",
		)
		return
	}

	if data.PartyType.ValueString() == PartyTypeExternal && data.Namespace.IsNull() {
		diagnostics.AddError(
			"Invalid Canton party type",
			"Namespace key is required for external parties. Set the party type to 'local' to use them.",
		)
		return
	}

	api.Keys = make([]Keys, 0)

	if !data.Namespace.IsNull() {
		api.Keys = append(api.Keys, Keys{
			ID:    data.Namespace.ValueString(),
			Usage: "namespace",
		})
	}

	if !data.SigningKeys.IsNull() {
		var signingKeys []string
		diagnostics.Append(data.SigningKeys.ElementsAs(ctx, &signingKeys, false)...)
		for _, key := range signingKeys {
			api.Keys = append(api.Keys, Keys{
				ID:    key,
				Usage: "protocol",
			})
		}
	}
}

func (api *CantonPartyAPIModel) toData(ctx context.Context, data *CantonPartyResourceModel, diagnostics *diag.Diagnostics) {
	data.Name = types.StringValue(api.Name)
	data.Party = types.StringValue(api.Party)
	data.Identifier = types.StringValue(api.Identifier)

	synchronizers := make([]types.String, 0)
	if api.Synchronizers != nil {
		for _, synchronizer := range api.Synchronizers {
			synchronizers = append(synchronizers, types.StringValue(synchronizer))
		}
	}

	// Will alway contain at least the synchronizer we created the party with
	syncList, d := types.ListValueFrom(ctx, types.StringType, synchronizers)
	diagnostics.Append(d...)
	data.Synchronizers = syncList

	signingKeys := make([]types.String, 0)
	if api.Keys != nil {
		for _, key := range api.Keys {
			switch key.Usage {
			case "namespace":
				data.Namespace = types.StringValue(key.ID)
			case "protocol":
				signingKeys = append(signingKeys, types.StringValue(key.ID))
			}
		}
	}

	if len(signingKeys) > 0 {
		tfList, d := types.ListValueFrom(ctx, types.StringType, signingKeys)
		diagnostics.Append(d...)
		data.SigningKeys = tfList
	} else {
		data.SigningKeys = types.ListNull(types.StringType)
	}
}

func (r *cantonPartyResource) apiPath(data *CantonPartyResourceModel) string {
	p := fmt.Sprintf("/endpoint/%s/%s/node/parties", data.Environment.ValueString(), data.Service.ValueString())
	if data.Party.ValueString() != "" {
		p = p + "/" + data.Party.ValueString()
	}
	return p
}

func (r *cantonPartyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data CantonPartyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	synchronizer := data.Synchronizer

	api := CantonPartyAPIModel{}
	data.toAPI(ctx, &api, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	apiPath := r.apiPath(&data)
	ok, _ := r.apiRequest(ctx, http.MethodPut, apiPath, api, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	api.toData(ctx, &data, &resp.Diagnostics)
	if !synchronizer.IsNull() {
		data.Synchronizer = synchronizer
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *cantonPartyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CantonPartyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	var api CantonPartyAPIModel
	apiPath := r.apiPath(&data)
	if resp.Diagnostics.HasError() {
		return
	}
	ok, status := r.apiRequest(ctx, http.MethodGet, apiPath, nil, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	// if party is not found, I need 404
	if status == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	api.toData(ctx, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *cantonPartyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data CantonPartyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("name"), &data.Name)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("party"), &data.Party)...)

	synchronizer := data.Synchronizer

	api := CantonPartyAPIModel{}
	data.toAPI(ctx, &api, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	apiPath := r.apiPath(&data)
	ok, _ := r.apiRequest(ctx, http.MethodPatch, apiPath, api, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	api.toData(ctx, &data, &resp.Diagnostics)
	if !synchronizer.IsNull() {
		data.Synchronizer = synchronizer
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Party deletion is not supported in API so just remove it from state
func (r *cantonPartyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CantonPartyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	resp.State.RemoveResource(ctx)
}
