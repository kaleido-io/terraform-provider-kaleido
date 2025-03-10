resource "kaleido_platform_runtime" "bnr" {
  type = "BesuNode"
  name = "besu_node_1"
  environment = kaleido_platform_environment.env.id
  config_json = jsonencode({})
  stack_id = kaleido_platform_stack.chain_infra_stack.id
}