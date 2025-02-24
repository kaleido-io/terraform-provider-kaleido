resource "kaleido_platform_service" "bns" {
  type = "BesuNode"
  name = "besu_node_1"
  environment = kaleido_platform_environment.env.id
  stack_id = kaleido_platform_stack.chain_infra_stack.id
  runtime = kaleido_platform_runtime.bnr.id
  config_json = jsonencode({
    network = {
      id = kaleido_platform_network.network.id
    }
  })
  // uncomment `force_delete = true` and run terraform apply before running terraform destory to successfully delete the besu nodes
  # force_delete = true
}