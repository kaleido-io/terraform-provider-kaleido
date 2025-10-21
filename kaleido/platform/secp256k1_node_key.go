// Copyright © Kaleido, Inc. 2025

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
	"encoding/hex"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hyperledger/firefly-signer/pkg/secp256k1"
)

func Secp256k1NodeKeyResourceFactory() resource.Resource {
	return &secp256k1NodeKeyResource{}
}

type secp256k1NodeKeyResource struct {
	commonResource
}

func (r *secp256k1NodeKeyResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_secp256k1_node_key"
}

func (r *secp256k1NodeKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `A secp256k1 node key is a key pair used by a blockchain node (typically Ethereum) to authenticate to peers, sign messages,
		 and validate blocks. Similar to a TLS certificate. This resource generates a new key pair each time it is created and stores it securely in the state.
		 For secp256k1 keys used for transaction signing, see the 'kaleido_platform_kms_key' resource within our Key Management Service (KMS).`,
		Attributes: map[string]schema.Attribute{
			"private_key": &schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "The private key of the key pair, can be uploaded to a blockchain node service's config when creating nodes.",
			},
			"public_key": &schema.StringAttribute{
				Computed:    true,
				Description: "The public key of the key pair, also node as enode ID in Ethereum.",
			},
			"address": &schema.StringAttribute{
				Computed:    true,
				Description: "The (public) address of the key pair, used for validator identities in Ethereum and BFT consensus.",
			},
		},
	}
}

func (r *secp256k1NodeKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.AddError("Import not yet supported", "Import is not yet supported for this resource")
}

func (r *secp256k1NodeKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	keypair, err := secp256k1.GenerateSecp256k1KeyPair()

	if err != nil {
		resp.Diagnostics.AddError("failed to generate secp256k1 keys", err.Error())
		return
	}

	resp.State.SetAttribute(ctx, path.Root("private_key"), hex.EncodeToString(keypair.PrivateKeyBytes()))
	resp.State.SetAttribute(ctx, path.Root("public_key"), hex.EncodeToString(keypair.PublicKeyBytes()))
	resp.State.SetAttribute(ctx, path.Root("address"), keypair.Address.String())
}

func (r *secp256k1NodeKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

func (r *secp256k1NodeKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

func (r *secp256k1NodeKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
}
