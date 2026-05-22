# Polygon Amoy testnet — 50 confirmations to handle reorg risk, per
# <upstream>/evm/ecosystems/polygon.yaml.

ecosystem   = { name = "polygon", displayName = "Polygon" }
network     = { name = "polygon-amoy-testnet", displayName = "Polygon Amoy Testnet", chainId = "80001" }
# Contact Kaleido support to get a JSONRPC URL for Arbitrum Sepolia testnet
jsonrpc_url = "https://polygon-amoy-testnet.rpc.example.com"
jsonrpc_auth = { username = "REPLACE_ME", password = "REPLACE_ME" }

confirmations = {
  count = 50
  resubmission = {
    enabled = true
  }
}
