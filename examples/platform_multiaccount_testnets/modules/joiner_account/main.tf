terraform {
  required_providers {
    kaleido = {
      source = "kaleido-io/kaleido"
    }
  }
}

# Variables passed from parent module
variable "account_index" {
  description = "Index of this joiner account (1-based, so joiner 1 is index 1, creates account node2)"
  type = number
}

variable "account_name_prefix" {
  description = "Prefix for account names"
  type = string
}

variable "besu_network_name" {
  description = "Base name for Besu networks"
  type = string
}

variable "ipfs_network_name" {
  description = "Base name for IPFS networks"
  type = string
}

variable "paladin_network_name" {
  description = "Base name for Paladin networks"
  type = string
}

variable "shared_swarm_key" {
  description = "Shared swarm key for IPFS networks"
  type = string
}

variable "originator_bootstrap_files" {
  description = "Bootstrap files from originator Besu network"
  type = any
}

variable "deployment_zone" {
  description = "Deployment zone for peerable nodes"
  type = string
}

variable "gateway_count" {
  description = "Number of gateways to create"
  type = number
}

variable "paladin_node_size" {
  description = "Size of Paladin nodes"
  type = string
}

variable "enable_force_delete" {
  description = "Enable force delete for services"
  type = bool
}

variable "child_accounts" {
  description = "List of child accounts"
  type = any
}

variable "originator_besu_network" {
  description = "Originator Besu network"
  type = any
}

variable "originator_besu_signer_services" {
  description = "Originator Besu signer services"
  type = any
}

# Local computed values
locals {
  account_name = "${var.account_name_prefix}${var.account_index + 1}"
  environment_name = "${var.account_name_prefix}${var.account_index + 1}_env"
}

# Users group for this joiner account
resource "kaleido_platform_group" "joiner_users_group" {
  name = "users"
  depends_on = [var.child_accounts]
}

# Environment for this joiner account
resource "kaleido_platform_environment" "joiner_environment" {
  name = local.environment_name
  depends_on = [var.child_accounts]
}

# Besu Network - Joiner
resource "kaleido_platform_network" "joiner_besu_network" {
  type = "BesuNetwork"
  name = "${var.besu_network_name}_${local.account_name}"
  environment = kaleido_platform_environment.joiner_environment.id
  init_mode = "manual"
  
  file_sets = var.originator_bootstrap_files != null ? {
    init = var.originator_bootstrap_files
  } : {}
  
  init_files = var.originator_bootstrap_files != null ? "init" : null
  config_json = jsonencode({})
  
  depends_on = [
    var.originator_besu_network,
    var.originator_besu_signer_services
  ]
}

# IPFS Network - Joiner
resource "kaleido_platform_network" "joiner_ipfs_network" {
  type = "IPFSNetwork"
  name = "${var.ipfs_network_name}_${local.account_name}"
  environment = kaleido_platform_environment.joiner_environment.id
  
  config_json = jsonencode({
    swarmKey = var.shared_swarm_key
  })
}

# Paladin Network - Joiner
resource "kaleido_platform_network" "joiner_paladin_network" {
  type = "PaladinNetwork"
  name = "${var.paladin_network_name}_${local.account_name}"
  environment = kaleido_platform_environment.joiner_environment.id
  
  config_json = jsonencode({
    type = "evmRegistry"
    evmRegistry = {
      registryContract = "existing"
      existingContract = {
        address = ""
      }
    }
  })
}

# Chain Infrastructure Stacks - Joiner
resource "kaleido_platform_stack" "joiner_besu_stack" {
  environment = kaleido_platform_environment.joiner_environment.id
  name = "chain_infrastructure_besu"
  type = "chain_infrastructure"
  network_id = kaleido_platform_network.joiner_besu_network.id
}

resource "kaleido_platform_stack" "joiner_ipfs_stack" {
  environment = kaleido_platform_environment.joiner_environment.id
  name = "chain_infrastructure_ipfs"
  type = "chain_infrastructure"
  network_id = kaleido_platform_network.joiner_ipfs_network.id
}

resource "kaleido_platform_stack" "joiner_paladin_stack" {
  environment = kaleido_platform_environment.joiner_environment.id
  name = "chain_infrastructure_paladin"
  type = "chain_infrastructure"
  sub_type = "PaladinStack"
  network_id = kaleido_platform_network.joiner_paladin_network.id
}

# Besu Peer Node - Joiner
resource "kaleido_platform_runtime" "joiner_besu_peer_runtime" {
  type = "BesuNode"
  name = "${local.account_name}_peer1"
  environment = kaleido_platform_environment.joiner_environment.id
  zone = var.deployment_zone
  config_json = jsonencode({})
  stack_id = kaleido_platform_stack.joiner_besu_stack.id
  
  depends_on = [kaleido_platform_network.joiner_besu_network]
}

resource "kaleido_platform_service" "joiner_besu_peer_service" {
  type = "BesuNode"
  name = "${local.account_name}_peer1"
  environment = kaleido_platform_environment.joiner_environment.id
  runtime = kaleido_platform_runtime.joiner_besu_peer_runtime.id
  
  config_json = jsonencode({
    network = {
      id = kaleido_platform_network.joiner_besu_network.id
    }
    signer = false
  })
  
  force_delete = var.enable_force_delete
  stack_id = kaleido_platform_stack.joiner_besu_stack.id
}

# EVM Gateway - Joiner
resource "kaleido_platform_runtime" "joiner_gateway_runtime" {
  count = var.gateway_count > 0 ? 1 : 0
  
  type = "EVMGateway"
  name = "${local.account_name}_gateway"
  environment = kaleido_platform_environment.joiner_environment.id
  config_json = jsonencode({})
  stack_id = kaleido_platform_stack.joiner_besu_stack.id
  
  depends_on = [kaleido_platform_network.joiner_besu_network]
}

resource "kaleido_platform_service" "joiner_gateway_service" {
  count = var.gateway_count > 0 ? 1 : 0
  
  type = "EVMGateway"
  name = "${local.account_name}_gateway"
  environment = kaleido_platform_environment.joiner_environment.id
  runtime = kaleido_platform_runtime.joiner_gateway_runtime[count.index].id
  
  config_json = jsonencode({
    network = {
      id = kaleido_platform_network.joiner_besu_network.id
    }
  })
  
  stack_id = kaleido_platform_stack.joiner_besu_stack.id
}

# IPFS Node - Joiner
resource "kaleido_platform_runtime" "joiner_ipfs_runtime" {
  type = "IPFSNode"
  name = "${local.account_name}_ipfs"
  zone = var.deployment_zone
  environment = kaleido_platform_environment.joiner_environment.id
  config_json = jsonencode({})
  stack_id = kaleido_platform_stack.joiner_ipfs_stack.id
}

resource "kaleido_platform_service" "joiner_ipfs_service" {
  type = "IPFSNode"
  name = "${local.account_name}_ipfs"
  environment = kaleido_platform_environment.joiner_environment.id
  runtime = kaleido_platform_runtime.joiner_ipfs_runtime.id
  
  config_json = jsonencode({
    network = {
      id = kaleido_platform_network.joiner_ipfs_network.id
    }
  })
  
  stack_id = kaleido_platform_stack.joiner_ipfs_stack.id
}

# Key Manager - Joiner
resource "kaleido_platform_runtime" "joiner_kms_runtime" {
  type = "KeyManager"
  name = "${local.account_name}_kms"
  environment = kaleido_platform_environment.joiner_environment.id
  config_json = jsonencode({})
}

resource "kaleido_platform_service" "joiner_kms_service" {
  type = "KeyManager"
  name = "${local.account_name}_kms"
  environment = kaleido_platform_environment.joiner_environment.id
  runtime = kaleido_platform_runtime.joiner_kms_runtime.id
  config_json = jsonencode({})
}

resource "kaleido_platform_kms_wallet" "paladin_wallet" {
  type = "hdwallet"
  name = "${local.account_name}_paladin_wallet"
  environment = kaleido_platform_environment.joiner_environment.id
  service = kaleido_platform_service.joiner_kms_service.id
  config_json = jsonencode({})
}

# Paladin Node - Joiner
resource "kaleido_platform_runtime" "joiner_paladin_runtime" {
  type = "PaladinNodeRuntime"
  name = "${local.account_name}_paladin"
  environment = kaleido_platform_environment.joiner_environment.id
  config_json = jsonencode({})
  stack_id = kaleido_platform_stack.joiner_paladin_stack.id
  size = var.paladin_node_size
  zone = var.deployment_zone
}

resource "kaleido_platform_service" "joiner_paladin_service" {
  type = "PaladinNodeService"
  name = "paladin-service-${local.account_name}"
  environment = kaleido_platform_environment.joiner_environment.id
  runtime = kaleido_platform_runtime.joiner_paladin_runtime.id
  
  config_json = jsonencode({
    network = {
      id = kaleido_platform_network.joiner_paladin_network.id
    }
    keyManager = {
      id = kaleido_platform_service.joiner_kms_service.id
    }
    baseLedgerEndpoint = {
      type = "local"
      local = {
        gateway = {
          id = kaleido_platform_service.joiner_gateway_service[0].id
        }
      }
    }
    wallets = {
      kmsKeyStore = kaleido_platform_kms_wallet.paladin_wallet.name
    }
    baseConfig = "{\"blockIndexer\":{\"fromBlock\": 0 }}"
    registryAdminIdentity = "registry.admin" // TODO can this not be hardcoded ?
  })
  
  stack_id = kaleido_platform_stack.joiner_paladin_stack.id
}

# Outputs
output "account_id" {
  description = "Account ID for this joiner"
  value = var.child_accounts[var.account_index].id
}

output "environment" {
  value = kaleido_platform_environment.joiner_environment
}

output "besu_network" {
  value = kaleido_platform_network.joiner_besu_network
}

output "ipfs_network" {
  value = kaleido_platform_network.joiner_ipfs_network
}

output "paladin_network" {
  value = kaleido_platform_network.joiner_paladin_network
}

output "besu_stack" {
  value = kaleido_platform_stack.joiner_besu_stack
}

output "ipfs_stack" {
  value = kaleido_platform_stack.joiner_ipfs_stack
}

output "paladin_stack" {
  value = kaleido_platform_stack.joiner_paladin_stack
}

output "besu_peer_service" {
  value = kaleido_platform_service.joiner_besu_peer_service
}

output "ipfs_service" {
  value = kaleido_platform_service.joiner_ipfs_service
}

output "paladin_service" {
  value = kaleido_platform_service.joiner_paladin_service
}

output "gateway_service" {
  value = var.gateway_count > 0 ? kaleido_platform_service.joiner_gateway_service[0] : null
}

output "users_group" {
  value = kaleido_platform_group.joiner_users_group
}