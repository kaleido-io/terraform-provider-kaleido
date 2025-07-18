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

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type PrivateDataManagerServiceResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Environment         types.String `tfsdk:"environment"`
	EnvironmentMemberID types.String `tfsdk:"environment_member_id"`
	Runtime             types.String `tfsdk:"runtime"`
	Name                types.String `tfsdk:"name"`
	StackID             types.String `tfsdk:"stack_id"`

	// Configuration fields
	StorageType      types.String `tfsdk:"storage_type"`
	DataExchangeType types.String `tfsdk:"data_exchange_type"`

	// Certificate configuration - flattened
	CertificateCA   types.String `tfsdk:"certificate_ca"`   // PEM file content
	CertificateCert types.String `tfsdk:"certificate_cert"` // PEM file content
	CertificateKey  types.String `tfsdk:"certificate_key"`  // PEM file content

	// Proxy configuration
	ProxyURL types.String `tfsdk:"proxy_url"`

	// TODO we don't really want to expose these as DXE Kafka is deprecated
	// Kaleido service configuration
	KaleidoServiceURL      types.String `tfsdk:"kaleido_service_url"`
	KaleidoServiceUsername types.String `tfsdk:"kaleido_service_username"`
	KaleidoServicePassword types.String `tfsdk:"kaleido_service_password"`
	KaleidoServicePeerID   types.String `tfsdk:"kaleido_service_peer_id"`

	// HTTPS configuration
	HTTPSNetworkingType types.String `tfsdk:"https_networking_type"`
	HTTPSPeerID         types.String `tfsdk:"https_peer_id"`

	ForceDelete types.Bool `tfsdk:"force_delete"`
}

func PrivateDataManagerServiceResourceFactory() resource.Resource {
	return &privateDataManagerServiceResource{}
}

type privateDataManagerServiceResource struct {
	commonResource
}

func (r *privateDataManagerServiceResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_privatedatamanager_service"
}

func (r *privateDataManagerServiceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A Private Data Manager service that provides secure data exchange capabilities for multiparty blockchain applications.",
		Attributes: map[string]schema.Attribute{
			"id": &schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"environment": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Environment ID where the Private Data Manager service will be deployed",
			},
			"environment_member_id": &schema.StringAttribute{
				Computed: true,
			},
			"runtime": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Runtime ID where the Private Data Manager service will be deployed",
			},
			"name": &schema.StringAttribute{
				Required:    true,
				Description: "Display name for the Private Data Manager service",
			},
			"stack_id": &schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Stack ID where the Private Data Manager service belongs (must be a FireflyStack)",
			},
			// TOODO
			"storage_type": &schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("instance_managed"),
				// TODO
				Validators: []validator.String{
					stringvalidator.OneOf("instance_managed", "instance_local"),
				},
				Description: "How storage should be managed for private data",
			},
			// TODO default to none and only allow https or none ?
			"data_exchange_type": &schema.StringAttribute{
				Optional:    true,
				Description: "Type of data exchange mechanism to use",
				Validators: []validator.String{
					stringvalidator.OneOf("none", "https"),
				},
			},

			// Certificate configuration
			"certificate_ca": &schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "PEM encoded CA certificate for data encryption",
			},
			"certificate_cert": &schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "PEM encoded certificate for data encryption",
			},
			"certificate_key": &schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "PEM encoded private key for data encryption",
			},

			// Proxy configuration
			"proxy_url": &schema.StringAttribute{
				Optional:    true,
				Description: "HTTP proxy URL for cloud blob and messaging requests",
			},

			// Kaleido service configuration
			"kaleido_service_url": &schema.StringAttribute{
				Optional:    true,
				Description: "Kaleido Data Exchange service endpoint URL",
			},
			"kaleido_service_username": &schema.StringAttribute{
				Optional:    true,
				Description: "Username for Kaleido Data Exchange service authentication",
			},
			"kaleido_service_password": &schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Password for Kaleido Data Exchange service authentication",
			},
			"kaleido_service_peer_id": &schema.StringAttribute{
				Optional:    true,
				Description: "Peer ID for Kaleido Data Exchange service",
			},

			// HTTPS configuration
			"https_networking_type": &schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("instance_local"),
				Validators: []validator.String{
					stringvalidator.OneOf("instance_local", "none"),
				},
				Description: "Type of networking to expose for data exchange with other runtimes",
			},
			"https_peer_id": &schema.StringAttribute{
				Optional:    true,
				Description: "Peer ID uniquely identifying the HTTPS Data Exchange",
			},

			"force_delete": &schema.BoolAttribute{
				Optional:    true,
				Description: "Set to true when you plan to delete a protected Private Data Manager service. You must apply this value before running terraform destroy.",
			},
		},
	}
}

func (data *PrivateDataManagerServiceResourceModel) toServiceAPI(ctx context.Context, api *ServiceAPIModel, diagnostics *diag.Diagnostics) {
	api.Type = "PrivateDataManagerService"
	api.Name = data.Name.ValueString()
	api.StackID = data.StackID.ValueString()
	api.Runtime.ID = data.Runtime.ValueString()
	api.Config = make(map[string]interface{})

	// Storage type
	if !data.StorageType.IsNull() && data.StorageType.ValueString() != "" {
		api.Config["storageType"] = data.StorageType.ValueString()
	}

	// Data exchange type
	if !data.DataExchangeType.IsNull() && data.DataExchangeType.ValueString() != "" {
		api.Config["dataExchangeType"] = data.DataExchangeType.ValueString()
	}

	// Certificate configuration - handle as file refs
	hasCertificate := false
	certificateConfig := make(map[string]interface{})

	if !data.CertificateCA.IsNull() && data.CertificateCA.ValueString() != "" {
		if api.Filesets == nil {
			api.Filesets = make(map[string]*FileSetAPI)
		}
		api.Filesets["certificateCA"] = &FileSetAPI{
			Name: "certificateCA",
			Files: map[string]*FileAPI{
				"ca.pem": {
					Data: FileDataAPI{
						Text: data.CertificateCA.ValueString(),
					},
				},
			},
		}
		certificateConfig["ca"] = map[string]interface{}{
			"fileRef": "certificateCA",
		}
		hasCertificate = true
	}

	if !data.CertificateCert.IsNull() && data.CertificateCert.ValueString() != "" {
		if api.Filesets == nil {
			api.Filesets = make(map[string]*FileSetAPI)
		}
		api.Filesets["certificateCert"] = &FileSetAPI{
			Name: "certificateCert",
			Files: map[string]*FileAPI{
				"cert.pem": {
					Data: FileDataAPI{
						Text: data.CertificateCert.ValueString(),
					},
				},
			},
		}
		certificateConfig["cert"] = map[string]interface{}{
			"fileRef": "certificateCert",
		}
		hasCertificate = true
	}

	if !data.CertificateKey.IsNull() && data.CertificateKey.ValueString() != "" {
		if api.Filesets == nil {
			api.Filesets = make(map[string]*FileSetAPI)
		}
		api.Filesets["certificateKey"] = &FileSetAPI{
			Name: "certificateKey",
			Files: map[string]*FileAPI{
				"key.pem": {
					Data: FileDataAPI{
						Text: data.CertificateKey.ValueString(),
					},
				},
			},
		}
		certificateConfig["key"] = map[string]interface{}{
			"fileRef": "certificateKey",
		}
		hasCertificate = true
	}

	if hasCertificate {
		api.Config["certificate"] = certificateConfig
	}

	// Proxy configuration
	if !data.ProxyURL.IsNull() && data.ProxyURL.ValueString() != "" {
		api.Config["proxy"] = map[string]interface{}{
			"url": data.ProxyURL.ValueString(),
		}
	}

	// Kaleido service configuration
	hasKaleidoService := false
	kaleidoServiceConfig := make(map[string]interface{})

	if !data.KaleidoServiceURL.IsNull() && data.KaleidoServiceURL.ValueString() != "" {
		kaleidoServiceConfig["url"] = data.KaleidoServiceURL.ValueString()
		hasKaleidoService = true
	}

	if !data.KaleidoServicePeerID.IsNull() && data.KaleidoServicePeerID.ValueString() != "" {
		kaleidoServiceConfig["peerId"] = data.KaleidoServicePeerID.ValueString()
		hasKaleidoService = true
	}

	// Handle Kaleido service credentials
	if (!data.KaleidoServiceUsername.IsNull() && data.KaleidoServiceUsername.ValueString() != "") ||
		(!data.KaleidoServicePassword.IsNull() && data.KaleidoServicePassword.ValueString() != "") {
		credSetName := "platformAuth"
		if api.Credsets == nil {
			api.Credsets = make(map[string]*CredSetAPI)
		}
		api.Credsets[credSetName] = &CredSetAPI{
			Name: credSetName,
			Type: "basic_auth",
			BasicAuth: &CredSetBasicAuthAPI{
				Username: data.KaleidoServiceUsername.ValueString(),
				Password: data.KaleidoServicePassword.ValueString(),
			},
		}
		kaleidoServiceConfig["auth"] = map[string]interface{}{
			"credSetRef": credSetName,
		}
		hasKaleidoService = true
	}

	if hasKaleidoService {
		api.Config["kaleidoService"] = kaleidoServiceConfig
	}

	// HTTPS configuration
	hasHTTPS := false
	httpsConfig := make(map[string]interface{})

	if !data.HTTPSNetworkingType.IsNull() && data.HTTPSNetworkingType.ValueString() != "" {
		httpsConfig["networkingType"] = data.HTTPSNetworkingType.ValueString()
		hasHTTPS = true
	}

	if !data.HTTPSPeerID.IsNull() && data.HTTPSPeerID.ValueString() != "" {
		httpsConfig["peerId"] = data.HTTPSPeerID.ValueString()
		hasHTTPS = true
	}

	if hasHTTPS {
		api.Config["https"] = httpsConfig
	}
}

func (api *ServiceAPIModel) toPrivateDataManagerServiceData(data *PrivateDataManagerServiceResourceModel, diagnostics *diag.Diagnostics) {
	data.ID = types.StringValue(api.ID)
	data.EnvironmentMemberID = types.StringValue(api.EnvironmentMemberID)
	data.Runtime = types.StringValue(api.Runtime.ID)
	data.Name = types.StringValue(api.Name)
	data.StackID = types.StringValue(api.StackID)

	// Storage type
	if v, ok := api.Config["storageType"].(string); ok {
		data.StorageType = types.StringValue(v)
	} else {
		data.StorageType = types.StringValue("instance_managed")
	}

	// Data exchange type
	if v, ok := api.Config["dataExchangeType"].(string); ok {
		data.DataExchangeType = types.StringValue(v)
	} else {
		data.DataExchangeType = types.StringNull()
	}

	// TODO dont think this is possible
	// Certificate configuration - read from filesets
	data.CertificateCA = types.StringNull()
	data.CertificateCert = types.StringNull()
	data.CertificateKey = types.StringNull()

	if api.Filesets != nil {
		if fileset, ok := api.Filesets["certificateCA"]; ok && fileset != nil {
			if file, exists := fileset.Files["ca.pem"]; exists && file != nil {
				data.CertificateCA = types.StringValue(file.Data.Text)
			}
		}
		if fileset, ok := api.Filesets["certificateCert"]; ok && fileset != nil {
			if file, exists := fileset.Files["cert.pem"]; exists && file != nil {
				data.CertificateCert = types.StringValue(file.Data.Text)
			}
		}
		if fileset, ok := api.Filesets["certificateKey"]; ok && fileset != nil {
			if file, exists := fileset.Files["key.pem"]; exists && file != nil {
				data.CertificateKey = types.StringValue(file.Data.Text)
			}
		}
	}

	// Proxy configuration
	if v, ok := api.Config["proxy"].(map[string]interface{}); ok {
		if url, ok := v["url"].(string); ok {
			data.ProxyURL = types.StringValue(url)
		}
	} else {
		data.ProxyURL = types.StringNull()
	}

	// Kaleido service configuration
	if v, ok := api.Config["kaleidoService"].(map[string]interface{}); ok {
		if url, ok := v["url"].(string); ok {
			data.KaleidoServiceURL = types.StringValue(url)
		} else {
			data.KaleidoServiceURL = types.StringNull()
		}
		if peerId, ok := v["peerId"].(string); ok {
			data.KaleidoServicePeerID = types.StringValue(peerId)
		} else {
			data.KaleidoServicePeerID = types.StringNull()
		}
	} else {
		data.KaleidoServiceURL = types.StringNull()
		data.KaleidoServicePeerID = types.StringNull()
	}

	// Handle Kaleido service credentials from credsets
	data.KaleidoServiceUsername = types.StringNull()
	data.KaleidoServicePassword = types.StringNull()
	if api.Credsets != nil {
		if credSet, ok := api.Credsets["platformAuth"]; ok && credSet != nil && credSet.BasicAuth != nil {
			data.KaleidoServiceUsername = types.StringValue(credSet.BasicAuth.Username)
			data.KaleidoServicePassword = types.StringValue(credSet.BasicAuth.Password)
		}
	}

	// HTTPS configuration
	if v, ok := api.Config["https"].(map[string]interface{}); ok {
		if networkingType, ok := v["networkingType"].(string); ok {
			data.HTTPSNetworkingType = types.StringValue(networkingType)
		} else {
			data.HTTPSNetworkingType = types.StringValue("instance_local")
		}
		if peerId, ok := v["peerId"].(string); ok {
			data.HTTPSPeerID = types.StringValue(peerId)
		} else {
			data.HTTPSPeerID = types.StringNull()
		}
	} else {
		data.HTTPSNetworkingType = types.StringValue("instance_local")
		data.HTTPSPeerID = types.StringNull()
	}
}

func (r *privateDataManagerServiceResource) apiPath(data *PrivateDataManagerServiceResourceModel) string {
	path := fmt.Sprintf("/api/v1/environments/%s/services", data.Environment.ValueString())
	if data.ID.ValueString() != "" {
		path = path + "/" + data.ID.ValueString()
	}
	if !data.ForceDelete.IsNull() && data.ForceDelete.ValueBool() {
		path = path + "?force=true"
	}
	return path
}

func (r *privateDataManagerServiceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PrivateDataManagerServiceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	var api ServiceAPIModel
	data.toServiceAPI(ctx, &api, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	ok, _ := r.apiRequest(ctx, http.MethodPost, r.apiPath(&data), api, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	api.toPrivateDataManagerServiceData(&data, &resp.Diagnostics)
	r.waitForReadyStatus(ctx, r.apiPath(&data), &resp.Diagnostics)

	api.ID = data.ID.ValueString()
	ok, _ = r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics)
	if !ok {
		return
	}

	api.toPrivateDataManagerServiceData(&data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *privateDataManagerServiceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data PrivateDataManagerServiceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &data.ID)...)

	var api ServiceAPIModel
	if ok, _ := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics); !ok {
		return
	}

	data.toServiceAPI(ctx, &api, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if ok, _ := r.apiRequest(ctx, http.MethodPut, r.apiPath(&data), api, &api, &resp.Diagnostics); !ok {
		return
	}

	api.toPrivateDataManagerServiceData(&data, &resp.Diagnostics)
	r.waitForReadyStatus(ctx, r.apiPath(&data), &resp.Diagnostics)

	if ok, _ := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics); !ok {
		return
	}
	api.toPrivateDataManagerServiceData(&data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *privateDataManagerServiceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PrivateDataManagerServiceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	var api ServiceAPIModel
	api.ID = data.ID.ValueString()
	ok, status := r.apiRequest(ctx, http.MethodGet, r.apiPath(&data), nil, &api, &resp.Diagnostics, Allow404())
	if !ok {
		return
	}
	if status == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	api.toPrivateDataManagerServiceData(&data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *privateDataManagerServiceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PrivateDataManagerServiceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	_, _ = r.apiRequest(ctx, http.MethodDelete, r.apiPath(&data), nil, nil, &resp.Diagnostics, Allow404())
	r.waitForRemoval(ctx, r.apiPath(&data), &resp.Diagnostics)
}
