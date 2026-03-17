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
          "url":var.rpc_url, 
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
       username = var.username
       password = var.password
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

output "key_address" {
  value = kaleido_platform_kms_key.key_0.address
}