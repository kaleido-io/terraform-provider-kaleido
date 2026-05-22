# Bitcoin Signet — designed for application testing, predictable block times,
# fixed minimum-fee rate so we don't depend on a meaningful fee market.

network = { name = "signet", displayName = "Bitcoin Signet" }

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
