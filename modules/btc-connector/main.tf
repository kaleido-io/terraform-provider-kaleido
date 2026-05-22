# ─── Stack + Runtime + Service ────────────────────────────────────────────────

resource "kaleido_platform_stack" "this" {
  environment = var.environment_id
  name        = var.stack_name
  type        = "web3_middleware"
  sub_type    = "BTCConnectorStack"
}

resource "kaleido_platform_runtime" "this" {
  type        = "BTCConnector"
  name        = var.connector_name
  environment = var.environment_id
  stack_id    = kaleido_platform_stack.this.id
  config_json = jsonencode({})
}

resource "kaleido_platform_service" "this" {
  type        = "BTCConnector"
  name        = var.connector_name
  environment = var.environment_id
  stack_id    = kaleido_platform_stack.this.id
  runtime     = kaleido_platform_runtime.this.id

  config_json = jsonencode(merge(
    { keyManager = { id = var.key_manager_service_id } },
    var.ecosystem != null ? { ecosystem = var.ecosystem } : {},
    var.network   != null ? { network   = var.network   } : {},
    var.rpc_url   != null ? { url       = var.rpc_url   } : {},
    var.rpc_auth  != null ? { auth      = var.rpc_auth  } : {},
  ))
}

# ─── Config types + profiles ──────────────────────────────────────────────────

locals {
  config_types = toset([
    "btc.feeRate",
    "btc.assembly",
    "btc.monitoring",
    "btc.transactionEventsConfig",
  ])

  profile_values = {
    "btc.feeRate"                 = var.fee_rate
    "btc.assembly"                = var.assembly
    "btc.monitoring"              = var.monitoring
    "btc.transactionEventsConfig" = var.transaction_events
  }
}

resource "kaleido_platform_connector_config_type" "this" {
  for_each    = local.config_types
  environment = var.environment_id
  service     = kaleido_platform_service.this.id
  name        = each.key
}

resource "kaleido_platform_connector_config_profile" "this" {
  for_each    = local.profile_values
  environment = var.environment_id
  service     = kaleido_platform_service.this.id
  name        = each.key
  config_type = each.key
  value_json  = jsonencode(each.value)
  depends_on  = [kaleido_platform_connector_config_type.this]
}

# ─── Connector flow ───────────────────────────────────────────────────────────

resource "kaleido_platform_connector_flow" "submission" {
  environment = var.environment_id
  service     = kaleido_platform_service.this.id
  name        = "submission"
  config_type_bindings = {
    "btc.feeRate"    = kaleido_platform_connector_config_profile.this["btc.feeRate"].name
    "btc.assembly"   = kaleido_platform_connector_config_profile.this["btc.assembly"].name
    "btc.monitoring" = kaleido_platform_connector_config_profile.this["btc.monitoring"].name
  }
}

# ─── Stream factory ───────────────────────────────────────────────────────────

resource "kaleido_platform_connector_stream_factory" "transaction_events" {
  environment = var.environment_id
  service     = kaleido_platform_service.this.id
  name        = "transactionEvents"
  depends_on  = [kaleido_platform_connector_config_type.this]
}

# ─── Standard API ─────────────────────────────────────────────────────────────

resource "kaleido_platform_connector_standard_api" "bitcoin" {
  environment = var.environment_id
  service     = kaleido_platform_service.this.id
  name        = "bitcoin"
  flow_type_bindings = {
    resolve = kaleido_platform_connector_flow.submission.name
    submit  = kaleido_platform_connector_flow.submission.name
  }
}
