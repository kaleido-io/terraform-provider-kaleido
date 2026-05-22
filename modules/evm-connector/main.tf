# ─── Stack + Runtime + Service ────────────────────────────────────────────────

resource "kaleido_platform_stack" "this" {
  environment = var.environment_id
  name        = var.stack_name
  type        = "web3_middleware"
  sub_type    = "EVMConnectorStack"
}

resource "kaleido_platform_runtime" "this" {
  type        = "EVMConnector"
  name        = var.connector_name
  environment = var.environment_id
  stack_id    = kaleido_platform_stack.this.id
  config_json = jsonencode({})
}

resource "kaleido_platform_service" "this" {
  type        = "EVMConnector"
  name        = var.connector_name
  environment = var.environment_id
  stack_id    = kaleido_platform_stack.this.id
  runtime     = kaleido_platform_runtime.this.id

  config_json = jsonencode(merge(
    { keyManager = { id = var.key_manager_service_id } },
    var.ecosystem              != null ? { ecosystem  = var.ecosystem  } : {},
    var.network                != null ? { network    = var.network    } : {},
    var.evm_gateway_service_id != null ? { evmGateway = { id = var.evm_gateway_service_id } } : {},
    var.jsonrpc_url            != null ? { url        = var.jsonrpc_url } : {},
    var.jsonrpc_auth           != null ? { auth       = var.jsonrpc_auth } : {},
  ))
}

# ─── Config types (template ensure / version pin) ─────────────────────────────

locals {
  config_types = toset([
    "evm.confirmations",
    "evm.gasEstimation",
    "evm.gasPricing",
    "evm.nonceAssignment",
    "evm.submission",
    "evm.transactionSerialization",
    "evm.blockEventsConfig",
    "evm.transactionEventsConfig",
    "evm.contractEventListener",
  ])

  profile_values = {
    "evm.confirmations"            = var.confirmations
    "evm.gasEstimation"            = var.gas_estimation
    "evm.gasPricing"               = var.gas_pricing
    "evm.nonceAssignment"          = var.nonce_assignment
    "evm.submission"               = var.submission
    "evm.transactionSerialization" = var.transaction_serialization
    "evm.blockEventsConfig"        = var.block_events
    "evm.transactionEventsConfig"  = var.transaction_events
    "evm.contractEventListener"    = var.contract_event_listener
  }
}

resource "kaleido_platform_connector_config_type" "this" {
  for_each    = local.config_types
  environment = var.environment_id
  service     = kaleido_platform_service.this.id
  name        = each.key
}

# ─── Config profiles (one per type) ───────────────────────────────────────────

resource "kaleido_platform_connector_config_profile" "this" {
  for_each    = local.profile_values
  environment = var.environment_id
  service     = kaleido_platform_service.this.id
  name        = each.key
  config_type = each.key
  value_json  = jsonencode(each.value)
  depends_on  = [kaleido_platform_connector_config_type.this]
}

# ─── Connector flows ──────────────────────────────────────────────────────────

resource "kaleido_platform_connector_flow" "submission" {
  environment = var.environment_id
  service     = kaleido_platform_service.this.id
  name        = "submission"
  config_type_bindings = {
    "evm.confirmations"            = kaleido_platform_connector_config_profile.this["evm.confirmations"].name
    "evm.gasEstimation"            = kaleido_platform_connector_config_profile.this["evm.gasEstimation"].name
    "evm.gasPricing"               = kaleido_platform_connector_config_profile.this["evm.gasPricing"].name
    "evm.nonceAssignment"          = kaleido_platform_connector_config_profile.this["evm.nonceAssignment"].name
    "evm.submission"               = kaleido_platform_connector_config_profile.this["evm.submission"].name
    "evm.transactionSerialization" = kaleido_platform_connector_config_profile.this["evm.transactionSerialization"].name
  }
}

resource "kaleido_platform_connector_flow" "query" {
  environment = var.environment_id
  service     = kaleido_platform_service.this.id
  name        = "query"
}

# ─── Stream factories ─────────────────────────────────────────────────────────

resource "kaleido_platform_connector_stream_factory" "block_events" {
  environment = var.environment_id
  service     = kaleido_platform_service.this.id
  name        = "blockEvents"
  depends_on  = [kaleido_platform_connector_config_type.this]
}

resource "kaleido_platform_connector_stream_factory" "transaction_events" {
  environment = var.environment_id
  service     = kaleido_platform_service.this.id
  name        = "transactionEvents"
  depends_on  = [kaleido_platform_connector_config_type.this]
}

# ─── Standard API ─────────────────────────────────────────────────────────────

resource "kaleido_platform_connector_standard_api" "evm" {
  environment = var.environment_id
  service     = kaleido_platform_service.this.id
  name        = "evm"
  flow_type_bindings = {
    resolve = kaleido_platform_connector_flow.submission.name
    submit  = kaleido_platform_connector_flow.submission.name
    query   = kaleido_platform_connector_flow.query.name
  }
}

# ─── Standard streams ─────────────────────────────────────────────────────────

resource "kaleido_platform_connector_standard_stream" "new_blocks" {
  environment               = var.environment_id
  service                   = kaleido_platform_service.this.id
  name                      = "newBlocks"
  config_profile_name_or_id = kaleido_platform_connector_config_profile.this["evm.blockEventsConfig"].name
}
