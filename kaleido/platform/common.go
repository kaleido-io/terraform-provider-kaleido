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
	"strconv"
	"strings"

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
	tflog.Debug(ctx, fmt.Sprintf("--> %s %s%s", method, r.Platform.BaseURL, path))

	res, err := r.Platform.R().
		SetContext(ctx).
		SetBody(body).
		SetHeader("Content-type", "application/json").
		SetDoNotParseResponse(true).
		Execute(method, path)
	var statusString string
	var rawBytes []byte
	if err != nil {
		statusString = err.Error()
	} else {
		statusString = strconv.Itoa(res.StatusCode())
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
		if res.StatusCode() == 404 {
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
				fmt.Sprintf("%s %s returned status code %d", method, path, res.StatusCode()),
			)
		}
	}
	return ok, res.StatusCode()
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

func Resources() []func() resource.Resource {
	return []func() resource.Resource{
		RuntimeResourceFactory,
		ServiceResourceFactory,
	}
}
