# Ethereum Sepolia testnet — 6 confirmations per the per-network override in
# <upstream>/evm/ecosystems/ethereum.yaml.

ecosystem   = { name = "ethereum", displayName = "Ethereum" }
network     = { name = "ethereum-sepolia-testnet", displayName = "Ethereum Sepolia Testnet", chainId = "11155111" }
jsonrpc_url = "https://sepolia.infura.io/v3/REPLACE_ME"

confirmations = {
  count = 6
  resubmission = {
    enabled = true
  }
}
