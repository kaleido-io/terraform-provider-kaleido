terraform {
  required_providers {
    kaleido = {
      source = "kaleido-io/kaleido"
      version = "1.1.0-rc.2"
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

resource "kaleido_platform_network" "net_0" {
  type = "Besu"
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
  name = "${var.environment_name}_chain_node_${count.index}"
  environment = kaleido_platform_environment.env_0.id
  config_json = jsonencode({})
  count = var.besu_node_count
}

resource "kaleido_platform_service" "bns" {
  type = "BesuNode"
  name = "${var.environment_name}_chain_node_${count.index}"
  environment = kaleido_platform_environment.env_0.id
  runtime = kaleido_platform_runtime.bnr[count.index].id
  config_json = jsonencode({
    network = {
      id = kaleido_platform_network.net_0.id
    }
  })
  count = var.besu_node_count
}

resource "kaleido_platform_network" "net_ipfs" {
  type = "IPFS"
  name = "${var.environment_name}_ipfs"
  environment = kaleido_platform_environment.env_0.id
  config_json = jsonencode({})
}


resource "kaleido_platform_runtime" "inr_0" {
  type = "IPFSNode"
  name = "ipfs_node"
  environment = kaleido_platform_environment.env_0.id
  config_json = jsonencode({ })
}

resource "kaleido_platform_service" "ins_0" {
  type = "IPFSNode"
  name = "ipfs_node"
  environment = kaleido_platform_environment.env_0.id
  runtime = kaleido_platform_runtime.inr_0.id
  config_json = jsonencode({
    network = {
      id = kaleido_platform_network.net_ipfs.id
    }
  })
}

resource "kaleido_platform_runtime" "gwr_0" {
  type = "EVMGateway"
  name = "${var.environment_name}_evm_gateway"
  environment = kaleido_platform_environment.env_0.id
  config_json = jsonencode({})
}

resource "kaleido_platform_service" "gws_0" {
  type = "EVMGateway"
  name = "${var.environment_name}_evm_gateway"
  environment = kaleido_platform_environment.env_0.id
  runtime = kaleido_platform_runtime.gwr_0.id
  config_json = jsonencode({
    network = {
      id =  kaleido_platform_network.net_0.id
    }
  })
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
}

resource "kaleido_platform_runtime" "ffr_0" {
  type = "FireFly"
  name = "firefly_runtime"
  environment = kaleido_platform_environment.env_0.id
  config_json = jsonencode({})
}

resource "kaleido_platform_service" "seed_firefly" {
  type = "FireFly"
  name = "seed"
  environment = kaleido_platform_environment.env_0.id
  runtime = kaleido_platform_runtime.ffr_0.id
  config_json = jsonencode({
    transactionManager = {
      id = kaleido_platform_service.tms_0.id
    }
  })
}

resource "kaleido_platform_runtime" "pdr_0" {
  type = "PrivateDataManager"
  name = "data_manager"
  environment = kaleido_platform_environment.env_0.id
  config_json = jsonencode({})
}


resource "kaleido_platform_service" "pds_0" {
  type = "PrivateDataManager"
  name = "data_manager"
  environment = kaleido_platform_environment.env_0.id
  runtime = kaleido_platform_runtime.pdr_0.id
  config_json = jsonencode({
    dataExchangeType = "https"
  })
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
  hostnames = {"${kaleido_platform_network.net_0.name}" = ["ui", "rest"]}
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
  firefly_namespace = kaleido_platform_service.seed_firefly.name
  signing_key = kaleido_platform_kms_key.seed_key.address
  depends_on = [ data.kaleido_platform_evm_netinfo.gws_0 ]
}

resource "kaleido_platform_service" "member_firefly" {
  type = "FireFly"
  name = "${each.key}_firefly"
  environment = kaleido_platform_environment.env_0.id
  runtime = kaleido_platform_runtime.ffr_0.id
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
}

