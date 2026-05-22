# Base mainnet — 20 confirmations, derived from <upstream>/evm/ecosystems/base.yaml.

ecosystem   = { name = "base", displayName = "Base" }
network     = { name = "base-mainnet", displayName = "Base Mainnet", chainId = "8453" }
jsonrpc_url = "https://mainnet.base.org"

confirmations = {
  count = 20
  resubmission = {
    enabled = true
  }
}
