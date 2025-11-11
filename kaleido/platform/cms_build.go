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
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kaleido-io/terraform-provider-kaleido/kaleido/kaleidobase"
)

type CMSBuildResourceModel struct {
	ID                      types.String                     `tfsdk:"id"`
	Environment             types.String                     `tfsdk:"environment"`
	Service                 types.String                     `tfsdk:"service"`
	Name                    types.String                     `tfsdk:"name"`
	Type                    types.String                     `tfsdk:"type"`
	Path                    types.String                     `tfsdk:"path"`
	Description             types.String                     `tfsdk:"description"`
	EVMVersion              types.String                     `tfsdk:"evm_version"`
	SolcVersion             types.String                     `tfsdk:"solc_version"`
	Precompiled             CMSBuildPrecompiledResourceModel `tfsdk:"precompiled"`
	GitHub                  CMSBuildGithubResourceModel      `tfsdk:"github"`
	Optimizer               *CMSBuildOptimizerResourceModel  `tfsdk:"optimizer"`
	SourceCode              CMSBuildSourceCodeResourceModel  `tfsdk:"source_code"`
	ABI                     types.String                     `tfsdk:"abi"`
	Bytecode                types.String                     `tfsdk:"bytecode"`
	DevDocs                 types.String                     `tfsdk:"dev_docs"`
	LibrariesJSON           types.String                     `tfsdk:"libraries_json"`
	CommitHash              types.String                     `tfsdk:"commit_hash"`
	CompilationMetadataJSON types.String                     `tfsdk:"compilation_metadata_json"`
	IgnoreDestroy           types.Bool                       `tfsdk:"ignore_destroy"`
}

type CMSBuildPrecompiledResourceModel struct {
	ABI      types.String `tfsdk:"abi"`
	Bytecode types.String `tfsdk:"bytecode"`
	DevDocs  types.String `tfsdk:"dev_docs"`
}

type CMSBuildGithubResourceModel struct {
	ContractURL  types.String `tfsdk:"contract_url"`
	ContractName types.String `tfsdk:"contract_name"`
	AuthToken    types.String `tfsdk:"auth_token"`
}

type CMSBuildOptimizerResourceModel struct {
	Enabled types.Bool  `tfsdk:"enabled"`
	Runs    types.Int64 `tfsdk:"runs"`
	ViaIR   types.Bool  `tfsdk:"via_ir"`
}

type CMSBuildSourceCodeResourceModel struct {
	ContractName types.String `tfsdk:"contract_name"`
	FileContents types.String `tfsdk:"file_contents"`
}

type CMSBuildAPIModel struct {
	ID                  string                      `json:"id,omitempty"`
	Created             *time.Time                  `json:"created,omitempty"`
	Updated             *time.Time                  `json:"updated,omitempty"`
	Name                string                      `json:"name"`
	Path                string                      `json:"path"`
	Description         string                      `json:"description,omitempty"`
	EVMVersion          string                      `json:"evmVersion,omitempty"`
	SolcVersion         string                      `json:"solcVersion,omitempty"`
	GitHub              *CMSBuildGithubAPIModel     `json:"github,omitempty"`
	Optimizer           *CMSBuildOptimizerAPIModel  `json:"optimizer,omitempty"`
	SourceCode          *CMSBuildSourceCodeAPIModel `json:"sourceCode,omitempty"`
	ABI                 interface{}                 `json:"abi,omitempty"`
	Bytecode            string                      `json:"bytecode,omitempty"`
	DevDocs             interface{}                 `json:"devDocs,omitempty"`
	CompileError        string                      `json:"compileError,omitempty"`
	Status              string                      `json:"status,omitempty"`
	Libraries           map[string]interface{}      `json:"libraries,omitempty"`
	CompilationMetadata map[string]interface{}      `json:"compilationMetadata,omitempty"` // json string of raw compiler metadata output
}

type CMSBuildGithubAPIModel struct {
	ContractURL  string `json:"contractUrl,omitempty"`
	ContractName string `json:"contractName,omitempty"`
	AuthToken    string `json:"oauthToken,omitempty"`
	CommitHash   string `json:"commitHash,omitempty"`
}

type CMSBuildOptimizerAPIModel struct {
	Enabled *bool   `json:"enabled,omitempty"`
	Runs    float64 `json:"runs,omitempty"`
	ViaIR   *bool   `json:"viaIR,omitempty"`
}

type CMSBuildSourceCodeAPIModel struct {
	ContractName string `json:"contractName,omitempty"`
	FileContents string `json:"fileContents,omitempty"`
}

func CMSBuildResourceFactory() resource.Resource {
	return &cms_buildResource{}
}

type cms_buildResource struct {
	commonResource
}

func (r *cms_buildResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_cms_build"
}

func (r *cms_buildResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"environment": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"service": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"type": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators: []validator.String{
					stringvalidator.OneOf(
						"github",
						"source_code",
						"precompiled",
					),
				},
			},
			"path": &schema.StringAttribute{
				Required: true,
			},
			"name": &schema.StringAttribute{
				Required: true,
			},
			"description": &schema.StringAttribute{
				Optional: true,
			},
			"evm_version": &schema.StringAttribute{
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Optional:      true,
			},
			"solc_version": &schema.StringAttribute{
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Optional:      true,
			},
			"github": &schema.SingleNestedAttribute{
				Optional: true,
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"contract_url": &schema.StringAttribute{
						Required:      true,
						PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
					},
					"contract_name": &schema.StringAttribute{
						Optional:      true,
						PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
					},
					"auth_token": &schema.StringAttribute{
						Optional:  true,
						Sensitive: true,
					},
				},
				Default: objectdefault.StaticValue(
					types.ObjectValueMust(
						map[string]attr.Type{
							"contract_url":  types.StringType,
							"contract_name": types.StringType,
							"auth_token":    types.StringType,
						},
						map[string]attr.Value{
							"contract_url":  types.StringValue(""),
							"contract_name": types.StringValue(""),
							"auth_token":    types.StringValue(""),
						},
					),
				),
			},
			"source_code": &schema.SingleNestedAttribute{
				Optional: true,
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"contract_name": &schema.StringAttribute{
						Required:      true,
						PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
					},
					"file_contents": &schema.StringAttribute{
						Optional:      true,
						PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
					},
				},
				Default: objectdefault.StaticValue(
					types.ObjectValueMust(
						map[string]attr.Type{
							"contract_name": types.StringType,
							"file_contents": types.StringType,
						},
						map[string]attr.Value{
							"contract_name": types.StringValue(""),
							"file_contents": types.StringValue(""),
						},
					),
				),
			},
			"precompiled": &schema.SingleNestedAttribute{
				Optional: true,
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"abi": &schema.StringAttribute{
						Required:      true,
						PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
					},
					"bytecode": &schema.StringAttribute{
						Required:      true,
						PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
					},
					"dev_docs": &schema.StringAttribute{
						Optional:      true,
						PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
					},
				},
				Default: objectdefault.StaticValue(
					types.ObjectValueMust(
						map[string]attr.Type{
							"abi":      types.StringType,
							"bytecode": types.StringType,
							"dev_docs": types.StringType,
						},
						map[string]attr.Value{
							"abi":      types.StringValue(""),
							"bytecode": types.StringValue(""),
							"dev_docs": types.StringValue(""),
						},
					),
				),
			},
			"libraries_json": &schema.StringAttribute{
				Optional: true,
			},
			"abi": &schema.StringAttribute{
				Computed: true,
			},
			"bytecode": &schema.StringAttribute{
				Computed: true,
			},
			"dev_docs": &schema.StringAttribute{
				Computed: true,
			},
			"commit_hash": &schema.StringAttribute{
				Computed: true,
			},
			"compilation_metadata_json": &schema.StringAttribute{
				Computed: true,
			},
			"optimizer": &schema.SingleNestedAttribute{
				Optional: true,
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"enabled": &schema.BoolAttribute{
						Optional:      true,
						PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
					},
					"runs": &schema.Int64Attribute{
						Optional:      true,
						PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
					},
					"via_ir": &schema.BoolAttribute{
						Optional:      true,
						PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
					},
				},
				Default: objectdefault.StaticValue(
					types.ObjectValueMust(
						map[string]attr.Type{
							"enabled": types.BoolType,
							"runs":    types.Int64Type,
							"via_ir":  types.BoolType,
						},
						map[string]attr.Value{
							"enabled": types.BoolValue(false),
							"runs":    types.Int64Value(200),
							"via_ir":  types.BoolValue(false),
						},
					),
				),
			},
			"ignore_destroy": &schema.BoolAttribute{
				Optional: true,
			},
		},
	}
}

func (data *CMSBuildResourceModel) toAPI(api *CMSBuildAPIModel, isUpdate bool) {
	api.Name = data.Name.ValueString()
	api.Path = data.Path.ValueString()
	api.Description = data.Description.ValueString()

	if !isUpdate {
		// The following fields are immutable, so we do not patch these fields if the values
		// haven't been changed. This section is required because if the value of `auth_token`
		// is updated, we don't want to trigger an replace as it's not going to result in
		// different source being retrieved.
		api.EVMVersion = data.EVMVersion.ValueString()
		api.SolcVersion = data.SolcVersion.ValueString()
		falseVar := false
		if data.Optimizer != nil {
			api.Optimizer = &CMSBuildOptimizerAPIModel{}

			if data.Optimizer.Enabled.ValueBoolPointer() != nil {
				api.Optimizer.Enabled = data.Optimizer.Enabled.ValueBoolPointer()
			} else {
				api.Optimizer.Enabled = &falseVar
			}

			if data.Optimizer.ViaIR.ValueBoolPointer() != nil {
				api.Optimizer.ViaIR = data.Optimizer.ViaIR.ValueBoolPointer()
			} else {
				api.Optimizer.ViaIR = &falseVar
			}

			if data.Optimizer.Runs.ValueInt64() != 200 {
				api.Optimizer.Runs = float64(data.Optimizer.Runs.ValueInt64())
			} else {
				api.Optimizer.Runs = 200
			}
		}

		switch data.Type.ValueString() {
		case "precompiled":
			_ = json.Unmarshal(([]byte)(data.Precompiled.ABI.ValueString()), &api.ABI)
			api.Bytecode = data.Precompiled.Bytecode.ValueString()
			_ = json.Unmarshal(([]byte)(data.Precompiled.DevDocs.ValueString()), &api.DevDocs)
		case "github":
			api.GitHub = &CMSBuildGithubAPIModel{
				ContractURL:  data.GitHub.ContractURL.ValueString(),
				ContractName: data.GitHub.ContractName.ValueString(),
				AuthToken:    data.GitHub.AuthToken.ValueString(),
			}
		case "source_code":
			api.SourceCode = &CMSBuildSourceCodeAPIModel{
				ContractName: data.SourceCode.ContractName.ValueString(),
				FileContents: data.SourceCode.FileContents.ValueString(),
			}
		}

		if data.LibrariesJSON.ValueString() != "" {
			_ = json.Unmarshal(([]byte)(data.LibrariesJSON.ValueString()), &api.Libraries)
		}
	}
}

func (api *CMSBuildAPIModel) toData(data *CMSBuildResourceModel) {
	data.ID = types.StringValue(api.ID)
	abiBytes, _ := json.Marshal(api.ABI)
	data.ABI = types.StringValue(string(abiBytes))
	data.Bytecode = types.StringValue(api.Bytecode)
	devDocsBytes, _ := json.Marshal(api.DevDocs)
	data.DevDocs = types.StringValue(string(devDocsBytes))
	if api.GitHub != nil {
		data.CommitHash = types.StringValue(api.GitHub.CommitHash)
	} else {
		data.CommitHash = types.StringValue("")
	}
	if api.Optimizer != nil {
		falseVar := false
		data.Optimizer = &CMSBuildOptimizerResourceModel{}

		if api.Optimizer.Enabled != nil {
			data.Optimizer.Enabled = types.BoolPointerValue(api.Optimizer.Enabled)
		} else {
			data.Optimizer.Enabled = types.BoolPointerValue(&falseVar)
		}

		if api.Optimizer.ViaIR != nil {
			data.Optimizer.ViaIR = types.BoolPointerValue(api.Optimizer.ViaIR)
		} else {
			data.Optimizer.ViaIR = types.BoolPointerValue(&falseVar)
		}

		if api.Optimizer.Runs != 200 {
			data.Optimizer.Runs = types.Int64Value(int64(api.Optimizer.Runs))
		} else {
			data.Optimizer.Runs = types.Int64Value(200)
		}
	}
	if api.Libraries != nil {
		librariesBytes, _ := json.Marshal(api.Libraries)
		data.LibrariesJSON = types.StringValue(string(librariesBytes))
	}
	if api.CompilationMetadata != nil {
		compilationMetadataBytes, _ := json.Marshal(api.CompilationMetadata)
		data.CompilationMetadataJSON = types.StringValue(string(compilationMetadataBytes))
	} else {
		data.CompilationMetadataJSON = types.StringValue("{}")
	}
}

func (r *cms_buildResource) apiPath(data *CMSBuildResourceModel) string {
	path := fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/builds", data.Environment.ValueString(), data.Service.ValueString())
	if data.ID.ValueString() != "" {
		path = path + "/" + data.ID.ValueString()
	}
	return path
}

func (r *cms_buildResource) waitForBuildStatus(ctx context.Context, data *CMSBuildResourceModel, api *CMSBuildAPIModel, diagnostics *diag.Diagnostics) {
	path := r.apiPath(data)
	cancelInfo := APICancelInfo()
	_ = kaleidobase.Retry.Do(ctx, fmt.Sprintf("build-check %s", path), func(attempt int) (retry bool, err error) {
		ok, _ := r.apiRequest(ctx, http.MethodGet, path, nil, &api, diagnostics, cancelInfo)
		if !ok {
			return false, fmt.Errorf("build-check failed") // already set in diag
		}
		cancelInfo.CancelInfo = fmt.Sprintf("(waiting for completion - status: %s)", api.Status)
		switch api.Status {
		case "succeeded":
			return false, nil
		case "failed":
			diagnostics.AddError("build failed", api.CompileError)
			return false, fmt.Errorf("build failed")
		default:
			return true, fmt.Errorf("not ready yet")
		}
	})
}

func (r *cms_buildResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var data CMSBuildResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	var api CMSBuildAPIModel
	data.toAPI(&api, false)
	ok, _ := r.apiRequest(ctx, http.MethodPost, r.apiPath(&data), api, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	api.toData(&data) // need the ID copied over
	r.waitForBuildStatus(ctx, &data, &api, &resp.Diagnostics)
	api.toData(&data) // capture the build info
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)

}

func (r *cms_buildResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

	var data CMSBuildResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)

	// Update from plan
	var api CMSBuildAPIModel
	data.toAPI(&api, true)
	if ok, _ := r.apiRequest(ctx, http.MethodPatch, r.apiPath(&data), api, &api, &resp.Diagnostics); !ok {
		return
	}

	api.toData(&data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *cms_buildResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CMSBuildResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	var api CMSBuildAPIModel
	api.ID = data.ID.ValueString()
	ok, status := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics, Allow404())
	if !ok {
		return
	}
	if status == 404 {
		resp.State.RemoveResource(ctx)
		return
	}
	api.toData(&data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *cms_buildResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CMSBuildResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if !data.IgnoreDestroy.IsNull() && data.IgnoreDestroy.ValueBool() {
		return
	}

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data), nil, nil, &resp.Diagnostics, Allow404())

	r.waitForRemoval(ctx, r.apiPath(&data), &resp.Diagnostics)
}
