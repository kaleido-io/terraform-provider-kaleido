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
  type = object({
    format = optional(object({
      name                 = optional(string)
      enableLegacyFallback = optional(bool)
    }))
    # `source` is a tagged union — set exactly one of fixedGasPrice / gasOracleAPI / RPCEndpoint.
    source = optional(object({
      fixedGasPrice = optional(object({
        enabled              = optional(bool)
        maxFeePerGas         = optional(string)
        maxPriorityFeePerGas = optional(string)
        gasPrice             = optional(string)
      }))
      gasOracleAPI = optional(object({
        enabled                   = optional(bool)
        enableRPCEndpointFallback = optional(bool)
        url                       = optional(string)
        method                    = optional(string)
        body                      = optional(string)
        bodyEncoding              = optional(string)
        httpHeaders               = optional(map(string))
        responseTemplate = optional(object({
          jsonata = optional(string)
        }))
        cache = optional(object({
          enabled = optional(bool)
          size    = optional(string)
          ttl     = optional(string)
        }))
      }))
      RPCEndpoint = optional(object({
        cache = optional(object({
          enabled = optional(bool)
          size    = optional(string)
          ttl     = optional(string)
        }))
        ethFeeHistory = optional(object({
          baseFeeBufferFactor   = optional(number)
          historyBlockCount     = optional(number)
          priorityFeePercentile = optional(number)
        }))
      }))
    }))
    autoIncrement = optional(object({
      enabled              = optional(bool)
      maxFeePerGas         = optional(object({ multiplier = optional(number) }))
      maxPriorityFeePerGas = optional(object({ multiplier = optional(number) }))
      gasPrice             = optional(object({ multiplier = optional(number) }))
    }))
    caps = optional(object({
      enabled              = optional(bool)
      maxFeePerGas         = optional(number)
      maxPriorityFeePerGas = optional(number)
      gasPrice             = optional(number)
    }))
  })
  default     = {}
  description = "evm.gasPricing — format (eip1559|legacy), source (tagged union: fixedGasPrice | gasOracleAPI | RPCEndpoint), auto-increment, and caps."
}

variable "nonce_assignment" {
  type = object({
    previousTxnsCondition = optional(string)
  })
  default     = {}
  description = "evm.nonceAssignment"
}

variable "submission" {
  type = object({
    # Keys are submission-error categories (gas_limit_error, gas_price_error, signature_error, …)
    # — the schema is open, so users may add their own categories.
    errorTypeMatchers = optional(map(object({
      containsIgnoreCase = optional(list(string))
      minInterval        = optional(string)
    })))
  })
  default     = {}
  description = "evm.submission — error-type matchers keyed by submission error category."
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
  # ABI fields (`abi`, `logFilters[].events`) accept native Terraform lists of objects;
  # the upstream JSON Schema is recursive (parameters have `components` of the same shape)
  # so we type them as `any` rather than re-encode the recursion.
  type = object({
    abi                              = optional(any)
    batchSize                        = optional(number)
    batchTimeout                     = optional(string)
    catchupBlockFetchAhead           = optional(number)
    catchupPageSize                  = optional(number)
    catchupPageSizeAdaptiveAlpha     = optional(number)
    catchupPageSizeAdaptiveMax       = optional(number)
    catchupPageSizeAdaptiveMin       = optional(number)
    catchupRetryMaxAttempts          = optional(number)
    catchupRetryRangeReductionFactor = optional(number)
    decodeConstructors               = optional(bool)
    enableBlockTrace                 = optional(bool)
    eventMode                        = optional(string)
    fromBlock                        = optional(string)
    includeBinaryInput               = optional(bool)
    includeBinaryLogs                = optional(bool)
    includeInputs                    = optional(bool)
    includeLogsBloom                 = optional(bool)
    logFilters = optional(list(object({
      addresses       = optional(list(string))
      eventSignatures = optional(list(string))
      events          = optional(any)
      topic0          = optional(list(string))
      topic1          = optional(list(string))
      topic2          = optional(list(string))
      topic3          = optional(list(string))
    })))
    omitSolidityDef = optional(bool)
    outputFormat    = optional(string)
    pollTimeout     = optional(string)
    requiredConfirmations = optional(number)
    traceFilters = optional(list(object({
      addresses   = optional(list(string))
      excludeFrom = optional(bool)
      excludeTo   = optional(bool)
    })))
    unfiltered = optional(bool)
  })
  default     = {}
  description = "evm.transactionEventsConfig — block-walking event stream tuning. eventMode is one of all|require_decoded|filter_decoded."
}

variable "contract_event_listener" {
  type = object({
    fromBlock    = optional(string)
    batchSize    = optional(number)
    batchTimeout = optional(string)
    pollTimeout  = optional(string)
    filters = optional(list(object({
      address = optional(string)
      # ABI event definition. Recursive schema — accept Terraform native objects.
      event = optional(any)
    })))
  })
  default     = {}
  description = "evm.contractEventListener — block-walking listener bound to a contract address + event ABI."
}
