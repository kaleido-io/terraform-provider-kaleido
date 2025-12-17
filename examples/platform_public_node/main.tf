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

resource "kaleido_platform_stack" "web3_middleware_stack" {
  environment = kaleido_platform_environment.env_0.id
  name = "web3_middleware_stack"
  type = "web3_middleware"
}

resource "kaleido_platform_runtime" "kmr_0" {
  type = "KeyManager"
  name = "kms1"
  environment = kaleido_platform_environment.env_0.id
  config_json = jsonencode({})
}

resource "kaleido_platform_service" "kms_0" {
  type = "KeyManager"
  name = "kms1"
  environment = kaleido_platform_environment.env_0.id
  runtime = kaleido_platform_runtime.kmr_0.id
  config_json = jsonencode({})
}

resource "kaleido_platform_kms_wallet" "wallet_0" {
  type = "hdwallet"
  name = "hdwallet1"
  environment = kaleido_platform_environment.env_0.id
  service = kaleido_platform_service.kms_0.id
  config_json = jsonencode({})
}

resource "kaleido_platform_kms_key" "key_0" {
  name = "key0"
  environment = kaleido_platform_environment.env_0.id
  service = kaleido_platform_service.kms_0.id
  wallet = kaleido_platform_kms_wallet.wallet_0.id
}

resource "kaleido_platform_runtime" "tmr_0" {
  type = "TransactionManager"
  name = "evmchain1_txmgr"
  environment = kaleido_platform_environment.env_0.id
  config_json = jsonencode({})
  stack_id = kaleido_platform_stack.web3_middleware_stack.id
}

resource "kaleido_platform_service" "tms_0" {
  type = "TransactionManager"
  name = "evmchain1_txmgr"
  environment = kaleido_platform_environment.env_0.id
  runtime = kaleido_platform_runtime.tmr_0.id
  stack_id = kaleido_platform_stack.web3_middleware_stack.id
  config_json = jsonencode({
    keyManager = {
      id: kaleido_platform_service.kms_0.id
    }
    type = "evm"
    evm = {
      confirmations = {
        required = 6
      }
      "connector":{
          "url":"insert public chain rpc endpoint", 
          "auth":{
              "credSetRef":"rpc_auth"
          },
          "events":{
              "catchupPageSize":5
          }
        }
      "transactions":{
        "handler":{
            "enterprise":{
              "gasPrice":{
                  "enableNodeGasPrice":true
              }
          }
        }
      }
    }
    })
  cred_sets = {
   "rpc_auth" = {
     type = "basic_auth"
     basic_auth = {
       username = "insert public chain rpc auth cred"
       password = "insert public chain rpc auth cred"
     }
   }
  }
}

resource "kaleido_platform_runtime" "ffr_0" {
  type = "FireFly"
  name = "firefly1"
  environment = kaleido_platform_environment.env_0.id
  config_json = jsonencode({})
  stack_id = kaleido_platform_stack.web3_middleware_stack.id
}

resource "kaleido_platform_service" "ffs_0" {
  type = "FireFly"
  name = "firefly1"
  environment = kaleido_platform_environment.env_0.id
  runtime = kaleido_platform_runtime.ffr_0.id
  stack_id = kaleido_platform_stack.web3_middleware_stack.id
  config_json = jsonencode({
    transactionManager = {
      id = kaleido_platform_service.tms_0.id
    }
  })
}

resource "kaleido_platform_runtime" "cmr_0" {
  type = "ContractManager"
  name = "smart_contract_manager"
  environment = kaleido_platform_environment.env_0.id
  config_json = jsonencode({})
}

resource "kaleido_platform_service" "cms_0" {
  type = "ContractManager"
  name = "smart_contract_manager"
  environment = kaleido_platform_environment.env_0.id
  runtime = kaleido_platform_runtime.cmr_0.id
  config_json = jsonencode({})
}


resource "kaleido_platform_cms_build" "erc20" {
  environment = kaleido_platform_environment.env_0.id
  service = kaleido_platform_service.cms_0.id
  type = "github"
  name = "ERC20WithData"
  path = "erc20_samples"
	github = {
		contract_url = "https://github.com/hyperledger/firefly-tokens-erc20-erc721/blob/main/samples/solidity/contracts/ERC20WithData.sol"
		contract_name = "ERC20WithData"
	}
}

/* This is where we could put the following resources to actually deploy a contract and generate an API for it, but currently it requires the manual step of getting gas for the key
resource "kaleido_platform_cms_action_deploy" "demotoken_erc20" {
  environment = kaleido_platform_environment.env_0.id
  service = kaleido_platform_service.cms_0.id
  build = kaleido_platform_cms_build.erc20.id
  name = "deploy_erc20"
  firefly_namespace = kaleido_platform_service.ffs_0.name
  signing_key = kaleido_platform_kms_key.key_0.address
  params_json = jsonencode([
    "DemoToken",
    "DTOK"
  ])
}

resource "kaleido_platform_cms_action_createapi" "erc20" {
  environment = kaleido_platform_environment.env_0.id
  service = kaleido_platform_service.cms_0.id
  build = kaleido_platform_cms_build.erc20.id
  name = "erc20withdata"
  firefly_namespace = kaleido_platform_service.ffs_0.name
  api_name = "erc20withdata"
}
*/