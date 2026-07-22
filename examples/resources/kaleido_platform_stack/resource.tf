resource "kaleido_platform_stack" "chain_infra_besu_stack" {
  environment = kaleido_platform_environment.env_0.id
  name = "chain_infra_besu_stack"
  type = "chain_infrastructure"
  network_id = kaleido_platform_network.besu_net.id
}

resource "kaleido_platform_stack" "chain_infra_ipfs_stack" {
  environment = kaleido_platform_environment.env_0.id
  name = "chain_infra_ipfs_stack"
  type = "chain_infrastructure"
  network_id = kaleido_platform_network.ipfs_net.id
}

resource "kaleido_platform_stack" "web3_middleware_stack" {
  environment = kaleido_platform_environment.env_0.id
  name = "web3_middleware_stack"
  type = "web3_middleware"
}

resource "kaleido_platform_stack" "digital_assets_stack" {
  environment = kaleido_platform_environment.env_0.id
  name = "digital_assets_stack"
  type = "digital_assets"
}