# Ethereum Sepolia testnet — 6 confirmations per the per-network override in
# <upstream>/evm/ecosystems/ethereum.yaml.

ecosystem   = { name = "ethereum", displayName = "Ethereum" }
network     = { name = "ethereum-sepolia-testnet", displayName = "Ethereum Sepolia Testnet", chainId = "11155111" }
# Contact Kaleido support to get a JSONRPC URL for Arbitrum Sepolia testnet
jsonrpc_url = "https://ethereum-sepolia-testnet.rpc.example.com"
jsonrpc_auth = { username = "REPLACE_ME", password = "REPLACE_ME" }

confirmations = {
  count = 6
  resubmission = {
    enabled = true
  }
}
