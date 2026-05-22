# Bitcoin Signet — designed for application testing, predictable block times,
# fixed minimum-fee rate so we don't depend on a meaningful fee market.

network = { name = "testnet3", displayName = "Bitcoin Testnet 3" }
# Contact Kaleido support to get a JSONRPC URL for Bitcoin testnet3
rpc_url = "https://bitcoin-testnet3.rpc.example.com"
rpc_auth = { username = "REPLACE_ME", password = "REPLACE_ME" }

fee_rate = {
  source = {
    fixedFeeRate = {
      enabled = true
      satVb   = 1
    }
  }
}

monitoring = {
  requiredConfirmations = 1
  staleTimeout          = "5m"
  monitoringInterval    = "2s"
}
