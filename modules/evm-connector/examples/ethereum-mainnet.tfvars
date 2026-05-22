# Ethereum mainnet — 12-confirmation finality, resubmission enabled.
# Derived from <upstream>/evm/ecosystems/ethereum.yaml.

ecosystem   = { name = "ethereum", displayName = "Ethereum" }
network     = { name = "ethereum-mainnet", displayName = "Ethereum Mainnet", chainId = "1" }
jsonrpc_url = "https://mainnet.infura.io/v3/REPLACE_ME"

confirmations = {
  count = 12
  resubmission = {
    enabled = true
  }
}
