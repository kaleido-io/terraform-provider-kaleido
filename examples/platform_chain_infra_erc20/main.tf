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

resource "kaleido_platform_stack" "chain_infra_stack" {
  environment = kaleido_platform_environment.env_0.id
  name = "chain_infra_besu_stack"
  type = "chain_infrastructure"
  network_id = kaleido_platform_network.net_0.id
}

resource "kaleido_platform_stack" "web3_middleware_stack" {
  environment = kaleido_platform_environment.env_0.id
  name = "web3_middleware_stack"
  type = "web3_middleware"
}

resource "kaleido_platform_network" "net_0" {
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
  name = "evmchain1_node_${count.index+1}"
  environment = kaleido_platform_environment.env_0.id
  config_json = jsonencode({})
  count = var.besu_node_count
  stack_id = kaleido_platform_stack.chain_infra_stack.id
  // uncomment `force_delete = true` and run terraform apply before running terraform destory to successfully delete the besu nodes
  # force_delete = true
}

resource "kaleido_platform_service" "bns" {
  type = "BesuNode"
  name = "evmchain1_node${count.index+1}"
  environment = kaleido_platform_environment.env_0.id
  stack_id = kaleido_platform_stack.chain_infra_stack.id
  runtime = kaleido_platform_runtime.bnr[count.index].id
  config_json = jsonencode({
    network = {
      id = kaleido_platform_network.net_0.id
    }
  })
  count = var.besu_node_count
  // uncomment `force_delete = true` and run terraform apply before running terraform destory to successfully delete the besu nodes
  # force_delete = true
}

resource "kaleido_platform_runtime" "gwr_0" {
  type = "EVMGateway"
  name = "evmchain1_gateway"
  stack_id = kaleido_platform_stack.chain_infra_stack.id
  environment = kaleido_platform_environment.env_0.id
  config_json = jsonencode({})
}

resource "kaleido_platform_service" "gws_0" {
  type = "EVMGateway"
  name = "evmchain1_gateway"
  stack_id = kaleido_platform_stack.chain_infra_stack.id
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

resource "kaleido_platform_runtime" "bir_0"{
  type = "BlockIndexer"
  name = "block_indexer"
  environment = kaleido_platform_environment.env_0.id
  config_json = jsonencode({})
  stack_id = kaleido_platform_stack.chain_infra_stack.id
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
  # Ensure hostname is valid: alphanumeric and - only, no underscores
  # Replace underscores and other invalid chars with dashes
  hostnames = {"${replace("${lower(replace(var.environment_name, "/[^\\w]/", ""))}-${kaleido_platform_network.net_0.name}", "_", "-")}" = ["ui", "rest"]}
  stack_id = kaleido_platform_stack.chain_infra_stack.id
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
  depends_on = [ data.kaleido_platform_evm_netinfo.gws_0 ]
}

resource "kaleido_platform_cms_action_createapi" "erc20" {
  environment = kaleido_platform_environment.env_0.id
  service = kaleido_platform_service.cms_0.id
  build = kaleido_platform_cms_build.erc20.id
  name = "erc20withdata"
  firefly_namespace = kaleido_platform_service.ffs_0.name
  api_name = "erc20withdata"
  depends_on = [ data.kaleido_platform_evm_netinfo.gws_0 ]
}

// Create FireFly subscription for blockchain events with webhook delivery
resource "kaleido_platform_firefly_subscription" "erc20_transfer_webhook" {
  environment = kaleido_platform_environment.env_0.id
  service = kaleido_platform_service.ffs_0.id
  namespace = kaleido_platform_service.ffs_0.name
  name = "erc20_transfer_webhook"
  config_json = jsonencode({
    transport = "webhooks"
    filter = {
      // Filter by event type - blockchain_event_received is emitted when contract listeners detect events
      events = "blockchain_event_received"
      // Filter by topic to match events from the contract listener (topic is set on the listener)
      topic = "erc20-transfer"
    }
    options = {
      withData = true
      url = "http://localhost:9090/webhook"
      // Webhook-specific options can be set here:
      // method = "POST"  // HTTP method (default: POST)
      // headers = {}     // Static headers to set on the webhook request
      // query = {}       // Static query params to set on the webhook request
      // tlsConfigName = "my-tls-config"  // Name of TLS config associated with the namespace
      // retry = {        // Retry options
      //   enabled = true
      //   count = 3
      //   initialDelay = "1s"
      //   maxDelay = "30s"
      // }
      // httpOptions = {  // HTTP connection options
      //   requestTimeout = "30s"
      //   connectionTimeout = "10s"
      // }
    }
  })
  depends_on = [
    kaleido_platform_service.ffs_0,
    kaleido_platform_cms_action_deploy.demotoken_erc20
  ]
}

// Create FireFly contract listener for ERC20 Transfer events
resource "kaleido_platform_firefly_contract_listener" "erc20_transfer" {
  environment = kaleido_platform_environment.env_0.id
  service = kaleido_platform_service.ffs_0.id
  namespace = kaleido_platform_service.ffs_0.name
  name = "erc20-transfer"  // Explicit name (using dash instead of underscore for better compatibility)
  config_json = jsonencode({
    location = {
      address = kaleido_platform_cms_action_deploy.demotoken_erc20.contract_address
    }
    event = {
      name = "Transfer"
      description = ""
      params = [
        {
          name = "from"
          schema = {
            type = "string"
            details = {
              type = "address"
              internalType = "address"
              indexed = true
            }
            description = "A hex encoded set of bytes, with an optional '0x' prefix"
          }
        },
        {
          name = "to"
          schema = {
            type = "string"
            details = {
              type = "address"
              internalType = "address"
              indexed = true
            }
            description = "A hex encoded set of bytes, with an optional '0x' prefix"
          }
        },
        {
          name = "value"
          schema = {
            oneOf = [
              {
                type = "string"
              },
              {
                type = "integer"
              }
            ]
            details = {
              type = "uint256"
              internalType = "uint256"
            }
            description = "An integer. You are recommended to use a JSON string. A JSON number can be used for values up to the safe maximum."
          }
        }
      ]
    }
    topic = "erc20-transfer"  // Topic must be at top level, not in options
    options = {
      firstEvent = "oldest"
    }
  })
  depends_on = [
    kaleido_platform_service.ffs_0,
    kaleido_platform_cms_action_deploy.demotoken_erc20,
    kaleido_platform_firefly_subscription.erc20_transfer_webhook
  ]
}
