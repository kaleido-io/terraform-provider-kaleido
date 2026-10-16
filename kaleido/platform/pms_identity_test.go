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
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/aidarkhanov/nanoid"
	"github.com/gorilla/mux"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
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

var pms_identity_ethereum_address = `
resource "kaleido_platform_pms_identity" "eth_identity" {
  environment = "test-env"
  service = "test-service"
  name = "eth-identity"
  verification_method = [
    {
      name = "primary-signing-key"
      type = "EthereumAddress"
      ethereum_address = "0x1234567890AbcdEF1234567890aBcdef12345678"
      key_uri = "kld:///keystore/ks1/key/signer"
    }
  ]
}
`

func TestPMSIdentityVerificationMethodEthereumAddress(t *testing.T) {
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

	pms_identity_resource := "kaleido_platform_pms_identity.eth_identity"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + pms_identity_ethereum_address,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(pms_identity_resource, "verification_method.0.type", "EthereumAddress"),
					resource.TestCheckResourceAttr(pms_identity_resource, "verification_method.0.ethereum_address", "0x1234567890AbcdEF1234567890aBcdef12345678"),
					resource.TestCheckResourceAttr(pms_identity_resource, "verification_method.0.key_uri", "kld:///keystore/ks1/key/signer"),
					resource.TestCheckNoResourceAttr(pms_identity_resource, "verification_method.0.public_key_jwk_json"),
					resource.TestCheckNoResourceAttr(pms_identity_resource, "verification_method.0.public_key_multibase"),
				),
			},
			{
				Config:             providerConfig + pms_identity_ethereum_address,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// pmsIdentityStateV0 is the state of an identity as written by a provider release built
// against the v1 Policy Manager API (schema version 0).
const pmsIdentityStateV0 = `{
	"id": "id:12345abcde",
	"environment": "test-env",
	"service": "test-service",
	"name": "treasury-signer",
	"description": "Treasury signer",
	"owner": "ap:owner12345",
	"preferred_assertion_method": "primary",
	"assertion_method": [
		{
			"id": "vm:eth12345",
			"identity_id": "id:12345abcde",
			"name": "primary",
			"type": "EthereumAddress",
			"signing_method": "offline",
			"verification_material": "0x12F62772C4652280d06E64CfBC9033d409559aD4",
			"created": "2026-01-02T03:04:05Z",
			"expires": null,
			"revoked": null
		},
		{
			"id": "vm:multi12345",
			"identity_id": "id:12345abcde",
			"name": "backup",
			"type": "Multikey",
			"signing_method": null,
			"verification_material": "zQ3shokFTS3brHcDQrn82RUDfCZESWL1ZdCEJwekUDPQiYBme",
			"created": "2026-01-02T03:04:05Z",
			"expires": null,
			"revoked": null
		}
	],
	"notification_method": [
		{ "name": "approvals", "type": "workflow", "value_json": "{\"workflow\":\"flw:1\"}" }
	]
}`

// State written against the v1 API must upgrade to the v2 attributes, as terraform does
// before the first refresh after a provider upgrade.
func TestPMSIdentityUpgradeStateV0(t *testing.T) {
	ctx := context.Background()
	server, err := testAccProviders["kaleido"]()
	assert.NoError(t, err)

	upgraded, err := server.UpgradeResourceState(ctx, &tfprotov6.UpgradeResourceStateRequest{
		TypeName: "kaleido_platform_pms_identity",
		Version:  0,
		RawState: &tfprotov6.RawState{JSON: []byte(pmsIdentityStateV0)},
	})
	assert.NoError(t, err)
	assert.Empty(t, upgraded.Diagnostics)

	var schemaResp fwresource.SchemaResponse
	(&policyIdentityResource{}).Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	raw, err := upgraded.UpgradedState.Unmarshal(schemaResp.Schema.Type().TerraformType(ctx))
	assert.NoError(t, err)
	var data PolicyIdentityResourceModel
	assert.False(t, tfsdk.State{Schema: schemaResp.Schema, Raw: raw}.Get(ctx, &data).HasError())

	assert.Equal(t, "id:12345abcde", data.ID.ValueString(), "the identity keeps its ID, so nothing is replaced")
	assert.Equal(t, "Treasury signer", data.Description.ValueString())
	assert.Equal(t, "ap:owner12345", data.Controller.ValueString(), "owner is the v2 controller")

	methods := data.VerificationMethods.Elements()
	if assert.Len(t, methods, 2) {
		eth := methods[0].(types.Object).Attributes()
		assert.Equal(t, "vm:eth12345", eth["id"].(types.String).ValueString())
		assert.Equal(t, "EthereumAddress", eth["type"].(types.String).ValueString())
		assert.Equal(t, "0x12F62772C4652280d06E64CfBC9033d409559aD4", eth["ethereum_address"].(types.String).ValueString(),
			"the verification material of an EthereumAddress method is its ethereum_address")
		assert.Equal(t, "2026-01-02T03:04:05Z", eth["created"].(types.String).ValueString())

		multi := methods[1].(types.Object).Attributes()
		assert.Equal(t, "Multikey", multi["type"].(types.String).ValueString())
		assert.True(t, multi["ethereum_address"].IsNull(), "only an EthereumAddress method carries an ethereum_address")
	}

	notifications := data.NotificationMethods.Elements()
	if assert.Len(t, notifications, 1) {
		nm := notifications[0].(types.Object).Attributes()
		assert.Equal(t, "approvals", nm["name"].(types.String).ValueString())
		assert.Equal(t, `{"workflow":"flw:1"}`, nm["value_json"].(jsonStringValue).ValueString())
		assert.True(t, nm["id"].IsNull(), "the refresh after the upgrade supplies the ID")
	}
}

// pmsIdentityStateV0FromV2 is the state of an identity written against the v2 API by a
// provider build from before the schema was versioned: it also carries schema version 0.
const pmsIdentityStateV0FromV2 = `{
	"id": "id:67890fghij",
	"environment": "test-env",
	"service": "test-service",
	"name": "alice",
	"description": null,
	"controller": "ap:alice12345",
	"verification_method": [
		{
			"id": "vm:jwk12345",
			"identity_id": "id:67890fghij",
			"name": "signing-key",
			"type": "JsonWebKey",
			"controller": null,
			"public_key_multibase": null,
			"public_key_jwk_json": "{\"kty\":\"EC\",\"crv\":\"secp256k1\",\"x\":\"abc\",\"y\":\"def\"}",
			"ethereum_address": null,
			"created": "2026-09-29T10:00:00Z",
			"expires": null,
			"revoked": null,
			"key_uri": "kld:///keystore/ks1/key/alice"
		}
	],
	"notification_method": null
}`

// State already written against the v2 API must come through the upgrade unchanged. In
// particular the JWK must keep its stored text: the configuration's jsonencode() matches
// that text, and a different spelling would plan a replacement of the identity.
func TestPMSIdentityUpgradeStateV0FromV2(t *testing.T) {
	ctx := context.Background()
	server, err := testAccProviders["kaleido"]()
	assert.NoError(t, err)

	upgraded, err := server.UpgradeResourceState(ctx, &tfprotov6.UpgradeResourceStateRequest{
		TypeName: "kaleido_platform_pms_identity",
		Version:  0,
		RawState: &tfprotov6.RawState{JSON: []byte(pmsIdentityStateV0FromV2)},
	})
	assert.NoError(t, err)
	assert.Empty(t, upgraded.Diagnostics)

	var schemaResp fwresource.SchemaResponse
	(&policyIdentityResource{}).Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	raw, err := upgraded.UpgradedState.Unmarshal(schemaResp.Schema.Type().TerraformType(ctx))
	assert.NoError(t, err)
	var data PolicyIdentityResourceModel
	assert.False(t, tfsdk.State{Schema: schemaResp.Schema, Raw: raw}.Get(ctx, &data).HasError())

	assert.Equal(t, "id:67890fghij", data.ID.ValueString())
	assert.Equal(t, "ap:alice12345", data.Controller.ValueString())
	assert.True(t, data.Description.IsNull())
	assert.True(t, data.NotificationMethods.IsNull())

	methods := data.VerificationMethods.Elements()
	if assert.Len(t, methods, 1) {
		vm := methods[0].(types.Object).Attributes()
		assert.Equal(t, "vm:jwk12345", vm["id"].(types.String).ValueString())
		assert.Equal(t, "JsonWebKey", vm["type"].(types.String).ValueString())
		assert.Equal(t, `{"kty":"EC","crv":"secp256k1","x":"abc","y":"def"}`, vm["public_key_jwk_json"].(jsonStringValue).ValueString(),
			"the stored JWK text is kept byte for byte")
		assert.Equal(t, "kld:///keystore/ks1/key/alice", vm["key_uri"].(types.String).ValueString())
		assert.Equal(t, "2026-09-29T10:00:00Z", vm["created"].(types.String).ValueString())
		assert.True(t, vm["controller"].IsNull())
	}
}
