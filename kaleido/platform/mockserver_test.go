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
	t        *testing.T
	lock     sync.Mutex
	router   *mux.Router
	server   *httptest.Server
	runtimes map[string]*RuntimeAPIModel
	services map[string]*ServiceAPIModel
	calls    []string
}

func startMockPlatformServer(t *testing.T) *mockPlatform {
	mp := &mockPlatform{
		t:        t,
		runtimes: make(map[string]*RuntimeAPIModel),
		services: make(map[string]*ServiceAPIModel),
		router:   mux.NewRouter(),
		calls:    []string{},
	}
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

	mp.server = httptest.NewServer(mp.router)
	return mp
}

func (mp *mockPlatform) checkClearCalls(expected []string) {
	assert.Equal(mp.t, mp.calls, expected)
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
