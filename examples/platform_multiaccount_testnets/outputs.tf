# ============================================================================
# Child Account Outputs
# ============================================================================

output "child_accounts_summary" {
  description = "Summary of all created child accounts"
  value = {
    for i in range(var.child_account_count) : "${var.account_name_prefix}${i + 1}" => {
      id   = kaleido_platform_account.child_accounts[i].id
      name = kaleido_platform_account.child_accounts[i].name
      role = i == 0 ? "originator" : "joiner"
    }
  }
}

output "originator_account_details" {
  description = "Detailed information for the originator account"
  value = {
    id             = kaleido_platform_account.child_accounts[0].id
    name           = kaleido_platform_account.child_accounts[0].name
    environment_id = kaleido_platform_environment.originator_environment.id
  }
}

output "joiner_account_details" {
  description = "Detailed information for all joiner accounts"
  value = {
    for i in range(var.child_account_count - 1) : "joiner_${i + 1}" => {
      id             = kaleido_platform_account.child_accounts[i + 1].id
      name           = kaleido_platform_account.child_accounts[i + 1].name
      environment_id = module.joiner_1[0].environment.id # This will need to be made dynamic
    }
  }
}

# ============================================================================
# Network and Service Outputs
# ============================================================================

output "originator_network_services" {
  description = "Network and service details for the originator account"
  value = {
    besu_network_id    = kaleido_platform_network.originator_besu_network.id
    ipfs_network_id    = kaleido_platform_network.originator_ipfs_network.id
    paladin_network_id = kaleido_platform_network.originator_paladin_network.id
    besu_signer_ids = [
      for s in kaleido_platform_service.originator_besu_signer_service : s.id
    ]
    ipfs_service_id    = kaleido_platform_service.originator_ipfs_service.id
    paladin_service_id = kaleido_platform_service.originator_paladin_service.id
  }
}

output "joiner_1_network_services" {
  description = "Network and service details for Joiner 1 (Account 2)"
  value = var.child_account_count > 1 ? {
    besu_network_id    = module.joiner_1[0].besu_network.id
    ipfs_network_id    = module.joiner_1[0].ipfs_network.id
    paladin_network_id = module.joiner_1[0].paladin_network.id
    besu_peer_id       = module.joiner_1[0].besu_peer_service.id
    ipfs_service_id    = module.joiner_1[0].ipfs_service.id
    paladin_service_id = module.joiner_1[0].paladin_service.id
  } : null
}

output "joiner_2_network_services" {
  description = "Network and service details for Joiner 2 (Account 3)"
  value = var.child_account_count > 2 ? {
    besu_network_id    = module.joiner_2[0].besu_network.id
    ipfs_network_id    = module.joiner_2[0].ipfs_network.id
    paladin_network_id = module.joiner_2[0].paladin_network.id
    besu_peer_id       = module.joiner_2[0].besu_peer_service.id
    ipfs_service_id    = module.joiner_2[0].ipfs_service.id
    paladin_service_id = module.joiner_2[0].paladin_service.id
  } : null
}

# ... (Add similar blocks for joiner_3 and joiner_4 if needed) ...

# ============================================================================
# Deployment Summary
# ============================================================================

output "deployment_summary" {
  description = "High-level summary of the deployment"
  value = {
    accounts_created     = var.child_account_count
    topology             = "full-mesh"
    originator_account   = "${var.account_name_prefix}1"
    joiner_accounts      = [for i in range(var.child_account_count - 1) : "${var.account_name_prefix}${i + 2}"]
    kubernetes_trust     = true
  }
}
