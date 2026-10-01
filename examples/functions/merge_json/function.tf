# Apply a few settings of your own on top of the platform catalog's default for a config profile.
# Settings you leave out, or leave null, keep the catalog's value.
data "kaleido_platform_catalog_web3ecosystem_defaults" "chain" {
  ecosystem = "ethereum"
  network   = "ethereum-mainnet"
}

variable "confirmations" {
  type = object({
    count = optional(number)
    resubmission = optional(object({
      enabled = optional(bool)
      timeout = optional(string)
    }))
  })
  default = { count = 20 }
}

resource "kaleido_platform_connector_config_profile" "confirmations" {
  environment = kaleido_platform_environment.env.id
  service     = kaleido_platform_service.evm_connector.id
  name        = "default-confirmations"
  config_type = "evm.confirmations"
  # With the catalog's {"count":12,"resubmission":{"enabled":true}}, this is
  # {"count":20,"resubmission":{"enabled":true}}.
  value_json = provider::kaleido::merge_json(
    lookup(data.kaleido_platform_catalog_web3ecosystem_defaults.chain.config_profiles, "evm.confirmations", "{}"),
    jsonencode(var.confirmations),
  )
}
