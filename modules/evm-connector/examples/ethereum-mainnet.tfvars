# Ethereum mainnet — 12-confirmation finality, resubmission enabled.
# Derived from <upstream>/evm/ecosystems/ethereum.yaml.

ecosystem   = { name = "ethereum", displayName = "Ethereum" }
network     = { name = "ethereum-mainnet", displayName = "Ethereum Mainnet", chainId = "1" }
# Contact Kaleido support to get a JSONRPC URL for Ethereum mainnet
jsonrpc_url = "https://ethereum-mainnet.rpc.example.com"
jsonrpc_auth = { username = "REPLACE_ME", password = "REPLACE_ME" }

confirmations = {
  count = 12
  resubmission = {
    enabled = true
  }
}
