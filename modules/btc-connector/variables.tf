variable "environment_id" {
  type        = string
  description = "ID of the environment to deploy the BTC connector into."
}

variable "stack_name" {
  type        = string
  default     = "btc"
  description = "Name of the BTCConnectorStack."
}

variable "connector_name" {
  type        = string
  default     = "btc-connector"
  description = "Name of the BTCConnector runtime and service."
}

variable "key_manager_service_id" {
  type        = string
  description = "ID of the KeyManager service used to sign Bitcoin transactions."
}

# ─── Service-level config ─────────────────────────────────────────────────────

variable "rpc_url" {
  type        = string
  default     = null
  description = "Bitcoin Core RPC URL."
}

variable "rpc_auth" {
  type = object({
    username = string
    password = string
  })
  default     = null
  description = "Basic auth credentials for the Bitcoin Core RPC endpoint."
}

variable "ecosystem" {
  type = object({
    name        = string
    displayName = optional(string)
  })
  default     = { name = "bitcoin", displayName = "Bitcoin" }
  description = "Ecosystem metadata."
}

variable "network" {
  type = object({
    name        = string
    displayName = optional(string)
  })
  description = "Network metadata (mainnet | testnet4 | testnet | signet)."
}

# ─── Config profile values ────────────────────────────────────────────────────
# Schemas mirror <upstream connector definitions source>/btc/config_types/*.yaml.

variable "fee_rate" {
  type        = any
  default     = {}
  description = "btc.feeRate — fee rate sources (rpcEndpoint | feeOracleAPI | fixedFeeRate), maxFeeRate cap, auto-increment policy."
}

variable "assembly" {
  type = object({
    changeOutputPosition = optional(string, "last")
  })
  default     = { changeOutputPosition = "last" }
  description = "btc.assembly — final transaction assembly (changeOutputPosition: \"last\" | \"random\")."
}

variable "monitoring" {
  type = object({
    monitoringInterval    = optional(string)
    requiredConfirmations = optional(number)
    staleTimeout          = optional(string)
  })
  default     = {}
  description = "btc.monitoring — confirmation count, polling interval, stale timeout for resubmission."
}

variable "transaction_events" {
  type = object({
    fromBlock             = optional(string)
    batchSize             = optional(number)
    batchTimeout          = optional(string)
    catchupPageSize       = optional(number)
    pollTimeout           = optional(string)
    requiredConfirmations = optional(number)
    unfiltered            = optional(bool)
  })
  default     = {}
  description = "btc.transactionEventsConfig — event stream tuning."
}
