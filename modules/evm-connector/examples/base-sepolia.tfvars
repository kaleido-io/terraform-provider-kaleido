# Base Sepolia — 6 confirmations per per-network override.

ecosystem   = { name = "base", displayName = "Base" }
network     = { name = "base-sepolia-testnet", displayName = "Base Sepolia Testnet", chainId = "84532" }
jsonrpc_url = "https://sepolia.base.org"

confirmations = {
  count = 6
  resubmission = {
    enabled = true
  }
}
