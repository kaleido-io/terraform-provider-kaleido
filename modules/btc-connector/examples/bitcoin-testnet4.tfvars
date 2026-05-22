# Bitcoin Testnet 4 — lower confirmation requirement, RPC-based fee estimation.

network = { name = "testnet4", displayName = "Bitcoin Testnet 4" }

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
