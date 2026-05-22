# Base mainnet — 20 confirmations, derived from <upstream>/evm/ecosystems/base.yaml.

ecosystem   = { name = "base", displayName = "Base" }
network     = { name = "base-mainnet", displayName = "Base Mainnet", chainId = "8453" }
# Contact Kaleido support to get a JSONRPC URL for Base mainnet
jsonrpc_url = "https://base-mainnet.rpc.example.com"
jsonrpc_auth = { username = "REPLACE_ME", password = "REPLACE_ME" }

confirmations = {
  count = 20
  resubmission = {
    enabled = true
  }
}
