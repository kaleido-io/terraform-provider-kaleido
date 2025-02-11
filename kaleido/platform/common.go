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
	"io"
	"net/http"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"gopkg.in/yaml.v3"

	"github.com/kaleido-io/terraform-provider-kaleido/kaleido/kaleidobase"
)

type APIRequestOption struct {
	allow404         bool
	captureLastError bool
	yamlBody         bool
	CancelInfo       string
}

type commonResource struct {
	*kaleidobase.ProviderData
}

func (r *commonResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.ProviderData = kaleidobase.ConfigureProviderData(req.ProviderData, &resp.Diagnostics)
}

type commonDataSource struct {
	*kaleidobase.ProviderData
}

func (r *commonDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	r.ProviderData = kaleidobase.ConfigureProviderData(req.ProviderData, &resp.Diagnostics)
}

func Allow404() *APIRequestOption {
	return &APIRequestOption{
		allow404: true,
	}
}

func YAMLBody() *APIRequestOption {
	return &APIRequestOption{
		yamlBody: true,
	}
}

func APICancelInfo() *APIRequestOption {
	return &APIRequestOption{
		captureLastError: true,
	}
}

func (r *commonResource) apiRequest(ctx context.Context, method, path string, body, result interface{}, diagnostics *diag.Diagnostics, options ...*APIRequestOption) (bool, int) {
	var bodyBytes []byte
	var err error
	bodyString := ""
	isYaml := false
	for _, o := range options {
		isYaml = isYaml || o.yamlBody
	}
	if body != nil {
		switch tBody := body.(type) {
		case []byte:
			bodyBytes = tBody
			bodyString = string(tBody)
		case string:
			bodyBytes = []byte(tBody)
			bodyString = tBody
		default:
			if isYaml {
				bodyBytes, err = yaml.Marshal(body)
			} else {
				bodyBytes, err = json.Marshal(body)
			}
		}
		if err == nil {
			bodyString = string(bodyBytes)
		}
	}
	tflog.Debug(ctx, fmt.Sprintf("--> %s %s%s %s", method, r.Platform.BaseURL, path, bodyString))

	var res *resty.Response
	if err == nil {
		req := r.Platform.R().
			SetContext(ctx).
			SetDoNotParseResponse(true)

		if isYaml {
			req = req.SetHeader("Content-type", "application/x-yaml")
		} else {
			req = req.SetHeader("Content-type", "application/json")
		}
		if bodyBytes != nil {
			req = req.SetBody(bodyBytes)
		}
		res, err = req.Execute(method, path)
	}
	var statusString string
	var rawBytes []byte
	statusCode := -1
	if err != nil {
		statusString = err.Error()
	} else {
		statusCode = res.StatusCode()
		statusString = fmt.Sprintf("%d %s", statusCode, res.Status())
	}
	tflog.Debug(ctx, fmt.Sprintf("<-- %s %s%s [%s]", method, r.Platform.BaseURL, path, statusString))
	if res != nil && res.RawResponse != nil {
		defer res.RawResponse.Body.Close()
		rawBytes, err = io.ReadAll(res.RawBody())
	}
	if rawBytes != nil {
		tflog.Debug(ctx, fmt.Sprintf("Response: %s", rawBytes))
	}
	if err == nil && res.IsSuccess() && result != nil {
		err = json.Unmarshal(rawBytes, &result)
	}
	ok := true
	if err != nil {
		ok = false
		isCancelled := false
		select {
		case <-ctx.Done():
			isCancelled = true
		default:
		}
		errorInfo := fmt.Sprintf("%s %s failed with error: %s", method, path, err)
		if isCancelled {
			for _, o := range options {
				if o.CancelInfo != "" {
					errorInfo = fmt.Sprintf("%s %s", errorInfo, o.CancelInfo)
				}
			}
			diagnostics.AddError(
				fmt.Sprintf("%s cancelled", method),
				errorInfo,
			)
		} else {
			diagnostics.AddError(
				fmt.Sprintf("%s failed", method),
				errorInfo,
			)
		}
	} else if !res.IsSuccess() {
		isOk404 := false
		if statusCode == 404 {
			for _, o := range options {
				if o.allow404 {
					isOk404 = true
				}
			}
		}
		if !isOk404 {
			ok = false
			errorInfo := fmt.Sprintf("%s %s returned status code %d: %s", method, path, statusCode, rawBytes)
			diagnostics.AddError(
				fmt.Sprintf("%s failed", method),
				errorInfo,
			)
		}
	}
	return ok, statusCode
}

func (r *commonDataSource) apiRequest(ctx context.Context, method, path string, body, result interface{}, diagnostics *diag.Diagnostics, options ...*APIRequestOption) (bool, int) {
	var bodyBytes []byte
	var err error
	bodyString := ""
	isYaml := false
	for _, o := range options {
		isYaml = isYaml || o.yamlBody
	}
	if body != nil {
		switch tBody := body.(type) {
		case []byte:
			bodyBytes = tBody
			bodyString = string(tBody)
		case string:
			bodyBytes = []byte(tBody)
			bodyString = tBody
		default:
			if isYaml {
				bodyBytes, err = yaml.Marshal(body)
			} else {
				bodyBytes, err = json.Marshal(body)
			}
		}
		if err == nil {
			bodyString = string(bodyBytes)
		}
	}
	tflog.Debug(ctx, fmt.Sprintf("--> %s %s%s %s", method, r.Platform.BaseURL, path, bodyString))

	var res *resty.Response
	if err == nil {
		req := r.Platform.R().
			SetContext(ctx).
			SetDoNotParseResponse(true)

		if isYaml {
			req = req.SetHeader("Content-type", "application/x-yaml")
		} else {
			req = req.SetHeader("Content-type", "application/json")
		}
		if bodyBytes != nil {
			req = req.SetBody(bodyBytes)
		}
		res, err = req.Execute(method, path)
	}
	var statusString string
	var rawBytes []byte
	statusCode := -1
	if err != nil {
		statusString = err.Error()
	} else {
		statusCode = res.StatusCode()
		statusString = fmt.Sprintf("%d %s", statusCode, res.Status())
	}
	tflog.Debug(ctx, fmt.Sprintf("<-- %s %s%s [%s]", method, r.Platform.BaseURL, path, statusString))
	if res != nil && res.RawResponse != nil {
		defer res.RawResponse.Body.Close()
		rawBytes, err = io.ReadAll(res.RawBody())
	}
	if rawBytes != nil {
		tflog.Debug(ctx, fmt.Sprintf("Response: %s", rawBytes))
	}
	if err == nil && res.IsSuccess() && result != nil {
		err = json.Unmarshal(rawBytes, &result)
	}
	ok := true
	if err != nil {
		ok = false
		isCancelled := false
		select {
		case <-ctx.Done():
			isCancelled = true
		default:
		}
		errorInfo := fmt.Sprintf("%s %s failed with error: %s", method, path, err)
		if isCancelled {
			for _, o := range options {
				if o.CancelInfo != "" {
					errorInfo = fmt.Sprintf("%s %s", errorInfo, o.CancelInfo)
				}
			}
			diagnostics.AddError(
				fmt.Sprintf("%s cancelled", method),
				errorInfo,
			)
		} else {
			diagnostics.AddError(
				fmt.Sprintf("%s failed", method),
				errorInfo,
			)
		}
	} else if !res.IsSuccess() {
		isOk404 := false
		if statusCode == 404 {
			for _, o := range options {
				if o.allow404 {
					isOk404 = true
				}
			}
		}
		if !isOk404 {
			ok = false
			errorInfo := fmt.Sprintf("%s %s returned status code %d: %s", method, path, statusCode, rawBytes)
			diagnostics.AddError(
				fmt.Sprintf("%s failed", method),
				errorInfo,
			)
		}
	}
	return ok, statusCode
}

func (r *commonResource) waitForReadyStatus(ctx context.Context, path string, diagnostics *diag.Diagnostics) {
	type statusResponse struct {
		Status string `json:"status"`
	}
	cancelInfo := APICancelInfo()
	_ = kaleidobase.Retry.Do(ctx, fmt.Sprintf("ready-check %s", path), func(attempt int) (retry bool, err error) {
		var res statusResponse
		ok, _ := r.apiRequest(ctx, http.MethodGet, path, nil, &res, diagnostics, cancelInfo)
		if !ok {
			return false, fmt.Errorf("ready-check failed") // already set in diag
		}
		cancelInfo.CancelInfo = fmt.Sprintf("(waiting for ready - status: %s)", res.Status)
		if !strings.EqualFold(res.Status, "ready") {
			return true, fmt.Errorf("not ready yet")
		}
		return false, nil
	})
}

func (r *commonResource) waitForRemoval(ctx context.Context, path string, diagnostics *diag.Diagnostics) {
	type statusResponse struct {
		Status string `json:"status"`
	}
	cancelInfo := APICancelInfo()
	cancelInfo.CancelInfo = "(waiting for removal)"
	_ = kaleidobase.Retry.Do(ctx, fmt.Sprintf("ready-check %s", path), func(attempt int) (retry bool, err error) {
		var res statusResponse
		ok, status := r.apiRequest(ctx, http.MethodGet, path, nil, &res, diagnostics, Allow404(), cancelInfo)
		if !ok {
			return false, fmt.Errorf("ready-check failed") // already set in diag
		}
		if status != 404 {
			return true, fmt.Errorf("not removed yet")
		}
		return false, nil
	})
}

func DataSources() []func() datasource.DataSource {
	return []func() datasource.DataSource{
		EVMNetInfoDataSourceFactory,
		NetworkBootstrapDatasourceModelFactory,
	}
}

func Resources() []func() resource.Resource {
	return []func() resource.Resource{
		EnvironmentResourceFactory,
		GroupFactory,
		RuntimeResourceFactory,
		ServiceResourceFactory,
		ServiceAccessResourceFactory,
		NetworkResourceFactory,
		KMSWalletResourceFactory,
		KMSKeyResourceFactory,
		CMSBuildResourceFactory,
		CMSActionDeployResourceFactory,
		CMSActionCreateAPIResourceFactory,
		AMSTaskResourceFactory,
		AMSPolicyResourceFactory,
		AMSFFListenerResourceFactory,
		AMSDMListenerResourceFactory,
		AMSDMUpsertResourceFactory,
		AMSVariableSetResourceFactory,
		FireFlyRegistrationResourceFactory,
		AuthenticatorResourceFactory,
		ApplicationResourceFactory,
		APIKeyResourceFactory,
	}
}

type FileSetAPI struct {
	Name  string              `json:"name"`
	Files map[string]*FileAPI `json:"files"`
}

type FileAPI struct {
	Type string      `json:"type,omitempty"`
	Data FileDataAPI `json:"data,omitempty"`
}

type FileDataAPI struct {
	Base64 string `json:"base64,omitempty"`
	Text   string `json:"text,omitempty"`
	Hex    string `json:"hex,omitempty"`
}

type CredSetAPI struct {
	Name      string               `json:"name"`
	Type      string               `json:"type,omitempty"`
	BasicAuth *CredSetBasicAuthAPI `json:"basicAuth,omitempty"`
	Key       *CredSetKeyAPI       `json:"key,omitempty"`
}

type CredSetBasicAuthAPI struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

type CredSetKeyAPI struct {
	Value string `json:"value,omitempty"`
}
