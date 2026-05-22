# Bitcoin Testnet 4 — lower confirmation requirement, RPC-based fee estimation.

network = { name = "testnet4", displayName = "Bitcoin Testnet 4" }
# Contact Kaleido support to get a JSONRPC URL for Bitcoin testnet4
rpc_url = "https://bitcoin-testnet4.rpc.example.com"
rpc_auth = { username = "REPLACE_ME", password = "REPLACE_ME" }

fee_rate = {
  maxFeeRate = { enabled = true, satVb = 100 }
  source = {
    rpcEndpoint = { enabled = true }
  }
}

monitoring = {
  requiredConfirmations = 2
  staleTimeout          = "5m"
  monitoringInterval    = "5s"
}
