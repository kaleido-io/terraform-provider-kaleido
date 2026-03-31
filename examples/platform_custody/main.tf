resource "kaleido_platform_environment" "env_0" {
  name = var.environment_name
}

## KMS

resource "kaleido_platform_runtime" "keys_runtime" {
  type = "KeyManager"
  name = "keys"
  environment = kaleido_platform_environment.env_0.id
  config_json = jsonencode({})
}

resource "kaleido_platform_service" "keys_service" {
  type = "KeyManager"
  name = "keys"
  environment = kaleido_platform_environment.env_0.id
  runtime = kaleido_platform_runtime.keys_runtime.id
  config_json = jsonencode({})

  database_name = var.databases != null ? var.databases.kms_db : null
}

resource "kaleido_platform_kms_wallet" "infra" {
  type = "hdwallet"
  name = "infra"
  environment = kaleido_platform_environment.env_0.id
  service = kaleido_platform_service.keys_service.id
  config_json = jsonencode({})
}

resource "kaleido_platform_kms_key" "deployer" {
  name = "deployer"
  environment = kaleido_platform_environment.env_0.id
  service = kaleido_platform_service.keys_service.id
  wallet = kaleido_platform_kms_wallet.infra.id
}


resource "kaleido_platform_kms_wallet" "users" {
  type = "hdwallet"
  name = "users"
  environment = kaleido_platform_environment.env_0.id
  service = kaleido_platform_service.keys_service.id
  config_json = jsonencode({})
}

## Contract management

resource "kaleido_platform_runtime" "contracts_runtime" {
  type = "ContractManager"
  name = "contracts"
  environment = kaleido_platform_environment.env_0.id
  config_json = jsonencode({})
}

resource "kaleido_platform_service" "contracts_service" {
  type = "ContractManager"
  name = "contracts"
  environment = kaleido_platform_environment.env_0.id
  runtime = kaleido_platform_runtime.contracts_runtime.id
  config_json = jsonencode({})

  database_name = var.databases != null ? var.databases.cms_db : null
}

resource "kaleido_platform_cms_build" "erc20" {
  environment = kaleido_platform_environment.env_0.id
  service = kaleido_platform_service.contracts_service.id
  type = "precompiled"
  name = "ERC20"
  path = "Samples"
  precompiled = {
    abi = ""
    bytecode = ""
  }
}

## FireFly v2 workflow engine

resource "kaleido_platform_runtime" "workflow_engine_runtime" {
  type = "WorkflowEngine"
  name = "flows"
  environment = kaleido_platform_environment.env_0.id
  config_json = jsonencode({})
}

resource "kaleido_platform_service" "workflow_engine_service" {
  type = "WorkflowEngine"
  name = "flows"
  environment = kaleido_platform_environment.env_0.id 
  runtime = kaleido_platform_runtime.workflow_engine_runtime.id
  config_json = jsonencode({})

  database_name = var.databases != null ? var.databases.wfe_db : null
}

## Chain infrastructure: Besu testnet

resource "kaleido_platform_network" "besu_network" {
  type = "BesuNetwork"
  name = "testnet"
  environment = kaleido_platform_environment.env_0.id
  init_mode = "automated"

  config_json = jsonencode({
    chainID = 3333
    bootstrapOptions = {
      qbft = {
        blockperiodseconds = 2
      }
      eipBlockConfig = {
        shanghaiTime = 0
      }
      blockConfigFlags = {
        zeroBaseFee = true
      }
    }
  })
}

resource "kaleido_platform_stack" "besu_stack" {
  environment = kaleido_platform_environment.env_0.id
  name        = "testnet"
  type        = "chain_infrastructure"
  sub_type    = "BesuStack"
  network_id  = kaleido_platform_network.besu_network.id
}

resource "kaleido_platform_runtime" "besu_node_runtime" {
  type        = "BesuNode"
  name        = "node-0"
  environment = kaleido_platform_environment.env_0.id
  config_json = jsonencode({})
  stack_id    = kaleido_platform_stack.besu_stack.id
}

resource "kaleido_platform_service" "besu_node_service" {
  type        = "BesuNode"
  name        = "node-0"
  environment = kaleido_platform_environment.env_0.id
  runtime     = kaleido_platform_runtime.besu_node_runtime.id
  stack_id    = kaleido_platform_stack.besu_stack.id
  config_json = jsonencode({
    network = {
      id = kaleido_platform_network.besu_network.id
    }
    signer            = true
    customBesuArgs    = ["--revert-reason-enabled=true"]
    dataStorageFormat = "BONSAI"
    logLevel          = "INFO"
  })
}

resource "kaleido_platform_runtime" "evmgw_runtime" {
  type        = "EVMGateway"
  name        = "evmgateway"
  environment = kaleido_platform_environment.env_0.id
  config_json = jsonencode({})
  stack_id    = kaleido_platform_stack.besu_stack.id
}

resource "kaleido_platform_service" "evmgw_service" {
  type        = "EVMGateway"
  name        = "evmgateway"
  environment = kaleido_platform_environment.env_0.id
  runtime     = kaleido_platform_runtime.evmgw_runtime.id
  stack_id    = kaleido_platform_stack.besu_stack.id
  config_json = jsonencode({
    network = {
      id = kaleido_platform_network.besu_network.id
    }
  })
}

resource "kaleido_platform_runtime" "blockindexer_runtime" {
  type        = "BlockIndexer"
  name        = "blockindexer"
  environment = kaleido_platform_environment.env_0.id
  config_json = jsonencode({})
  stack_id    = kaleido_platform_stack.besu_stack.id
}

resource "kaleido_platform_service" "blockindexer_service" {
  type        = "BlockIndexer"
  name        = "blockindexer"
  environment = kaleido_platform_environment.env_0.id
  runtime     = kaleido_platform_runtime.blockindexer_runtime.id
  stack_id    = kaleido_platform_stack.besu_stack.id
  config_json = jsonencode({
    contractManager = {
      id = kaleido_platform_service.contracts_service.id
    }
    evmGateway = {
      id = kaleido_platform_service.evmgw_service.id
    }
  })

  database_name = var.databases != null ? var.databases.bis_db : null
}

## Web3 middleware: EVM connector

resource "kaleido_platform_stack" "evmconnector_stack" {
  environment = kaleido_platform_environment.env_0.id
  name        = "evm"
  type        = "web3_middleware"
  sub_type    = "EVMConnectorStack"
}

resource "kaleido_platform_runtime" "evmconnector_runtime" {
  type        = "EVMConnector"
  name        = "connector"
  environment = kaleido_platform_environment.env_0.id
  config_json = jsonencode({})
  stack_id    = kaleido_platform_stack.evmconnector_stack.id
}

resource "kaleido_platform_service" "evmconnector_service" {
  type        = "EVMConnector"
  name        = "connector"
  environment = kaleido_platform_environment.env_0.id
  runtime     = kaleido_platform_runtime.evmconnector_runtime.id
  stack_id    = kaleido_platform_stack.evmconnector_stack.id
  config_json = jsonencode({
    evmGateway = {
      id = kaleido_platform_service.evmgw_service.id
    }
    keyManager = {
      id = kaleido_platform_service.keys_service.id
    }
    ecosystem = {
      name        = "besu"
      displayName = "Besu"
    }
    network = {
      chainId = "3333"
      name    = "testnet"
    }
  })

  database_name = var.databases != null ? var.databases.evmconnector_db : null

  depends_on = [
    kaleido_platform_service.evmgw_service,
  ]
}

## Digital assets: tokenization stack

resource "kaleido_platform_stack" "tokenization_stack" {
  environment = kaleido_platform_environment.env_0.id
  name = "tokenization"
  type = "digital_assets"
  sub_type = "TokenizationStack"
}

resource "kaleido_platform_runtime" "tokenization_runtime" {
  type = "AssetManager"
  name = "assets"
  environment = kaleido_platform_environment.env_0.id
  config_json = jsonencode({})
  stack_id = kaleido_platform_stack.tokenization_stack.id
}

resource "kaleido_platform_service" "tokenization_service" {
  type = "AssetManager"
  name = "assets"
  environment = kaleido_platform_environment.env_0.id
  runtime = kaleido_platform_runtime.tokenization_runtime.id
  config_json = jsonencode({
    keyManager = {
      id = kaleido_platform_service.keys_service.id
    }
  })
  stack_id = kaleido_platform_stack.tokenization_stack.id
}

## Digital assets: custody

resource "kaleido_platform_stack" "custody_stack" {
  environment = kaleido_platform_environment.env_0.id
  name = "custody"
  type = "digital_assets"
  sub_type = "CustodyStack"
}

resource "kaleido_platform_runtime" "custody_runtime" {
  type = "WalletManager"
  name = "wallets"
  environment = kaleido_platform_environment.env_0.id
  config_json = jsonencode({})
  stack_id = kaleido_platform_stack.custody_stack.id
}

resource "kaleido_platform_service" "custody_service" {
  type        = "WalletManager"
  name        = "wallets"
  environment = kaleido_platform_environment.env_0.id
  runtime     = kaleido_platform_runtime.custody_runtime.id
  config_json = jsonencode({
    keyManager = {
      id = kaleido_platform_service.keys_service.id
    }
    assetManagerService = {
      name = kaleido_platform_service.tokenization_service.name
      service = {
        id = kaleido_platform_service.tokenization_service.id
      }
    }
  })
  stack_id   = kaleido_platform_stack.custody_stack.id
  depends_on = [kaleido_platform_service.workflow_engine_service]
}

resource "kaleido_platform_runtime" "policy_manager_runtime" {
  type = "PolicyManager"
  name = "policies"
  environment = kaleido_platform_environment.env_0.id
  config_json = jsonencode({})
}

resource "kaleido_platform_service" "policy_manager_service" {
  type        = "PolicyManager"
  name        = "policies"
  environment = kaleido_platform_environment.env_0.id
  runtime     = kaleido_platform_runtime.policy_manager_runtime.id
  config_json = jsonencode({})
}