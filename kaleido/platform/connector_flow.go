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
	"regexp"
	"sort"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ConnectorFlowResourceModel struct {
	ID                    types.String `tfsdk:"id"`
	Environment           types.String `tfsdk:"environment"`
	Service               types.String `tfsdk:"service"`
	Name                  types.String `tfsdk:"name"`
	Description           types.String `tfsdk:"description"`
	ConfigTypeBindings    types.Map    `tfsdk:"config_type_bindings"`
	ConfigProfiles        types.Map    `tfsdk:"config_profiles"`
	ConfigProfileBindings types.Map    `tfsdk:"config_profile_bindings"`
	Version               types.String `tfsdk:"version"`
	FlowType              types.String `tfsdk:"flow_type"`
	CurrentVersion        types.String `tfsdk:"current_version"`
}

type ConfigProfileDynamicMappingInput struct {
	JSONata string `json:"jsonata"`
}

type ConfigProfileBindingTargetInput struct {
	ConfigProfile   string                            `json:"configProfile,omitempty"`
	ConfigProfileID string                            `json:"configProfileId,omitempty"`
	DynamicMapping  *ConfigProfileDynamicMappingInput `json:"dynamicMapping,omitempty"`
}

// A nil entry is sent as JSON null, which the connector service treats as "unbind this optional
// config type". See unbindRemovedOptionalTypes.
type ConnectorFlowDeployAPIModel struct {
	Name               string                                      `json:"name,omitempty"`
	Description        string                                      `json:"description,omitempty"`
	Version            string                                      `json:"version,omitempty"`
	ConfigTypeBindings map[string]*ConfigProfileBindingTargetInput `json:"configTypeBindings,omitempty"`
}

type ConnectorFlowUpgradeAPIModel struct {
	Version            string                                      `json:"version,omitempty"`
	ConfigTypeBindings map[string]*ConfigProfileBindingTargetInput `json:"configTypeBindings,omitempty"`
}

type ConfigProfileDynamicMappingAPIModel struct {
	NamePrefix string `json:"namePrefix,omitempty"`
	JSONata    string `json:"jsonata,omitempty"`
}

type ConfigProfileBindingTargetAPIModel struct {
	ConfigProfileID string                               `json:"configProfileId,omitempty"`
	DynamicMapping  *ConfigProfileDynamicMappingAPIModel `json:"dynamicMapping,omitempty"`
}

type ConnectorFlowAPIModel struct {
	ID                    string                                        `json:"id,omitempty"`
	Name                  string                                        `json:"name,omitempty"`
	FlowType              string                                        `json:"flowType,omitempty"`
	CurrentVersion        string                                        `json:"currentVersion,omitempty"`
	Description           string                                        `json:"description,omitempty"`
	Labels                map[string]string                             `json:"labels,omitempty"`
	ConfigProfileBindings map[string]ConfigProfileBindingTargetAPIModel `json:"configProfileBindings,omitempty"`
}

// ConnectorFlowTemplateAPIModel is the part of GET /metadata/connector-flows/{name}[/versions/{v}]
// this resource needs: which binding names each config type fans out to, and which are optional.
type ConnectorFlowTemplateAPIModel struct {
	Version                string            `json:"version,omitempty"`
	RequiredConfigProfiles map[string]string `json:"requiredConfigProfiles,omitempty"` // binding name -> config type
	OptionalConfigProfiles map[string]string `json:"optionalConfigProfiles,omitempty"` // binding name -> config type
}

// The label the connector service stamps with the template version a flow was deployed or last
// upgraded from. It is what /upgrade's forward-only check compares against, so it is the
// authoritative source for `version` - unlike currentVersion, which an out-of-band PATCH can
// roll back to an older workflow version without changing the template the flow tracks.
const connectorReferenceVersionLabel = "connector_reference_version"

// configProfileIDPattern matches a workflow engine config profile ID: the "fcp" KID prefix, then
// ten alphanumerics, in its regular (fcp:) or Kubernetes-safe (fcp-) form. The engine documents the
// KID format as a stable integration contract (kap-utils/pkg/kid).
var configProfileIDPattern = regexp.MustCompile(`^fcp[-:][A-Za-z0-9]{10}$`)

var configProfileBindingAttrTypes = map[string]attr.Type{
	"config_profile_id": types.StringType,
	"jsonata":           types.StringType,
}

func ConnectorFlowResourceFactory() resource.Resource {
	return &connectorFlowResource{}
}

type connectorFlowResource struct {
	commonResource
}

var _ resource.ResourceWithModifyPlan = &connectorFlowResource{}
var _ resource.ResourceWithValidateConfig = &connectorFlowResource{}

func (r *connectorFlowResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_connector_flow"
}

func (r *connectorFlowResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Deploys a connector flow (workflow template) from a connector service's stored template versions, binding it to config profiles. " +
			"The resource owns all of the deployed flow's config profile bindings: a binding or dynamic mapping set outside this resource on a config type it binds is overwritten on apply.",
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"environment": &schema.StringAttribute{
				Required:      true,
				Description:   "Environment ID",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"service": &schema.StringAttribute{
				Required:      true,
				Description:   "Connector service ID",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": &schema.StringAttribute{
				Required:      true,
				Description:   "Name of the connector flow template (e.g. submission, query)",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"description": &schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "Description of the deployed flow. If omitted, the flow takes its template's description. " +
					"Once set it is kept in step with the connector, like the bindings; removing it from configuration leaves the current description in place.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"config_type_bindings": &schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Deprecated: use config_profiles. A fixed config profile per config type, keyed by config type name (e.g. evm.confirmations); " +
					"the value is the name or ID of an existing config profile of that type, which every slot of that type in the flow uses. " +
					"A config type may appear here or in config_profiles, not both.",
				DeprecationMessage: "Use config_profiles instead: \"<type>\" = \"<profile>\" becomes \"<type>\" = { profile_id = <profile resource>.id }. " +
					"config_profiles binds by ID, so the flow follows a profile that is replaced, and also supports selecting a profile per transaction.",
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
			},
			"config_profiles": &schema.MapNestedAttribute{
				Optional: true,
				Description: "Which config profile each config type in the flow uses. Keyed by config type name (e.g. evm.gasPricing); " +
					"every slot of that type in the flow uses the profile chosen here. Each entry sets `profile_id`, `jsonata`, or both: " +
					"`profile_id` is a fixed profile, and `jsonata` selects one by name per transaction, falling back to `profile_id` when it yields nothing. " +
					"A config type may appear here or in config_type_bindings, not both. Removing `jsonata` from an entry leaves its fixed profile in place; " +
					"removing an optional config type's entry entirely unbinds it.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"profile_id": &schema.StringAttribute{
							Optional: true,
							Description: "ID of an existing config profile of this config type - reference the profile resource's `id` " +
								"(e.g. kaleido_platform_connector_config_profile.default.id). A name is rejected: the connector resolves the profile once, " +
								"when the flow is deployed or upgraded, so binding by name would leave the flow pointing at a deleted profile if one were " +
								"ever replaced under the same name. With `jsonata`, it is the fallback used when the expression yields no profile.",
						},
						"jsonata": &schema.StringAttribute{
							Optional: true,
							Description: "JSONata expression evaluated against each transaction's state; its result is the name of the config profile to use. " +
								"The named profile need not be bound - it only has to exist on the service and be of this config type. " +
								"Only config types bound to a flow stage can use this; those bound to the monitor (for EVM: confirmations, nonceAssignment, prioritization) cannot. " +
								"Without `profile_id`, a transaction for which the expression yields nothing fails.",
						},
					},
				},
			},
			"config_profile_bindings": &schema.MapNestedAttribute{
				Computed: true,
				Description: "The deployed flow's config profile bindings as the connector service reports them, keyed by binding name " +
					"(a config type can fan out to several binding names).",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"config_profile_id": &schema.StringAttribute{Computed: true},
						"jsonata":           &schema.StringAttribute{Computed: true},
					},
				},
			},
			"version": &schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "Template version to deploy or upgrade to. It must be a version the connector service stores and supports - see the " +
					"kaleido_platform_connector_template_versions data source. Raising it upgrades the flow; lowering it is refused at plan time. " +
					"If omitted, the flow is deployed at the latest version and then held there: it is only upgraded when this is set.",
				PlanModifiers: []planmodifier.String{
					// Without this an omitted version plans as unknown on every update, so a
					// binding-only change would be sent with no version and the connector
					// service would treat it as "upgrade to latest".
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"flow_type": &schema.StringAttribute{
				Computed:    true,
				Description: "The flow type as reported by the deployed workflow (e.g. submission, query).",
			},
			"current_version": &schema.StringAttribute{
				Computed:    true,
				Description: "The workflow version currently active on the deployed flow. Normally equal to version; differs only if the flow was rolled back outside Terraform.",
			},
		},
	}
}

func (r *connectorFlowResource) metadataPath(data *ConnectorFlowResourceModel, suffix string) string {
	return fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/metadata/connector-flows/%s%s",
		data.Environment.ValueString(), data.Service.ValueString(), data.Name.ValueString(), suffix)
}

func (r *connectorFlowResource) instancePath(data *ConnectorFlowResourceModel) string {
	return fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/connector-flows/%s",
		data.Environment.ValueString(), data.Service.ValueString(), data.Name.ValueString())
}

// boundConfigTypes returns every config type the model binds, statically or dynamically.
func boundConfigTypes(data *ConnectorFlowResourceModel) map[string]bool {
	bound := map[string]bool{}
	for _, m := range []types.Map{data.ConfigTypeBindings, data.ConfigProfiles} {
		if m.IsNull() || m.IsUnknown() {
			continue
		}
		for k := range m.Elements() {
			bound[k] = true
		}
	}
	return bound
}

// configProfileEntry is one config_profiles value.
type configProfileEntry struct {
	ProfileID types.String `tfsdk:"profile_id"`
	JSONata   types.String `tfsdk:"jsonata"`
}

// checkProfileID reports a known profile_id that is not a config profile ID. An unknown value - a
// reference to a profile not created yet - is checked again at apply, once it is known.
func checkProfileID(configType string, e configProfileEntry, diagnostics *diag.Diagnostics) {
	if e.ProfileID.IsNull() || e.ProfileID.IsUnknown() || configProfileIDPattern.MatchString(e.ProfileID.ValueString()) {
		return
	}
	diagnostics.AddAttributeError(path.Root("config_profiles").AtMapKey(configType).AtName("profile_id"), "Not a config profile ID",
		fmt.Sprintf("config_profiles[%q].profile_id is %q, which is not a config profile ID (fcp:…). "+
			"Reference the profile resource's id, e.g. kaleido_platform_connector_config_profile.<name>.id, so the flow follows the profile if it is ever replaced.",
			configType, e.ProfileID.ValueString()))
}

// addConfiguredTwiceError reports a config type set in both config_type_bindings and
// config_profiles. Shared by ValidateConfig (plan time) and collectBindings (apply time), so the
// two checks cannot drift apart.
func addConfiguredTwiceError(configType string, diagnostics *diag.Diagnostics) {
	diagnostics.AddAttributeError(path.Root("config_profiles").AtMapKey(configType), "Config type configured twice",
		fmt.Sprintf("%q appears in both config_type_bindings and config_profiles. Configure it in config_profiles only - config_type_bindings is deprecated.", configType))
}

// collectBindings builds the one entry per config type that /deploy and /upgrade take, from
// config_type_bindings (a fixed profile) and config_profiles (a fixed profile, a JSONata
// selection, or both). Two checks from ValidateConfig are repeated here, now every value is
// known: a profile_id that was unknown at plan time, and a config type in both maps - which
// ValidateConfig cannot see when config_profiles as a whole is unknown at plan time (e.g. built
// from a for expression over something not yet created). Without the second, the config_profiles
// entry would silently win.
func (r *connectorFlowResource) collectBindings(ctx context.Context, data *ConnectorFlowResourceModel, diagnostics *diag.Diagnostics) map[string]*ConfigProfileBindingTargetInput {
	bindings := map[string]*ConfigProfileBindingTargetInput{}
	if !data.ConfigTypeBindings.IsNull() && !data.ConfigTypeBindings.IsUnknown() {
		for k, v := range data.ConfigTypeBindings.Elements() {
			s, ok := v.(types.String)
			if !ok {
				diagnostics.AddError("Invalid binding", fmt.Sprintf("config_type_bindings[%s] is not a string", k))
				return nil
			}
			bindings[k] = &ConfigProfileBindingTargetInput{ConfigProfile: s.ValueString()}
		}
	}
	if !data.ConfigProfiles.IsNull() && !data.ConfigProfiles.IsUnknown() {
		var entries map[string]configProfileEntry
		diagnostics.Append(data.ConfigProfiles.ElementsAs(ctx, &entries, false)...)
		if diagnostics.HasError() {
			return nil
		}
		for k, e := range entries {
			if _, inShorthand := bindings[k]; inShorthand {
				addConfiguredTwiceError(k, diagnostics)
			}
			checkProfileID(k, e, diagnostics)
			if diagnostics.HasError() {
				return nil
			}
			// Sent as configProfile rather than the ID-only configProfileId: that way the connector
			// still looks the profile up, so a well-formed ID for a profile that does not exist is
			// refused at deploy/upgrade rather than bound.
			b := &ConfigProfileBindingTargetInput{}
			if !e.ProfileID.IsNull() {
				b.ConfigProfile = e.ProfileID.ValueString()
			}
			if !e.JSONata.IsNull() {
				b.DynamicMapping = &ConfigProfileDynamicMappingInput{JSONata: e.JSONata.ValueString()}
			}
			bindings[k] = b
		}
	}
	return bindings
}

// ValidateConfig catches mistakes the schema cannot express: an entry that chooses no profile at
// all, a profile_id that is not a config profile ID, and a config type configured in both maps -
// where there would be no honest answer to which one wins.
func (r *connectorFlowResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data ConnectorFlowResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() || data.ConfigProfiles.IsNull() || data.ConfigProfiles.IsUnknown() {
		return
	}
	var entries map[string]configProfileEntry
	resp.Diagnostics.Append(data.ConfigProfiles.ElementsAs(ctx, &entries, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	for k, e := range entries {
		if e.ProfileID.IsNull() && e.JSONata.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("config_profiles").AtMapKey(k), "No config profile chosen",
				fmt.Sprintf("config_profiles[%q] must set `profile_id`, `jsonata`, or both.", k))
		}
		checkProfileID(k, e, &resp.Diagnostics)
		if e.ProfileID.IsNull() && !e.JSONata.IsNull() {
			// Legal, and occasionally intended - but it is also exactly what a misspelt profile_id
			// produces: Terraform silently drops unknown attributes from a nested object literal,
			// so the provider cannot see the typo, only its result.
			resp.Diagnostics.AddAttributeWarning(path.Root("config_profiles").AtMapKey(k), "No fallback config profile",
				fmt.Sprintf("config_profiles[%q] sets jsonata but no profile_id, so a transaction for which the expression yields no profile name will fail. "+
					"If you meant to set a fallback, check the attribute is spelled profile_id - Terraform ignores misspelt attributes here.", k))
		}
	}
	if data.ConfigTypeBindings.IsNull() || data.ConfigTypeBindings.IsUnknown() {
		return
	}
	var both []string
	for k := range data.ConfigTypeBindings.Elements() {
		if _, ok := entries[k]; ok {
			both = append(both, k)
		}
	}
	sort.Strings(both)
	for _, k := range both {
		addConfiguredTwiceError(k, &resp.Diagnostics)
	}
}

// removedConfigTypes returns, sorted, each config type bound in prior state but bound by neither
// map in the plan.
func removedConfigTypes(plan, state *ConnectorFlowResourceModel) []string {
	planned := boundConfigTypes(plan)
	var removed []string
	for t := range boundConfigTypes(state) {
		if !planned[t] {
			removed = append(removed, t)
		}
	}
	sort.Strings(removed)
	return removed
}

// templateConfigTypes reads which config types a template version requires and which it declares
// as optional. An empty version reads the latest.
//
// Only a plan or apply that removes a config type calls this, and on a connector runtime that
// predates stored template versions the versioned route does not exist. Removing a config type is
// genuinely unsafe there - unbinding sends a null binding, which such a runtime cannot handle - so
// the 404 becomes a clear refusal naming the fix, rather than a raw API error.
func (r *connectorFlowResource) templateConfigTypes(ctx context.Context, data *ConnectorFlowResourceModel, templateVersion string, diagnostics *diag.Diagnostics) (required, optional map[string]bool, ok bool) {
	suffix := ""
	if templateVersion != "" {
		suffix = "/versions/" + templateVersion
	}
	var tmpl ConnectorFlowTemplateAPIModel
	ok, status := r.apiRequest(ctx, http.MethodGet, r.metadataPath(data, suffix), nil, &tmpl, diagnostics, Allow404())
	if !ok {
		return nil, nil, false
	}
	if status == http.StatusNotFound {
		diagnostics.AddAttributeError(path.Root("config_profiles"), "Connector runtime cannot remove config types",
			fmt.Sprintf("Connector flow %q is losing a config type, but its connector runtime predates stored template versions and cannot unbind one. "+
				"Upgrade the connector runtime first, or keep the config type configured.", data.Name.ValueString()))
		return nil, nil, false
	}
	required, optional = map[string]bool{}, map[string]bool{}
	for _, t := range tmpl.RequiredConfigProfiles {
		required[t] = true
	}
	for _, t := range tmpl.OptionalConfigProfiles {
		optional[t] = true
	}
	return required, optional, true
}

// unbindRemovedOptionalTypes adds a null entry for each optional config type that was bound in
// prior state but is bound by neither map in the plan. /upgrade deliberately leaves alone any
// binding it is not sent, so without the explicit null an optional binding removed from
// configuration would stay bound. A removed required type never reaches here - ModifyPlan refuses it.
func (r *connectorFlowResource) unbindRemovedOptionalTypes(ctx context.Context, plan, state *ConnectorFlowResourceModel, bindings map[string]*ConfigProfileBindingTargetInput, diagnostics *diag.Diagnostics) {
	removed := removedConfigTypes(plan, state)
	if len(removed) == 0 {
		return
	}
	targetVersion := ""
	if !plan.Version.IsNull() && !plan.Version.IsUnknown() {
		targetVersion = plan.Version.ValueString()
	}
	_, optional, ok := r.templateConfigTypes(ctx, plan, targetVersion, diagnostics)
	if !ok {
		return
	}
	for _, t := range removed {
		if optional[t] {
			bindings[t] = nil // sent as JSON null: unbind
		}
		// A type the target version no longer references at all is removed by the upgrade itself.
	}
}

func (r *connectorFlowResource) toData(api *ConnectorFlowAPIModel, data *ConnectorFlowResourceModel, diagnostics *diag.Diagnostics) {
	if api.ID != "" {
		data.ID = types.StringValue(api.ID)
	} else if data.ID.IsNull() || data.ID.IsUnknown() {
		data.ID = types.StringValue(api.Name)
	}
	data.FlowType = types.StringValue(api.FlowType)
	data.CurrentVersion = types.StringValue(api.CurrentVersion)
	data.Description = optionalString(api.Description)
	// `version` reflects the template version the service reports, so an out-of-band upgrade
	// surfaces as a plan diff against a pinned value.
	data.Version = types.StringValue(deployedTemplateVersion(api))

	bindings := map[string]attr.Value{}
	for name, b := range api.ConfigProfileBindings {
		jsonata := types.StringNull()
		if b.DynamicMapping != nil && b.DynamicMapping.JSONata != "" {
			jsonata = types.StringValue(b.DynamicMapping.JSONata)
		}
		profileID := types.StringNull()
		if b.ConfigProfileID != "" {
			profileID = types.StringValue(b.ConfigProfileID)
		}
		obj, d := types.ObjectValue(configProfileBindingAttrTypes, map[string]attr.Value{
			"config_profile_id": profileID,
			"jsonata":           jsonata,
		})
		diagnostics.Append(d...)
		bindings[name] = obj
	}
	m, d := types.MapValue(types.ObjectType{AttrTypes: configProfileBindingAttrTypes}, bindings)
	diagnostics.Append(d...)
	data.ConfigProfileBindings = m
}

func deployedTemplateVersion(api *ConnectorFlowAPIModel) string {
	if v := api.Labels[connectorReferenceVersionLabel]; v != "" {
		return v
	}
	return api.CurrentVersion
}

// checkVersionPin fails the apply when the service deployed a different template version than was
// requested. A version-aware connector service refuses an unknown or unsupported version outright,
// so reaching this means the service ignored the request - a connector runtime that predates
// deploying at a version.
func (r *connectorFlowResource) checkVersionPin(requested types.String, api *ConnectorFlowAPIModel, diagnostics *diag.Diagnostics) {
	deployed := deployedTemplateVersion(api)
	if requested.IsNull() || requested.IsUnknown() || requested.ValueString() == deployed {
		return
	}
	diagnostics.AddError(
		"Connector flow version mismatch",
		fmt.Sprintf("Requested version %q of connector flow %q, but the connector service deployed %q. "+
			"The connector runtime does not support deploying or upgrading to a specific version - upgrade the connector runtime, or set `version` to %q.",
			requested.ValueString(), api.Name, deployed, deployed),
	)
}

// ModifyPlan refuses a decrease in `version`. The connector service is forward-only, and would
// refuse it at apply, but failing here keeps the refusal in the plan where it belongs - and, unlike
// RequiresReplace, never destroys the flow and every subflow binding to it.
func (r *connectorFlowResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return // create or destroy
	}
	var plan, state ConnectorFlowResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.refuseLowerVersion(&plan, &state, &resp.Diagnostics)
	r.refuseRemovingRequiredTypes(ctx, &plan, &state, &resp.Diagnostics)
}

// refuseLowerVersion refuses a decrease in `version`. The connector service is forward-only, and
// would refuse it at apply, but failing here keeps the refusal in the plan where it belongs - and,
// unlike RequiresReplace, never destroys the flow and every subflow binding to it.
func (r *connectorFlowResource) refuseLowerVersion(plan, state *ConnectorFlowResourceModel, diagnostics *diag.Diagnostics) {
	if plan.Version.IsNull() || plan.Version.IsUnknown() || state.Version.IsNull() || state.Version.IsUnknown() {
		return
	}
	planned, err1 := version.NewVersion(plan.Version.ValueString())
	deployed, err2 := version.NewVersion(state.Version.ValueString())
	if err1 != nil || err2 != nil {
		return // not semver: let the connector service decide
	}
	if planned.LessThan(deployed) {
		diagnostics.AddAttributeError(path.Root("version"), "Connector flow version cannot be lowered",
			fmt.Sprintf("Connector flow is deployed at %q; %q is older. Connector flows only move forward. "+
				"To run an older template, roll back the workflow outside Terraform (PATCH /connector-flows/{name} with currentVersion), then set `version` to match.",
				state.Version.ValueString(), plan.Version.ValueString()))
	}
}

// refuseRemovingRequiredTypes refuses a plan that stops configuring a config type the flow
// requires. The flow cannot run without it, so it can never be unbound, and because the resource
// owns its bindings (see reconcileConfigProfiles) leaving it bound but unconfigured would show as a
// diff on every plan that no apply could resolve. The template is read only when the plan
// actually drops a config type, so an ordinary plan makes no extra call.
func (r *connectorFlowResource) refuseRemovingRequiredTypes(ctx context.Context, plan, state *ConnectorFlowResourceModel, diagnostics *diag.Diagnostics) {
	if plan.ConfigTypeBindings.IsUnknown() || plan.ConfigProfiles.IsUnknown() {
		return // cannot tell yet what is being removed
	}
	removed := removedConfigTypes(plan, state)
	if len(removed) == 0 {
		return
	}
	targetVersion := state.Version.ValueString()
	if !plan.Version.IsNull() && !plan.Version.IsUnknown() {
		targetVersion = plan.Version.ValueString()
	}
	required, _, ok := r.templateConfigTypes(ctx, plan, targetVersion, diagnostics)
	if !ok {
		return
	}
	for _, t := range removed {
		if required[t] {
			diagnostics.AddAttributeError(path.Root("config_profiles"), "Required config type cannot be removed",
				fmt.Sprintf("%q is required by version %q of connector flow %q, so it must stay configured in config_profiles.", t, targetVersion, plan.Name.ValueString()))
		}
	}
}

func (r *connectorFlowResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ConnectorFlowResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	bindings := r.collectBindings(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	body := ConnectorFlowDeployAPIModel{
		Name:               data.Name.ValueString(),
		ConfigTypeBindings: bindings,
	}
	if !data.Description.IsNull() {
		body.Description = data.Description.ValueString()
	}
	if !data.Version.IsNull() && !data.Version.IsUnknown() {
		body.Version = data.Version.ValueString()
	}
	var api ConnectorFlowAPIModel
	ok, _ := r.apiRequest(ctx, http.MethodPost, r.metadataPath(&data, "/deploy"), &body, &api, &resp.Diagnostics)
	if !ok {
		return
	}
	requestedVersion := data.Version
	r.toData(&api, &data, &resp.Diagnostics)
	// Set state even on a version mismatch so the deployed flow is tracked (and
	// re-created on the next apply) rather than orphaned.
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	r.checkVersionPin(requestedVersion, &api, &resp.Diagnostics)
}

func (r *connectorFlowResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ConnectorFlowResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var api ConnectorFlowAPIModel
	ok, status := r.apiRequest(ctx, http.MethodGet, r.instancePath(&data), nil, &api, &resp.Diagnostics, Allow404())
	if !ok {
		return
	}
	if status == 404 {
		resp.State.RemoveResource(ctx)
		return
	}
	r.toData(&api, &data, &resp.Diagnostics)
	r.reconcileConfigProfiles(ctx, &data, &api, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *connectorFlowResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state ConnectorFlowResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)
	if resp.Diagnostics.HasError() {
		return
	}
	bindings := r.collectBindings(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.unbindRemovedOptionalTypes(ctx, &data, &state, bindings, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	// One /upgrade carries both the version move (if any) and the full binding set: forward to a
	// newer version, or - at the same version - a pure binding reconcile. The version is always
	// sent when known, including when it is unchanged, so a binding-only change never becomes an
	// implicit upgrade to latest.
	body := ConnectorFlowUpgradeAPIModel{ConfigTypeBindings: bindings}
	if !data.Version.IsNull() && !data.Version.IsUnknown() {
		body.Version = data.Version.ValueString()
	}
	var api ConnectorFlowAPIModel
	ok, _ := r.apiRequest(ctx, http.MethodPost, r.metadataPath(&data, "/upgrade"), &body, &api, &resp.Diagnostics)
	if !ok {
		return
	}
	// /upgrade carries no description, so a change to it goes to the workflow itself as a sparse
	// update - otherwise it would be recorded in state and never reach the connector.
	if !data.Description.IsNull() && !data.Description.IsUnknown() && !data.Description.Equal(state.Description) {
		patch := map[string]string{"description": data.Description.ValueString()}
		if ok, _ := r.apiRequest(ctx, http.MethodPatch, r.instancePath(&data), &patch, &api, &resp.Diagnostics); !ok {
			return
		}
	}
	requestedVersion := data.Version
	r.toData(&api, &data, &resp.Diagnostics)
	// Record the actual deployed version even when it doesn't match the pin.
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	r.checkVersionPin(requestedVersion, &api, &resp.Diagnostics)
}

// observedBinding is the part of one slot's binding config_profiles can express. An empty field
// means absent.
type observedBinding struct {
	profileID string
	jsonata   string
}

func observedFromTarget(b ConfigProfileBindingTargetAPIModel) observedBinding {
	o := observedBinding{profileID: b.ConfigProfileID}
	if b.DynamicMapping != nil {
		o.jsonata = b.DynamicMapping.JSONata
	}
	return o
}

// observedForType collapses the slots of one config type into the single value config_profiles
// holds for that type. Every shipped flow has one slot per config type (AD-3), so normally there is
// nothing to collapse. If the slots disagree - possible only through the engine's per-slot API - it
// returns a slot that differs from what is expected, so the plan shows drift against the type's
// entry and the next apply converges every slot of the type. slots must be non-empty and in a
// stable order.
func observedForType(slots []observedBinding, expected *observedBinding) observedBinding {
	if expected != nil {
		for _, s := range slots {
			if s != *expected {
				return s
			}
		}
		return *expected
	}
	return slots[0]
}

// reconcileConfigProfiles rebuilds config_profiles from the bindings the connector service reports,
// so a binding changed outside Terraform - a dynamic mapping added in the UI, a profile re-pointed,
// an optional type bound or unbound - shows as a plan diff and is reverted by the next apply. The
// flow resource owns its bindings; without this it would only overwrite such a change when
// something else happened to trigger an update.
//
// The service reports bindings per slot, so this reads the deployed template to map each slot back
// to its config type. Config types written in the deprecated config_type_bindings are left alone:
// that map holds profile names and the service stores IDs, so there is nothing to compare.
//
// It never fails a refresh. A connector runtime that predates stored template versions has no
// route to read, and is simply not reconciled.
func (r *connectorFlowResource) reconcileConfigProfiles(ctx context.Context, data *ConnectorFlowResourceModel, api *ConnectorFlowAPIModel, diagnostics *diag.Diagnostics) {
	if data.ConfigProfiles.IsUnknown() {
		return
	}
	deployed := deployedTemplateVersion(api)
	if deployed == "" {
		return
	}
	var tmpl ConnectorFlowTemplateAPIModel
	var lookup diag.Diagnostics
	ok, status := r.apiRequest(ctx, http.MethodGet, r.metadataPath(data, "/versions/"+deployed), nil, &tmpl, &lookup, Allow404())
	if !ok || status == 404 {
		if lookup.HasError() {
			diagnostics.AddWarning("Connector flow bindings not checked for drift",
				fmt.Sprintf("Could not read template version %q of connector flow %q, so changes made to its config profile bindings outside Terraform will not be detected on this refresh.", deployed, data.Name.ValueString()))
		}
		return
	}
	slotType := map[string]string{}
	for name, t := range tmpl.RequiredConfigProfiles {
		slotType[name] = t
	}
	for name, t := range tmpl.OptionalConfigProfiles {
		slotType[name] = t
	}

	shorthand := map[string]bool{}
	if !data.ConfigTypeBindings.IsNull() && !data.ConfigTypeBindings.IsUnknown() {
		for t := range data.ConfigTypeBindings.Elements() {
			shorthand[t] = true
		}
	}
	prior := map[string]configProfileEntry{}
	if !data.ConfigProfiles.IsNull() {
		diagnostics.Append(data.ConfigProfiles.ElementsAs(ctx, &prior, false)...)
		if diagnostics.HasError() {
			return
		}
	}

	slotNames := make([]string, 0, len(api.ConfigProfileBindings))
	for name := range api.ConfigProfileBindings {
		slotNames = append(slotNames, name)
	}
	sort.Strings(slotNames)
	byType := map[string][]observedBinding{}
	for _, name := range slotNames {
		t := slotType[name]
		if t == "" || shorthand[t] {
			continue // a slot the deployed template no longer declares, or one the shorthand owns
		}
		byType[t] = append(byType[t], observedFromTarget(api.ConfigProfileBindings[name]))
	}

	if data.ConfigProfiles.IsNull() && len(byType) == 0 {
		return // nothing managed here and nothing unexpected bound: keep null rather than {}
	}
	// A type in prior state with no slot on the service has been unbound outside Terraform; leaving
	// it out makes the plan show it being bound again.
	entries := map[string]attr.Value{}
	for t, slots := range byType {
		var expected *observedBinding
		if e, ok := prior[t]; ok {
			expected = &observedBinding{profileID: e.ProfileID.ValueString(), jsonata: e.JSONata.ValueString()}
		}
		o := observedForType(slots, expected)
		obj, d := types.ObjectValue(configProfileEntryAttrTypes, map[string]attr.Value{
			"profile_id": optionalString(o.profileID),
			"jsonata":    optionalString(o.jsonata),
		})
		diagnostics.Append(d...)
		entries[t] = obj
	}
	m, d := types.MapValue(types.ObjectType{AttrTypes: configProfileEntryAttrTypes}, entries)
	diagnostics.Append(d...)
	data.ConfigProfiles = m
}

var configProfileEntryAttrTypes = map[string]attr.Type{
	"profile_id": types.StringType,
	"jsonata":    types.StringType,
}

func optionalString(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}

func (r *connectorFlowResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ConnectorFlowResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.apiRequest(ctx, http.MethodDelete, r.instancePath(&data), nil, nil, &resp.Diagnostics, Allow404())
}
