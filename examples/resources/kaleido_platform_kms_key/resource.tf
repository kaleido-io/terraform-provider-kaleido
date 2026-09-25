// Recommended: set spec (or keystore_name) so every entry in
// public_identifier_types is actually honoured, and public_identifiers
// exposes each one's value.
resource "kaleido_platform_kms_key" "example" {
  environment             = "e:1234abcd"
  service                 = "s:1234abcd"
  wallet                  = "w:1234abcd"
  name                    = "my-key"
  spec                    = "secp256k1"
  public_identifier_types = ["address_ethereum", "address_ethereum_checksum"]

  // Pin a specific HD wallet derivation path — this API has no top-level
  // path field, so this goes in attributes instead.
  attributes = {
    "bip44_path" = "m/44'/60'/0'/0/0"
  }
}

output "checksum_address" {
  value = kaleido_platform_kms_key.example.public_identifiers["address_ethereum_checksum"]
}

// Without spec/keystore_name: only address_ethereum is ever created,
// regardless of public_identifier_types. Kept for backwards compatibility.
resource "kaleido_platform_kms_key" "legacy" {
  environment             = "e:1234abcd"
  service                 = "s:1234abcd"
  wallet                  = "w:1234abcd"
  name                    = "my-legacy-key"
  public_identifier_types = ["address_ethereum"]
}
