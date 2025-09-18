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

func BesuNodeKeyResourceFactory() resource.Resource {
	return &besuNodeKeyResource{}
}

type besuNodeKeyResource struct {
	commonResource
}

func (r *besuNodeKeyResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_besu_node_key"
}

func (r *besuNodeKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A Besu node key is a key pair used to sign transactions and messages on the Besu network.",
		Attributes: map[string]schema.Attribute{
			"private_key": &schema.StringAttribute{
				Computed:  true,
				Sensitive: true,
			},
			"public_key": &schema.StringAttribute{
				Computed: true,
			},
			"enode": &schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (r *besuNodeKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.AddError("Import not yet supported", "Import is not yet supported for this resource")
}

func (r *besuNodeKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	keypair, err := secp256k1.GenerateSecp256k1KeyPair()

	if err != nil {
		resp.Diagnostics.AddError("failed to generate secp256k1 keys", err.Error())
		return
	}

	resp.State.SetAttribute(ctx, path.Root("private_key"), hex.EncodeToString(keypair.PrivateKeyBytes()))
	resp.State.SetAttribute(ctx, path.Root("public_key"), hex.EncodeToString(keypair.PublicKeyBytes()))
	resp.State.SetAttribute(ctx, path.Root("enode"), keypair.Address.String())
}

func (r *besuNodeKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

func (r *besuNodeKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

func (r *besuNodeKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
}
