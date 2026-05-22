# Polygon mainnet — 50 confirmations to handle reorg risk, per
# <upstream>/evm/ecosystems/polygon.yaml.

ecosystem   = { name = "polygon", displayName = "Polygon" }
network     = { name = "polygon-mainnet", displayName = "Polygon Mainnet", chainId = "137" }
jsonrpc_url = "https://polygon-rpc.com"

confirmations = {
  count = 50
  resubmission = {
    enabled = true
  }
}
