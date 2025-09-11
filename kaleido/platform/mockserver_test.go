// Copyright © Kaleido, Inc. 2024-2025

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
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

type mockPlatform struct {
	t                       *testing.T
	lock                    sync.Mutex
	router                  *mux.Router
	server                  *httptest.Server
	environments            map[string]*EnvironmentAPIModel
	runtimes                map[string]*RuntimeAPIModel
	services                map[string]*ServiceAPIModel
	networks                map[string]*NetworkAPIModel
	stacks                  map[string]*StacksAPIModel
	connectors              map[string]*ConnectorAPIModel
	networkinitdatas        map[string]*NetworkInitData
	kmsWallets              map[string]*KMSWalletAPIModel
	kmsKeys                 map[string]*KMSKeyAPIModel
	cmsBuilds               map[string]*CMSBuildAPIModel
	cmsActions              map[string]CMSActionAPIBaseAccessor
	amsTasks                map[string]*AMSTaskAPIModel
	amsTaskVersions         map[string]map[string]interface{}
	amsPolicies             map[string]*AMSPolicyAPIModel
	amsPolicyVersions       map[string]*AMSPolicyVersionAPIModel
	amsDMUpserts            map[string]map[string]interface{}
	amsFFListeners          map[string]*AMSFFListenerAPIModel
	amsDMListeners          map[string]*AMSDMListenerAPIModel
	amsVariableSets         map[string]*AMSVariableSetAPIModel
	amsCollections          map[string]*AMSCollectionAPIModel
	groups                  map[string]*GroupAPIModel
	ffsNode                 *FireFlyStatusNodeAPIModel
	ffsOrg                  *FireFlyStatusOrgAPIModel
	calls                   []string
	applications            map[string]*ApplicationAPIModel
	apiKeys                 map[string]*APIKeyAPIModel
	serviceAccess           map[string]*ServiceAccessAPIModel
	stackAccess             map[string]*StackAccessAPIModel
	wmsWallets              map[string]*WMSWalletAPIModel
	wmsAssets               map[string]*WMSAssetAPIModel
	wmsAssetIcons           map[string]*struct{}
	wmsAccounts             map[string]*WMSAccountAPIModel
	policyIdentities        map[string]*PolicyIdentityAPIModel
	pmsIdentityLists        map[string]*PMSIdentityListAPIModel
	pmsIdentityListVersions map[string]map[string]*PMSIdentityListVersionAPIModel
}

func startMockPlatformServer(t *testing.T) *mockPlatform {
	mp := &mockPlatform{
		t:                       t,
		environments:            make(map[string]*EnvironmentAPIModel),
		runtimes:                make(map[string]*RuntimeAPIModel),
		services:                make(map[string]*ServiceAPIModel),
		networks:                make(map[string]*NetworkAPIModel),
		stacks:                  make(map[string]*StacksAPIModel),
		connectors:              make(map[string]*ConnectorAPIModel),
		networkinitdatas:        make(map[string]*NetworkInitData),
		kmsWallets:              make(map[string]*KMSWalletAPIModel),
		kmsKeys:                 make(map[string]*KMSKeyAPIModel),
		cmsBuilds:               make(map[string]*CMSBuildAPIModel),
		cmsActions:              make(map[string]CMSActionAPIBaseAccessor),
		amsTasks:                make(map[string]*AMSTaskAPIModel),
		amsTaskVersions:         make(map[string]map[string]interface{}),
		amsPolicies:             make(map[string]*AMSPolicyAPIModel),
		amsPolicyVersions:       make(map[string]*AMSPolicyVersionAPIModel),
		amsDMUpserts:            make(map[string]map[string]interface{}),
		amsFFListeners:          make(map[string]*AMSFFListenerAPIModel),
		amsDMListeners:          make(map[string]*AMSDMListenerAPIModel),
		amsVariableSets:         make(map[string]*AMSVariableSetAPIModel),
		amsCollections:          make(map[string]*AMSCollectionAPIModel),
		groups:                  make(map[string]*GroupAPIModel),
		applications:            make(map[string]*ApplicationAPIModel),
		serviceAccess:           make(map[string]*ServiceAccessAPIModel),
		stackAccess:             make(map[string]*StackAccessAPIModel),
		apiKeys:                 make(map[string]*APIKeyAPIModel),
		wmsWallets:              make(map[string]*WMSWalletAPIModel),
		wmsAssets:               make(map[string]*WMSAssetAPIModel),
		wmsAssetIcons:           make(map[string]*struct{}),
		wmsAccounts:             make(map[string]*WMSAccountAPIModel),
		policyIdentities:        make(map[string]*PolicyIdentityAPIModel),
		pmsIdentityLists:        make(map[string]*PMSIdentityListAPIModel),
		pmsIdentityListVersions: make(map[string]map[string]*PMSIdentityListVersionAPIModel),
		router:                  mux.NewRouter(),
		calls:                   []string{},
	}

	// See account_test.go
	mp.register("/api/v1/self/identity", http.MethodGet, mp.getSelfIdentity)

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

	// See network_connector_test.go
	mp.register("/api/v1/environments/{env}/networks/{net}/connectors", http.MethodPost, mp.postConnector)
	mp.register("/api/v1/environments/{env}/networks/{net}/connectors/{connector}", http.MethodGet, mp.getConnector)
	mp.register("/api/v1/environments/{env}/networks/{net}/connectors/{connector}", http.MethodPut, mp.putConnector)
	mp.register("/api/v1/environments/{env}/networks/{net}/connectors/{connector}", http.MethodDelete, mp.deleteConnector)

	// See network_bootstrap_test.go
	mp.register("/api/v1/environments/{env}/networks/{network}/initdata", http.MethodGet, mp.getNetworkInitData)

	// See kms_wallet.go
	//mp.register("/endpoint/{env}/{service}/rest/api/v1/wallets", http.MethodPost, mp.postKMSWallet)
	//mp.register("/endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}", http.MethodGet, mp.getKMSWallet)
	//mp.register("/endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}", http.MethodPut, mp.putKMSWallet)
	//mp.register("/endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}", http.MethodDelete, mp.deleteKMSWallet)

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

	// See cms_actions_base.go
	mp.register("/endpoint/{env}/{service}/rest/api/v1/actions", http.MethodPost, mp.postCMSAction)
	mp.register("/endpoint/{env}/{service}/rest/api/v1/actions/{action}", http.MethodGet, mp.getCMSAction)
	mp.register("/endpoint/{env}/{service}/rest/api/v1/actions/{action}", http.MethodPatch, mp.patchCMSAction)
	mp.register("/endpoint/{env}/{service}/rest/api/v1/actions/{action}", http.MethodDelete, mp.deleteCMSAction)

	// See ams_task.go
	mp.register("/endpoint/{env}/{service}/rest/api/v1/tasks/{task}", http.MethodGet, mp.getAMSTask)
	mp.register("/endpoint/{env}/{service}/rest/api/v1/tasks/{task}", http.MethodPut, mp.putAMSTask)
	mp.register("/endpoint/{env}/{service}/rest/api/v1/tasks/{task}/versions", http.MethodPost, mp.postAMSTaskVersion)
	mp.register("/endpoint/{env}/{service}/rest/api/v1/tasks/{task}", http.MethodPatch, mp.patchAMSTask)
	mp.register("/endpoint/{env}/{service}/rest/api/v1/tasks/{task}", http.MethodDelete, mp.deleteAMSTask)

	// See ams_policy.go
	mp.register("/endpoint/{env}/{service}/rest/api/v1/policies/{policy}", http.MethodGet, mp.getAMSPolicy)
	mp.register("/endpoint/{env}/{service}/rest/api/v1/policies/{policy}", http.MethodPut, mp.putAMSPolicy)
	mp.register("/endpoint/{env}/{service}/rest/api/v1/policies/{policy}/versions", http.MethodPost, mp.postAMSPolicyVersion)
	mp.register("/endpoint/{env}/{service}/rest/api/v1/policies/{policy}", http.MethodPatch, mp.patchAMSPolicy)
	mp.register("/endpoint/{env}/{service}/rest/api/v1/policies/{policy}", http.MethodDelete, mp.deleteAMSPolicy)

	// See ams_dmupsert.go
	mp.register("/endpoint/{env}/{service}/rest/api/v1/bulk/datamodel", http.MethodPut, mp.putAMSDMUpsert)

	// See ams_fflistener.go
	mp.register("/endpoint/{env}/{service}/rest/api/v1/listeners/firefly/{listener}", http.MethodGet, mp.getAMSFFListener)
	mp.register("/endpoint/{env}/{service}/rest/api/v1/listeners/firefly/{listener}", http.MethodPut, mp.putAMSFFListener)
	mp.register("/endpoint/{env}/{service}/rest/api/v1/listeners/firefly/{listener}", http.MethodDelete, mp.deleteAMSFFListener)

	// See ams_dmlistener.go
	mp.register("/endpoint/{env}/{service}/rest/api/v1/listeners/datamodel/{listener}", http.MethodGet, mp.getAMSDMListener)
	mp.register("/endpoint/{env}/{service}/rest/api/v1/listeners/datamodel/{listener}", http.MethodPut, mp.putAMSDMListener)
	mp.register("/endpoint/{env}/{service}/rest/api/v1/listeners/datamodel/{listener}", http.MethodDelete, mp.deleteAMSDMListener)

	// See ams_dmlistener.go
	mp.register("/endpoint/{env}/{service}/rest/api/v1/variable-sets/{variable-set}", http.MethodGet, mp.getAMSVariableSet)
	mp.register("/endpoint/{env}/{service}/rest/api/v1/variable-sets/{variable-set}", http.MethodPut, mp.putAMSVariableSet)
	mp.register("/endpoint/{env}/{service}/rest/api/v1/variable-sets/{variable-set}", http.MethodDelete, mp.deleteAMSVariableSet)

	// See ams_collection.go
	mp.register("/endpoint/{env}/{service}/rest/api/v1/collections/{collection}", http.MethodGet, mp.getAMSCollection)
	mp.register("/endpoint/{env}/{service}/rest/api/v1/collections/{collection}", http.MethodPut, mp.putAMSCollection)
	mp.register("/endpoint/{env}/{service}/rest/api/v1/collections/{collection}", http.MethodDelete, mp.deleteAMSCollection)

	// See firefly_registration.go
	mp.register("/endpoint/{env}/{service}/rest/api/v1/network/nodes/self", http.MethodPost, mp.postFireFlyRegistrationNode)
	mp.register("/endpoint/{env}/{service}/rest/api/v1/network/organizations/self", http.MethodPost, mp.postFireFlyRegistrationOrg)
	mp.register("/endpoint/{env}/{service}/rest/api/v1/status", http.MethodGet, mp.getFireFlyStatus)

	// See wms_wallet.go
	mp.register("/endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}", http.MethodGet, mp.getWMSWallet)
	mp.register("/endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}", http.MethodPut, mp.putWMSWallet)
	mp.register("/endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}", http.MethodDelete, mp.deleteWMSWallet)
	mp.register("/endpoint/{env}/{service}/rest/api/v1/wallets", http.MethodPost, mp.postWMSWallet)
	mp.register("/endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}", http.MethodPatch, mp.patchWMSWallet)

	// See wms_asset.go
	mp.register("/endpoint/{env}/{service}/rest/api/v1/assets/{asset}", http.MethodGet, mp.getWMSAsset)
	mp.register("/endpoint/{env}/{service}/rest/api/v1/assets/{asset}", http.MethodPut, mp.putWMSAsset)
	mp.register("/endpoint/{env}/{service}/rest/api/v1/assets/{asset}", http.MethodDelete, mp.deleteWMSAsset)
	mp.register("/endpoint/{env}/{service}/rest/api/v1/assets", http.MethodPost, mp.postWMSAsset)
	mp.register("/endpoint/{env}/{service}/rest/api/v1/assets/{asset}", http.MethodPatch, mp.patchWMSAsset)

	// See wms_asset_icon.go
	mp.register("/endpoint/{env}/{service}/rest/api/v1/assets/{asset}/icon", http.MethodGet, mp.getWMSAssetIcon)
	mp.register("/endpoint/{env}/{service}/rest/api/v1/assets/{asset}/icon", http.MethodPost, mp.postWMSAssetIcon)
	mp.register("/endpoint/{env}/{service}/rest/api/v1/assets/{asset}/icon", http.MethodDelete, mp.deleteWMSAssetIcon)

	// See wms_account.go
	mp.register("/endpoint/{env}/{service}/rest/api/v1/assets/{asset}/connect/{wallet}", http.MethodPost, mp.connectWMSAccount)
	mp.register("/endpoint/{env}/{service}/rest/api/v1/accounts/{account}", http.MethodGet, mp.getWMSAccount)
	mp.register("/endpoint/{env}/{service}/rest/api/v1/accounts/{account}", http.MethodDelete, mp.deleteWMSAccount)

	// See policy_identity.go
	mp.register("/endpoint/{env}/{service}/rest/api/v1/identities", http.MethodPost, mp.postPolicyIdentity)
	mp.register("/endpoint/{env}/{service}/rest/api/v1/identities/{identity}", http.MethodGet, mp.getPolicyIdentity)
	mp.register("/endpoint/{env}/{service}/rest/api/v1/identities/{identity}", http.MethodPut, mp.putPolicyIdentity)
	mp.register("/endpoint/{env}/{service}/rest/api/v1/identities/{identity}", http.MethodDelete, mp.deletePolicyIdentity)

	// See pms_identity_list.go
	mp.register("/endpoint/{env}/{service}/rest/api/v1/identity-lists/{identityList}", http.MethodPut, mp.putPMSIdentityList)
	mp.register("/endpoint/{env}/{service}/rest/api/v1/identity-lists/{identityList}", http.MethodGet, mp.getPMSIdentityList)
	mp.register("/endpoint/{env}/{service}/rest/api/v1/identity-lists/{identityList}", http.MethodPatch, mp.patchPMSIdentityList)
	mp.register("/endpoint/{env}/{service}/rest/api/v1/identity-lists/{identityList}", http.MethodDelete, mp.deletePMSIdentityList)
	mp.register("/endpoint/{env}/{service}/rest/api/v1/identity-lists/{identityList}/versions", http.MethodPost, mp.postPMSIdentityListVersion)

	// See group_test.go
	mp.register("/api/v1/groups", http.MethodPost, mp.postGroup)
	mp.register("/api/v1/groups/{group}", http.MethodGet, mp.getGroup)
	mp.register("/api/v1/groups/{group}", http.MethodPatch, mp.patchGroup)
	mp.register("/api/v1/groups/{group}", http.MethodDelete, mp.deleteGroup)

	// See stacks_test.go
	mp.register("/api/v1/environments/{env}/stacks", http.MethodPost, mp.postStacks)
	mp.register("/api/v1/environments/{env}/stacks/{stack}", http.MethodGet, mp.getStacks)
	mp.register("/api/v1/environments/{env}/stacks/{stack}", http.MethodPut, mp.putStacks)
	mp.register("/api/v1/environments/{env}/stacks/{stack}", http.MethodDelete, mp.deleteStacks)
	// See application_test.go
	mp.register("/api/v1/applications", http.MethodPost, mp.postApplication)
	mp.register("/api/v1/applications/{application}", http.MethodGet, mp.getApplication)
	mp.register("/api/v1/applications/{application}", http.MethodPatch, mp.patchApplication)
	mp.register("/api/v1/applications/{application}", http.MethodDelete, mp.deleteApplication)

	// See apikey_test.go
	mp.register("/api/v1/applications/{application}/api-keys", http.MethodPost, mp.postApiKey)
	mp.register("/api/v1/applications/{application}/api-keys/{api-key}", http.MethodGet, mp.getApiKey)
	mp.register("/api/v1/applications/{application}/api-keys/{api-key}", http.MethodDelete, mp.deleteApiKey)

	// See service_access_test.go
	mp.register("/api/v1/service-access/{service}/permissions", http.MethodPost, mp.postServiceAccessPermission)
	mp.register("/api/v1/service-access/{service}/permissions/{permission}", http.MethodGet, mp.getServiceAccessPermission)
	mp.register("/api/v1/service-access/{service}/permissions/{permission}", http.MethodDelete, mp.deleteServiceAccessPermission)

	// See stack_access_test.go
	mp.register("/api/v1/stack-access/{stack}/permissions", http.MethodPost, mp.postStackAccessPermission)
	mp.register("/api/v1/stack-access/{stack}/permissions/{permission}", http.MethodGet, mp.getStackAccessPermission)
	mp.register("/api/v1/stack-access/{stack}/permissions/{permission}", http.MethodDelete, mp.deleteStackAccessPermission)

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
		defer func() {
			mp.lock.Unlock()
			err := recover()
			assert.Nil(mp.t, err)
			if err != nil {
				resString := fmt.Sprintf("%s", err)
				res.Header().Set("Content-Length", strconv.Itoa(len(resString)))
				res.WriteHeader(500)
				res.Write([]byte(resString))
				mp.t.Logf(resString + ": " + string(debug.Stack()))
			}
		}()
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
	if strings.HasPrefix(req.Header.Get("Content-Type"), "application/x-yaml") {
		err := yaml.NewDecoder(req.Body).Decode(body)
		assert.NoError(mp.t, err)
	} else {
		err := json.NewDecoder(req.Body).Decode(body)
		assert.NoError(mp.t, err)
	}
}

func (mp *mockPlatform) peekBody(req *http.Request, body interface{}) []byte {
	rawBody, err := io.ReadAll(req.Body)
	assert.NoError(mp.t, err)
	err = json.Unmarshal(rawBody, &body)
	assert.NoError(mp.t, err)
	return rawBody
}
