// Copyright 2020 Kaleido, a ConsenSys business

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
	"net/http/httptest"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/kaleido-io/terraform-provider-kaleido/kaleido/kaleidobase"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

var testAccProviders map[string]func() (tfprotov6.ProviderServer, error)

func testSetup(t *testing.T) (mp *mockPlatform, providerConfig string) {
	mp = startMockPlatformServer(t)
	providerConfig = fmt.Sprintf(`
provider "kaleido" {
	platform_api = "%s"
}
`,
		mp.server.URL)
	return
}

func init() {
	kaleidoProvider := kaleidobase.New(
		"0.0.1-unittest",
		Resources(),
		DataSources(),
	)
	testAccProviders = map[string]func() (tfprotov6.ProviderServer, error){
		"kaleido": providerserver.NewProtocol6WithError(kaleidoProvider),
	}
}

func testJSONEqual(t *testing.T, obj interface{}, expected string) {
	assert.NotNil(t, obj)
	jsonObj, err := json.Marshal(obj)
	assert.NoError(t, err)
	t.Logf("%s\n", jsonObj)
	assert.JSONEq(t, expected, string(jsonObj))
}

func testYAMLEqual(t *testing.T, obj interface{}, expected string) {
	assert.NotNil(t, obj)
	yamlObj, err := yaml.Marshal(obj)
	assert.NoError(t, err)
	assert.YAMLEq(t, expected, string(yamlObj))
}

// TestAPIRequestAllow409 verifies that apiRequest with Allow409() treats HTTP 409 as success.
func TestAPIRequestAllow409(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
	}))
	defer srv.Close()

	client := resty.New().SetBaseURL(srv.URL)
	r := &commonResource{ProviderData: &kaleidobase.ProviderData{Platform: client}}
	ctx := context.Background()
	var diag diag.Diagnostics

	ok, statusCode := r.apiRequest(ctx, http.MethodPost, "/api/v1/foo", nil, nil, &diag, Allow409())

	assert.True(t, ok, "apiRequest with Allow409() should return ok=true for 409")
	assert.Equal(t, http.StatusConflict, statusCode)
	assert.False(t, diag.HasError(), "diagnostics should have no error for allowed 409")
}

// TestAPIRequest409WithoutOption verifies that apiRequest without Allow409() treats HTTP 409 as failure.
func TestAPIRequest409WithoutOption(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
	}))
	defer srv.Close()

	client := resty.New().SetBaseURL(srv.URL)
	r := &commonResource{ProviderData: &kaleidobase.ProviderData{Platform: client}}
	ctx := context.Background()
	var diag diag.Diagnostics

	ok, statusCode := r.apiRequest(ctx, http.MethodPost, "/api/v1/foo", nil, nil, &diag)

	assert.False(t, ok)
	assert.Equal(t, http.StatusConflict, statusCode)
	assert.True(t, diag.HasError(), "diagnostics should contain error for 409 when Allow409 not set")
}

// TestAPIRequestAllow404 verifies that apiRequest with Allow404() treats HTTP 404 as success.
func TestAPIRequestAllow404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := resty.New().SetBaseURL(srv.URL)
	r := &commonResource{ProviderData: &kaleidobase.ProviderData{Platform: client}}
	ctx := context.Background()
	var diag diag.Diagnostics

	ok, statusCode := r.apiRequest(ctx, http.MethodGet, "/api/v1/foo", nil, nil, &diag, Allow404())

	assert.True(t, ok, "apiRequest with Allow404() should return ok=true for 404")
	assert.Equal(t, http.StatusNotFound, statusCode)
	assert.False(t, diag.HasError(), "diagnostics should have no error for allowed 404")
}

// TestAPIRequest404WithoutOption verifies that apiRequest without Allow404() treats HTTP 404 as failure.
func TestAPIRequest404WithoutOption(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := resty.New().SetBaseURL(srv.URL)
	r := &commonResource{ProviderData: &kaleidobase.ProviderData{Platform: client}}
	ctx := context.Background()
	var diag diag.Diagnostics

	ok, statusCode := r.apiRequest(ctx, http.MethodGet, "/api/v1/foo", nil, nil, &diag)

	assert.False(t, ok)
	assert.Equal(t, http.StatusNotFound, statusCode)
	assert.True(t, diag.HasError(), "diagnostics should contain error for 404 when Allow404 not set")
}
