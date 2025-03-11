resource "kaleido_platform_network" "network" {
  type = "BesuNetwork"
  name = "besu_network"
  environment = kaleido_platform_environment.env.id
  config_json = jsonencode({
    bootstrapOptions = {
      qbft = {
        blockperiodseconds = 2
      }
    }
  })
}