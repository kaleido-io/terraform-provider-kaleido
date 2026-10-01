# A connector flow is deployed from a template the connector service stores, and each config type
# it uses is pointed at a config profile. The config types and profiles are separate resources:
# the flow only chooses between profiles that already exist.

locals {
  env     = kaleido_platform_environment.env.id
  service = kaleido_platform_service.evm_connector.id

  # The config types the submission flow requires - each needs a profile.
  submission_config_types = toset([
    "evm.confirmations",
    "evm.gasEstimation",
    "evm.gasPricing",
    "evm.nonceAssignment",
    "evm.submission",
    "evm.transactionSerialization",
  ])

  # Every config type the flow declares, including optional ones it can run without.
  # All are deployed; only the required ones are given a profile below.
  all_config_types = setunion(local.submission_config_types, ["evm.prioritization"])

  # Profile values, validated against each config type's JSON Schema. A type not listed here
  # takes every field's schema default - check those suit your network. In particular the
  # schema default for confirmations is zero blocks.
  profile_values = {
    "evm.confirmations" = {
      count        = 12 # blocks mined after the one containing the transaction
      resubmission = { enabled = true }
    }
  }
}

# 1. Deploy each config type's schema (shipped with the connector) onto the service.
resource "kaleido_platform_connector_config_type" "submission" {
  for_each    = local.all_config_types
  environment = local.env
  service     = local.service
  name        = each.key
}

# 2. A default profile for each required config type, with values from local.profile_values.
resource "kaleido_platform_connector_config_profile" "default" {
  for_each    = local.submission_config_types
  environment = local.env
  service     = local.service
  name        = "default-${trimprefix(each.key, "evm.")}" # e.g. default-gasPricing
  config_type = each.key
  value_json  = jsonencode(lookup(local.profile_values, each.key, {}))
  depends_on  = [kaleido_platform_connector_config_type.submission]
}

# 3. Extra gas pricing profiles a transaction can select by name. They are never bound to the
#    flow: the jsonata expression below finds them by name when a transaction asks for one.
resource "kaleido_platform_connector_config_profile" "gas" {
  for_each = {
    gas-fast = "60000000000"
    gas-slow = "5000000000"
  }
  environment = local.env
  service     = local.service
  name        = each.key
  config_type = "evm.gasPricing"
  value_json  = jsonencode({ source = { fixedGasPrice = { enabled = true, gasPrice = each.value } } })
  depends_on  = [kaleido_platform_connector_config_type.submission]
}

# 4. The flow: which config profile each of its config types uses. Profiles are referenced by
#    their resource's id, so the flow follows a profile if it is ever replaced.
resource "kaleido_platform_connector_flow" "submission" {
  environment = local.env
  service     = local.service
  name        = "submission"

  # Pinned: deployed at exactly this version, and upgraded only when you raise it. It can never
  # be lowered - connector flows only move forward.
  version = "2026.09.0"

  config_profiles = {
    "evm.confirmations"            = { profile_id = kaleido_platform_connector_config_profile.default["evm.confirmations"].id }
    "evm.gasEstimation"            = { profile_id = kaleido_platform_connector_config_profile.default["evm.gasEstimation"].id }
    "evm.nonceAssignment"          = { profile_id = kaleido_platform_connector_config_profile.default["evm.nonceAssignment"].id }
    "evm.submission"               = { profile_id = kaleido_platform_connector_config_profile.default["evm.submission"].id }
    "evm.transactionSerialization" = { profile_id = kaleido_platform_connector_config_profile.default["evm.transactionSerialization"].id }

    # Selected per transaction by name (e.g. "gas-fast"), falling back to the default profile
    # when a transaction selects nothing.
    "evm.gasPricing" = {
      profile_id = kaleido_platform_connector_config_profile.default["evm.gasPricing"].id
      jsonata    = "state.input.options.gasPricing.configProfileName"
    }
  }

  # The selectable profiles must exist before transactions can pick them.
  depends_on = [kaleido_platform_connector_config_profile.gas]
}

# Tracking latest: version comes from the versions the connector stores, so when a connector
# upgrade brings a newer template the next plan shows the flow being upgraded, and the apply does it.
# Omit version entirely to hold the flow at whatever version it was first deployed at instead.
# (The query flow uses no config types, so it needs no profiles.)
data "kaleido_platform_connector_template_versions" "query" {
  environment = local.env
  service     = local.service
  kind        = "connector_flow"
  name        = "query"
}

resource "kaleido_platform_connector_flow" "query" {
  environment = local.env
  service     = local.service
  name        = "query"
  version     = data.kaleido_platform_connector_template_versions.query.latest
}
