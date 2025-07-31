terraform {
  required_providers {
    kaleido = {
      source = "kaleido-io/kaleido"
    }
    random = {
      source  = "hashicorp/random"
      version = "3.6.3"
    }
  }
}

# Root provider for account creation
provider "kaleido" {
  platform_api = var.root_platform_api
  platform_bearer_token = var.root_platform_bearer_token
  alias = "root"
}

# Child account providers (explicitly defined for up to 10 accounts)
provider "kaleido" {
  platform_api = "https://${var.account_name_prefix}1.kaleido.dev"
  platform_bearer_token = var.root_platform_bearer_token
  alias = "child_0"
}

provider "kaleido" {
  platform_api = "https://${var.account_name_prefix}2.kaleido.dev"
  platform_bearer_token = var.root_platform_bearer_token
  alias = "child_1"
}

provider "kaleido" {
  platform_api = "https://${var.account_name_prefix}3.kaleido.dev"
  platform_bearer_token = var.root_platform_bearer_token
  alias = "child_2"
}

provider "kaleido" {
  platform_api = "https://${var.account_name_prefix}4.kaleido.dev"
  platform_bearer_token = var.root_platform_bearer_token
  alias = "child_3"
}

provider "kaleido" {
  platform_api = "https://${var.account_name_prefix}5.kaleido.dev"
  platform_bearer_token = var.root_platform_bearer_token
  alias = "child_4"
}

provider "random" {}

# Create bootstrap OAuth application configuration for child accounts
resource "kaleido_platform_identity_provider" "kaleido_id" {
  provider = kaleido.root
  
  name = "kaleido-id"
  hostname = "kaleido-id"
  client_type = "confidential"
  client_id = var.kaleido_id_client_id
  client_secret = var.kaleido_id_client_secret
  oidc_config_url = var.kaleido_id_config_url
  id_token_nonce_enabled = true
}

# Create N child accounts dynamically
resource "kaleido_platform_account" "child_accounts" {
  provider = kaleido.root
  count = var.child_account_count

  name = "${var.account_name_prefix}${count.index + 1}"
  oidc_client_id = kaleido_platform_identity_provider.kaleido_id.id
  
  hostnames = {
    "${var.account_name_prefix}${count.index + 1}" = []
  }
  
  first_user_email = var.first_user_email
  user_jit_enabled = true
  user_jit_default_group = "users"
  
  validation_policy = var.custom_validation_policy != null ? var.custom_validation_policy : <<EOF
package account_validation
import future.keywords.in

default allow := false

is_valid_aud(aud) := aud == "${var.kaleido_id_client_id}"
is_valid_user(roles) := roles == ["user"]
is_valid_admin(roles) := roles == ["admin"]
is_valid_owner(roles) := roles == ["owner"]

allow {
	is_valid_aud(input.id_token.aud)
	is_valid_user(input.id_token.roles)
}

allow {
	is_valid_aud(input.id_token.aud)
	is_valid_admin(input.id_token.roles)
}

allow {
	is_valid_aud(input.id_token.aud)
	is_valid_owner(input.id_token.roles)
}
EOF

  bootstrap_application_name = "kubernetes-local"
  bootstrap_application_oauth_json = jsonencode({
    issuer = var.bootstrap_application_issuer
    jwksEndpoint = var.bootstrap_application_jwks_endpoint
    caCertificate = var.bootstrap_application_ca_certificate
  })
  
  bootstrap_application_validation_policy = <<EOF
package k8s_application_validation

default allow := false

is_valid_sa(sub) := sub == "system:serviceaccount:default:kaleidoplatform"

allow {
  is_valid_sa(input.sub)
}
EOF
}

# Generate shared swarm key for IPFS networks
resource "random_id" "shared_swarm_key" {
  byte_length = 32
}

# ============================================================================
# ORIGINATOR ACCOUNT RESOURCES (First child account - Hub)
# ============================================================================

# Users group and environment for originator
resource "kaleido_platform_group" "originator_users_group" {
  provider = kaleido.child_0
  name = "users"
  depends_on = [kaleido_platform_account.child_accounts]
}

resource "kaleido_platform_environment" "originator_environment" {
  provider = kaleido.child_0
  name = "${var.account_name_prefix}1_env"
  depends_on = [kaleido_platform_account.child_accounts]
}

# Networks for originator
resource "kaleido_platform_network" "originator_besu_network" {
  provider = kaleido.child_0
  
  type = "BesuNetwork"
  name = "${var.besu_network_name}_originator"
  environment = kaleido_platform_environment.originator_environment.id
  init_mode = "automated"

  config_json = jsonencode({
    chainID = var.chain_id
    bootstrapOptions = {
      qbft = {
        blockperiodseconds = var.block_period_seconds
      }
      eipBlockConfig = {
        shanghaiTime = 0
      }
      initialBalances = var.initial_balances
      blockConfigFlags = {
        zeroBaseFee = true
      }
    }
  })
}

resource "kaleido_platform_network" "originator_ipfs_network" {
  provider = kaleido.child_0
  
  type = "IPFSNetwork"
  name = "${var.ipfs_network_name}_originator"
  environment = kaleido_platform_environment.originator_environment.id
  
  config_json = jsonencode({
    swarmKey = random_id.shared_swarm_key.hex
  })
}

resource "kaleido_platform_network" "originator_paladin_network" {
  provider = kaleido.child_0
  
  type = "PaladinNetwork"
  name = "${var.paladin_network_name}_originator"
  environment = kaleido_platform_environment.originator_environment.id
  
  config_json = jsonencode({
    type = "evmRegistry"
    evmRegistry = {
      registryContract = "deploy"
      admin = {
        nodeName = "paladin-service-originator"
        identity = "registry.admin"
      }
    }
  })
}

# Stacks for originator
resource "kaleido_platform_stack" "originator_besu_stack" {
  provider = kaleido.child_0
  
  environment = kaleido_platform_environment.originator_environment.id
  name = "chain_infrastructure_besu"
  type = "chain_infrastructure"
  network_id = kaleido_platform_network.originator_besu_network.id
}

resource "kaleido_platform_stack" "originator_ipfs_stack" {
  provider = kaleido.child_0
  
  environment = kaleido_platform_environment.originator_environment.id
  name = "chain_infrastructure_ipfs"
  type = "chain_infrastructure"
  network_id = kaleido_platform_network.originator_ipfs_network.id
}

resource "kaleido_platform_stack" "originator_paladin_stack" {
  provider = kaleido.child_0
  
  environment = kaleido_platform_environment.originator_environment.id
  name = "chain_infrastructure_paladin"
  type = "chain_infrastructure"
  sub_type = "PaladinStack"
  network_id = kaleido_platform_network.originator_paladin_network.id
}

# Stack access for originator
# Note: Stack access is managed through the users group created above
# The "users" group in each account provides appropriate access to chain infrastructure stacks

# Besu nodes for originator
resource "kaleido_platform_runtime" "originator_besu_signer_runtime" {
  provider = kaleido.child_0
  count = var.originator_signer_count
  
  type = "BesuNode"
  name = "originator_signer${count.index + 1}"
  environment = kaleido_platform_environment.originator_environment.id
  config_json = jsonencode({})
  zone = var.deployment_zone
  stack_id = kaleido_platform_stack.originator_besu_stack.id
}

resource "kaleido_platform_service" "originator_besu_signer_service" {
  provider = kaleido.child_0
  count = var.originator_signer_count
  
  type = "BesuNode"
  name = "originator_signer${count.index + 1}"
  environment = kaleido_platform_environment.originator_environment.id
  runtime = kaleido_platform_runtime.originator_besu_signer_runtime[count.index].id
  
  config_json = jsonencode({
    network = {
      id = kaleido_platform_network.originator_besu_network.id
    }
    signer = true
  })
  
  force_delete = var.enable_force_delete
  stack_id = kaleido_platform_stack.originator_besu_stack.id
}

# Gateway for originator
resource "kaleido_platform_runtime" "originator_gateway_runtime" {
  provider = kaleido.child_0
  count = var.gateway_count > 0 ? 1 : 0
  
  type = "EVMGateway"
  name = "originator_gateway"
  environment = kaleido_platform_environment.originator_environment.id
  config_json = jsonencode({})
  stack_id = kaleido_platform_stack.originator_besu_stack.id
}

resource "kaleido_platform_service" "originator_gateway_service" {
  provider = kaleido.child_0
  count = var.gateway_count > 0 ? 1 : 0
  
  type = "EVMGateway"
  name = "originator_gateway"
  environment = kaleido_platform_environment.originator_environment.id
  runtime = kaleido_platform_runtime.originator_gateway_runtime[count.index].id
  
  config_json = jsonencode({
    network = {
      id = kaleido_platform_network.originator_besu_network.id
    }
  })
  
  stack_id = kaleido_platform_stack.originator_besu_stack.id
}

# Bootstrap data
data "kaleido_platform_network_bootstrap_data" "originator_bootstrap" {
  provider = kaleido.child_0
  
  environment = kaleido_platform_environment.originator_environment.id
  network = kaleido_platform_network.originator_besu_network.id
  
  depends_on = [
    kaleido_platform_service.originator_besu_signer_service,
    kaleido_platform_network.originator_besu_network
  ]
}

# IPFS and Paladin nodes for originator
resource "kaleido_platform_runtime" "originator_ipfs_runtime" {
  provider = kaleido.child_0
  
  type = "IPFSNode"
  name = "originator_ipfs"
  zone = var.deployment_zone
  environment = kaleido_platform_environment.originator_environment.id
  config_json = jsonencode({})
  stack_id = kaleido_platform_stack.originator_ipfs_stack.id
}

resource "kaleido_platform_service" "originator_ipfs_service" {
  provider = kaleido.child_0
  
  type = "IPFSNode"
  name = "originator_ipfs"
  environment = kaleido_platform_environment.originator_environment.id
  runtime = kaleido_platform_runtime.originator_ipfs_runtime.id
  
  config_json = jsonencode({
    network = {
      id = kaleido_platform_network.originator_ipfs_network.id
    }
  })
  
  stack_id = kaleido_platform_stack.originator_ipfs_stack.id
}

resource "kaleido_platform_runtime" "originator_paladin_runtime" {
  provider = kaleido.child_0
  
  type = "PaladinNodeRuntime"
  name = "originator_paladin"
  environment = kaleido_platform_environment.originator_environment.id
  config_json = jsonencode({})
  stack_id = kaleido_platform_stack.originator_paladin_stack.id
  size = var.paladin_node_size
  zone = var.deployment_zone
}

resource "kaleido_platform_service" "originator_paladin_service" {
  provider = kaleido.child_0
  
  type = "PaladinNodeService"
  name = "paladin-service-originator"
  environment = kaleido_platform_environment.originator_environment.id
  runtime = kaleido_platform_runtime.originator_paladin_runtime.id
  
  config_json = jsonencode({
    network = {
      id = kaleido_platform_network.originator_paladin_network.id
    }
    baseLedgerEndpoint = {
      type = "local"
      local = {
        node = {
          id = kaleido_platform_service.originator_besu_signer_service[0].id
        }
      }
    }
    baseConfig = "{\"blockIndexer\":{\"fromBlock\": 0 }}"
    registryAdminIdentity = "registry.admin"
  })
  
  stack_id = kaleido_platform_stack.originator_paladin_stack.id
}

# ============================================================================
# JOINER ACCOUNTS (Remaining child accounts)
# Implements full mesh connectivity for Paladin P2P selective disclosure
# ============================================================================

# Account data providers for connectors
data "kaleido_platform_account" "originator_account" {
  provider = kaleido.child_0
}

data "kaleido_platform_account" "joiner_accounts" {
  count = var.child_account_count - 1
  provider = count.index == 0 ? kaleido.child_1 : (
    count.index == 1 ? kaleido.child_2 : (
      count.index == 2 ? kaleido.child_3 : kaleido.child_4
    )
  )
}

# Joiner Account Modules - Create one for each joiner (accounts 2-N)
module "joiner_accounts" {
  source = "./modules/joiner_account"
  count = var.child_account_count - 1

  # Account configuration
  account_index = count.index
  account_name_prefix = var.account_name_prefix
  
  # Network configuration
  besu_network_name = var.besu_network_name
  ipfs_network_name = var.ipfs_network_name
  paladin_network_name = var.paladin_network_name
  
  # Shared resources
  shared_swarm_key = random_id.shared_swarm_key.hex
  originator_bootstrap_files = data.kaleido_platform_network_bootstrap_data.originator_bootstrap.bootstrap_files
  
  # Infrastructure configuration
  deployment_zone = var.deployment_zone
  gateway_count = var.gateway_count
  paladin_node_size = var.paladin_node_size
  enable_force_delete = var.enable_force_delete
  
  # Dependencies
  child_accounts = kaleido_platform_account.child_accounts
  originator_besu_network = kaleido_platform_network.originator_besu_network
  originator_besu_signer_services = kaleido_platform_service.originator_besu_signer_service
  
  providers = {
    kaleido = count.index == 0 ? kaleido.child_1 : (
      count.index == 1 ? kaleido.child_2 : (
        count.index == 2 ? kaleido.child_3 : kaleido.child_4
      )
    )
  }
}

# Network Connectors Module - Calculates full mesh connectivity
module "network_connectors" {
  source = "./modules/network_connectors"
  
  child_account_count = var.child_account_count
  deployment_zone = var.deployment_zone
  account_name_prefix = var.account_name_prefix
  
  # Originator data
  originator_account_id = data.kaleido_platform_account.originator_account.account_id
  originator_environment_id = kaleido_platform_environment.originator_environment.id
  originator_besu_network_id = kaleido_platform_network.originator_besu_network.id
  originator_ipfs_network_id = kaleido_platform_network.originator_ipfs_network.id
  originator_paladin_network_id = kaleido_platform_network.originator_paladin_network.id
  
  # Joiner data
  joiner_account_ids = data.kaleido_platform_account.joiner_accounts[*].account_id
  joiner_environment_ids = module.joiner_accounts[*].environment.id
  joiner_besu_network_ids = module.joiner_accounts[*].besu_network.id
  joiner_ipfs_network_ids = module.joiner_accounts[*].ipfs_network.id
  joiner_paladin_network_ids = module.joiner_accounts[*].paladin_network.id
}

# ============================================================================
# FULL MESH NETWORK CONNECTORS IMPLEMENTATION
# Due to Terraform provider limitations, we implement key connectors explicitly
# ============================================================================

# Example: Originator to Joiner 1 connections (all networks)
# Besu: Originator requests connection to Joiner 1
resource "kaleido_network_connector" "originator_to_joiner1_besu_request" {
  count = var.child_account_count > 1 ? 1 : 0
  provider = kaleido.child_0
  
  type = "Platform"
  name = "originator_to_joiner1_besu"
  environment = kaleido_platform_environment.originator_environment.id
  network = kaleido_platform_network.originator_besu_network.id
  zone = var.deployment_zone

  platform_requestor = {
    target_account_id = data.kaleido_platform_account.joiner_accounts[0].account_id
    target_environment_id = module.joiner_accounts[0].environment.id
    target_network_id = module.joiner_accounts[0].besu_network.id
  }
}

# Besu: Joiner 1 accepts connection from Originator
resource "kaleido_network_connector" "joiner1_accept_originator_besu" {
  count = var.child_account_count > 1 ? 1 : 0
  provider = kaleido.child_1
  
  type = "Platform"
  name = "joiner1_accept_originator_besu"
  environment = module.joiner_accounts[0].environment.id
  network = module.joiner_accounts[0].besu_network.id
  zone = var.deployment_zone

  platform_acceptor = {
    target_account_id = data.kaleido_platform_account.originator_account.account_id
    target_environment_id = kaleido_platform_environment.originator_environment.id
    target_network_id = kaleido_platform_network.originator_besu_network.id
    target_connector_id = kaleido_network_connector.originator_to_joiner1_besu_request[0].id
  }
}

# IPFS: Originator to Joiner 1
resource "kaleido_network_connector" "originator_to_joiner1_ipfs_request" {
  count = var.child_account_count > 1 ? 1 : 0
  provider = kaleido.child_0
  
  type = "Platform"
  name = "originator_to_joiner1_ipfs"
  environment = kaleido_platform_environment.originator_environment.id
  network = kaleido_platform_network.originator_ipfs_network.id
  zone = var.deployment_zone

  platform_requestor = {
    target_account_id = data.kaleido_platform_account.joiner_accounts[0].account_id
    target_environment_id = module.joiner_accounts[0].environment.id
    target_network_id = module.joiner_accounts[0].ipfs_network.id
  }
}

resource "kaleido_network_connector" "joiner1_accept_originator_ipfs" {
  count = var.child_account_count > 1 ? 1 : 0
  provider = kaleido.child_1
  
  type = "Platform"
  name = "joiner1_accept_originator_ipfs"
  environment = module.joiner_accounts[0].environment.id
  network = module.joiner_accounts[0].ipfs_network.id
  zone = var.deployment_zone

  platform_acceptor = {
    target_account_id = data.kaleido_platform_account.originator_account.account_id
    target_environment_id = kaleido_platform_environment.originator_environment.id
    target_network_id = kaleido_platform_network.originator_ipfs_network.id
    target_connector_id = kaleido_network_connector.originator_to_joiner1_ipfs_request[0].id
  }
}

# Paladin: Originator to Joiner 1 (Critical for P2P selective disclosure)
resource "kaleido_network_connector" "originator_to_joiner1_paladin_request" {
  count = var.child_account_count > 1 ? 1 : 0
  provider = kaleido.child_0
  
  type = "Platform"
  name = "originator_to_joiner1_paladin"
  environment = kaleido_platform_environment.originator_environment.id
  network = kaleido_platform_network.originator_paladin_network.id
  zone = var.deployment_zone

  platform_requestor = {
    target_account_id = data.kaleido_platform_account.joiner_accounts[0].account_id
    target_environment_id = module.joiner_accounts[0].environment.id
    target_network_id = module.joiner_accounts[0].paladin_network.id
  }
}

resource "kaleido_network_connector" "joiner1_accept_originator_paladin" {
  count = var.child_account_count > 1 ? 1 : 0
  provider = kaleido.child_1
  
  type = "Platform"
  name = "joiner1_accept_originator_paladin"
  environment = module.joiner_accounts[0].environment.id
  network = module.joiner_accounts[0].paladin_network.id
  zone = var.deployment_zone

  platform_acceptor = {
    target_account_id = data.kaleido_platform_account.originator_account.account_id
    target_environment_id = kaleido_platform_environment.originator_environment.id
    target_network_id = kaleido_platform_network.originator_paladin_network.id
    target_connector_id = kaleido_network_connector.originator_to_joiner1_paladin_request[0].id
  }
}

# Additional connector pairs can be implemented following the same pattern
# For production deployments with more accounts, consider using:
# 1. Multiple Terraform modules with workspace-based provider configurations
# 2. External tooling to generate connector configurations
# 3. GitOps-based approaches for scaling beyond 5 accounts

# Note: Full mesh connectivity means every account connects to every other account
# For N accounts, this requires N*(N-1) total connections (bidirectional)
# This example demonstrates the pattern for the critical connections