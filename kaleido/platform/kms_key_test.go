// Copyright © Kaleido, Inc. 2024-2026

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
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/aidarkhanov/nanoid"
	"github.com/gorilla/mux"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/assert"

	_ "embed"
)

var kms_keyStep1 = `
resource "kaleido_platform_kms_key" "kms_key1" {
    environment = "env1"
	service = "service1"
	wallet = "wallet1_id"
    name = "kms_key1"
	path = "some/path"
	attributes = {
		"attribute1" = "value1"
		"attribute2" = "value2"
	}
	public_identifier_types = ["address_ethereum"]
}
`

// Mutable update only — path/attributes are immutable (RequireRecreate).
var kms_keyStep2 = `
resource "kaleido_platform_kms_key" "kms_key1" {
    environment = "env1"
	service = "service1"
	wallet = "wallet1_id"
    name = "kms_key1_renamed"
	path = "some/path"
	attributes = {
		"attribute1" = "value1"
		"attribute2" = "value2"
	}
	public_identifier_types = ["address_ethereum"]
}
`

// Changing an immutable field must fail planning (no automatic replace).
var kms_keyStepImmutablePath = `
resource "kaleido_platform_kms_key" "kms_key1" {
    environment = "env1"
	service = "service1"
	wallet = "wallet1_id"
    name = "kms_key1_renamed"
	path = "other/path"
	attributes = {
		"attribute1" = "value1"
		"attribute2" = "value2"
	}
	public_identifier_types = ["address_ethereum"]
}
`

var kms_keyStepImmutablePublicIdentifiers = `
resource "kaleido_platform_kms_key" "kms_key1" {
    environment = "env1"
	service = "service1"
	wallet = "wallet1_id"
    name = "kms_key1_renamed"
	path = "some/path"
	attributes = {
		"attribute1" = "value1"
		"attribute2" = "value2"
	}
	public_identifier_types = ["address_ethereum", "address_ethereum_checksum"]
}
`

func TestKMSKey1(t *testing.T) {

	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			// Create (v1 default path)
			"GET /endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}",
			"PUT /endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}/keys",
			// Post-create refresh, pre-step2-plan refresh, Update's own pre-PATCH
			// fetch — each a single v2 call now, no separate wallet lookup needed
			"GET /endpoint/{env}/{service}/rest/api/v2/keys/{key}",
			"GET /endpoint/{env}/{service}/rest/api/v2/keys/{key}",
			"GET /endpoint/{env}/{service}/rest/api/v2/keys/{key}",
			"PATCH /endpoint/{env}/{service}/rest/api/v2/keys/{key}",
			// Post-update refresh, pre-step3-plan refresh, pre-step4-plan refresh
			// (both ExpectError steps refresh then fail during plan)
			"GET /endpoint/{env}/{service}/rest/api/v2/keys/{key}",
			"GET /endpoint/{env}/{service}/rest/api/v2/keys/{key}",
			"GET /endpoint/{env}/{service}/rest/api/v2/keys/{key}",
			// Destroy
			"DELETE /endpoint/{env}/{service}/rest/api/v2/keys/{key}",
			"GET /endpoint/{env}/{service}/rest/api/v2/keys/{key}",
		})
		mp.server.Close()
	}()

	// KMS requires an external ID->Name resolution before making API key calls
	mp.kmsWallets["env1/service1/wallet1_id"] = &KMSWalletAPIModel{Name: "wallet1"}

	kms_key1Resource := "kaleido_platform_kms_key.kms_key1"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + kms_keyStep1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(kms_key1Resource, "id"),
					resource.TestCheckResourceAttr(kms_key1Resource, "name", `kms_key1`),
					resource.TestCheckResourceAttr(kms_key1Resource, "path", `some/path`),
					resource.TestCheckResourceAttr(kms_key1Resource, "uri", `uri/for/kms_key1`),
					resource.TestCheckResourceAttrSet(kms_key1Resource, "address"),
					resource.TestCheckResourceAttr(kms_key1Resource, "attributes.attribute1", `value1`),
					resource.TestCheckResourceAttr(kms_key1Resource, "attributes.attribute2", `value2`),
					resource.TestCheckResourceAttr(kms_key1Resource, "public_identifier_types.0", `address_ethereum`),
				),
			},
			{
				Config: providerConfig + kms_keyStep2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(kms_key1Resource, "id"),
					resource.TestCheckResourceAttr(kms_key1Resource, "name", `kms_key1_renamed`),
					resource.TestCheckResourceAttr(kms_key1Resource, "path", `some/path`),
					resource.TestCheckResourceAttr(kms_key1Resource, "uri", `uri/for/kms_key1_renamed`),
					resource.TestCheckResourceAttrSet(kms_key1Resource, "address"),
					resource.TestCheckResourceAttr(kms_key1Resource, "attributes.attribute1", `value1`),
					resource.TestCheckResourceAttr(kms_key1Resource, "attributes.attribute2", `value2`),
					resource.TestCheckResourceAttr(kms_key1Resource, "public_identifier_types.0", `address_ethereum`),
					func(s *terraform.State) error {
						// Compare the final result on the mock-server side
						id := s.RootModule().Resources[kms_key1Resource].Primary.Attributes["id"]
						obj := mp.kmsKeys[fmt.Sprintf("env1/service1/wallet1/%s", id)]
						testJSONEqual(t, obj, fmt.Sprintf(`
						{
							"id": "%[1]s",
							"created": "%[2]s",
							"updated": "%[3]s",
							"name": "kms_key1_renamed",
							"path": "some/path",
							"address": "%[4]s",
							"uri": "uri/for/kms_key1_renamed",
							"attributes": {
								"attribute1": "value1",
								"attribute2": "value2"
							},
							"publicIdentifiers": [
								{"type": "address_ethereum", "value": "%[4]s"}
							]
						}
						`,
							// generated fields that vary per test run
							id,
							obj.Created.UTC().Format(time.RFC3339Nano),
							obj.Updated.UTC().Format(time.RFC3339Nano),
							obj.Address,
						))
						return nil
					},
				),
			},
			{
				Config:      providerConfig + kms_keyStepImmutablePath,
				ExpectError: regexp.MustCompile(`Immutable attribute cannot be updated`),
			},
			{
				Config:      providerConfig + kms_keyStepImmutablePublicIdentifiers,
				ExpectError: regexp.MustCompile(`Immutable attribute cannot be updated`),
			},
		},
	})
}

// putKMSKey mirrors this resource's v1 create behavior: it creates a real,
// persisted address_ethereum identifier (findable later via v2's
// fetchDetail) and nothing else, regardless of what publicIdentifierTypes
// requested.
func (mp *mockPlatform) putKMSKey(res http.ResponseWriter, req *http.Request) {
	var obj KMSKeyAPIModel
	mp.getBody(req, &obj)
	obj.ID = nanoid.New()
	now := time.Now().UTC()
	obj.Created = &now
	obj.Updated = &now
	obj.Address = nanoid.New()
	if obj.URI == "" {
		obj.URI = "uri/for/" + obj.Name
	}
	obj.PublicIdentifierTypes = nil
	obj.PublicIdentifiers = []PublicIdentifierWire{{Type: "address_ethereum", Value: obj.Address}}
	// Mirror the real API: merge the wallet's default_key_attributes into the
	// key's attributes. Per-key values take precedence; every default the key
	// didn't explicitly set is added.
	walletName := mux.Vars(req)["wallet"]
	envSvcPrefix := mux.Vars(req)["env"] + "/" + mux.Vars(req)["service"] + "/"
	for k, w := range mp.kmsWallets {
		if !strings.HasPrefix(k, envSvcPrefix) || w.Name != walletName {
			continue
		}
		if len(w.DefaultKeyAttributes) > 0 {
			if obj.Attributes == nil {
				obj.Attributes = map[string]string{}
			}
			for ak, av := range w.DefaultKeyAttributes {
				if _, set := obj.Attributes[ak]; !set {
					obj.Attributes[ak] = av
				}
			}
		}
		break
	}
	walletKey := mux.Vars(req)["env"] + "/" + mux.Vars(req)["service"] + "/" + mux.Vars(req)["wallet"] + "/" + obj.ID
	idKey := mux.Vars(req)["env"] + "/" + mux.Vars(req)["service"] + "/" + obj.ID
	mp.kmsKeys[walletKey] = &obj
	mp.kmsKeysByID[idKey] = &obj
	mp.respond(res, &obj, 201)
}

// postKMSKeyV2 mirrors the real v2 POST /keys behavior: unlike putKMSKey, it
// honours publicIdentifierTypes and, when returnPublicIdentifiers is set,
// returns them inline. Its response has no address/path — v2 calls the
// derivation path keyHandle and has no address field, only publicIdentifiers
// entries.
func (mp *mockPlatform) postKMSKeyV2(res http.ResponseWriter, req *http.Request) {
	var obj KMSKeyAPIModel
	mp.getBody(req, &obj)
	obj.ID = nanoid.New()
	now := time.Now().UTC()
	obj.Created = &now
	obj.Updated = &now
	if obj.URI == "" {
		obj.URI = "uri/for/" + obj.Name
	}
	obj.KeyHandle = "m/44'/60'/0'/0/0"

	// Mirror the real API: merge the wallet's default_key_attributes into the
	// key's attributes. Per-key values take precedence; every default the key
	// didn't explicitly set is added.
	walletName := obj.KeystoreName
	envSvcPrefix := mux.Vars(req)["env"] + "/" + mux.Vars(req)["service"] + "/"
	for k, w := range mp.kmsWallets {
		if !strings.HasPrefix(k, envSvcPrefix) || w.Name != walletName {
			continue
		}
		if len(w.DefaultKeyAttributes) > 0 {
			if obj.Attributes == nil {
				obj.Attributes = map[string]string{}
			}
			for ak, av := range w.DefaultKeyAttributes {
				if _, set := obj.Attributes[ak]; !set {
					obj.Attributes[ak] = av
				}
			}
		}
		break
	}

	if obj.ReturnPublicIdentifiers {
		for _, piType := range obj.PublicIdentifierTypes {
			obj.PublicIdentifiers = append(obj.PublicIdentifiers, PublicIdentifierWire{Type: piType, Value: "0x" + nanoid.New()})
		}
	}

	walletKey := envSvcPrefix + walletName + "/" + obj.ID
	idKey := mux.Vars(req)["env"] + "/" + mux.Vars(req)["service"] + "/" + obj.ID
	mp.kmsKeys[walletKey] = &obj
	mp.kmsKeysByID[idKey] = &obj
	mp.respond(res, &obj, 201)
}

// getKMSKeyByIDV2 mirrors v2 GET /keys/{id}: never returns address (v2 has no
// such field, regardless of how the key was created), and only includes
// publicIdentifiers when fetchDetail=true is requested, same as the real API.
func (mp *mockPlatform) getKMSKeyByIDV2(res http.ResponseWriter, req *http.Request) {
	idKey := mux.Vars(req)["env"] + "/" + mux.Vars(req)["service"] + "/" + mux.Vars(req)["key"]
	obj := mp.kmsKeysByID[idKey]
	if obj == nil {
		mp.respond(res, nil, 404)
		return
	}
	out := *obj
	out.Address = ""
	if req.URL.Query().Get("fetchDetail") != "true" {
		out.PublicIdentifiers = nil
	}
	mp.respond(res, &out, 200)
}

// patchKMSKeyByIDV2 mirrors v2 PATCH /keys/{id}: name/labels only (no
// attributes/uri input, unlike v1's PATCH), and its response carries no
// address/publicIdentifiers detail — PATCH has no fetchDetail option at all.
func (mp *mockPlatform) patchKMSKeyByIDV2(res http.ResponseWriter, req *http.Request) {
	idKey := mux.Vars(req)["env"] + "/" + mux.Vars(req)["service"] + "/" + mux.Vars(req)["key"]
	obj := mp.kmsKeysByID[idKey]
	assert.NotNil(mp.t, obj)
	var in KMSKeyAPIModel
	mp.getBody(req, &in)
	assert.NotEmpty(mp.t, in.Name, "v2 PATCH must send name")

	updated := *obj
	updated.Name = in.Name
	updated.URI = "uri/for/" + in.Name // server regenerates URI on rename
	now := time.Now().UTC()
	updated.Updated = &now
	mp.kmsKeysByID[idKey] = &updated
	for k, v := range mp.kmsKeys {
		if v.ID == obj.ID {
			mp.kmsKeys[k] = &updated
		}
	}

	out := updated
	out.Address = ""
	out.PublicIdentifiers = nil
	mp.respond(res, &out, 200)
}

func (mp *mockPlatform) deleteKMSKeyByIDV2(res http.ResponseWriter, req *http.Request) {
	idKey := mux.Vars(req)["env"] + "/" + mux.Vars(req)["service"] + "/" + mux.Vars(req)["key"]
	obj := mp.kmsKeysByID[idKey]
	assert.NotNil(mp.t, obj)
	delete(mp.kmsKeysByID, idKey)
	for k, v := range mp.kmsKeys {
		if v.ID == obj.ID {
			delete(mp.kmsKeys, k)
		}
	}
	mp.respond(res, nil, 204)
}

var kms_keyFolderStep = `
resource "kaleido_platform_kms_key" "kms_key_folder" {
    environment = "env1"
	service = "service1"
	wallet = "wallet1_id"
    name = "folder_key"
	folder_path = "keys/folder"
	public_identifier_types = ["address_ethereum"]
}
`

var kms_keyFolderStepRename = `
resource "kaleido_platform_kms_key" "kms_key_folder" {
    environment = "env1"
	service = "service1"
	wallet = "wallet1_id"
    name = "folder_key_renamed"
	folder_path = "keys/folder"
	public_identifier_types = ["address_ethereum"]
}
`

func TestKMSKeyFolderUpdateAndDelete(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			// Create (v1 default path)
			"GET /endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}",
			"PUT /endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}/keys",
			// Post-create refresh, pre-update-plan refresh, Update's own pre-PATCH
			// fetch — v2's by-ID route is folder-agnostic, so unlike the old v1
			// wallet-scoped route there's no 404-preserve special case needed here.
			"GET /endpoint/{env}/{service}/rest/api/v2/keys/{key}",
			"GET /endpoint/{env}/{service}/rest/api/v2/keys/{key}",
			"GET /endpoint/{env}/{service}/rest/api/v2/keys/{key}",
			"PATCH /endpoint/{env}/{service}/rest/api/v2/keys/{key}",
			// Post-update refresh
			"GET /endpoint/{env}/{service}/rest/api/v2/keys/{key}",
			// Destroy
			"DELETE /endpoint/{env}/{service}/rest/api/v2/keys/{key}",
			"GET /endpoint/{env}/{service}/rest/api/v2/keys/{key}",
		})
		mp.server.Close()
	}()

	mp.kmsWallets["env1/service1/wallet1_id"] = &KMSWalletAPIModel{Name: "wallet1"}

	resourceName := "kaleido_platform_kms_key.kms_key_folder"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + kms_keyFolderStep,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "folder_path", "keys/folder"),
					resource.TestCheckResourceAttr(resourceName, "name", "folder_key"),
				),
			},
			{
				Config: providerConfig + kms_keyFolderStepRename,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "folder_path", "keys/folder"),
					resource.TestCheckResourceAttr(resourceName, "name", "folder_key_renamed"),
					func(s *terraform.State) error {
						id := s.RootModule().Resources[resourceName].Primary.Attributes["id"]
						obj := mp.kmsKeysByID[fmt.Sprintf("env1/service1/%s", id)]
						assert.NotNil(t, obj)
						assert.Equal(t, "folder_key_renamed", obj.Name)
						return nil
					},
				),
			},
		},
	})

	assert.Empty(t, mp.kmsKeysByID, "folder key should be deleted via global /keys/{id} path")
}

// kms_keyInheritedAttributes exercises a wallet that has default_key_attributes:
// creating a key without attributes should inherit them from the wallet, and the
// provider must not error with "inconsistent result after apply" and must not
// show a drift on the next plan.
var kms_keyInheritedAttributesStep = `
resource "kaleido_platform_kms_key" "kms_key_inherited" {
    environment = "env1"
	service = "service1"
	wallet = "wallet_defaults_id"
    name = "inherited_key"
}
`

func TestKMSKeyInheritsWalletDefaultAttributes(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer mp.server.Close()

	mp.kmsWallets["env1/service1/wallet_defaults_id"] = &KMSWalletAPIModel{
		Name: "wallet_defaults",
		DefaultKeyAttributes: map[string]string{
			"attr1": "value1",
		},
	}

	resourceName := "kaleido_platform_kms_key.kms_key_inherited"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				// First apply: wallet defaults get injected server-side; state must
				// accept the returned map even though config didn't set attributes.
				Config: providerConfig + kms_keyInheritedAttributesStep,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "name", "inherited_key"),
					resource.TestCheckResourceAttr(resourceName, "attributes.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "attributes.attr1", "value1"),
				),
			},
			{
				// Re-apply the same config: with UseStateForUnknown, the plan reuses
				// the stored value and produces no diff.
				Config:   providerConfig + kms_keyInheritedAttributesStep,
				PlanOnly: true,
			},
		},
	})
}

// kms_keyExplicitAttributes covers the case where the user sets attributes in
// config directly: the sent value must round-trip through create + refresh with
// no drift on the next plan.
var kms_keyExplicitAttributesStep = `
resource "kaleido_platform_kms_key" "kms_key_explicit" {
    environment = "env1"
	service = "service1"
	wallet = "wallet1_id"
    name = "explicit_key"
	attributes = {
		"custom" = "yes"
	}
}
`

func TestKMSKeyExplicitAttributesRoundTrip(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer mp.server.Close()

	mp.kmsWallets["env1/service1/wallet1_id"] = &KMSWalletAPIModel{Name: "wallet1"}

	resourceName := "kaleido_platform_kms_key.kms_key_explicit"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + kms_keyExplicitAttributesStep,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "attributes.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "attributes.custom", "yes"),
				),
			},
			{
				Config:   providerConfig + kms_keyExplicitAttributesStep,
				PlanOnly: true,
			},
		},
	})
}

// kms_keyMergedAttributes covers the merge path: the wallet has defaults and
// the key sets its own attributes. The state must contain the union — the
// per-key value wins on conflicts and the extra wallet-default entries are
// included.
//
// Note on re-plan drift: if the user's config sets `attributes` and the wallet
// contributes additional defaults, the state has more entries than the config.
// The RequireRecreateMap plan modifier then flags that as an immutable change
// on the NEXT plan and errors telling the user to recreate. The clean-round-
// trip options for users are (a) list every merged entry in config, or
// (b) don't set attributes in config at all and let the wallet defaults flow
// through (covered by TestKMSKeyInheritsWalletDefaultAttributes). This test
// therefore only asserts state after create.
var kms_keyMergedAttributesStep = `
resource "kaleido_platform_kms_key" "kms_key_merged" {
    environment = "env1"
	service = "service1"
	wallet = "wallet_defaults_id"
    name = "merged_key"
	attributes = {
		"custom" = "yes"
		"attr1"  = "override"
	}
}
`

func TestKMSKeyMergesWalletDefaultAttributes(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer mp.server.Close()

	// Wallet defaults: attr1=default1, shared=fromWallet.
	// Key config: custom=yes, attr1=override (overrides wallet default).
	// Server-side merge: attr1=override, custom=yes, shared=fromWallet.
	// Terraform state (after the plan-preservation logic): only the two config
	// entries — extras from wallet defaults are hidden from state so that
	// plan-consistency holds and re-plans are clean.
	mp.kmsWallets["env1/service1/wallet_defaults_id"] = &KMSWalletAPIModel{
		Name: "wallet_defaults",
		DefaultKeyAttributes: map[string]string{
			"attr1":  "default1",
			"shared": "fromWallet",
		},
	}

	resourceName := "kaleido_platform_kms_key.kms_key_merged"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + kms_keyMergedAttributesStep,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					// State mirrors config: 2 entries, key's override wins.
					resource.TestCheckResourceAttr(resourceName, "attributes.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "attributes.attr1", "override"),
					resource.TestCheckResourceAttr(resourceName, "attributes.custom", "yes"),
					resource.TestCheckNoResourceAttr(resourceName, "attributes.shared"),
					func(s *terraform.State) error {
						id := s.RootModule().Resources[resourceName].Primary.Attributes["id"]
						obj := mp.kmsKeysByID[fmt.Sprintf("env1/service1/%s", id)]
						assert.NotNil(t, obj)
						assert.Equal(t, map[string]string{
							"attr1":  "override",
							"custom": "yes",
							"shared": "fromWallet",
						}, obj.Attributes, "server-side attributes should be the merge of config + wallet defaults, config winning on conflicts")
						return nil
					},
				),
			},
			{
				// Round-trip: re-apply the same config; the wallet-merged extras must
				// not leak back into state and re-plan must be a no-op.
				Config:   providerConfig + kms_keyMergedAttributesStep,
				PlanOnly: true,
			},
		},
	})
}

// kms_keyV2Step opts into v2 create (via spec) with two identifier types — v1
// PUT can only ever produce address_ethereum, so this only passes if Create
// actually took the v2 path.
var kms_keyV2Step = `
resource "kaleido_platform_kms_key" "kms_key_v2" {
    environment = "env1"
	service = "service1"
	wallet = "wallet1_id"
    name = "kms_key_v2"
	spec = "secp256k1"
	public_identifier_types = ["address_ethereum", "address_ethereum_checksum"]
}
`

func TestKMSKeyV2CreateHonoursAllIdentifierTypes(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			// Create: no adopt-check on the v2 path, straight to POST
			"GET /endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}",
			"POST /endpoint/{env}/{service}/rest/api/v2/keys",
			// Post-apply refresh: Read always uses the v2 global by-ID route,
			// regardless of which API created the key — no wallet lookup needed.
			"GET /endpoint/{env}/{service}/rest/api/v2/keys/{key}",
			// Destroy
			"DELETE /endpoint/{env}/{service}/rest/api/v2/keys/{key}",
			"GET /endpoint/{env}/{service}/rest/api/v2/keys/{key}",
		})
		mp.server.Close()
	}()

	mp.kmsWallets["env1/service1/wallet1_id"] = &KMSWalletAPIModel{Name: "wallet1"}

	resourceName := "kaleido_platform_kms_key.kms_key_v2"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + kms_keyV2Step,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "address"),
					resource.TestCheckResourceAttr(resourceName, "spec", "secp256k1"),
					resource.TestCheckResourceAttr(resourceName, "public_identifier_types.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "public_identifier_types.0", "address_ethereum"),
					resource.TestCheckResourceAttr(resourceName, "public_identifier_types.1", "address_ethereum_checksum"),
					resource.TestCheckResourceAttr(resourceName, "public_identifiers.%", "2"),
					resource.TestCheckResourceAttrSet(resourceName, "public_identifiers.address_ethereum"),
					resource.TestCheckResourceAttrSet(resourceName, "public_identifiers.address_ethereum_checksum"),
					func(s *terraform.State) error {
						// Compare against the mock's server-side record: this is what
						// actually got sent to v2, not just what's restored into state.
						id := s.RootModule().Resources[resourceName].Primary.Attributes["id"]
						obj := mp.kmsKeysByID[fmt.Sprintf("env1/service1/%s", id)]
						assert.NotNil(t, obj)
						assert.Equal(t, []string{"address_ethereum", "address_ethereum_checksum"}, obj.PublicIdentifierTypes,
							"v2 must receive and honour every requested identifier type, unlike v1 PUT")
						assert.Len(t, obj.PublicIdentifiers, 2, "returnPublicIdentifiers should have produced one entry per requested type")
						return nil
					},
				),
			},
		},
	})
}

// kms_keyV2PathConflictStep sets both spec (a v2 trigger) and path (a v1-only
// field with no v2 equivalent) — Create must reject this combination rather
// than silently drop path.
var kms_keyV2PathConflictStep = `
resource "kaleido_platform_kms_key" "kms_key_v2_conflict" {
    environment = "env1"
	service = "service1"
	wallet = "wallet1_id"
    name = "kms_key_v2_conflict"
	spec = "secp256k1"
	path = "m/44'/60'/0'/0/1"
}
`

func TestKMSKeyV2RejectsPath(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			// Create fails validation after resolving the wallet name, before any create call
			"GET /endpoint/{env}/{service}/rest/api/v1/wallets/{wallet}",
		})
		mp.server.Close()
	}()

	mp.kmsWallets["env1/service1/wallet1_id"] = &KMSWalletAPIModel{Name: "wallet1"}

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      providerConfig + kms_keyV2PathConflictStep,
				ExpectError: regexp.MustCompile(`path is not supported with keystore_name/spec`),
			},
		},
	})
}
