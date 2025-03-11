terraform {
  required_providers {
    kaleido = {
      source = "kaleido-io/kaleido"
    }
  }
}

provider "kaleido" {
  platform_api = var.kaleido_platform_api
  platform_username = var.kaleido_platform_username
  platform_password = var.kaleido_platform_password
}

resource "kaleido_platform_environment" "env_0" {
  name = var.environment_name
}

resource "kaleido_platform_stack" "chain_infra_besu_stack" {
  environment = kaleido_platform_environment.env_0.id
  name = "chain_infra_besu_stack"
  type = "chain_infrastructure"
  network_id = kaleido_platform_network.besu_net.id
}

resource "kaleido_platform_stack" "chain_infra_ipfs_stack" {
  environment = kaleido_platform_environment.env_0.id
  name = "chain_infra_ipfs_stack"
  type = "chain_infrastructure"
  network_id = kaleido_platform_network.ipfs_net.id
}

resource "kaleido_platform_stack" "web3_middleware_stack" {
  for_each = toset(var.members)
  environment = kaleido_platform_environment.env_0.id
  name = "${each.key}_web3_middleware_stack"
  type = "web3_middleware"
}

resource "kaleido_platform_stack" "digital_assets_stack" {
  for_each = toset(var.members)
  environment = kaleido_platform_environment.env_0.id
  name = "${each.key}_digital_assets_stack"
  type = "digital_assets"
}

resource "kaleido_platform_network" "besu_net" {
  type = "BesuNetwork"
  name = "evmchain1"
  environment = kaleido_platform_environment.env_0.id
  config_json = jsonencode({
    bootstrapOptions = {
      qbft = {
        blockperiodseconds = 2
      }
    }
  })
}


resource "kaleido_platform_runtime" "bnr" {
  type = "BesuNode"
  name = "${var.environment_name}_chain_node_${count.index+1}"
  environment = kaleido_platform_environment.env_0.id
  config_json = jsonencode({})
  count = var.besu_node_count
  stack_id = kaleido_platform_stack.chain_infra_besu_stack.id
  // uncomment `force_delete = true` and run terraform apply before running terraform destory to successfully delete the besu nodes
  # force_delete = true
}

resource "kaleido_platform_service" "bns" {
  type = "BesuNode"
  name = "${var.environment_name}_chain_node_${count.index+1}"
  environment = kaleido_platform_environment.env_0.id
  runtime = kaleido_platform_runtime.bnr[count.index].id
  config_json = jsonencode({
    network = {
      id = kaleido_platform_network.besu_net.id
    }
  })
  count = var.besu_node_count
  stack_id = kaleido_platform_stack.chain_infra_besu_stack.id
  // uncomment `force_delete = true` and run terraform apply before running terraform destory to successfully delete the besu nodes
  # force_delete = true
}

resource "kaleido_platform_network" "ipfs_net" {
  type = "IPFSNetwork"
  name = "${var.environment_name}_ipfs"
  environment = kaleido_platform_environment.env_0.id
  config_json = jsonencode({})
}


resource "kaleido_platform_runtime" "inr_0" {
  type = "IPFSNode"
  name = "ipfs_node"
  environment = kaleido_platform_environment.env_0.id
  config_json = jsonencode({ })
  stack_id = kaleido_platform_stack.chain_infra_ipfs_stack.id
}

resource "kaleido_platform_service" "ins_0" {
  type = "IPFSNode"
  name = "ipfs_node"
  environment = kaleido_platform_environment.env_0.id
  runtime = kaleido_platform_runtime.inr_0.id
  config_json = jsonencode({
    network = {
      id = kaleido_platform_network.ipfs_net.id
    }
  })
  stack_id = kaleido_platform_stack.chain_infra_ipfs_stack.id
}

resource "kaleido_platform_runtime" "gwr_0" {
  type = "EVMGateway"
  name = "${var.environment_name}_evm_gateway"
  environment = kaleido_platform_environment.env_0.id
  config_json = jsonencode({})
  stack_id = kaleido_platform_stack.chain_infra_besu_stack.id
}

resource "kaleido_platform_service" "gws_0" {
  type = "EVMGateway"
  name = "${var.environment_name}_evm_gateway"
  environment = kaleido_platform_environment.env_0.id
  runtime = kaleido_platform_runtime.gwr_0.id
  config_json = jsonencode({
    network = {
      id =  kaleido_platform_network.besu_net.id
    }
  })
  stack_id = kaleido_platform_stack.chain_infra_besu_stack.id
}

data "kaleido_platform_evm_netinfo" "gws_0" {
  environment = kaleido_platform_environment.env_0.id
  service = kaleido_platform_service.gws_0.id
  depends_on = [
    kaleido_platform_service.bns,
    kaleido_platform_service.gws_0
  ]
}

resource "kaleido_platform_runtime" "kmr_0" {
  type = "KeyManager"
  name = "key_manager"
  environment = kaleido_platform_environment.env_0.id
  config_json = jsonencode({})
}

resource "kaleido_platform_service" "kms_0" {
  type = "KeyManager"
  name = "key_manager"
  environment = kaleido_platform_environment.env_0.id
  runtime = kaleido_platform_runtime.kmr_0.id
  config_json = jsonencode({})
}

resource "kaleido_platform_kms_wallet" "seed_wallet" {
  type = "hdwallet"
  name = "seed_wallet"
  environment = kaleido_platform_environment.env_0.id
  service = kaleido_platform_service.kms_0.id
  config_json = jsonencode({})
}

resource "kaleido_platform_kms_wallet" "org_wallets" {
  type = "hdwallet"
  name = "${each.key}_wallet"
  environment = kaleido_platform_environment.env_0.id
  service = kaleido_platform_service.kms_0.id
  config_json = jsonencode({})
  for_each = toset(var.members)
}

resource "kaleido_platform_kms_key" "seed_key" {
  name = "seed_key"
  environment = kaleido_platform_environment.env_0.id
  service = kaleido_platform_service.kms_0.id
  wallet = kaleido_platform_kms_wallet.seed_wallet.id
}

resource "kaleido_platform_kms_key" "org_keys" {
  name = "${each.key}_org_key"
  environment = kaleido_platform_environment.env_0.id
  service = kaleido_platform_service.kms_0.id
  wallet = kaleido_platform_kms_wallet.org_wallets[each.key].id
  for_each = toset(var.members)
}

resource "kaleido_platform_runtime" "tmr_0" {
  type = "TransactionManager"
  name = "${var.environment_name}_chain_txmanager"
  environment = kaleido_platform_environment.env_0.id
  config_json = jsonencode({})
  stack_id = kaleido_platform_stack.web3_middleware_stack[tolist(var.members)[0]].id
}

resource "kaleido_platform_service" "tms_0" {
  type = "TransactionManager"
  name = "${var.environment_name}_chain_txmanager"
  environment = kaleido_platform_environment.env_0.id
  runtime = kaleido_platform_runtime.tmr_0.id
  config_json = jsonencode({
    keyManager = {
      id: kaleido_platform_service.kms_0.id
    }
    type = "evm"
    evm = {
      confirmations = {
        required = 0
      }
      connector = {
        evmGateway = {
          id =  kaleido_platform_service.gws_0.id
        }
      }
    }
  })
  stack_id = kaleido_platform_stack.web3_middleware_stack[tolist(var.members)[0]].id
}


resource "kaleido_platform_runtime" "pdr_0" {
  type = "PrivateDataManager"
  name = "data_manager"
  environment = kaleido_platform_environment.env_0.id
  config_json = jsonencode({})
  stack_id = kaleido_platform_stack.web3_middleware_stack[tolist(var.members)[0]].id
}

resource "tls_private_key" "pdr_ca_private_key" {
  count = var.pdm_manage_p2p_tls ? 1 : 0

  algorithm = "RSA"
  rsa_bits = 4096
}

resource "tls_self_signed_cert" "pdr_ca_cert" {
  count = var.pdm_manage_p2p_tls ? 1 : 0

  private_key_pem = tls_private_key.pdr_ca_private_key[0].private_key_pem

  subject {
    common_name = "test.net"
    organization = "Multiparty Test Network"
  }

  allowed_uses = ["cert_signing"]

  is_ca_certificate = true
  validity_period_hours = 87660 # 10 years
}

resource "tls_private_key" "pdr_p2p_private_key" {
  count = var.pdm_manage_p2p_tls ? 1 : 0

  algorithm = "RSA"
  rsa_bits = 4096
}

resource "tls_cert_request" "pdr_p2p_cert_request" {
  count = var.pdm_manage_p2p_tls ? 1 : 0

  private_key_pem = tls_private_key.pdr_p2p_private_key[0].private_key_pem

  subject {
    common_name = "${replace(kaleido_platform_runtime.pdr_0.id, ":", "-")}-pdr.${var.pdm_runtime_endpoint_domain}"
    organization = var.pdm_service_peer_id
  }

  dns_names = ["${replace(kaleido_platform_runtime.pdr_0.id, ":", "-")}-pdr.${var.pdm_runtime_endpoint_domain}"]
}

resource "tls_locally_signed_cert" "pdr_p2p_cert" {
  count = var.pdm_manage_p2p_tls ? 1 : 0

  cert_request_pem = tls_cert_request.pdr_p2p_cert_request[0].cert_request_pem
  ca_private_key_pem = tls_private_key.pdr_ca_private_key[0].private_key_pem
  ca_cert_pem = tls_self_signed_cert.pdr_ca_cert[0].cert_pem

  allowed_uses = ["server_auth", "client_auth"]
  is_ca_certificate = false
  validity_period_hours = 87660 # 10 years
}


resource "kaleido_platform_service" "pds_0" {
  type = "PrivateDataManager"
  name = "data_manager"
  environment = kaleido_platform_environment.env_0.id
  runtime = kaleido_platform_runtime.pdr_0.id
  stack_id = kaleido_platform_stack.web3_middleware_stack[tolist(var.members)[0]].id
  config_json = var.pdm_manage_p2p_tls ? jsonencode({
    dataExchangeType = "https"
    https = var.pdm_manage_p2p_tls ? {
      peerId = var.pdm_service_peer_id
    } : null
    certificate =  {
      ca = {
        fileRef = "#certificate.ca"
      }
      cert = {
        fileRef = "#certificate.cert"
      }
      key = {
        fileRef = "#certificate.key"
      }
    }
  }) : jsonencode({ dataExchangeType = "https" })
  file_sets = var.pdm_manage_p2p_tls ? {
    certificate = {
      name = "certificate"
      files = {
        ca = {
          type = "pem"
          data = {
            text = tls_self_signed_cert.pdr_ca_cert[0].cert_pem
          }
        }
        cert = {
          type = "pem"
          data = {
            text = tls_locally_signed_cert.pdr_p2p_cert[0].cert_pem
          }
        }
        key = {
          type = "pem"
          data = {
            text = tls_private_key.pdr_p2p_private_key[0].private_key_pem
          }
        }
      }
    }
  } : null
}


resource "kaleido_platform_runtime" "cmr_0" {
  type = "ContractManager"
  name = "contract_manager"
  environment = kaleido_platform_environment.env_0.id
  config_json = jsonencode({})
}

resource "kaleido_platform_service" "cms_0" {
  type = "ContractManager"
  name = "contract_manager"
  environment = kaleido_platform_environment.env_0.id
  runtime = kaleido_platform_runtime.cmr_0.id
  config_json = jsonencode({})
}

resource "kaleido_platform_runtime" "bir_0"{
  type = "BlockIndexer"
  name = "block_indexer"
  environment = kaleido_platform_environment.env_0.id
  config_json = jsonencode({})
  stack_id = kaleido_platform_stack.chain_infra_besu_stack.id
}

resource "kaleido_platform_service" "bis_0"{
  type = "BlockIndexer"
  name = "block_indexer"
  environment = kaleido_platform_environment.env_0.id
  runtime = kaleido_platform_runtime.bir_0.id
  config_json=jsonencode(
    {
      contractManager = {
        id = kaleido_platform_service.cms_0.id
      }
      evmGateway = {
        id = kaleido_platform_service.gws_0.id
      }
    }
  )
  hostnames = {"${kaleido_platform_network.besu_net.name}" = ["ui", "rest"]}
  stack_id = kaleido_platform_stack.chain_infra_besu_stack.id
}

resource "kaleido_platform_cms_build" "firefly_batch_pin" {
  environment = kaleido_platform_environment.env_0.id
  service = kaleido_platform_service.cms_0.id
  type = "github"
  name = "batch_pin_v1.3"
  path = "firefly"
	github = {
		contract_url = "https://github.com/hyperledger/firefly/blob/main/smart_contracts/ethereum/solidity_firefly/contracts/Firefly.sol"
		contract_name = "Firefly"
	}
}

resource "kaleido_platform_cms_action_deploy" "firefly_batch_pin" {
  environment = kaleido_platform_environment.env_0.id
  service = kaleido_platform_service.cms_0.id
  build = kaleido_platform_cms_build.firefly_batch_pin.id
  name = "firefly_batch_pin"
  transaction_manager = kaleido_platform_service.tms_0.id
  signing_key = kaleido_platform_kms_key.seed_key.address
  depends_on = [ data.kaleido_platform_evm_netinfo.gws_0 ]
}

resource "kaleido_platform_runtime" "ffr_0" {
  for_each = toset(var.members)
  type = "FireFly"
  name = "${each.key}_firefly_runtime"
  environment = kaleido_platform_environment.env_0.id
  config_json = jsonencode({})
  stack_id = kaleido_platform_stack.web3_middleware_stack[each.key].id
}

resource "kaleido_platform_service" "member_firefly" {
  type = "FireFly"
  name = "${each.key}_firefly"
  environment = kaleido_platform_environment.env_0.id
  runtime = kaleido_platform_runtime.ffr_0[each.key].id
  config_json = jsonencode({
    transactionManager = {
      id = kaleido_platform_service.tms_0.id
    }
    ipfs = {
      ipfsService = {
        id = kaleido_platform_service.ins_0.id
      }
    }
    privatedatamanager = {
      id = kaleido_platform_service.pds_0.id
    }
    multiparty = {
      enabled = true
      networkNamespace = var.environment_name
      orgName = each.key
      orgKey = kaleido_platform_kms_key.org_keys[each.key].address
      contracts = [
          {
            address = kaleido_platform_cms_action_deploy.firefly_batch_pin.contract_address
            blockNumber= kaleido_platform_cms_action_deploy.firefly_batch_pin.block_number
          }
      ]
    }
  })
  for_each = toset(var.members)
  stack_id = kaleido_platform_stack.web3_middleware_stack[each.key].id
}

resource "kaleido_platform_firefly_registration" "registrations" {
  environment = kaleido_platform_environment.env_0.id
  service = kaleido_platform_service.member_firefly[each.key].id
  for_each = toset(var.members)
}

resource "kaleido_platform_runtime" "asset_managers" {
  type = "AssetManager"
  name = "${each.key}_assets"
  environment = kaleido_platform_environment.env_0.id
  config_json = jsonencode({})
  for_each = toset(var.members)
  stack_id = kaleido_platform_stack.digital_assets_stack[each.key].id
}

resource "kaleido_platform_service" "asset_managers" {
  type = "AssetManager"
  name = "${each.key}_assets"
  environment = kaleido_platform_environment.env_0.id
  runtime = kaleido_platform_runtime.asset_managers[each.key].id
  config_json = jsonencode({
    keyManager = {
      id: kaleido_platform_service.kms_0.id
    }
  })
  for_each = toset(var.members)
  stack_id = kaleido_platform_stack.digital_assets_stack[each.key].id
}

