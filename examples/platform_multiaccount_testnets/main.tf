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
  platform_api = "https://${var.root_platform_account_name}.${var.platform_instance_domain}"
  platform_username = var.root_platform_username
  platform_password = var.root_platform_password
  alias = "root"
}

// TODO make domain variables
# Child account providers (explicitly defined for up to 5 accounts)
provider "kaleido" {
  platform_api = "https://${var.account_name_prefix}1.${var.platform_instance_domain}"
  platform_bearer_token = var.bootstrap_platform_bearer_token
  alias = "child_0"
}

provider "kaleido" {
  platform_api = "https://${var.account_name_prefix}2.${var.platform_instance_domain}"
  platform_bearer_token = var.bootstrap_platform_bearer_token
  alias = "child_1"
}

provider "kaleido" {
  platform_api = "https://${var.account_name_prefix}3.${var.platform_instance_domain}"
  platform_bearer_token = var.bootstrap_platform_bearer_token
  alias = "child_2"
}

provider "kaleido" {
  platform_api = "https://${var.account_name_prefix}4.${var.platform_instance_domain}"
  platform_bearer_token = var.bootstrap_platform_bearer_token
  alias = "child_3"
}

provider "kaleido" {
  platform_api = "https://${var.account_name_prefix}5.${var.platform_instance_domain}"
  platform_bearer_token = var.bootstrap_platform_bearer_token
  alias = "child_4"
}

provider "random" {}

# Create bootstrap OAuth application configuration for child accounts
resource "kaleido_platform_identity_provider" "kaleido_id" {
  provider = kaleido.root
  
  // TODO make configurable
  name = var.child_identity_provider_name
  hostname = var.child_identity_provider_hostname
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

is_valid_sa(sub) := sub == "system:serviceaccount:${var.bootstrap_application_sa_namespace}:${var.bootstrap_application_sa_name}"

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

resource "kaleido_platform_runtime" "originator_cms_runtime" {
  provider = kaleido.child_0

  type = "ContractManager"
  name = "originator_cms"
  environment = kaleido_platform_environment.originator_environment.id
  config_json = jsonencode({})
  size = "Medium"
}

resource "kaleido_platform_service" "originator_cms_service" {
  provider = kaleido.child_0

  type = "ContractManager"
  name = "originator_cms"
  environment = kaleido_platform_environment.originator_environment.id
  runtime = kaleido_platform_runtime.originator_cms_runtime.id
  config_json = jsonencode({})
  // TODO proxy option ?
}

resource "kaleido_platform_runtime" "originator_kms_runtime" {
  provider = kaleido.child_0

  type = "KeyManager"
  name = "originator_kms"
  environment = kaleido_platform_environment.originator_environment.id
  config_json = jsonencode({})
}

resource "kaleido_platform_service" "originator_kms_service" {
  provider = kaleido.child_0

  type = "KeyManager"
  name = "originator_kms"
  environment = kaleido_platform_environment.originator_environment.id
  runtime = kaleido_platform_runtime.originator_kms_runtime.id
  config_json = jsonencode({})
}

resource "kaleido_platform_kms_wallet" "paladin_wallet" {
  provider = kaleido.child_0

  type = "hdwallet"
  name = "paladin_wallet"
  environment = kaleido_platform_environment.originator_environment.id
  service = kaleido_platform_service.originator_kms_service.id
  config_json = jsonencode({})
}

resource "kaleido_platform_kms_wallet" "admin_wallet" {
  provider = kaleido.child_0

  type = "hdwallet"
  name = "admin_wallet"
  environment = kaleido_platform_environment.originator_environment.id
  service = kaleido_platform_service.originator_kms_service.id
  config_json = jsonencode({})
}

resource "kaleido_platform_kms_key" "contract_deployer_key" {
  provider = kaleido.child_0

  name = "contract_deployer_key"
  environment = kaleido_platform_environment.originator_environment.id
  service = kaleido_platform_service.originator_kms_service.id
  wallet = kaleido_platform_kms_wallet.admin_wallet.name
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
  size = var.node_runtime_size
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
    customBesuArgs = ["--revert-reason-enabled=true"]
    dataStorageFormat = "BONSAI"
    logLevel = "DEBUG"
  })
  
  force_delete = var.enable_force_delete
  stack_id = kaleido_platform_stack.originator_besu_stack.id
}

resource "kaleido_platform_runtime" "originator_block_indexer" {
  provider = kaleido.child_0

  type = "BlockIndexer"
  name = "originator_block_indexer"
  config_json = jsonencode({})
  environment = kaleido_platform_environment.originator_environment.id
  stack_id = kaleido_platform_stack.originator_besu_stack.id
}

resource "kaleido_platform_service" "originator_block_indexer_service" {
  provider = kaleido.child_0

  type = "BlockIndexer"
  name = "originator_block_indexer"
  environment = kaleido_platform_environment.originator_environment.id
  runtime = kaleido_platform_runtime.originator_block_indexer.id
  config_json = jsonencode({
    node = {
      id = kaleido_platform_service.originator_besu_signer_service[0].id
    }
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
  size = var.node_runtime_size
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
  size = var.node_runtime_size
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
    keyManager = {
      id = kaleido_platform_service.originator_kms_service.id
    }
    baseLedgerEndpoint = {
      type = "local"
      local = {
        gateway = {
          id = kaleido_platform_service.originator_gateway_service[0].id
        }
      }
    }
    wallets = {
      kmsKeyStore = kaleido_platform_kms_wallet.paladin_wallet.name
    }
    registryAdminIdentity = "registry.admin" // TODO can this not be hardcoded ?
    baseConfig = jsonencode({
      blockIndexer = {
        fromBlock = 0
      }
      domains = {
        noto = {
          plugin = {
            library = "/app/domains/libnoto.so"
            type = "c-shared"
          }
          registryAddress = kaleido_platform_cms_action_deploy.contracts["noto_factory"].contract_address
        }
        pente = {
          plugin = {
            class = "io.kaleido.paladin.pente.domain.PenteDomainFactory"
            library = "/app/domains/pente.jar"
            type = "jar"
          }
          registryAddress = kaleido_platform_cms_action_deploy.contracts["pente_factory"].contract_address
        }
      }
    })
  })
  
  stack_id = kaleido_platform_stack.originator_paladin_stack.id
}

resource "kaleido_platform_stack" "originator_web3_middleware_stack" {
  provider = kaleido.child_0

  environment = kaleido_platform_environment.originator_environment.id
  name = "web3_middleware"
  type = "web3_middleware"
}

resource "kaleido_platform_runtime" "originator_tms_runtime" {
  provider = kaleido.child_0

  type = "TransactionManager"
  name = "originator_tms"
  environment = kaleido_platform_environment.originator_environment.id
  config_json = jsonencode({})
  stack_id = kaleido_platform_stack.originator_web3_middleware_stack.id
}

resource "kaleido_platform_service" "originator_tms_service" {
  provider = kaleido.child_0

  type = "TransactionManager"
  name = "originator_tms"
  environment = kaleido_platform_environment.originator_environment.id
  runtime = kaleido_platform_runtime.originator_tms_runtime.id
  config_json = jsonencode({
    keyManager = {
      id = kaleido_platform_service.originator_kms_service.id
    }
    type = "evm"
    evm = {
      confirmations = {
        required = 0
      }
      connector = {
        evmGateway = {
          id =  kaleido_platform_service.originator_gateway_service[0].id
        }
      }
    }
  })
  stack_id = kaleido_platform_stack.originator_web3_middleware_stack.id
}

locals {
  contracts = {
    noto_factory = {
      name = "NotoFactory"
      contract_url = "hhttps://github.com/kaleido-io/paladin/blob/9318ea383074b1bc5d3196587aa07109c8cf17e0/solidity/contracts/domains/noto/NotoFactory.sol"
      path = "Paladin"
    }
    pente_factory = {
      name = "PenteFactory"
      contract_url = "https://github.com/kaleido-io/paladin/blob/9318ea383074b1bc5d3196587aa07109c8cf17e0/solidity/contracts/domains/pente/PenteFactory.sol"
      path = "Paladin"
    }
  }
}


resource "kaleido_platform_cms_build" "contracts" {
  for_each = local.contracts
  environment = kaleido_platform_environment.originator_environment.id
  service     = kaleido_platform_service.originator_cms_service.id
  type        = "github"
  name        = each.value.name
  path        = each.value.path
  evm_version = "shanghai"
  github = {
    contract_url  = each.value.contract_url
    contract_name = each.value.name
  }

  provider = kaleido.child_0
}

resource "kaleido_platform_cms_action_deploy" "contracts" {
  for_each = local.contracts
  environment         = kaleido_platform_environment.originator_environment.id
  service             = kaleido_platform_service.originator_cms_service.id
  build               = kaleido_platform_cms_build.contracts[each.key].id
  name                = "deploy_${each.value.name}"
  transaction_manager = kaleido_platform_service.originator_tms_service.id
  signing_key         = kaleido_platform_kms_key.contract_deployer_key.address

  provider = kaleido.child_0
}

# ============================================================================
# JOINER ACCOUNTS (Statically defined modules for each potential joiner)
# ============================================================================

# Joiner 1 (Account 2)
module "joiner_1" {
  source = "./modules/joiner_account"
  count = var.child_account_count > 1 ? 1 : 0

  providers = {
    kaleido = kaleido.child_1
  }

  account_index = 1
  account_name_prefix = var.account_name_prefix
  besu_network_name = var.besu_network_name
  ipfs_network_name = var.ipfs_network_name
  paladin_network_name = var.paladin_network_name
  shared_swarm_key = random_id.shared_swarm_key.hex
  originator_bootstrap_files = data.kaleido_platform_network_bootstrap_data.originator_bootstrap.bootstrap_files
  deployment_zone = var.deployment_zone
  gateway_count = var.gateway_count
  node_runtime_size = var.node_runtime_size
  enable_force_delete = var.enable_force_delete
  child_accounts = kaleido_platform_account.child_accounts
  originator_besu_network = kaleido_platform_network.originator_besu_network
  originator_besu_signer_services = kaleido_platform_service.originator_besu_signer_service
  paladin_noto_factory_address = kaleido_platform_cms_action_deploy.contracts["noto_factory"].contract_address
  paladin_pente_factory_address = kaleido_platform_cms_action_deploy.contracts["pente_factory"].contract_address

  depends_on = [
    kaleido_platform_service.originator_besu_signer_service,
    kaleido_platform_service.originator_paladin_service,
    kaleido_platform_network.originator_besu_network,
    kaleido_platform_network.originator_ipfs_network,
    kaleido_platform_network.originator_paladin_network,
  ]
}

# Joiner 2 (Account 3)
module "joiner_2" {
  source = "./modules/joiner_account"
  count = var.child_account_count > 2 ? 1 : 0

  providers = {
    kaleido = kaleido.child_2
  }

  account_index = 2
  account_name_prefix = var.account_name_prefix
  besu_network_name = var.besu_network_name
  ipfs_network_name = var.ipfs_network_name
  paladin_network_name = var.paladin_network_name
  shared_swarm_key = random_id.shared_swarm_key.hex
  originator_bootstrap_files = data.kaleido_platform_network_bootstrap_data.originator_bootstrap.bootstrap_files
  deployment_zone = var.deployment_zone
  gateway_count = var.gateway_count
  node_runtime_size = var.node_runtime_size
  enable_force_delete = var.enable_force_delete
  child_accounts = kaleido_platform_account.child_accounts
  originator_besu_network = kaleido_platform_network.originator_besu_network
  originator_besu_signer_services = kaleido_platform_service.originator_besu_signer_service
  paladin_noto_factory_address = kaleido_platform_cms_action_deploy.contracts["noto_factory"].contract_address
  paladin_pente_factory_address = kaleido_platform_cms_action_deploy.contracts["pente_factory"].contract_address

  depends_on = [
    kaleido_platform_service.originator_besu_signer_service,
    kaleido_platform_service.originator_paladin_service,
    kaleido_platform_network.originator_besu_network,
    kaleido_platform_network.originator_ipfs_network,
    kaleido_platform_network.originator_paladin_network,
  ]
}

# Joiner 3 (Account 4)
module "joiner_3" {
  source = "./modules/joiner_account"
  count = var.child_account_count > 3 ? 1 : 0

  providers = {
    kaleido = kaleido.child_3
  }

  account_index = 3
  account_name_prefix = var.account_name_prefix
  besu_network_name = var.besu_network_name
  ipfs_network_name = var.ipfs_network_name
  paladin_network_name = var.paladin_network_name
  shared_swarm_key = random_id.shared_swarm_key.hex
  originator_bootstrap_files = data.kaleido_platform_network_bootstrap_data.originator_bootstrap.bootstrap_files
  deployment_zone = var.deployment_zone
  gateway_count = var.gateway_count
  node_runtime_size = var.node_runtime_size
  enable_force_delete = var.enable_force_delete
  child_accounts = kaleido_platform_account.child_accounts
  originator_besu_network = kaleido_platform_network.originator_besu_network
  originator_besu_signer_services = kaleido_platform_service.originator_besu_signer_service
  paladin_noto_factory_address = kaleido_platform_cms_action_deploy.contracts["noto_factory"].contract_address
  paladin_pente_factory_address = kaleido_platform_cms_action_deploy.contracts["pente_factory"].contract_address

  depends_on = [
    kaleido_platform_service.originator_besu_signer_service,
    kaleido_platform_service.originator_paladin_service,
    kaleido_platform_network.originator_besu_network,
    kaleido_platform_network.originator_ipfs_network,
    kaleido_platform_network.originator_paladin_network,
  ]
}

# Joiner 4 (Account 5)
module "joiner_4" {
  source = "./modules/joiner_account"
  count = var.child_account_count > 4 ? 1 : 0

  providers = {
    kaleido = kaleido.child_4
  }

  account_index = 4
  account_name_prefix = var.account_name_prefix
  besu_network_name = var.besu_network_name
  ipfs_network_name = var.ipfs_network_name
  paladin_network_name = var.paladin_network_name
  shared_swarm_key = random_id.shared_swarm_key.hex
  originator_bootstrap_files = data.kaleido_platform_network_bootstrap_data.originator_bootstrap.bootstrap_files
  deployment_zone = var.deployment_zone
  gateway_count = var.gateway_count
  node_runtime_size = var.node_runtime_size
  enable_force_delete = var.enable_force_delete
  child_accounts = kaleido_platform_account.child_accounts
  originator_besu_network = kaleido_platform_network.originator_besu_network
  originator_besu_signer_services = kaleido_platform_service.originator_besu_signer_service
  paladin_noto_factory_address = kaleido_platform_cms_action_deploy.contracts["noto_factory"].contract_address
  paladin_pente_factory_address = kaleido_platform_cms_action_deploy.contracts["pente_factory"].contract_address

  depends_on = [
    kaleido_platform_service.originator_besu_signer_service,
    kaleido_platform_service.originator_paladin_service,
    kaleido_platform_network.originator_besu_network,
    kaleido_platform_network.originator_ipfs_network,
    kaleido_platform_network.originator_paladin_network,
  ]
}

# ============================================================================
# FULL MESH NETWORK CONNECTORS IMPLEMENTATION
# ============================================================================



resource "kaleido_network_connector" "account1_to_account2_besu_request" {
  count = var.child_account_count > 1 ? 1 : 0
  provider = kaleido.child_0
  
  type = "Platform"
  name = "account1_to_account2_besu"
  environment = kaleido_platform_environment.originator_environment.id
  network = kaleido_platform_network.originator_besu_network.id
  zone = var.deployment_zone

  platform_requestor = {
    target_account_id = kaleido_platform_account.child_accounts[1].id
    target_environment_id = module.joiner_1[0].environment.id
    target_network_id = module.joiner_1[0].besu_network.id
  }
}

resource "kaleido_network_connector" "account2_accept_account1_besu" {
  count = var.child_account_count > 1 ? 1 : 0
  provider = kaleido.child_1
  
  type = "Platform"
  name = "account2_accept_account1_besu"
  environment = module.joiner_1[0].environment.id
  network = module.joiner_1[0].besu_network.id
  zone = var.deployment_zone

  platform_acceptor = {
    target_account_id = kaleido_platform_account.child_accounts[0].id
    target_environment_id = kaleido_platform_environment.originator_environment.id
    target_network_id = kaleido_platform_network.originator_besu_network.id
    target_connector_id = kaleido_network_connector.account1_to_account2_besu_request[0].id
  }
}

resource "kaleido_network_connector" "account1_to_account2_ipfs_request" {
  count = var.child_account_count > 1 ? 1 : 0
  provider = kaleido.child_0
  
  type = "Platform"
  name = "account1_to_account2_ipfs"
  environment = kaleido_platform_environment.originator_environment.id
  network = kaleido_platform_network.originator_ipfs_network.id
  zone = var.deployment_zone

  platform_requestor = {
    target_account_id = kaleido_platform_account.child_accounts[1].id
    target_environment_id = module.joiner_1[0].environment.id
    target_network_id = module.joiner_1[0].ipfs_network.id
  }
}

resource "kaleido_network_connector" "account2_accept_account1_ipfs" {
  count = var.child_account_count > 1 ? 1 : 0
  provider = kaleido.child_1
  
  type = "Platform"
  name = "account2_accept_account1_ipfs"
  environment = module.joiner_1[0].environment.id
  network = module.joiner_1[0].ipfs_network.id
  zone = var.deployment_zone

  platform_acceptor = {
    target_account_id = kaleido_platform_account.child_accounts[0].id
    target_environment_id = kaleido_platform_environment.originator_environment.id
    target_network_id = kaleido_platform_network.originator_ipfs_network.id
    target_connector_id = kaleido_network_connector.account1_to_account2_ipfs_request[0].id
  }
}

resource "kaleido_network_connector" "account1_to_account2_paladin_request" {
  count = var.child_account_count > 1 ? 1 : 0
  provider = kaleido.child_0
  
  type = "Platform"
  name = "account1_to_account2_paladin"
  environment = kaleido_platform_environment.originator_environment.id
  network = kaleido_platform_network.originator_paladin_network.id
  zone = var.deployment_zone

  platform_requestor = {
    target_account_id = kaleido_platform_account.child_accounts[1].id
    target_environment_id = module.joiner_1[0].environment.id
    target_network_id = module.joiner_1[0].paladin_network.id
  }
}

resource "kaleido_network_connector" "account2_accept_account1_paladin" {
  count = var.child_account_count > 1 ? 1 : 0
  provider = kaleido.child_1
  
  type = "Platform"
  name = "account2_accept_account1_paladin"
  environment = module.joiner_1[0].environment.id
  network = module.joiner_1[0].paladin_network.id
  zone = var.deployment_zone

  platform_acceptor = {
    target_account_id = kaleido_platform_account.child_accounts[0].id
    target_environment_id = kaleido_platform_environment.originator_environment.id
    target_network_id = kaleido_platform_network.originator_paladin_network.id
    target_connector_id = kaleido_network_connector.account1_to_account2_paladin_request[0].id
  }
}

resource "kaleido_network_connector" "account1_to_account3_besu_request" {
  count = var.child_account_count > 2 ? 1 : 0
  provider = kaleido.child_0
  
  type = "Platform"
  name = "account1_to_account3_besu"
  environment = kaleido_platform_environment.originator_environment.id
  network = kaleido_platform_network.originator_besu_network.id
  zone = var.deployment_zone

  platform_requestor = {
    target_account_id = kaleido_platform_account.child_accounts[2].id
    target_environment_id = module.joiner_2[0].environment.id
    target_network_id = module.joiner_2[0].besu_network.id
  }
}

resource "kaleido_network_connector" "account3_accept_account1_besu" {
  count = var.child_account_count > 2 ? 1 : 0
  provider = kaleido.child_2
  
  type = "Platform"
  name = "account3_accept_account1_besu"
  environment = module.joiner_2[0].environment.id
  network = module.joiner_2[0].besu_network.id
  zone = var.deployment_zone

  platform_acceptor = {
    target_account_id = kaleido_platform_account.child_accounts[0].id
    target_environment_id = kaleido_platform_environment.originator_environment.id
    target_network_id = kaleido_platform_network.originator_besu_network.id
    target_connector_id = kaleido_network_connector.account1_to_account3_besu_request[0].id
  }
}

resource "kaleido_network_connector" "account1_to_account3_ipfs_request" {
  count = var.child_account_count > 2 ? 1 : 0
  provider = kaleido.child_0
  
  type = "Platform"
  name = "account1_to_account3_ipfs"
  environment = kaleido_platform_environment.originator_environment.id
  network = kaleido_platform_network.originator_ipfs_network.id
  zone = var.deployment_zone

  platform_requestor = {
    target_account_id = kaleido_platform_account.child_accounts[2].id
    target_environment_id = module.joiner_2[0].environment.id
    target_network_id = module.joiner_2[0].ipfs_network.id
  }
}

resource "kaleido_network_connector" "account3_accept_account1_ipfs" {
  count = var.child_account_count > 2 ? 1 : 0
  provider = kaleido.child_2
  
  type = "Platform"
  name = "account3_accept_account1_ipfs"
  environment = module.joiner_2[0].environment.id
  network = module.joiner_2[0].ipfs_network.id
  zone = var.deployment_zone

  platform_acceptor = {
    target_account_id = kaleido_platform_account.child_accounts[0].id
    target_environment_id = kaleido_platform_environment.originator_environment.id
    target_network_id = kaleido_platform_network.originator_ipfs_network.id
    target_connector_id = kaleido_network_connector.account1_to_account3_ipfs_request[0].id
  }
}

resource "kaleido_network_connector" "account1_to_account3_paladin_request" {
  count = var.child_account_count > 2 ? 1 : 0
  provider = kaleido.child_0
  
  type = "Platform"
  name = "account1_to_account3_paladin"
  environment = kaleido_platform_environment.originator_environment.id
  network = kaleido_platform_network.originator_paladin_network.id
  zone = var.deployment_zone

  platform_requestor = {
    target_account_id = kaleido_platform_account.child_accounts[2].id
    target_environment_id = module.joiner_2[0].environment.id
    target_network_id = module.joiner_2[0].paladin_network.id
  }
}

resource "kaleido_network_connector" "account3_accept_account1_paladin" {
  count = var.child_account_count > 2 ? 1 : 0
  provider = kaleido.child_2
  
  type = "Platform"
  name = "account3_accept_account1_paladin"
  environment = module.joiner_2[0].environment.id
  network = module.joiner_2[0].paladin_network.id
  zone = var.deployment_zone

  platform_acceptor = {
    target_account_id = kaleido_platform_account.child_accounts[0].id
    target_environment_id = kaleido_platform_environment.originator_environment.id
    target_network_id = kaleido_platform_network.originator_paladin_network.id
    target_connector_id = kaleido_network_connector.account1_to_account3_paladin_request[0].id
  }
}

resource "kaleido_network_connector" "account1_to_account4_besu_request" {
  count = var.child_account_count > 3 ? 1 : 0
  provider = kaleido.child_0
  
  type = "Platform"
  name = "account1_to_account4_besu"
  environment = kaleido_platform_environment.originator_environment.id
  network = kaleido_platform_network.originator_besu_network.id
  zone = var.deployment_zone

  platform_requestor = {
    target_account_id = kaleido_platform_account.child_accounts[3].id
    target_environment_id = module.joiner_3[0].environment.id
    target_network_id = module.joiner_3[0].besu_network.id
  }
}

resource "kaleido_network_connector" "account4_accept_account1_besu" {
  count = var.child_account_count > 3 ? 1 : 0
  provider = kaleido.child_3
  
  type = "Platform"
  name = "account4_accept_account1_besu"
  environment = module.joiner_3[0].environment.id
  network = module.joiner_3[0].besu_network.id
  zone = var.deployment_zone

  platform_acceptor = {
    target_account_id = kaleido_platform_account.child_accounts[0].id
    target_environment_id = kaleido_platform_environment.originator_environment.id
    target_network_id = kaleido_platform_network.originator_besu_network.id
    target_connector_id = kaleido_network_connector.account1_to_account4_besu_request[0].id
  }
}

resource "kaleido_network_connector" "account1_to_account4_ipfs_request" {
  count = var.child_account_count > 3 ? 1 : 0
  provider = kaleido.child_0
  
  type = "Platform"
  name = "account1_to_account4_ipfs"
  environment = kaleido_platform_environment.originator_environment.id
  network = kaleido_platform_network.originator_ipfs_network.id
  zone = var.deployment_zone

  platform_requestor = {
    target_account_id = kaleido_platform_account.child_accounts[3].id
    target_environment_id = module.joiner_3[0].environment.id
    target_network_id = module.joiner_3[0].ipfs_network.id
  }
}

resource "kaleido_network_connector" "account4_accept_account1_ipfs" {
  count = var.child_account_count > 3 ? 1 : 0
  provider = kaleido.child_3
  
  type = "Platform"
  name = "account4_accept_account1_ipfs"
  environment = module.joiner_3[0].environment.id
  network = module.joiner_3[0].ipfs_network.id
  zone = var.deployment_zone

  platform_acceptor = {
    target_account_id = kaleido_platform_account.child_accounts[0].id
    target_environment_id = kaleido_platform_environment.originator_environment.id
    target_network_id = kaleido_platform_network.originator_ipfs_network.id
    target_connector_id = kaleido_network_connector.account1_to_account4_ipfs_request[0].id
  }
}

resource "kaleido_network_connector" "account1_to_account4_paladin_request" {
  count = var.child_account_count > 3 ? 1 : 0
  provider = kaleido.child_0
  
  type = "Platform"
  name = "account1_to_account4_paladin"
  environment = kaleido_platform_environment.originator_environment.id
  network = kaleido_platform_network.originator_paladin_network.id
  zone = var.deployment_zone

  platform_requestor = {
    target_account_id = kaleido_platform_account.child_accounts[3].id
    target_environment_id = module.joiner_3[0].environment.id
    target_network_id = module.joiner_3[0].paladin_network.id
  }
}

resource "kaleido_network_connector" "account4_accept_account1_paladin" {
  count = var.child_account_count > 3 ? 1 : 0
  provider = kaleido.child_3
  
  type = "Platform"
  name = "account4_accept_account1_paladin"
  environment = module.joiner_3[0].environment.id
  network = module.joiner_3[0].paladin_network.id
  zone = var.deployment_zone

  platform_acceptor = {
    target_account_id = kaleido_platform_account.child_accounts[0].id
    target_environment_id = kaleido_platform_environment.originator_environment.id
    target_network_id = kaleido_platform_network.originator_paladin_network.id
    target_connector_id = kaleido_network_connector.account1_to_account4_paladin_request[0].id
  }
}

resource "kaleido_network_connector" "account1_to_account5_besu_request" {
  count = var.child_account_count > 4 ? 1 : 0
  provider = kaleido.child_0
  
  type = "Platform"
  name = "account1_to_account5_besu"
  environment = kaleido_platform_environment.originator_environment.id
  network = kaleido_platform_network.originator_besu_network.id
  zone = var.deployment_zone

  platform_requestor = {
    target_account_id = kaleido_platform_account.child_accounts[4].id
    target_environment_id = module.joiner_4[0].environment.id
    target_network_id = module.joiner_4[0].besu_network.id
  }
}

resource "kaleido_network_connector" "account5_accept_account1_besu" {
  count = var.child_account_count > 4 ? 1 : 0
  provider = kaleido.child_4
  
  type = "Platform"
  name = "account5_accept_account1_besu"
  environment = module.joiner_4[0].environment.id
  network = module.joiner_4[0].besu_network.id
  zone = var.deployment_zone

  platform_acceptor = {
    target_account_id = kaleido_platform_account.child_accounts[0].id
    target_environment_id = kaleido_platform_environment.originator_environment.id
    target_network_id = kaleido_platform_network.originator_besu_network.id
    target_connector_id = kaleido_network_connector.account1_to_account5_besu_request[0].id
  }
}

resource "kaleido_network_connector" "account1_to_account5_ipfs_request" {
  count = var.child_account_count > 4 ? 1 : 0
  provider = kaleido.child_0
  
  type = "Platform"
  name = "account1_to_account5_ipfs"
  environment = kaleido_platform_environment.originator_environment.id
  network = kaleido_platform_network.originator_ipfs_network.id
  zone = var.deployment_zone

  platform_requestor = {
    target_account_id = kaleido_platform_account.child_accounts[4].id
    target_environment_id = module.joiner_4[0].environment.id
    target_network_id = module.joiner_4[0].ipfs_network.id
  }
}

resource "kaleido_network_connector" "account5_accept_account1_ipfs" {
  count = var.child_account_count > 4 ? 1 : 0
  provider = kaleido.child_4
  
  type = "Platform"
  name = "account5_accept_account1_ipfs"
  environment = module.joiner_4[0].environment.id
  network = module.joiner_4[0].ipfs_network.id
  zone = var.deployment_zone

  platform_acceptor = {
    target_account_id = kaleido_platform_account.child_accounts[0].id
    target_environment_id = kaleido_platform_environment.originator_environment.id
    target_network_id = kaleido_platform_network.originator_ipfs_network.id
    target_connector_id = kaleido_network_connector.account1_to_account5_ipfs_request[0].id
  }
}

resource "kaleido_network_connector" "account1_to_account5_paladin_request" {
  count = var.child_account_count > 4 ? 1 : 0
  provider = kaleido.child_0
  
  type = "Platform"
  name = "account1_to_account5_paladin"
  environment = kaleido_platform_environment.originator_environment.id
  network = kaleido_platform_network.originator_paladin_network.id
  zone = var.deployment_zone

  platform_requestor = {
    target_account_id = kaleido_platform_account.child_accounts[4].id
    target_environment_id = module.joiner_4[0].environment.id
    target_network_id = module.joiner_4[0].paladin_network.id
  }
}

resource "kaleido_network_connector" "account5_accept_account1_paladin" {
  count = var.child_account_count > 4 ? 1 : 0
  provider = kaleido.child_4
  
  type = "Platform"
  name = "account5_accept_account1_paladin"
  environment = module.joiner_4[0].environment.id
  network = module.joiner_4[0].paladin_network.id
  zone = var.deployment_zone

  platform_acceptor = {
    target_account_id = kaleido_platform_account.child_accounts[0].id
    target_environment_id = kaleido_platform_environment.originator_environment.id
    target_network_id = kaleido_platform_network.originator_paladin_network.id
    target_connector_id = kaleido_network_connector.account1_to_account5_paladin_request[0].id
  }
}


resource "kaleido_network_connector" "account2_to_account3_besu_request" {
  count = var.child_account_count > 2 ? 1 : 0
  provider = kaleido.child_1
  
  type = "Platform"
  name = "account2_to_account3_besu"
  environment = module.joiner_1[0].environment.id
  network = module.joiner_1[0].besu_network.id
  zone = var.deployment_zone

  platform_requestor = {
    target_account_id = kaleido_platform_account.child_accounts[2].id
    target_environment_id = module.joiner_2[0].environment.id
    target_network_id = module.joiner_2[0].besu_network.id
  }
}

resource "kaleido_network_connector" "account3_accept_account2_besu" {
  count = var.child_account_count > 2 ? 1 : 0
  provider = kaleido.child_2
  
  type = "Platform"
  name = "account3_accept_account2_besu"
  environment = module.joiner_2[0].environment.id
  network = module.joiner_2[0].besu_network.id
  zone = var.deployment_zone

  platform_acceptor = {
    target_account_id = kaleido_platform_account.child_accounts[1].id
    target_environment_id = module.joiner_1[0].environment.id
    target_network_id = module.joiner_1[0].besu_network.id
    target_connector_id = kaleido_network_connector.account2_to_account3_besu_request[0].id
  }
}

resource "kaleido_network_connector" "account2_to_account3_ipfs_request" {
  count = var.child_account_count > 2 ? 1 : 0
  provider = kaleido.child_1
  
  type = "Platform"
  name = "account2_to_account3_ipfs"
  environment = module.joiner_1[0].environment.id
  network = module.joiner_1[0].ipfs_network.id
  zone = var.deployment_zone

  platform_requestor = {
    target_account_id = kaleido_platform_account.child_accounts[2].id
    target_environment_id = module.joiner_2[0].environment.id
    target_network_id = module.joiner_2[0].ipfs_network.id
  }
}

resource "kaleido_network_connector" "account3_accept_account2_ipfs" {
  count = var.child_account_count > 2 ? 1 : 0
  provider = kaleido.child_2
  
  type = "Platform"
  name = "account3_accept_account2_ipfs"
  environment = module.joiner_2[0].environment.id
  network = module.joiner_2[0].ipfs_network.id
  zone = var.deployment_zone

  platform_acceptor = {
    target_account_id = kaleido_platform_account.child_accounts[1].id
    target_environment_id = module.joiner_1[0].environment.id
    target_network_id = module.joiner_1[0].ipfs_network.id
    target_connector_id = kaleido_network_connector.account2_to_account3_ipfs_request[0].id
  }
}

resource "kaleido_network_connector" "account2_to_account3_paladin_request" {
  count = var.child_account_count > 2 ? 1 : 0
  provider = kaleido.child_1
  
  type = "Platform"
  name = "account2_to_account3_paladin"
  environment = module.joiner_1[0].environment.id
  network = module.joiner_1[0].paladin_network.id
  zone = var.deployment_zone

  platform_requestor = {
    target_account_id = kaleido_platform_account.child_accounts[2].id
    target_environment_id = module.joiner_2[0].environment.id
    target_network_id = module.joiner_2[0].paladin_network.id
  }
}

resource "kaleido_network_connector" "account3_accept_account2_paladin" {
  count = var.child_account_count > 2 ? 1 : 0
  provider = kaleido.child_2
  
  type = "Platform"
  name = "account3_accept_account2_paladin"
  environment = module.joiner_2[0].environment.id
  network = module.joiner_2[0].paladin_network.id
  zone = var.deployment_zone

  platform_acceptor = {
    target_account_id = kaleido_platform_account.child_accounts[1].id
    target_environment_id = module.joiner_1[0].environment.id
    target_network_id = module.joiner_1[0].paladin_network.id
    target_connector_id = kaleido_network_connector.account2_to_account3_paladin_request[0].id
  }
}

resource "kaleido_network_connector" "account2_to_account4_besu_request" {
  count = var.child_account_count > 3 ? 1 : 0
  provider = kaleido.child_1
  
  type = "Platform"
  name = "account2_to_account4_besu"
  environment = module.joiner_1[0].environment.id
  network = module.joiner_1[0].besu_network.id
  zone = var.deployment_zone

  platform_requestor = {
    target_account_id = kaleido_platform_account.child_accounts[3].id
    target_environment_id = module.joiner_3[0].environment.id
    target_network_id = module.joiner_3[0].besu_network.id
  }
}

resource "kaleido_network_connector" "account4_accept_account2_besu" {
  count = var.child_account_count > 3 ? 1 : 0
  provider = kaleido.child_3
  
  type = "Platform"
  name = "account4_accept_account2_besu"
  environment = module.joiner_3[0].environment.id
  network = module.joiner_3[0].besu_network.id
  zone = var.deployment_zone

  platform_acceptor = {
    target_account_id = kaleido_platform_account.child_accounts[1].id
    target_environment_id = module.joiner_1[0].environment.id
    target_network_id = module.joiner_1[0].besu_network.id
    target_connector_id = kaleido_network_connector.account2_to_account4_besu_request[0].id
  }
}

resource "kaleido_network_connector" "account2_to_account4_ipfs_request" {
  count = var.child_account_count > 3 ? 1 : 0
  provider = kaleido.child_1
  
  type = "Platform"
  name = "account2_to_account4_ipfs"
  environment = module.joiner_1[0].environment.id
  network = module.joiner_1[0].ipfs_network.id
  zone = var.deployment_zone

  platform_requestor = {
    target_account_id = kaleido_platform_account.child_accounts[3].id
    target_environment_id = module.joiner_3[0].environment.id
    target_network_id = module.joiner_3[0].ipfs_network.id
  }
}

resource "kaleido_network_connector" "account4_accept_account2_ipfs" {
  count = var.child_account_count > 3 ? 1 : 0
  provider = kaleido.child_3
  
  type = "Platform"
  name = "account4_accept_account2_ipfs"
  environment = module.joiner_3[0].environment.id
  network = module.joiner_3[0].ipfs_network.id
  zone = var.deployment_zone

  platform_acceptor = {
    target_account_id = kaleido_platform_account.child_accounts[1].id
    target_environment_id = module.joiner_1[0].environment.id
    target_network_id = module.joiner_1[0].ipfs_network.id
    target_connector_id = kaleido_network_connector.account2_to_account4_ipfs_request[0].id
  }
}

resource "kaleido_network_connector" "account2_to_account4_paladin_request" {
  count = var.child_account_count > 3 ? 1 : 0
  provider = kaleido.child_1
  
  type = "Platform"
  name = "account2_to_account4_paladin"
  environment = module.joiner_1[0].environment.id
  network = module.joiner_1[0].paladin_network.id
  zone = var.deployment_zone

  platform_requestor = {
    target_account_id = kaleido_platform_account.child_accounts[3].id
    target_environment_id = module.joiner_3[0].environment.id
    target_network_id = module.joiner_3[0].paladin_network.id
  }
}

resource "kaleido_network_connector" "account4_accept_account2_paladin" {
  count = var.child_account_count > 3 ? 1 : 0
  provider = kaleido.child_3
  
  type = "Platform"
  name = "account4_accept_account2_paladin"
  environment = module.joiner_3[0].environment.id
  network = module.joiner_3[0].paladin_network.id
  zone = var.deployment_zone

  platform_acceptor = {
    target_account_id = kaleido_platform_account.child_accounts[1].id
    target_environment_id = module.joiner_1[0].environment.id
    target_network_id = module.joiner_1[0].paladin_network.id
    target_connector_id = kaleido_network_connector.account2_to_account4_paladin_request[0].id
  }
}

resource "kaleido_network_connector" "account2_to_account5_besu_request" {
  count = var.child_account_count > 4 ? 1 : 0
  provider = kaleido.child_1
  
  type = "Platform"
  name = "account2_to_account5_besu"
  environment = module.joiner_1[0].environment.id
  network = module.joiner_1[0].besu_network.id
  zone = var.deployment_zone

  platform_requestor = {
    target_account_id = kaleido_platform_account.child_accounts[4].id
    target_environment_id = module.joiner_4[0].environment.id
    target_network_id = module.joiner_4[0].besu_network.id
  }
}

resource "kaleido_network_connector" "account5_accept_account2_besu" {
  count = var.child_account_count > 4 ? 1 : 0
  provider = kaleido.child_4
  
  type = "Platform"
  name = "account5_accept_account2_besu"
  environment = module.joiner_4[0].environment.id
  network = module.joiner_4[0].besu_network.id
  zone = var.deployment_zone

  platform_acceptor = {
    target_account_id = kaleido_platform_account.child_accounts[1].id
    target_environment_id = module.joiner_1[0].environment.id
    target_network_id = module.joiner_1[0].besu_network.id
    target_connector_id = kaleido_network_connector.account2_to_account5_besu_request[0].id
  }
}

resource "kaleido_network_connector" "account2_to_account5_ipfs_request" {
  count = var.child_account_count > 4 ? 1 : 0
  provider = kaleido.child_1
  
  type = "Platform"
  name = "account2_to_account5_ipfs"
  environment = module.joiner_1[0].environment.id
  network = module.joiner_1[0].ipfs_network.id
  zone = var.deployment_zone

  platform_requestor = {
    target_account_id = kaleido_platform_account.child_accounts[4].id
    target_environment_id = module.joiner_4[0].environment.id
    target_network_id = module.joiner_4[0].ipfs_network.id
  }
}

resource "kaleido_network_connector" "account5_accept_account2_ipfs" {
  count = var.child_account_count > 4 ? 1 : 0
  provider = kaleido.child_4
  
  type = "Platform"
  name = "account5_accept_account2_ipfs"
  environment = module.joiner_4[0].environment.id
  network = module.joiner_4[0].ipfs_network.id
  zone = var.deployment_zone

  platform_acceptor = {
    target_account_id = kaleido_platform_account.child_accounts[1].id
    target_environment_id = module.joiner_1[0].environment.id
    target_network_id = module.joiner_1[0].ipfs_network.id
    target_connector_id = kaleido_network_connector.account2_to_account5_ipfs_request[0].id
  }
}

resource "kaleido_network_connector" "account2_to_account5_paladin_request" {
  count = var.child_account_count > 4 ? 1 : 0
  provider = kaleido.child_1
  
  type = "Platform"
  name = "account2_to_account5_paladin"
  environment = module.joiner_1[0].environment.id
  network = module.joiner_1[0].paladin_network.id
  zone = var.deployment_zone

  platform_requestor = {
    target_account_id = kaleido_platform_account.child_accounts[4].id
    target_environment_id = module.joiner_4[0].environment.id
    target_network_id = module.joiner_4[0].paladin_network.id
  }
}

resource "kaleido_network_connector" "account5_accept_account2_paladin" {
  count = var.child_account_count > 4 ? 1 : 0
  provider = kaleido.child_4
  
  type = "Platform"
  name = "account5_accept_account2_paladin"
  environment = module.joiner_4[0].environment.id
  network = module.joiner_4[0].paladin_network.id
  zone = var.deployment_zone

  platform_acceptor = {
    target_account_id = kaleido_platform_account.child_accounts[1].id
    target_environment_id = module.joiner_1[0].environment.id
    target_network_id = module.joiner_1[0].paladin_network.id
    target_connector_id = kaleido_network_connector.account2_to_account5_paladin_request[0].id
  }
}

resource "kaleido_network_connector" "account3_to_account4_besu_request" {
  count = var.child_account_count > 3 ? 1 : 0
  provider = kaleido.child_2
  
  type = "Platform"
  name = "account3_to_account4_besu"
  environment = module.joiner_2[0].environment.id
  network = module.joiner_2[0].besu_network.id
  zone = var.deployment_zone

  platform_requestor = {
    target_account_id = kaleido_platform_account.child_accounts[3].id
    target_environment_id = module.joiner_3[0].environment.id
    target_network_id = module.joiner_3[0].besu_network.id
  }
}

resource "kaleido_network_connector" "account4_accept_account3_besu" {
  count = var.child_account_count > 3 ? 1 : 0
  provider = kaleido.child_3
  
  type = "Platform"
  name = "account4_accept_account3_besu"
  environment = module.joiner_3[0].environment.id
  network = module.joiner_3[0].besu_network.id
  zone = var.deployment_zone

  platform_acceptor = {
    target_account_id = kaleido_platform_account.child_accounts[2].id
    target_environment_id = module.joiner_2[0].environment.id
    target_network_id = module.joiner_2[0].besu_network.id
    target_connector_id = kaleido_network_connector.account3_to_account4_besu_request[0].id
  }
}

resource "kaleido_network_connector" "account3_to_account4_ipfs_request" {
  count = var.child_account_count > 3 ? 1 : 0
  provider = kaleido.child_2
  
  type = "Platform"
  name = "account3_to_account4_ipfs"
  environment = module.joiner_2[0].environment.id
  network = module.joiner_2[0].ipfs_network.id
  zone = var.deployment_zone

  platform_requestor = {
    target_account_id = kaleido_platform_account.child_accounts[3].id
    target_environment_id = module.joiner_3[0].environment.id
    target_network_id = module.joiner_3[0].ipfs_network.id
  }
}

resource "kaleido_network_connector" "account4_accept_account3_ipfs" {
  count = var.child_account_count > 3 ? 1 : 0
  provider = kaleido.child_3
  
  type = "Platform"
  name = "account4_accept_account3_ipfs"
  environment = module.joiner_3[0].environment.id
  network = module.joiner_3[0].ipfs_network.id
  zone = var.deployment_zone

  platform_acceptor = {
    target_account_id = kaleido_platform_account.child_accounts[2].id
    target_environment_id = module.joiner_2[0].environment.id
    target_network_id = module.joiner_2[0].ipfs_network.id
    target_connector_id = kaleido_network_connector.account3_to_account4_ipfs_request[0].id
  }
}

resource "kaleido_network_connector" "account3_to_account4_paladin_request" {
  count = var.child_account_count > 3 ? 1 : 0
  provider = kaleido.child_2
  
  type = "Platform"
  name = "account3_to_account4_paladin"
  environment = module.joiner_2[0].environment.id
  network = module.joiner_2[0].paladin_network.id
  zone = var.deployment_zone

  platform_requestor = {
    target_account_id = kaleido_platform_account.child_accounts[3].id
    target_environment_id = module.joiner_3[0].environment.id
    target_network_id = module.joiner_3[0].paladin_network.id
  }
}

resource "kaleido_network_connector" "account4_accept_account3_paladin" {
  count = var.child_account_count > 3 ? 1 : 0
  provider = kaleido.child_3
  
  type = "Platform"
  name = "account4_accept_account3_paladin"
  environment = module.joiner_3[0].environment.id
  network = module.joiner_3[0].paladin_network.id
  zone = var.deployment_zone

  platform_acceptor = {
    target_account_id = kaleido_platform_account.child_accounts[2].id
    target_environment_id = module.joiner_2[0].environment.id
    target_network_id = module.joiner_2[0].paladin_network.id
    target_connector_id = kaleido_network_connector.account3_to_account4_paladin_request[0].id
  }
}

resource "kaleido_network_connector" "account3_to_account5_besu_request" {
  count = var.child_account_count > 4 ? 1 : 0
  provider = kaleido.child_2
  
  type = "Platform"
  name = "account3_to_account5_besu"
  environment = module.joiner_2[0].environment.id
  network = module.joiner_2[0].besu_network.id
  zone = var.deployment_zone

  platform_requestor = {
    target_account_id = kaleido_platform_account.child_accounts[4].id
    target_environment_id = module.joiner_4[0].environment.id
    target_network_id = module.joiner_4[0].besu_network.id
  }
}

resource "kaleido_network_connector" "account5_accept_account3_besu" {
  count = var.child_account_count > 4 ? 1 : 0
  provider = kaleido.child_4
  
  type = "Platform"
  name = "account5_accept_account3_besu"
  environment = module.joiner_4[0].environment.id
  network = module.joiner_4[0].besu_network.id
  zone = var.deployment_zone

  platform_acceptor = {
    target_account_id = kaleido_platform_account.child_accounts[2].id
    target_environment_id = module.joiner_2[0].environment.id
    target_network_id = module.joiner_2[0].besu_network.id
    target_connector_id = kaleido_network_connector.account3_to_account5_besu_request[0].id
  }
}

resource "kaleido_network_connector" "account3_to_account5_ipfs_request" {
  count = var.child_account_count > 4 ? 1 : 0
  provider = kaleido.child_2
  
  type = "Platform"
  name = "account3_to_account5_ipfs"
  environment = module.joiner_2[0].environment.id
  network = module.joiner_2[0].ipfs_network.id
  zone = var.deployment_zone

  platform_requestor = {
    target_account_id = kaleido_platform_account.child_accounts[4].id
    target_environment_id = module.joiner_4[0].environment.id
    target_network_id = module.joiner_4[0].ipfs_network.id
  }
}

resource "kaleido_network_connector" "account5_accept_account3_ipfs" {
  count = var.child_account_count > 4 ? 1 : 0
  provider = kaleido.child_4
  
  type = "Platform"
  name = "account5_accept_account3_ipfs"
  environment = module.joiner_4[0].environment.id
  network = module.joiner_4[0].ipfs_network.id
  zone = var.deployment_zone

  platform_acceptor = {
    target_account_id = kaleido_platform_account.child_accounts[2].id
    target_environment_id = module.joiner_2[0].environment.id
    target_network_id = module.joiner_2[0].ipfs_network.id
    target_connector_id = kaleido_network_connector.account3_to_account5_ipfs_request[0].id
  }
}

resource "kaleido_network_connector" "account3_to_account5_paladin_request" {
  count = var.child_account_count > 4 ? 1 : 0
  provider = kaleido.child_2
  
  type = "Platform"
  name = "account3_to_account5_paladin"
  environment = module.joiner_2[0].environment.id
  network = module.joiner_2[0].paladin_network.id
  zone = var.deployment_zone

  platform_requestor = {
    target_account_id = kaleido_platform_account.child_accounts[4].id
    target_environment_id = module.joiner_4[0].environment.id
    target_network_id = module.joiner_4[0].paladin_network.id
  }
}

resource "kaleido_network_connector" "account5_accept_account3_paladin" {
  count = var.child_account_count > 4 ? 1 : 0
  provider = kaleido.child_4
  
  type = "Platform"
  name = "account5_accept_account3_paladin"
  environment = module.joiner_4[0].environment.id
  network = module.joiner_4[0].paladin_network.id
  zone = var.deployment_zone

  platform_acceptor = {
    target_account_id = kaleido_platform_account.child_accounts[2].id
    target_environment_id = module.joiner_2[0].environment.id
    target_network_id = module.joiner_2[0].paladin_network.id
    target_connector_id = kaleido_network_connector.account3_to_account5_paladin_request[0].id
  }
}

resource "kaleido_network_connector" "account4_to_account5_besu_request" {
  count = var.child_account_count > 4 ? 1 : 0
  provider = kaleido.child_3
  
  type = "Platform"
  name = "account4_to_account5_besu"
  environment = module.joiner_3[0].environment.id
  network = module.joiner_3[0].besu_network.id
  zone = var.deployment_zone

  platform_requestor = {
    target_account_id = kaleido_platform_account.child_accounts[4].id
    target_environment_id = module.joiner_4[0].environment.id
    target_network_id = module.joiner_4[0].besu_network.id
  }
}

resource "kaleido_network_connector" "account5_accept_account4_besu" {
  count = var.child_account_count > 4 ? 1 : 0
  provider = kaleido.child_4
  
  type = "Platform"
  name = "account5_accept_account4_besu"
  environment = module.joiner_4[0].environment.id
  network = module.joiner_4[0].besu_network.id
  zone = var.deployment_zone

  platform_acceptor = {
    target_account_id = kaleido_platform_account.child_accounts[3].id
    target_environment_id = module.joiner_3[0].environment.id
    target_network_id = module.joiner_3[0].besu_network.id
    target_connector_id = kaleido_network_connector.account4_to_account5_besu_request[0].id
  }
}

resource "kaleido_network_connector" "account4_to_account5_ipfs_request" {
  count = var.child_account_count > 4 ? 1 : 0
  provider = kaleido.child_3
  
  type = "Platform"
  name = "account4_to_account5_ipfs"
  environment = module.joiner_3[0].environment.id
  network = module.joiner_3[0].ipfs_network.id
  zone = var.deployment_zone

  platform_requestor = {
    target_account_id = kaleido_platform_account.child_accounts[4].id
    target_environment_id = module.joiner_4[0].environment.id
    target_network_id = module.joiner_4[0].ipfs_network.id
  }
}

resource "kaleido_network_connector" "account5_accept_account4_ipfs" {
  count = var.child_account_count > 4 ? 1 : 0
  provider = kaleido.child_4
  
  type = "Platform"
  name = "account5_accept_account4_ipfs"
  environment = module.joiner_4[0].environment.id
  network = module.joiner_4[0].ipfs_network.id
  zone = var.deployment_zone

  platform_acceptor = {
    target_account_id = kaleido_platform_account.child_accounts[3].id
    target_environment_id = module.joiner_3[0].environment.id
    target_network_id = module.joiner_3[0].ipfs_network.id
    target_connector_id = kaleido_network_connector.account4_to_account5_ipfs_request[0].id
  }
}

resource "kaleido_network_connector" "account4_to_account5_paladin_request" {
  count = var.child_account_count > 4 ? 1 : 0
  provider = kaleido.child_3
  
  type = "Platform"
  name = "account4_to_account5_paladin"
  environment = module.joiner_3[0].environment.id
  network = module.joiner_3[0].paladin_network.id
  zone = var.deployment_zone

  platform_requestor = {
    target_account_id = kaleido_platform_account.child_accounts[4].id
    target_environment_id = module.joiner_4[0].environment.id
    target_network_id = module.joiner_4[0].paladin_network.id
  }
}

resource "kaleido_network_connector" "account5_accept_account4_paladin" {
  count = var.child_account_count > 4 ? 1 : 0
  provider = kaleido.child_4
  
  type = "Platform"
  name = "account5_accept_account4_paladin"
  environment = module.joiner_4[0].environment.id
  network = module.joiner_4[0].paladin_network.id
  zone = var.deployment_zone

  platform_acceptor = {
    target_account_id = kaleido_platform_account.child_accounts[3].id
    target_environment_id = module.joiner_3[0].environment.id
    target_network_id = module.joiner_3[0].paladin_network.id
    target_connector_id = kaleido_network_connector.account4_to_account5_paladin_request[0].id
  }
}


