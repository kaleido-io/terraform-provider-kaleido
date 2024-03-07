terraform {
  required_providers {
    kaleido = {
      source = "kaleido-io/kaleido"
      version = "0.2.15"
    }
  }
}

provider "kaleido" {
  platform_api = var.kaleido_platform_api
  platform_username = var.kaleido_platform_username
  platform_password = var.kaleido_platform_password
}

resource "kaleido_platform_runtime" "kmr_0" {
  type = "KeyManager"
  name = "kmr_0"
  environment = var.environment
  config_json = jsonencode({})
}

resource "kaleido_platform_service" "kms_0" {
  type = "KeyManager"
  name = "kms_0"
  environment = var.environment
  runtime = kaleido_platform_runtime.kmr_0.id
  config_json = jsonencode({})
}

resource "kaleido_platform_runtime" "tmr_0" {
  type = "TransactionManager"
  name = "tmr_0"
  environment = var.environment
  config_json = jsonencode({})
}

resource "kaleido_platform_service" "tms_0" {
  type = "TransactionManager"
  name = "tms_0"
  environment = var.environment
  runtime = kaleido_platform_runtime.tmr_0.id
  config_json = jsonencode({
    keyManager = {
      id: kaleido_platform_service.kms_0.id
    }
    type = "evm"
    evm = {
      connector = {
        url = var.json_rpc_url
        auth = {
          credSetRef = "rpc_auth"
        }
      }
    }
  })
  cred_sets = {
    "rpc_auth" = {
      type = "basic_auth"
      basic_auth = {
        username = var.json_rpc_username
        password = var.json_rpc_password
      }
    }
  }
}

resource "kaleido_platform_runtime" "ffr_0" {
  type = "FireFly"
  name = "ffr_0"
  environment = var.environment
  config_json = jsonencode({})
}

resource "kaleido_platform_service" "ffs_0" {
  type = "FireFly"
  name = "ffs_0"
  environment = var.environment
  runtime = kaleido_platform_runtime.ffr_0.id
  config_json = jsonencode({
    transactionManager = kaleido_platform_service.tms_0.id
  })
}
