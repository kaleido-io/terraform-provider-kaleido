# Besu — private EVM network, immediate finality, fixed zero-fee gas pricing.
# Derived from <upstream>/evm/ecosystems/besu.yaml.

ecosystem = { name = "besu", displayName = "Besu" }
network   = { name = "besu-private", displayName = "Besu Private", chainId = "3333" }

confirmations = { count = 0 }

gas_pricing = {
  source = {
    fixedGasPrice = {
      enabled              = true
      maxPriorityFeePerGas = "0x0"
      maxFeePerGas         = "0x0"
    }
  }
}

block_events = { minWait = "100ms", maxWait = "500ms" }
