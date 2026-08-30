// Copyright © Kaleido, Inc. 2026

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
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/aidarkhanov/nanoid"
	"github.com/gorilla/mux"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/assert"

	_ "embed"
)

var pms_identity_step1 = `
resource "kaleido_platform_pms_identity" "test_identity" {
  environment = "test-env"
  service = "test-service"
  name = "test-identity"
  description = "Test identity for policy management"
  controller = "u:12345abcde"
  verification_method = [
    {
      name = "key-1"
      type = "Multikey"
      public_key_multibase = "z6DtLLbUmR3TQFTLcbTLnfDMHUKPQwLXDBpS7mCWvzGqEHwn"
      controller = "did:kaleido:controller"
      key_uri = "kld:///keystore/ks1/key/key-1"
    }
  ]
}
`

func TestPMSIdentity1(t *testing.T) {
	mp, providerConfig := testSetup(t)
	mp.expectIdentityController = "u:12345abcde"
	mp.expectKeyURI = "kld:///keystore/ks1/key/key-1"
	defer func() {
		mp.checkClearCalls([]string{
			"POST /endpoint/{env}/{service}/rest/api/v2/identities",
			"GET /endpoint/{env}/{service}/rest/api/v2/identities/{identity}",
			"GET /endpoint/{env}/{service}/rest/api/v2/identities/{identity}",
			"DELETE /endpoint/{env}/{service}/rest/api/v2/identities/{identity}",
		})
		mp.server.Close()
	}()

	pms_identity_resource := "kaleido_platform_pms_identity.test_identity"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + pms_identity_step1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(pms_identity_resource, "id"),
					resource.TestCheckResourceAttr(pms_identity_resource, "environment", "test-env"),
					resource.TestCheckResourceAttr(pms_identity_resource, "service", "test-service"),
					resource.TestCheckResourceAttr(pms_identity_resource, "name", "test-identity"),
					resource.TestCheckResourceAttr(pms_identity_resource, "description", "Test identity for policy management"),
					resource.TestCheckResourceAttr(pms_identity_resource, "verification_method.0.name", "key-1"),
					resource.TestCheckResourceAttr(pms_identity_resource, "verification_method.0.type", "Multikey"),
					resource.TestCheckResourceAttr(pms_identity_resource, "verification_method.0.public_key_multibase", "z6DtLLbUmR3TQFTLcbTLnfDMHUKPQwLXDBpS7mCWvzGqEHwn"),
					resource.TestCheckResourceAttr(pms_identity_resource, "verification_method.0.controller", "did:kaleido:controller"),
					resource.TestCheckResourceAttr(pms_identity_resource, "verification_method.0.key_uri", "kld:///keystore/ks1/key/key-1"),
					resource.TestCheckResourceAttr(pms_identity_resource, "controller", "u:12345abcde"),
				),
			},
		},
	})
}

var pms_identity_jwk = `
resource "kaleido_platform_pms_identity" "jwk_identity" {
  environment = "test-env"
  service = "test-service"
  name = "jwk-identity"
  verification_method = [
    {
      name = "jwk-key"
      type = "JsonWebKey"
      public_key_jwk_json = jsonencode({
        kty = "EC"
        crv = "secp256k1"
        x   = "test-x"
        y   = "test-y"
      })
    }
  ]
}
`

func TestPMSIdentityVerificationMethodJWK(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"POST /endpoint/{env}/{service}/rest/api/v2/identities",
			"GET /endpoint/{env}/{service}/rest/api/v2/identities/{identity}",
			"GET /endpoint/{env}/{service}/rest/api/v2/identities/{identity}",
			"GET /endpoint/{env}/{service}/rest/api/v2/identities/{identity}",
			"DELETE /endpoint/{env}/{service}/rest/api/v2/identities/{identity}",
		})
		mp.server.Close()
	}()

	pms_identity_resource := "kaleido_platform_pms_identity.jwk_identity"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + pms_identity_jwk,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(pms_identity_resource, "verification_method.0.name", "jwk-key"),
					resource.TestCheckResourceAttr(pms_identity_resource, "verification_method.0.type", "JsonWebKey"),
					func(s *terraform.State) error {
						// The JWK must have been stored server-side as a structured object,
						// not as a string containing JSON
						id := s.RootModule().Resources[pms_identity_resource].Primary.Attributes["id"]
						obj := mp.policyIdentities[id]
						assert.Len(t, obj.VerificationMethods, 1)
						assert.JSONEq(t, `{
							"kty": "EC",
							"crv": "secp256k1",
							"x":   "test-x",
							"y":   "test-y"
						}`, string(obj.VerificationMethods[0].PublicKeyJwk))
						return nil
					},
				),
			},
			{
				// Re-planning against the value the API returns must produce no diff
				Config:             providerConfig + pms_identity_jwk,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

var pms_identity_jwk_key_order = `
resource "kaleido_platform_pms_identity" "ordered_identity" {
  environment = "test-env"
  service = "test-service"
  name = "ordered-identity"
  verification_method = [
    {
      name = "key-1"
      type = "JsonWebKey"
      # kty first, which is NOT the order the server returns the keys in
      public_key_jwk_json = "{\"kty\":\"EC\",\"crv\":\"secp256k1\",\"x\":\"test-x\",\"y\":\"test-y\"}"
    }
  ]
}
`

func TestPMSIdentityJWKKeyOrderPreserved(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"POST /endpoint/{env}/{service}/rest/api/v2/identities",
			"GET /endpoint/{env}/{service}/rest/api/v2/identities/{identity}",
			"GET /endpoint/{env}/{service}/rest/api/v2/identities/{identity}",
			"GET /endpoint/{env}/{service}/rest/api/v2/identities/{identity}",
			"DELETE /endpoint/{env}/{service}/rest/api/v2/identities/{identity}",
		})
		mp.server.Close()
	}()

	pms_identity_resource := "kaleido_platform_pms_identity.ordered_identity"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + pms_identity_jwk_key_order,
				Check: resource.ComposeAggregateTestCheckFunc(
					// The configured formatting must survive, not the server's key order
					resource.TestCheckResourceAttr(pms_identity_resource, "verification_method.0.public_key_jwk_json",
						`{"kty":"EC","crv":"secp256k1","x":"test-x","y":"test-y"}`),
				),
			},
			{
				// A re-ordered JWK from the API must not produce a perpetual diff
				Config:             providerConfig + pms_identity_jwk_key_order,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// assertPublicKeyJwksAreObjects fails the test if a verification method JWK
// reaches the API as a JSON string rather than as a JSON object
func (mp *mockPlatform) assertPublicKeyJwksAreObjects(rawBody []byte) {
	var wire struct {
		Controller          string `json:"controller"`
		VerificationMethods []struct {
			PublicKeyJwk json.RawMessage `json:"publicKeyJwk"`
			KeyURI       string          `json:"keyUri"`
		} `json:"verificationMethods"`
	}
	err := json.Unmarshal(rawBody, &wire)
	assert.NoError(mp.t, err)
	if mp.expectIdentityController != "" {
		assert.Equal(mp.t, mp.expectIdentityController, wire.Controller,
			"controller must be sent at the top level of the identity, got: %s", rawBody)
	}
	if mp.expectKeyURI != "" {
		assert.Equal(mp.t, mp.expectKeyURI, wire.VerificationMethods[0].KeyURI,
			"keyUri must be sent on the verification method, got: %s", rawBody)
	}
	for _, vm := range wire.VerificationMethods {
		if len(vm.PublicKeyJwk) == 0 {
			continue
		}
		assert.Equal(mp.t, uint8('{'), vm.PublicKeyJwk[0],
			"publicKeyJwk must be sent as a JSON object, got: %s", vm.PublicKeyJwk)
	}
}

func (mp *mockPlatform) postPolicyIdentity(res http.ResponseWriter, req *http.Request) {
	var obj PolicyIdentityAPIModel
	rawBody := mp.peekBody(req, &obj)
	mp.assertPublicKeyJwksAreObjects(rawBody)
	obj.ID = nanoid.New()
	now := time.Now().UTC()
	obj.Created = &now
	obj.Updated = &now
	mp.policyIdentities[obj.ID] = &obj
	// The real API inserts verification methods separately and does not return them
	// on create - only a subsequent fetchDetails read carries them
	// The server re-serializes the JWK, which sorts its keys
	for i, vm := range obj.VerificationMethods {
		if len(vm.PublicKeyJwk) == 0 {
			continue
		}
		var reserialized map[string]interface{}
		assert.NoError(mp.t, json.Unmarshal(vm.PublicKeyJwk, &reserialized))
		sorted, err := json.Marshal(reserialized)
		assert.NoError(mp.t, err)
		obj.VerificationMethods[i].PublicKeyJwk = sorted
	}
	created := obj
	created.VerificationMethods = nil
	created.NotificationMethods = nil
	mp.respond(res, &created, 201)
}

func (mp *mockPlatform) getPolicyIdentity(res http.ResponseWriter, req *http.Request) {
	obj := mp.policyIdentities[mux.Vars(req)["identity"]]
	if obj == nil {
		mp.respond(res, nil, 404)
	} else {
		mp.respond(res, obj, 200)
	}
}

func (mp *mockPlatform) deletePolicyIdentity(res http.ResponseWriter, req *http.Request) {
	obj := mp.policyIdentities[mux.Vars(req)["identity"]]
	assert.NotNil(mp.t, obj)
	delete(mp.policyIdentities, mux.Vars(req)["identity"])
	mp.respond(res, nil, 204)
}
