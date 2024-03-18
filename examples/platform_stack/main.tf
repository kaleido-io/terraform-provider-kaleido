terraform {
  required_providers {
    kaleido = {
      source = "kaleido-io/kaleido"
      version = "1.1.0-rc.1"
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
  name = "net_0"
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
  name = "bnr_${count.index}"
  environment = kaleido_platform_environment.env_0.id
  config_json = jsonencode({})
  count = var.node_count
}

resource "kaleido_platform_service" "bns" {
  type = "BesuNode"
  name = "bns_${count.index}"
  environment = kaleido_platform_environment.env_0.id
  runtime = kaleido_platform_runtime.bnr[count.index].id
  config_json = jsonencode({
    network = {
      id = kaleido_platform_network.net_0.id
    }
  })
  count = var.node_count
}

resource "kaleido_platform_runtime" "gwr_0" {
  type = "EVMGateway"
  name = "gwr_0"
  environment = kaleido_platform_environment.env_0.id
  config_json = jsonencode({})
}

resource "kaleido_platform_service" "gws_0" {
  type = "EVMGateway"
  name = "gws_0"
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
  name = "kmr_0"
  environment = kaleido_platform_environment.env_0.id
  config_json = jsonencode({})
}

resource "kaleido_platform_service" "kms_0" {
  type = "KeyManager"
  name = "kms_0"
  environment = kaleido_platform_environment.env_0.id
  runtime = kaleido_platform_runtime.kmr_0.id
  config_json = jsonencode({})
}

resource "kaleido_platform_kms_wallet" "wallet_0" {
  type = "hdwallet"
  name = "wallet_0"
  environment = kaleido_platform_environment.env_0.id
  service = kaleido_platform_service.kms_0.id
  config_json = jsonencode({})
}

resource "kaleido_platform_kms_key" "key_0" {
  name = "key_0"
  environment = kaleido_platform_environment.env_0.id
  service = kaleido_platform_service.kms_0.id
  wallet = kaleido_platform_kms_wallet.wallet_0.id
}

resource "kaleido_platform_runtime" "tmr_0" {
  type = "TransactionManager"
  name = "tmr_0"
  environment = kaleido_platform_environment.env_0.id
  config_json = jsonencode({})
}

resource "kaleido_platform_service" "tms_0" {
  type = "TransactionManager"
  name = "tms_0"
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
        # url = var.json_rpc_url
        # auth = {
        #   credSetRef = "rpc_auth"
        # }
      }
    }
  })
  # cred_sets = {
  #   "rpc_auth" = {
  #     type = "basic_auth"
  #     basic_auth = {
  #       username = var.json_rpc_username
  #       password = var.json_rpc_password
  #     }
  #   }
  # }
}

resource "kaleido_platform_runtime" "ffr_0" {
  type = "FireFly"
  name = "ffr_0"
  environment = kaleido_platform_environment.env_0.id
  config_json = jsonencode({})
}

resource "kaleido_platform_service" "ffs_0" {
  type = "FireFly"
  name = "ffs_0"
  environment = kaleido_platform_environment.env_0.id
  runtime = kaleido_platform_runtime.ffr_0.id
  config_json = jsonencode({
    transactionManager = {
      id = kaleido_platform_service.tms_0.id
    }
  })
}

resource "kaleido_platform_runtime" "cmr_0" {
  type = "ContractManager"
  name = "cmr_0"
  environment = kaleido_platform_environment.env_0.id
  config_json = jsonencode({})
}

resource "kaleido_platform_service" "cms_0" {
  type = "ContractManager"
  name = "cms_0"
  environment = kaleido_platform_environment.env_0.id
  runtime = kaleido_platform_runtime.cmr_0.id
  config_json = jsonencode({})
}

resource "kaleido_platform_runtime" "amr_0" {
  type = "AssetManager"
  name = "amr_0"
  environment = kaleido_platform_environment.env_0.id
  config_json = jsonencode({})
}

resource "kaleido_platform_service" "ams_0" {
  type = "AssetManager"
  name = "ams_0"
  environment = kaleido_platform_environment.env_0.id
  runtime = kaleido_platform_runtime.amr_0.id
  config_json = jsonencode({
    keyManager = {
      id: kaleido_platform_service.kms_0.id
    }
  })
}

resource "kaleido_platform_cms_build" "contract_0" {
  environment = kaleido_platform_environment.env_0.id
  service = kaleido_platform_service.cms_0.id
  type = "github"
  name = "ff"
  path = "firefly"
	github = {
		contract_url = "https://github.com/hyperledger/firefly/blob/main/smart_contracts/ethereum/solidity_firefly/contracts/Firefly.sol"
		contract_name = "Firefly"
	}
}

resource "kaleido_platform_cms_action_deploy" "deploy_0" {
  environment = kaleido_platform_environment.env_0.id
  service = kaleido_platform_service.cms_0.id
  build = kaleido_platform_cms_build.contract_0.id
  name = "deploy_0"
  firefly_namespace = kaleido_platform_service.ffs_0.name
  signing_key = kaleido_platform_kms_key.key_0.address
  depends_on = [ data.kaleido_platform_evm_netinfo.gws_0 ]
}

resource "kaleido_platform_cms_action_creatapi" "api_0" {
  environment = kaleido_platform_environment.env_0.id
  service = kaleido_platform_service.cms_0.id
  build = kaleido_platform_cms_build.contract_0.id
  name = "api_0"
  firefly_namespace = kaleido_platform_service.ffs_0.name
  api_name = "firefly"
  contract_address = kaleido_platform_cms_action_deploy.deploy_0.contract_address
  depends_on = [ data.kaleido_platform_evm_netinfo.gws_0 ]
}

resource "kaleido_platform_ams_task" "task_0" {
  environment = kaleido_platform_environment.env_0.id
  service = kaleido_platform_service.ams_0.id
  task_yaml = <<EOT
    name: task1
    steps:
    - name: demostep1
      type: jsonata_template
      options:
        template: |-
          "hello world"
  EOT
}