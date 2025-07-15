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
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kaleido-io/terraform-provider-kaleido/kaleido/kaleidobase"
)

type CMSActionBaseAPIModel struct {
	ID          string     `json:"id,omitempty"`
	Created     *time.Time `json:"created,omitempty"`
	Updated     *time.Time `json:"updated,omitempty"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	Type        string     `json:"type,omitempty"`
}

type CMSActionOutputBaseAPIModel struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type cms_action_baseResource struct {
	commonResource
}

type CMSActionAPIBaseAccessor interface {
	ActionBase() *CMSActionBaseAPIModel
	OutputBase() *CMSActionOutputBaseAPIModel
}

type CMSActionResourceBaseAccessor interface {
	ResourceIdentifiers() (types.String, types.String, types.String)
}

func (a *CMSActionBaseAPIModel) ActionBase() *CMSActionBaseAPIModel {
	return a
}

func (r *cms_action_baseResource) apiPath(data CMSActionResourceBaseAccessor) string {
	environment, service, id := data.ResourceIdentifiers()
	path := fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/actions", environment.ValueString(), service.ValueString())
	if id.ValueString() != "" {
		path = path + "/" + id.ValueString()
	}
	return path
}

func (r *cms_action_baseResource) waitForActionStatus(ctx context.Context, data CMSActionResourceBaseAccessor, api CMSActionAPIBaseAccessor, diagnostics *diag.Diagnostics) {
	path := r.apiPath(data)
	cancelInfo := APICancelInfo()
	_ = kaleidobase.Retry.Do(ctx, fmt.Sprintf("build-check %s", path), func(attempt int) (retry bool, err error) {
		ok, statusCode := r.apiRequest(ctx, http.MethodGet, path, nil, &api, diagnostics, cancelInfo)
		if !ok {
			if statusCode == 429 {
				return true, fmt.Errorf("rate limit exceeded")
			}
			return false, fmt.Errorf("action-check failed") // already set in diag
		}
		cancelInfo.CancelInfo = fmt.Sprintf("waiting for completion - status: %s", api.OutputBase().Status)
		switch api.OutputBase().Status {
		case "succeeded":
			return false, nil
		case "failed":
			diagnostics.AddError("action failed", api.OutputBase().Error)
			return false, fmt.Errorf("action failed")
		default:
			return true, fmt.Errorf("not ready yet")
		}
	})
}
