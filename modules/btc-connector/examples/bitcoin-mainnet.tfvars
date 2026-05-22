# Bitcoin mainnet — derived from <upstream>/btc/ecosystems/bitcoin.yaml.

network  = { name = "mainnet", displayName = "Bitcoin Mainnet" }
# Contact Kaleido support to get a JSONRPC URL for Bitcoin mainnet
rpc_url  = "https://bitcoin-mainnet.rpc.example/REPLACE_ME"
rpc_auth = { username = "REPLACE_ME", password = "REPLACE_ME" }

fee_rate = {
  maxFeeRate = {
    enabled = true
    satVb   = 100
  }
  source = {
    rpcEndpoint = {
      enabled            = true
      confirmationTarget = 6
      estimateMode       = "CONSERVATIVE"
    }
  }
}

assembly = { changeOutputPosition = "last" }

monitoring = {
  requiredConfirmations = 6
  staleTimeout          = "10m"
  monitoringInterval    = "5s"
}
