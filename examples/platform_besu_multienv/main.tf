
terraform {
  required_providers {
    kaleido = {
      source = "kaleido-io/kaleido"
    }
  }
}

data "kaleido_platform_account" "account" {}

provider "kaleido" {
  platform_api = var.kaleido_platform_api
  platform_username = var.kaleido_platform_api_key_name
  platform_password = var.kaleido_platform_api_key_secret
}

resource "kaleido_platform_besu_node_key" "besu_node_keys" {
  for_each = var.validators
}

locals {
  validator_keys = [
    for k, v in kaleido_platform_besu_node_key.besu_node_keys : v.enode
  ]
}

data "kaleido_platform_besu_qbft_network_genesis" "besu_qbft_network_genesis" {
  chain_id = var.chain_id
  validator_keys = local.validator_keys
  qbft_config = {
    block_period_seconds = var.block_period
    epoch_length = 30000
    request_timeout_seconds = 10
  }
}

resource "kaleido_platform_environment" "environments" {
  for_each = var.validators
  name = "${each.value.name}-${var.network_name}"
}

resource "kaleido_platform_network" "besu_network" {
  for_each = var.validators
  name = var.network_name
  type = "Besu"
  init_mode = "manual"
  environment = kaleido_platform_environment.environments[each.value.name].id
  file_sets = {
    init = {
      files = {
        "genesis.json" = {
          type = "json"
          data = {
          "text" = data.kaleido_platform_besu_qbft_network_genesis.besu_qbft_network_genesis.genesis
        }
        }
      }
    }
  }
  init_files = "init"
  config_json = jsonencode({})
}

resource "kaleido_platform_stack" "stack" {
  for_each = var.validators
  name = var.network_name
  environment = kaleido_platform_environment.environments[each.value.name].id
  type = "chain_infrastructure"
  network_id = kaleido_platform_network.besu_network[each.value.name].id
}

resource "kaleido_platform_runtime" "runtime" {
  for_each = var.validators
  type = "BesuNode"
  name = "${each.value.name}-runtime"
  environment = kaleido_platform_environment.environments[each.value.name].id
  config_json = jsonencode({})
  zone = var.node_peerable_zone
  stack_id = kaleido_platform_stack.stack[each.value.name].id
  size = "Small"
}

resource "kaleido_platform_service" "service" {
  for_each = var.validators
  type = "BesuNode"
  name = "${each.value.name}"
  environment = kaleido_platform_environment.environments[each.value.name].id
  config_json = jsonencode({
    network = {
      id = kaleido_platform_network.besu_network[each.value.name].id
    }
    nodeKey = {
      credSetRef = "nodeKey"
    }
    signer = true
  })
  cred_sets = {
    nodeKey = {
      type = "key"
      key = {
        value = kaleido_platform_besu_node_key.besu_node_keys[each.value.name].private_key
      }
    }
  }
  runtime = kaleido_platform_runtime.runtime[each.value.name].id
  stack_id = kaleido_platform_stack.stack[each.value.name].id
}

locals {
  validator_names = sort(keys(var.validators))
  mesh_connectors = {
    for pair in setproduct(local.validator_names, local.validator_names) :
    "${pair[0]}-${pair[1]}" => {
      requestor_name = pair[0]
      acceptor_name  = pair[1]
    } if index(local.validator_names, pair[0]) < index(local.validator_names, pair[1])
  }
}

module "mesh_connectors" {
  source = "./connector"
  for_each = local.mesh_connectors
  requestor_name = each.value.requestor_name
  acceptor_name = each.value.acceptor_name
  requestor_network_id = kaleido_platform_network.besu_network[each.value.requestor_name].id
  requestor_environment_id = kaleido_platform_environment.environments[each.value.requestor_name].id
  acceptor_network_id = kaleido_platform_network.besu_network[each.value.acceptor_name].id
  acceptor_environment_id = kaleido_platform_environment.environments[each.value.acceptor_name].id
  account_id = data.kaleido_platform_account.account.account_id
  node_peerable_zone = var.node_peerable_zone
}

