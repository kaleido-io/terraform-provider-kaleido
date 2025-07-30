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

	"github.com/hashicorp/terraform-plugin-framework/diag"

	"github.com/kaleido-io/terraform-provider-kaleido/kaleido/kaleidobase"
)

// Shared API models for all CMS build types
type CMSBuildAPIModel struct {
	ID           string                      `json:"id,omitempty"`
	Created      *time.Time                  `json:"created,omitempty"`
	Updated      *time.Time                  `json:"updated,omitempty"`
	Name         string                      `json:"name"`
	Path         string                      `json:"path"`
	Description  string                      `json:"description,omitempty"`
	EVMVersion   string                      `json:"evmVersion,omitempty"`
	SolcVersion  string                      `json:"solcVersion,omitempty"`
	GitHub       *CMSBuildGithubAPIModel     `json:"github,omitempty"`
	Optimizer    *CMSBuildOptimizerAPIModel  `json:"optimizer,omitempty"`
	SourceCode   *CMSBuildSourceCodeAPIModel `json:"sourceCode,omitempty"`
	ABI          interface{}                 `json:"abi,omitempty"`
	Bytecode     string                      `json:"bytecode,omitempty"`
	DevDocs      interface{}                 `json:"devDocs,omitempty"`
	CompileError string                      `json:"compileError,omitempty"`
	Status       string                      `json:"status,omitempty"`
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

// Shared utility functions for CMS build resources
type cmsBuildResourceBase struct {
	commonResource
}

func (r *cmsBuildResourceBase) buildAPIPath(environment, service, id string) string {
	path := fmt.Sprintf("/endpoint/%s/%s/rest/api/v1/builds", environment, service)
	if id != "" {
		path = path + "/" + id
	}
	return path
}

func (r *cmsBuildResourceBase) waitForBuildStatus(ctx context.Context, path string, api *CMSBuildAPIModel, diagnostics *diag.Diagnostics) {
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

// Helper function to convert interface{} to JSON string safely
func interfaceToJSONString(data interface{}) string {
	if data == nil {
		return ""
	}
	bytes, _ := json.Marshal(data)
	return string(bytes)
}

// Helper function to unmarshal JSON string to interface{} safely
func jsonStringToInterface(jsonStr string, target *interface{}) {
	if jsonStr != "" {
		_ = json.Unmarshal([]byte(jsonStr), target)
	}
}
