# Arbitrum Sepolia — 6 confirmations per per-network override in
# <upstream>/evm/ecosystems/arbitrum.yaml.

ecosystem   = { name = "arbitrum", displayName = "Arbitrum" }
network     = { name = "arbitrum-sepolia-testnet", displayName = "Arbitrum Sepolia Testnet", chainId = "421613" }
jsonrpc_url = "https://sepolia-rollup.arbitrum.io/rpc"

confirmations = {
  count = 6
  resubmission = {
    enabled = true
  }
}
