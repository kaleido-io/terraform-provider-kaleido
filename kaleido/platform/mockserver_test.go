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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

type mockPlatform struct {
	t            *testing.T
	lock         sync.Mutex
	router       *mux.Router
	server       *httptest.Server
	environments map[string]*EnvironmentAPIModel
	runtimes     map[string]*RuntimeAPIModel
	services     map[string]*ServiceAPIModel
	networks     map[string]*NetworkAPIModel
	kmsWallets   map[string]*KMSWalletAPIModel
	kmsKeys      map[string]*KMSKeyAPIModel
	cmsBuilds    map[string]*CMSBuildAPIModel
	calls        []string
}

func startMockPlatformServer(t *testing.T) *mockPlatform {
	mp := &mockPlatform{
		t:            t,
		environments: make(map[string]*EnvironmentAPIModel),
		runtimes:     make(map[string]*RuntimeAPIModel),
		services:     make(map[string]*ServiceAPIModel),
		networks:     make(map[string]*NetworkAPIModel),
		kmsWallets:   make(map[string]*KMSWalletAPIModel),
		kmsKeys:      make(map[string]*KMSKeyAPIModel),
		cmsBuilds:    make(map[string]*CMSBuildAPIModel),
		router:       mux.NewRouter(),
		calls:        []string{},
	}
	// See environment_test.go
	mp.register("/api/v1/environments", http.MethodPost, mp.postEnvironment)
	mp.register("/api/v1/environments/{env}", http.MethodGet, mp.getEnvironment)
	mp.register("/api/v1/environments/{env}", http.MethodPut, mp.putEnvironment)
	mp.register("/api/v1/environments/{env}", http.MethodDelete, mp.deleteEnvironment)

	// See runtime_test.go
	mp.register("/api/v1/environments/{env}/runtimes", http.MethodPost, mp.postRuntime)
	mp.register("/api/v1/environments/{env}/runtimes/{runtime}", http.MethodGet, mp.getRuntime)
	mp.register("/api/v1/environments/{env}/runtimes/{runtime}", http.MethodPut, mp.putRuntime)
	mp.register("/api/v1/environments/{env}/runtimes/{runtime}", http.MethodDelete, mp.deleteRuntime)

	// See service_test.go
	mp.register("/api/v1/environments/{env}/services", http.MethodPost, mp.postService)
	mp.register("/api/v1/environments/{env}/services/{service}", http.MethodGet, mp.getService)
	mp.register("/api/v1/environments/{env}/services/{service}", http.MethodPut, mp.putService)
	mp.register("/api/v1/environments/{env}/services/{service}", http.MethodDelete, mp.deleteService)

	// See network_test.go
	mp.register("/api/v1/environments/{env}/networks", http.MethodPost, mp.postNetwork)
	mp.register("/api/v1/environments/{env}/networks/{network}", http.MethodGet, mp.getNetwork)
	mp.register("/api/v1/environments/{env}/networks/{network}", http.MethodPut, mp.putNetwork)
	mp.register("/api/v1/environments/{env}/networks/{network}", http.MethodDelete, mp.deleteNetwork)

	// See kms_wallet.go
	mp.register("/endpoint/{env}/{service}/rest/api/v1/wallets", http.MethodPost, mp.postKMSWallet)
	mp.register("/endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}", http.MethodGet, mp.getKMSWallet)
	mp.register("/endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}", http.MethodPut, mp.putKMSWallet)
	mp.register("/endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}", http.MethodDelete, mp.deleteKMSWallet)

	// See kms_key.go
	mp.register("/endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}/keys", http.MethodPut, mp.putKMSKey)
	mp.register("/endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}/keys/{key}", http.MethodGet, mp.getKMSKey)
	mp.register("/endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}/keys/{key}", http.MethodPatch, mp.patchKMSKey)
	mp.register("/endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}/keys/{key}", http.MethodDelete, mp.deleteKMSKey)

	// See cms_build.go
	mp.register("/endpoint/{env}/{service}/rest/api/v1/builds", http.MethodPost, mp.postCMSBuild)
	mp.register("/endpoint/{env}/{service}/rest/api/v1/builds/{build}", http.MethodGet, mp.getCMSBuild)
	mp.register("/endpoint/{env}/{service}/rest/api/v1/builds/{build}", http.MethodPatch, mp.patchCMSBuild)
	mp.register("/endpoint/{env}/{service}/rest/api/v1/builds/{build}", http.MethodDelete, mp.deleteCMSBuild)

	mp.server = httptest.NewServer(mp.router)
	return mp
}

func (mp *mockPlatform) checkClearCalls(expected []string) {
	assert.Equal(mp.t, expected, mp.calls)
	mp.calls = []string{}
}

func (mp *mockPlatform) register(pathMatch, method string, handler http.HandlerFunc) {
	mp.router.HandleFunc(pathMatch, func(res http.ResponseWriter, req *http.Request) {
		mp.lock.Lock()
		defer mp.lock.Unlock()
		sniffed, err := io.ReadAll(req.Body)
		assert.NoError(mp.t, err)
		req.Body = io.NopCloser(bytes.NewBuffer(sniffed))

		genericCall := fmt.Sprintf("%s %s", method, pathMatch)
		fmt.Printf("%s (%s): %s\n", req.URL, genericCall, sniffed)
		mp.calls = append(mp.calls, genericCall)

		handler(res, req)
	}).Methods(method)
}

func (mp *mockPlatform) respond(res http.ResponseWriter, body interface{}, status int) {
	var bytes []byte
	var err error
	if body != nil {
		bytes, err = json.Marshal(body)
		assert.NoError(mp.t, err)
	}
	res.Header().Set("Content-Length", strconv.Itoa(len(bytes)))
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(status)
	if len(bytes) > 0 {
		res.Write(bytes)
	}
}

func (mp *mockPlatform) getBody(req *http.Request, body interface{}) {
	err := json.NewDecoder(req.Body).Decode(&body)
	assert.NoError(mp.t, err)
}
