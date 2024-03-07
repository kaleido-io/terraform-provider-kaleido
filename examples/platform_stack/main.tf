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

resource "kaleido_platform_network" "net_0" {
  type = "Besu"
  name = "net_0"
  environment = kaleido_platform_environment.env_0.id
  config_json = jsonencode({})
}


resource "kaleido_platform_runtime" "bnr" {
  type = "BesuNode"
  name = "bnr_${count.index}"
  environment = kaleido_platform_environment.env_0.id
  config_json = jsonencode({})
  count = var.node_count
}

# resource "kaleido_platform_service" "bns" {
#   type = "BesuNode"
#   name = "bns_${count.index}"
#   environment = kaleido_platform_environment.env_0.id
#   runtime = kaleido_platform_runtime.bnr[count.index].id
#   config_json = jsonencode({
#     network = {
#       id = kaleido_platform_network.net_0.id
#     }
#   })
#   count = var.node_count
# }

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
    key = kaleido_platform_kms_key.key_0.uri
  })
}
