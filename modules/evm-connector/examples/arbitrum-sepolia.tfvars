# Arbitrum Sepolia — 6 confirmations per per-network override in
# <upstream>/evm/ecosystems/arbitrum.yaml.

ecosystem   = { name = "arbitrum", displayName = "Arbitrum" }
network     = { name = "arbitrum-sepolia-testnet", displayName = "Arbitrum Sepolia Testnet", chainId = "421613" }
# Contact Kaleido support to get a JSONRPC URL for Arbitrum Sepolia testnet
jsonrpc_url = "https://arbitrum-sepolia-testnet.rpc.example.com"
jsonrpc_auth = { username = "REPLACE_ME", password = "REPLACE_ME" }

confirmations = {
  count = 6
  resubmission = {
    enabled = true
  }
}
