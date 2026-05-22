variable "environment_id" {
  type        = string
  description = "ID of the environment to deploy the EVM connector into."
}

variable "stack_name" {
  type        = string
  default     = "evm"
  description = "Name of the EVMConnectorStack."
}

variable "connector_name" {
  type        = string
  default     = "evm-connector"
  description = "Name of the EVMConnector runtime and service."
}

variable "key_manager_service_id" {
  type        = string
  description = "ID of the KeyManager service used to sign EVM transactions."
}

# ─── Service-level config ─────────────────────────────────────────────────────

variable "evm_gateway_service_id" {
  type        = string
  default     = null
  description = "Optional EVMGateway service ID for Kaleido-managed Besu networks."
}

variable "jsonrpc_url" {
  type        = string
  default     = null
  description = "Optional external JSON-RPC URL (for public networks)."
}

variable "jsonrpc_auth" {
  type = object({
    username = string
    password = string
  })
  default     = null
  description = "Optional basic auth credentials for the JSON-RPC endpoint."
}

variable "ecosystem" {
  type = object({
    name        = string
    displayName = optional(string)
  })
  default     = null
  description = "Ecosystem metadata block (e.g. { name = \"ethereum\", displayName = \"Ethereum\" })."
}

variable "network" {
  type = object({
    name        = string
    displayName = optional(string)
    chainId     = optional(string)
  })
  default     = null
  description = "Network metadata block (e.g. { name = \"ethereum-mainnet\", chainId = \"1\" })."
}

# ─── Config profile values (one variable per upstream config type) ────────────
# Schemas mirror <upstream connector definitions source>/evm/config_types/*.yaml.

variable "confirmations" {
  type = object({
    count = optional(number, 0)
    resubmission = optional(object({
      enabled = optional(bool, false)
      timeout = optional(string, "5m")
    }))
  })
  default     = {}
  description = "evm.confirmations — number of confirmations before a transaction is considered final, plus optional resubmission policy."
}

variable "gas_estimation" {
  type = object({
    scaleFactor = optional(number, 1.0)
  })
  default     = {}
  description = "evm.gasEstimation"
}

variable "gas_pricing" {
  type        = any
  default     = {}
  description = "evm.gasPricing — { format = \"eip1559\"|\"legacy\", source = { fixedGasPrice|gasOracleAPI|rpcEndpoint = {...} } }. Wide tagged-union shape; pass through as-is."
}

variable "nonce_assignment" {
  type = object({
    previousTxnsCondition = optional(string)
  })
  default     = {}
  description = "evm.nonceAssignment"
}

variable "submission" {
  type        = any
  default     = {}
  description = "evm.submission — { errorTypeMatchers = [...] } and similar policy knobs."
}

variable "transaction_serialization" {
  type = object({
    useOriginalFormat = optional(bool, false)
  })
  default     = {}
  description = "evm.transactionSerialization"
}

variable "block_events" {
  type = object({
    minWait = optional(string, "500ms")
    maxWait = optional(string, "5s")
  })
  default     = {}
  description = "evm.blockEventsConfig — debounce timings for the latest-block poller."
}

variable "transaction_events" {
  type        = any
  default     = {}
  description = "evm.transactionEventsConfig — fromBlock, batchSize, logFilters, abi."
}

variable "contract_event_listener" {
  type        = any
  default     = {}
  description = "evm.contractEventListener — fromBlock, batchSize, filters, abi."
}
