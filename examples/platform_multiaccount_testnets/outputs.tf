# ============================================================================
# Child Account Outputs
# ============================================================================

output "child_accounts" {
  description = "Details of all created child accounts"
  value = {
    for i in range(var.child_account_count) : "${var.account_name_prefix}${i + 1}" => {
      id               = kaleido_platform_account.child_accounts[i].id
      name             = kaleido_platform_account.child_accounts[i].name
      account_id       = kaleido_platform_account.child_accounts[i].account_id
      api_url          = "https://${var.account_name_prefix}${i + 1}.kaleido.dev"
      oidc_client_id   = kaleido_platform_account.child_accounts[i].oidc_client_id
      environment_id   = kaleido_platform_environment.child_environments[i].id
      role             = i == 0 ? "originator" : "joiner"
    }
  }
}

output "originator_account" {
  description = "Details of the originator (hub) account"
  value = {
    id               = kaleido_platform_account.child_accounts[0].id
    name             = kaleido_platform_account.child_accounts[0].name
    account_id       = kaleido_platform_account.child_accounts[0].account_id
    api_url          = "https://${var.account_name_prefix}1.kaleido.dev"
    environment_id   = kaleido_platform_environment.child_environments[0].id
    besu_network_id  = kaleido_platform_network.originator_besu_network.id
    ipfs_network_id  = kaleido_platform_network.originator_ipfs_network.id
    paladin_network_id = kaleido_platform_network.originator_paladin_network.id
  }
}

output "joiner_accounts" {
  description = "Details of all joiner (spoke) accounts"
  value = {
    for i in range(var.child_account_count - 1) : "${var.account_name_prefix}${i + 2}" => {
      id               = kaleido_platform_account.child_accounts[i + 1].id
      name             = kaleido_platform_account.child_accounts[i + 1].name
      account_id       = kaleido_platform_account.child_accounts[i + 1].account_id
      api_url          = "https://${var.account_name_prefix}${i + 2}.kaleido.dev"
      note             = "Network resources would be created following the originator pattern"
    }
  }
}

# ============================================================================
# Network Infrastructure Outputs
# ============================================================================

output "network_topology" {
  description = "Network topology summary"
  value = {
    total_accounts     = var.child_account_count
    originator_count   = 1
    joiner_count       = var.child_account_count - 1
    topology_type      = "full-mesh"
    chain_id          = var.chain_id
    deployment_zone   = var.deployment_zone
    connections_required = var.child_account_count * (var.child_account_count - 1)
    paladin_p2p_enabled = true
  }
}

output "besu_networks" {
  description = "Besu network details"
  value = {
    originator = {
      network_id = kaleido_platform_network.originator_besu_network.id
      name       = kaleido_platform_network.originator_besu_network.name
      chain_id   = var.chain_id
      stack_id   = kaleido_platform_stack.originator_besu_stack.id
      signer_nodes = var.originator_signer_count
      peer_nodes   = var.originator_peer_count
    }
    note = "Joiner networks would be created with init_mode='manual' using originator bootstrap data"
  }
}

output "ipfs_networks" {
  description = "IPFS network details"
  value = {
    shared_swarm_key = random_id.shared_swarm_key.hex
    originator = {
      network_id = kaleido_platform_network.originator_ipfs_network.id
      name       = kaleido_platform_network.originator_ipfs_network.name
      stack_id   = kaleido_platform_stack.originator_ipfs_stack.id
    }
    note = "Joiner IPFS networks would use the same shared swarm key for connectivity"
  }
}

output "paladin_networks" {
  description = "Paladin network details"
  value = {
    originator = {
      network_id = kaleido_platform_network.originator_paladin_network.id
      name       = kaleido_platform_network.originator_paladin_network.name
      stack_id   = kaleido_platform_stack.originator_paladin_stack.id
      registry_admin = "registry.admin"
    }
    note = "Joiner Paladin networks would use 'existing' registry contract configuration"
  }
}

# ============================================================================
# Stack and Access Control Outputs
# ============================================================================

output "stack_details" {
  description = "Details of all chain infrastructure stacks"
  value = {
    originator = {
      besu_stack_id    = kaleido_platform_stack.originator_besu_stack.id
      ipfs_stack_id    = kaleido_platform_stack.originator_ipfs_stack.id
      paladin_stack_id = kaleido_platform_stack.originator_paladin_stack.id
      users_group      = kaleido_platform_group.originator_users_group.name
    }
    note = "Joiner stacks would follow the same pattern for additional accounts"
  }
}

# ============================================================================
# Network Connector Outputs
# ============================================================================

output "network_connections" {
  description = "Full mesh network connection configuration"
  value = {
    topology_type = "full-mesh"
    originator_account = "${var.account_name_prefix}1"
    joiner_accounts = [for i in range(var.child_account_count - 1) : "${var.account_name_prefix}${i + 2}"]
    total_connections = var.child_account_count * (var.child_account_count - 1)
    connections_per_network = {
      besu = var.child_account_count * (var.child_account_count - 1)
      ipfs = var.child_account_count * (var.child_account_count - 1)
      paladin = var.child_account_count * (var.child_account_count - 1)
    }
    submodules_used = ["joiner_account", "network_connectors"]
    note = "Full mesh connectivity implemented for Paladin P2P selective disclosure requirements"
  }
}

# ============================================================================
# Service and Runtime Outputs
# ============================================================================

output "service_endpoints" {
  description = "Service endpoints for applications"
  value = {
    originator_services = {
      besu_signers = [
        for i in range(var.originator_signer_count) : 
        kaleido_platform_service.originator_besu_signer_service[i].id
      ]
      besu_peers = [
        for i in range(var.originator_peer_count) : 
        kaleido_platform_service.originator_besu_peer_service[i].id
      ]
      ipfs_node     = kaleido_platform_service.originator_ipfs_service.id
      paladin_node  = kaleido_platform_service.originator_paladin_service.id
      gateway       = var.gateway_count > 0 ? kaleido_platform_service.originator_gateway_service[0].id : null
    }
    joiner_services = {
      for i in range(var.child_account_count - 1) : "${var.account_name_prefix}${i + 2}" => {
        besu_peer    = module.joiner_accounts[i].besu_peer_service.id
        ipfs_node    = module.joiner_accounts[i].ipfs_service.id
        paladin_node = module.joiner_accounts[i].paladin_service.id
        gateway      = var.gateway_count > 0 ? module.joiner_accounts[i].gateway_service.id : null
        environment  = module.joiner_accounts[i].environment.id
      }
    }
  }
}

# Submodule outputs for operational visibility
output "submodule_status" {
  description = "Status and details of submodules"
  value = {
    joiner_accounts_created = var.child_account_count - 1
    joiner_account_modules = {
      for i in range(var.child_account_count - 1) : "${var.account_name_prefix}${i + 2}" => {
        besu_network_id = module.joiner_accounts[i].besu_network.id
        ipfs_network_id = module.joiner_accounts[i].ipfs_network.id
        paladin_network_id = module.joiner_accounts[i].paladin_network.id
        environment_id = module.joiner_accounts[i].environment.id
        users_group = module.joiner_accounts[i].users_group.name
      }
    }
    network_connectors = module.network_connectors.connection_config
  }
}

# ============================================================================
# Bootstrap and Configuration Outputs
# ============================================================================

output "bootstrap_data" {
  description = "Bootstrap data for manual network setup (if needed)"
  value = {
    originator_bootstrap_available = data.kaleido_platform_network_bootstrap_data.originator_bootstrap.bootstrap_files != null
    shared_swarm_key_id           = random_id.shared_swarm_key.id
    identity_provider_id          = kaleido_platform_identity_provider.kaleido_id.id
  }
}

# ============================================================================
# Deployment Summary
# ============================================================================

output "deployment_summary" {
  description = "High-level summary of the deployment"
  value = {
    accounts_created              = var.child_account_count
    networks_per_account         = 3  # Besu, IPFS, Paladin
    total_networks               = var.child_account_count * 3
    originator_account           = "${var.account_name_prefix}1"
    joiner_accounts              = [for i in range(var.child_account_count - 1) : "${var.account_name_prefix}${i + 2}"]
    deployment_zone              = var.deployment_zone
    topology                     = "full-mesh"
    paladin_p2p_selective_disclosure = true
    oauth_provider_configured    = true
    kubernetes_service_account_trust_enabled = true
    stack_access_configured      = true
    submodules_utilized          = ["joiner_account", "network_connectors"]
  }
}

# ============================================================================
# Operational Information
# ============================================================================

output "operational_info" {
  description = "Information useful for operations and monitoring"
  value = {
    chain_infrastructure_stacks = {
      total = var.child_account_count * 3
      by_account = {
        for i in range(var.child_account_count) : "${var.account_name_prefix}${i + 1}" => {
          besu_stack    = i == 0 ? kaleido_platform_stack.originator_besu_stack.id : kaleido_platform_stack.joiner_besu_stacks[i - 1].id
          ipfs_stack    = i == 0 ? kaleido_platform_stack.originator_ipfs_stack.id : kaleido_platform_stack.joiner_ipfs_stacks[i - 1].id
          paladin_stack = i == 0 ? kaleido_platform_stack.originator_paladin_stack.id : kaleido_platform_stack.joiner_paladin_stacks[i - 1].id
        }
      }
    }
    users_groups_created = var.child_account_count
    total_nodes = {
      besu_signers = var.originator_signer_count
      besu_peers   = var.originator_peer_count + (var.child_account_count - 1)  # 1 peer per joiner
      ipfs_nodes   = var.child_account_count
      paladin_nodes = var.child_account_count
      gateways     = var.gateway_count > 0 ? var.child_account_count : 0
    }
  }
}