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
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/kaleido-io/terraform-provider-kaleido/kaleido/kaleidobase"
)

type HttpOptions int

const (
	Allow404 HttpOptions = iota
)

type commonResource struct {
	*kaleidobase.ProviderData
}

func (r *commonResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.ProviderData = kaleidobase.ConfigureProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *commonResource) apiRequest(ctx context.Context, method, path string, body, result interface{}, diagnostics *diag.Diagnostics, options ...HttpOptions) (bool, int) {
	var bodyBytes []byte
	var err error
	bodyString := ""
	if body != nil {
		bodyBytes, err = json.Marshal(body)
		if err == nil {
			bodyString = string(bodyBytes)
		}
	}
	tflog.Debug(ctx, fmt.Sprintf("--> %s %s%s %s", method, r.Platform.BaseURL, path, bodyString))

	var res *resty.Response
	if err == nil {
		req := r.Platform.R().
			SetContext(ctx).
			SetHeader("Content-type", "application/json").
			SetDoNotParseResponse(true)
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
		diagnostics.AddError(
			fmt.Sprintf("%s failed", method),
			fmt.Sprintf("%s %s failed with error: %s", method, path, err),
		)
	} else if !res.IsSuccess() {
		isOk404 := false
		if statusCode == 404 {
			for _, o := range options {
				if o == Allow404 {
					isOk404 = true
				}
			}
		}
		if !isOk404 {
			ok = false
			diagnostics.AddError(
				fmt.Sprintf("%s failed", method),
				fmt.Sprintf("%s %s returned status code %d: %s", method, path, statusCode, rawBytes),
			)
		}
	}
	return ok, statusCode
}

func (r *commonResource) waitForReadyStatus(ctx context.Context, path string, diagnostics *diag.Diagnostics) {
	type statusResponse struct {
		Status string `json:"status"`
	}
	_ = kaleidobase.Retry.Do(ctx, fmt.Sprintf("ready-check %s", path), func(attempt int) (retry bool, err error) {
		var res statusResponse
		ok, _ := r.apiRequest(ctx, http.MethodGet, path, nil, &res, diagnostics)
		if !ok {
			return false, fmt.Errorf("ready-check failed") // already set in diag
		}
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
	_ = kaleidobase.Retry.Do(ctx, fmt.Sprintf("ready-check %s", path), func(attempt int) (retry bool, err error) {
		var res statusResponse
		ok, status := r.apiRequest(ctx, http.MethodGet, path, nil, &res, diagnostics, Allow404)
		if !ok {
			return false, fmt.Errorf("ready-check failed") // already set in diag
		}
		if status != 404 {
			return true, fmt.Errorf("not removed yet")
		}
		return false, nil
	})
}

func Resources() []func() resource.Resource {
	return []func() resource.Resource{
		EnvironmentResourceFactory,
		RuntimeResourceFactory,
		ServiceResourceFactory,
		NetworkResourceFactory,
		KMSWalletResourceFactory,
		KMSKeyResourceFactory,
	}
}
