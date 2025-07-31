terraform {
  required_providers {
    kaleido = {
      source = "kaleido-io/kaleido"
    }
  }
}

# Variables
variable "child_account_count" {
  description = "Total number of child accounts"
  type = number
}

variable "deployment_zone" {
  description = "Deployment zone for connectors"
  type = string
}

variable "account_name_prefix" {
  description = "Prefix for account names"
  type = string
}

# Originator data
variable "originator_account_id" {
  description = "Originator account ID"
  type = string
}

variable "originator_environment_id" {
  description = "Originator environment ID"
  type = string
}

variable "originator_besu_network_id" {
  description = "Originator Besu network ID"
  type = string
}

variable "originator_ipfs_network_id" {
  description = "Originator IPFS network ID"
  type = string
}

variable "originator_paladin_network_id" {
  description = "Originator Paladin network ID"
  type = string
}

# Joiner data arrays
variable "joiner_account_ids" {
  description = "List of joiner account IDs"
  type = list(string)
}

variable "joiner_environment_ids" {
  description = "List of joiner environment IDs"
  type = list(string)
}

variable "joiner_besu_network_ids" {
  description = "List of joiner Besu network IDs"
  type = list(string)
}

variable "joiner_ipfs_network_ids" {
  description = "List of joiner IPFS network IDs"
  type = list(string)
}

variable "joiner_paladin_network_ids" {
  description = "List of joiner Paladin network IDs"
  type = list(string)
}

# Provider aliases for different accounts (passed from parent)
# Since we can't dynamically create providers, we need these passed in

# ============================================================================
# FULL MESH CONNECTIVITY IMPLEMENTATION
# Paladin requires P2P networking to all fellow nodes for selective disclosure
# ============================================================================

locals {
  # Calculate all connection pairs for full mesh
  # For N accounts, each account connects to every other account
  connection_pairs = flatten([
    for i in range(var.child_account_count) : [
      for j in range(var.child_account_count) : {
        from_index = i
        to_index = j
        from_account = i == 0 ? "originator" : "joiner_${i - 1}"
        to_account = j == 0 ? "originator" : "joiner_${j - 1}"
        from_account_id = i == 0 ? var.originator_account_id : var.joiner_account_ids[i - 1]
        to_account_id = j == 0 ? var.originator_account_id : var.joiner_account_ids[j - 1]
        from_env_id = i == 0 ? var.originator_environment_id : var.joiner_environment_ids[i - 1]
        to_env_id = j == 0 ? var.originator_environment_id : var.joiner_environment_ids[j - 1]
        from_besu_network_id = i == 0 ? var.originator_besu_network_id : var.joiner_besu_network_ids[i - 1]
        to_besu_network_id = j == 0 ? var.originator_besu_network_id : var.joiner_besu_network_ids[j - 1]
        from_ipfs_network_id = i == 0 ? var.originator_ipfs_network_id : var.joiner_ipfs_network_ids[i - 1]
        to_ipfs_network_id = j == 0 ? var.originator_ipfs_network_id : var.joiner_ipfs_network_ids[j - 1]
        from_paladin_network_id = i == 0 ? var.originator_paladin_network_id : var.joiner_paladin_network_ids[i - 1]
        to_paladin_network_id = j == 0 ? var.originator_paladin_network_id : var.joiner_paladin_network_ids[j - 1]
      }
      if i != j  # Don't connect to self
    ]
  ])
  
  # Filter to only requestor connections (each pair will have a requestor and acceptor)
  # We'll implement the requestor side here, acceptor side will be in main.tf with proper providers
  requestor_connections = [
    for pair in local.connection_pairs : pair
    if pair.from_index < pair.to_index  # Only create one direction per pair
  ]
}

# Output the connection configuration for main.tf to implement
# Due to Terraform provider limitations, actual resources must be created in main.tf
output "connection_config" {
  description = "Full mesh connection configuration for implementation in main.tf"
  value = {
    total_connections = length(local.connection_pairs)
    requestor_connections = length(local.requestor_connections) 
    connection_pairs = local.connection_pairs
    requestor_pairs = local.requestor_connections
    
    # Connection summary for each account
    connection_summary = {
      for i in range(var.child_account_count) : "${var.account_name_prefix}${i + 1}" => {
        connects_to = [
          for j in range(var.child_account_count) : "${var.account_name_prefix}${j + 1}"
          if i != j
        ]
        outbound_connections = length([for j in range(var.child_account_count) : j if i != j])
      }
    }
    
    # Network-specific connection mappings
    besu_connections = {
      for pair in local.requestor_connections : "${pair.from_account}_to_${pair.to_account}" => {
        requestor_account_id = pair.from_account_id
        requestor_env_id = pair.from_env_id
        requestor_network_id = pair.from_besu_network_id
        target_account_id = pair.to_account_id
        target_env_id = pair.to_env_id
        target_network_id = pair.to_besu_network_id
      }
    }
    
    ipfs_connections = {
      for pair in local.requestor_connections : "${pair.from_account}_to_${pair.to_account}" => {
        requestor_account_id = pair.from_account_id
        requestor_env_id = pair.from_env_id
        requestor_network_id = pair.from_ipfs_network_id
        target_account_id = pair.to_account_id
        target_env_id = pair.to_env_id
        target_network_id = pair.to_ipfs_network_id
      }
    }
    
    paladin_connections = {
      for pair in local.requestor_connections : "${pair.from_account}_to_${pair.to_account}" => {
        requestor_account_id = pair.from_account_id
        requestor_env_id = pair.from_env_id
        requestor_network_id = pair.from_paladin_network_id
        target_account_id = pair.to_account_id
        target_env_id = pair.to_env_id
        target_network_id = pair.to_paladin_network_id
      }
    }
  }
}

# Template for network connector resources that would be implemented in main.tf
output "connector_resource_template" {
  description = "Template for implementing network connectors in main.tf"
  value = {
    example_besu_requestor = {
      type = "Platform"
      name = "account1_to_account2_besu"
      environment = "env_id"
      network = "network_id"
      zone = var.deployment_zone
      platform_requestor = {
        target_account_id = "target_account_id"
        target_environment_id = "target_env_id"
        target_network_id = "target_network_id"
      }
    }
    
    example_besu_acceptor = {
      type = "Platform"
      name = "account2_accept_account1_besu"
      environment = "env_id"
      network = "network_id"
      zone = var.deployment_zone
      platform_acceptor = {
        target_account_id = "target_account_id"
        target_environment_id = "target_env_id"
        target_network_id = "target_network_id"
        target_connector_id = "requestor_connector_id"
      }
    }
  }
}