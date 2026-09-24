# The platform's default config profile values for the chain a connector runs against - for
# example, how many confirmations to wait for. Some networks (typically testnets) override the
# ecosystem's values.
data "kaleido_platform_catalog_web3ecosystem_defaults" "chain" {
  ecosystem = "ethereum"
  network   = "ethereum-sepolia-testnet"
}

# Use the chain's default as a profile's value, falling back to the schema default when the
# catalog has none for this config type.
resource "kaleido_platform_connector_config_profile" "confirmations" {
  environment = kaleido_platform_environment.env.id
  service     = kaleido_platform_service.evm_connector.id
  name        = "default-confirmations"
  config_type = "evm.confirmations"
  value_json  = lookup(data.kaleido_platform_catalog_web3ecosystem_defaults.chain.config_profiles, "evm.confirmations", "{}")
}
