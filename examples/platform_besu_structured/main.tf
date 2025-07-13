terraform {
  required_providers {
    kaleido = {
      source = "kaleido-io/kaleido"
      version = "1.1.0"
    }
  }
}

provider "kaleido" {
  platform_api = var.kaleido_platform_api
  platform_username = var.kaleido_platform_username
  platform_password = var.kaleido_platform_password
}

variable "kaleido_platform_api" {
  description = "Kaleido platform API URL"
  type        = string
}

variable "kaleido_platform_username" {
  description = "Kaleido platform username"
  type        = string
}

variable "kaleido_platform_password" {
  description = "Kaleido platform password"
  type        = string
  sensitive   = true
}

variable "environment_name" {
  description = "Name for the environment"
  type        = string
  default     = "besu-structured-example"
}

variable "besu_node_count" {
  description = "Number of Besu nodes to create"
  type        = number
  default     = 3
}

// Create the environment
resource "kaleido_platform_environment" "env_0" {
  name = var.environment_name
}

// Create the Besu stack
resource "kaleido_platform_stack" "chain_infra_besu_stack" {
  name = "besu-chain-infrastructure"
  environment = kaleido_platform_environment.env_0.id
  type = "BesuStack"
}

// Create the Besu network using the structured resource
resource "kaleido_platform_besu_network" "besu_net" {
  name = "besu-mainnet"
  environment = kaleido_platform_environment.env_0.id
  
  // QBFT consensus configuration
  bootstrap_options = {
    qbft = {
      block_period_seconds = 3
      epoch_length = 30000
      request_timeout = 10000
      message_queue_limit = 1000
      duplicate_queue_limit = 100
      future_messages_limit = 1000
      future_messages_max_distance = 10
    }
  }
  
  // Let the network auto-generate chain ID
  consensus_type = "qbft"
  init_mode = "automated"
}

// Create runtimes for the Besu nodes
resource "kaleido_platform_runtime" "bnr" {
  count = var.besu_node_count
  name = "${var.environment_name}_besu_runtime_${count.index+1}"
  environment = kaleido_platform_environment.env_0.id
  type = "BesuRuntime"
  config_json = jsonencode({
    size = "medium"
    zone = "us-east-1"
  })
}

// Create Besu nodes using the structured resource
resource "kaleido_platform_besunode_service" "besu_nodes" {
  count = var.besu_node_count
  name = "${var.environment_name}_besu_node_${count.index+1}"
  environment = kaleido_platform_environment.env_0.id
  runtime = kaleido_platform_runtime.bnr[count.index].id
  stack_id = kaleido_platform_stack.chain_infra_besu_stack.id
  
  network = {
    id = kaleido_platform_besu_network.besu_net.id
  }
  
  // Node configuration
  mode = "active"
  signer = true
  log_level = "INFO"
  sync_mode = "FULL"
  data_storage_format = "FOREST"
  
  // Storage configuration
  storage = {
    size = "20Gi"
  }
  
  // Enable additional APIs
  apis_enabled = ["TRACE", "DEBUG"]
  
  // Gas configuration
  gas_price = "0"
  target_gas_limit = 30000000
  
  // Custom Besu arguments
  custom_besu_args = [
    "--tx-pool-retention-hours=999",
    "--tx-pool-limit-by-account-percentage=0.1"
  ]
  
  depends_on = [kaleido_platform_besu_network.besu_net]
}

// Output the network information
output "besu_network_id" {
  value = kaleido_platform_besu_network.besu_net.id
}

output "besu_network_info" {
  value = kaleido_platform_besu_network.besu_net.info
}

output "besu_node_endpoints" {
  value = { for idx, node in kaleido_platform_besunode_service.besu_nodes : 
    "node_${idx+1}" => node.endpoints
  }
}

output "besu_node_connectivity" {
  value = { for idx, node in kaleido_platform_besunode_service.besu_nodes : 
    "node_${idx+1}" => node.connectivity_json
  }
  sensitive = true
} 