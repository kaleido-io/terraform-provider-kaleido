package platform

import (
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/aidarkhanov/nanoid"
	"github.com/gorilla/mux"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"
)

var localPartyWithNamespaceConfig = `
resource "kaleido_platform_canton_party" "party1" {
	environment = "env1"
	service = "service1"
	name = "party1"
	synchronizer = "synchronizer1"
	type = "local"
	namespace = "namespace1"
}
`

var localPartyWithSigningKeysConfig = `
resource "kaleido_platform_canton_party" "party1" {
	environment = "env1"
	service = "service1"
	name = "party1"
	synchronizer = "synchronizer1"
	type = "local"
	signing_keys = ["key1"]
}
`

var localParty1Config = `
resource "kaleido_platform_canton_party" "party1" {
	environment = "env1"
	service = "service1"
	name = "party1"
	synchronizer = "synchronizer1"
	type = "local"
}
`

var localPartyRenamedConfig = `
resource "kaleido_platform_canton_party" "party1" {
	environment = "env1"
	service = "service1"
	name = "party1_renamed"
	synchronizer = "synchronizer1"
	type = "local"
}
`

func TestCantonPartyLocal(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"PUT /endpoint/{env}/{service}/node/parties",
			"GET /endpoint/{env}/{service}/node/parties/{party}",
			"GET /endpoint/{env}/{service}/node/parties/{party}",
		})
		mp.server.Close()
	}()

	party1Data := "kaleido_platform_canton_party.party1"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      providerConfig + localPartyWithNamespaceConfig,
				ExpectError: regexp.MustCompile(`Namespace and signing keys are only supported for external parties`),
			},
			{
				Config:      providerConfig + localPartyWithSigningKeysConfig,
				ExpectError: regexp.MustCompile(`Namespace and signing keys are only supported for external parties`),
			},
			{
				Config: providerConfig + localParty1Config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(party1Data, "environment", "env1"),
					resource.TestCheckResourceAttr(party1Data, "service", "service1"),
					resource.TestCheckResourceAttr(party1Data, "name", "party1"),
					resource.TestCheckResourceAttr(party1Data, "type", "local"),
					resource.TestCheckResourceAttrSet(party1Data, "party"),
					resource.TestCheckResourceAttrSet(party1Data, "identifier"),
				),
			},
			{
				Config:      providerConfig + localPartyRenamedConfig,
				ExpectError: regexp.MustCompile(`Immutable attribute cannot be updated`),
			},
		},
	})
}

var externalPartyWithoutNamespaceConfig = `
resource "kaleido_platform_canton_party" "party1" {
	environment = "env1"
	service = "service1"
	name = "party1"
	type = "external"
	synchronizer = "synchronizer1"
}
`

var externalParty1Config = `
resource "kaleido_platform_canton_party" "party1" {
	environment = "env1"
	service = "service1"
	name = "party1"
	type = "external"
	synchronizer = "synchronizer1"
	namespace = "namespace1"
}
`

var externalParty1ConfigWithSigningKeys = `
resource "kaleido_platform_canton_party" "party1" {
	environment = "env1"
	service = "service1"
	name = "party1"
	type = "external"
	synchronizer = "synchronizer1"
	namespace = "namespace1"
	signing_keys = ["key1", "key2"]
}
`

var externalPartyRenamedConfig = `
resource "kaleido_platform_canton_party" "party1" {
	environment = "env1"
	service = "service1"
	name = "party1_renamed"
	type = "external"
	synchronizer = "synchronizer1"
	namespace = "namespace1"
	signing_keys = ["key1", "key2"]
}
`

var externalPartyNamespaceChangedConfig = `
resource "kaleido_platform_canton_party" "party1" {
	environment = "env1"
	service = "service1"
	name = "party1"
	type = "external"
	synchronizer = "synchronizer1"
	namespace = "namespace2"
	signing_keys = ["key1", "key2"]
}
`

func TestCantonPartyExternal(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"PUT /endpoint/{env}/{service}/node/parties",
			"GET /endpoint/{env}/{service}/node/parties/{party}",
			"GET /endpoint/{env}/{service}/node/parties/{party}",
			"PATCH /endpoint/{env}/{service}/node/parties/{party}",
			"GET /endpoint/{env}/{service}/node/parties/{party}",
			"GET /endpoint/{env}/{service}/node/parties/{party}",
			"GET /endpoint/{env}/{service}/node/parties/{party}",
		})
		mp.server.Close()
	}()

	party1Data := "kaleido_platform_canton_party.party1"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      providerConfig + externalPartyWithoutNamespaceConfig,
				ExpectError: regexp.MustCompile(`Namespace key is required for external parties`),
			},
			{
				Config: providerConfig + externalParty1Config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(party1Data, "environment", "env1"),
					resource.TestCheckResourceAttr(party1Data, "service", "service1"),
					resource.TestCheckResourceAttr(party1Data, "name", "party1"),
					resource.TestCheckResourceAttr(party1Data, "type", "external"),
					resource.TestCheckResourceAttr(party1Data, "namespace", "namespace1"),
					resource.TestCheckResourceAttrSet(party1Data, "party"),
					resource.TestCheckResourceAttrSet(party1Data, "identifier"),
				),
			},
			{
				Config: providerConfig + externalParty1ConfigWithSigningKeys,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(party1Data, "environment", "env1"),
					resource.TestCheckResourceAttr(party1Data, "service", "service1"),
					resource.TestCheckResourceAttr(party1Data, "name", "party1"),
					resource.TestCheckResourceAttr(party1Data, "type", "external"),
					resource.TestCheckResourceAttrSet(party1Data, "party"),
					resource.TestCheckResourceAttrSet(party1Data, "identifier"),
					resource.TestCheckResourceAttr(party1Data, "signing_keys.0", "key1"),
					resource.TestCheckResourceAttr(party1Data, "signing_keys.1", "key2"),
					resource.TestCheckResourceAttr(party1Data, "synchronizers.0", "synchronizer1"),
				),
			},
			{
				Config:      providerConfig + externalPartyRenamedConfig,
				ExpectError: regexp.MustCompile(`Immutable attribute cannot be updated`),
			},
			{
				Config:      providerConfig + externalPartyNamespaceChangedConfig,
				ExpectError: regexp.MustCompile(`Immutable attribute cannot be updated`),
			},
		},
	})
}

func cantonPartyPrefix(req *http.Request) string {
	vars := mux.Vars(req)
	return vars["env"] + "/" + vars["service"] + "/"
}

func (mp *mockPlatform) lookupCantonParty(req *http.Request) *CantonPartyAPIModel {
	id := mux.Vars(req)["party"]
	prefix := cantonPartyPrefix(req)
	if obj := mp.cantonParties[prefix+id]; obj != nil {
		return obj
	}
	if name, _, ok := strings.Cut(id, "::"); ok {
		return mp.cantonParties[prefix+name]
	}
	return nil
}

func (mp *mockPlatform) storeCantonParty(req *http.Request, obj *CantonPartyAPIModel) {
	prefix := cantonPartyPrefix(req)
	mp.cantonParties[prefix+obj.Name] = obj
	if obj.Party != "" {
		mp.cantonParties[prefix+obj.Party] = obj
	}
}

func (mp *mockPlatform) putCantonParty(res http.ResponseWriter, req *http.Request) {
	var obj CantonPartyAPIModel
	mp.getBody(req, &obj)
	assert.NotEmpty(mp.t, obj.Name)
	obj.Identifier = nanoid.New()
	obj.Party = obj.Name + "::" + obj.Identifier
	if obj.Synchronizer != "" {
		obj.Synchronizers = []string{obj.Synchronizer}
	}
	mp.storeCantonParty(req, &obj)
	mp.respond(res, &obj, 200)
}

func (mp *mockPlatform) getCantonParty(res http.ResponseWriter, req *http.Request) {
	obj := mp.lookupCantonParty(req)
	if obj == nil {
		mp.respond(res, nil, 404)
	} else {
		mp.respond(res, obj, 200)
	}
}

func (mp *mockPlatform) patchCantonParty(res http.ResponseWriter, req *http.Request) {
	obj := mp.lookupCantonParty(req)
	assert.NotNil(mp.t, obj)
	var newObj CantonPartyAPIModel
	mp.getBody(req, &newObj)
	newObj.Name = obj.Name
	newObj.Party = obj.Party
	newObj.Identifier = obj.Identifier
	if newObj.PartyType == "" {
		newObj.PartyType = obj.PartyType
	}
	if newObj.Synchronizer != "" {
		newObj.Synchronizers = []string{newObj.Synchronizer}
	} else {
		newObj.Synchronizers = obj.Synchronizers
	}
	mp.storeCantonParty(req, &newObj)
	mp.respond(res, &newObj, 200)
}
