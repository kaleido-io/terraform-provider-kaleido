# Base Sepolia — 6 confirmations per per-network override.

ecosystem   = { name = "base", displayName = "Base" }
network     = { name = "base-sepolia-testnet", displayName = "Base Sepolia Testnet", chainId = "84532" }
# Contact Kaleido support to get a JSONRPC URL for Base Sepolia testnet
jsonrpc_url = "https://base-sepolia-testnet.rpc.example.com"
jsonrpc_auth = { username = "REPLACE_ME", password = "REPLACE_ME" }

confirmations = {
  count = 6
  resubmission = {
    enabled = true
  }
}
