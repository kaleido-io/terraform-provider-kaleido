terraform {
  required_providers {
    kaleido = {
      source = "kaleido-io/kaleido"
      version = "1.1.0-rc.3"
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
  name = "evmchain1_node${count.index+1}"
  environment = kaleido_platform_environment.env_0.id
  config_json = jsonencode({})
  count = var.node_count
}

resource "kaleido_platform_service" "bns" {
  type = "BesuNode"
  name = "evmchain1_node${count.index+1}"
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
  name = "evmchain1_gateway"
  environment = kaleido_platform_environment.env_0.id
  config_json = jsonencode({})
}

resource "kaleido_platform_service" "gws_0" {
  type = "EVMGateway"
  name = "evmchain1_gateway"
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
}

resource "kaleido_platform_service" "tms_0" {
  type = "TransactionManager"
  name = "evmchain1_txmgr"
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
  name = "firefly1"
  environment = kaleido_platform_environment.env_0.id
  config_json = jsonencode({})
}

resource "kaleido_platform_service" "ffs_0" {
  type = "FireFly"
  name = "firefly1"
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

resource "kaleido_platform_runtime" "amr_0" {
  type = "AssetManager"
  name = "asset_manager1"
  environment = kaleido_platform_environment.env_0.id
  config_json = jsonencode({})
}

resource "kaleido_platform_service" "ams_0" {
  type = "AssetManager"
  name = "asset_manager1"
  environment = kaleido_platform_environment.env_0.id
  runtime = kaleido_platform_runtime.amr_0.id
  config_json = jsonencode({
    keyManager = {
      id: kaleido_platform_service.kms_0.id
    }
  })
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

resource "kaleido_platform_cms_build" "erc721" {
  environment = kaleido_platform_environment.env_0.id
  service = kaleido_platform_service.cms_0.id
  type = "github"
  name = "ERC721WithData"
  path = "erc721_samples"
	github = {
		contract_url = "https://github.com/hyperledger/firefly-tokens-erc20-erc721/blob/main/samples/solidity/contracts/ERC721WithData.sol"
		contract_name = "ERC721WithData"
	}
  depends_on = [ kaleido_platform_cms_build.erc20 ] // don't compile in parallel (excessive disk usage for npm)
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

resource "kaleido_platform_cms_action_deploy" "demotoken_erc721" {
  environment = kaleido_platform_environment.env_0.id
  service = kaleido_platform_service.cms_0.id
  build = kaleido_platform_cms_build.erc721.id
  name = "deploy_erc721"
  firefly_namespace = kaleido_platform_service.ffs_0.name
  signing_key = kaleido_platform_kms_key.key_0.address
  params_json = jsonencode([
    "DemoNFT",
    "DNFT",
    "demo://token/"
  ])
  depends_on = [ data.kaleido_platform_evm_netinfo.gws_0 ]
}

resource "kaleido_platform_cms_action_creatapi" "erc20" {
  environment = kaleido_platform_environment.env_0.id
  service = kaleido_platform_service.cms_0.id
  build = kaleido_platform_cms_build.erc20.id
  name = "erc20withdata"
  firefly_namespace = kaleido_platform_service.ffs_0.name
  api_name = "erc20withdata"
  depends_on = [ data.kaleido_platform_evm_netinfo.gws_0 ]
}

resource "kaleido_platform_cms_action_creatapi" "erc721" {
  environment = kaleido_platform_environment.env_0.id
  service = kaleido_platform_service.cms_0.id
  build = kaleido_platform_cms_build.erc721.id
  name = "erc721withdata"
  firefly_namespace = kaleido_platform_service.ffs_0.name
  api_name = "erc721withdata"
  depends_on = [ data.kaleido_platform_evm_netinfo.gws_0 ]
}

resource "kaleido_platform_ams_task" "erc20_indexer" {
  environment = kaleido_platform_environment.env_0.id
  service = kaleido_platform_service.ams_0.id
  depends_on = [ kaleido_platform_ams_task.erc20_indexer ]
  name = "erc20_indexer"
  task_yaml = <<EOT
    steps:
    - dynamicOptions:
        addresses: |-
          [{
              "updateType": "create_or_ignore",
              "address": input.blockchainEvent.info.address
          }]
        assets: |-
          [{
              "updateType": "create_or_ignore",
              "name": "erc20_" & input.blockchainEvent.info.address
          }]
        pools: |-
          [{
              "updateType": "create_or_ignore",
              "name": "erc20",
              "asset": "erc20_" & input.blockchainEvent.info.address,
              "address": input.blockchainEvent.info.address
          }]
        transfers: |-
          [{
              "protocolId": input.blockchainEvent.protocolId,
              "asset": "erc20_" & input.blockchainEvent.info.address,
              "from": input.blockchainEvent.output.from,
              "to": input.blockchainEvent.output.to,
              "amount": input.blockchainEvent.output.value,
              "transactionHash": input.blockchainEvent.info.transactionHash,
              "parent": {
                  "type": "pool",
                  "ref": input.blockchainEvent.info.address & "/erc20"
              }
          }]
      name: upsert_transfer
      options: {}
      skip: input.blockchainEvent.info.signature !=
        "Transfer(address,address,uint256)" or
        $boolean([input.blockchainEvent.output.value]) = false
      type: data_model_update
  EOT
}

resource "kaleido_platform_ams_fflistener" "erc20_indexer" {
  environment = kaleido_platform_environment.env_0.id
  service = kaleido_platform_service.ams_0.id
  name = "erc20_indexer"
  depends_on = [ kaleido_platform_ams_task.erc20_indexer ]
  config_json = jsonencode({
		namespace = kaleido_platform_service.ffs_0.name,
		taskName = "erc20_indexer",
		blockchainEvents = {
      createOptions = {
  		  firstEvent = "0"
      },
			abiEvents = [
				{
          "anonymous": false,
          "inputs": [
              {
                  "indexed": true,
                  "name": "from",
                  "type": "address"
              },
              {
                  "indexed": true,
                  "name": "to",
                  "type": "address"
              },
              {
                  "indexed": false,
                  "name": "value",
                  "type": "uint256"
              }
          ],
          "name": "Transfer",
          "type": "event"
        }
			]
		}
    })
}

resource "kaleido_platform_ams_task" "erc721_indexer" {
  environment = kaleido_platform_environment.env_0.id
  service = kaleido_platform_service.ams_0.id
  name = "erc721_indexer"
  task_yaml = <<EOT
    steps:
    - dynamicOptions:
        body: |-
          {
              "input": {
                  "tokenId": $string(input.blockchainEvent.output.tokenId)
              },
              "key": "${kaleido_platform_kms_key.key_0.address}",
              "location": {
                  "address": input.blockchainEvent.info.address
              }
          }
      name: query_uri
      options:
        bodyJSONEncoding: json
        method: POST
        namespace: ${kaleido_platform_service.ffs_0.name}
        path: apis/erc721withdata/query/tokenURI
      skip: |-
        input.blockchainEvent.info.signature !=
              "Transfer(address,address,uint256)" or
              $boolean([input.blockchainEvent.output.tokenId]) = false
      type: firefly_request
    - dynamicOptions:
        addresses: |-
          [{
              "updateType": "create_or_ignore",
              "address": input.blockchainEvent.info.address
          }]
        collections: >-
          [{
              "updateType": "create_or_ignore",
              "name": "erc721_" & input.blockchainEvent.info.address
          }]
        assets: >-
          [{
              "updateType": "create_or_update",
              "collection": "erc721_" & input.blockchainEvent.info.address,
              "name": "erc721_" & input.blockchainEvent.info.address & "_" & input.blockchainEvent.output.tokenId
          }]
        nfts: >-
          [{
              "updateType": "create_or_ignore",
              "asset": "erc721_" & input.blockchainEvent.info.address & "_" & input.blockchainEvent.output.tokenId,
              "name": input.blockchainEvent.output.tokenId,
              "standard": "ERC-721",
              "address": input.blockchainEvent.info.address,
              "tokenIndex": input.blockchainEvent.output.tokenId,
              "uri": steps.query_uri.data.output
          }]
        transfers: >-
          [{
              "protocolId": input.blockchainEvent.protocolId,
              "asset": "erc721_" & input.blockchainEvent.info.address & "_" & input.blockchainEvent.output.tokenId,
              "from": input.blockchainEvent.output.from,
              "to": input.blockchainEvent.output.to,
              "amount": "1",
              "transactionHash": input.blockchainEvent.info.transactionHash,
              "parent": {
                  "type": "nft",
                  "ref": input.blockchainEvent.info.address & "/" & input.blockchainEvent.output.tokenId
              }
          }]        
      name: upsert_transfer
      options: {}
      skip: |-
        input.blockchainEvent.info.signature !=
              "Transfer(address,address,uint256)" or
              $boolean([input.blockchainEvent.output.tokenId]) = false
      type: data_model_update
  EOT
}

resource "kaleido_platform_ams_fflistener" "erc721_indexer" {
  environment = kaleido_platform_environment.env_0.id
  service = kaleido_platform_service.ams_0.id
  name = "erc721_indexer"
  depends_on = [ kaleido_platform_ams_task.erc721_indexer ]
  config_json = jsonencode({
		namespace = kaleido_platform_service.ffs_0.name,
		taskName = "erc721_indexer",
		blockchainEvents = {
      createOptions = {
  		  firstEvent = "0"
      },
			abiEvents = [
				{
          "anonymous": false,
          "inputs": [
              {
                  "indexed": true,
                  "internalType": "address",
                  "name": "from",
                  "type": "address"
              },
              {
                  "indexed": true,
                  "internalType": "address",
                  "name": "to",
                  "type": "address"
              },
              {
                  "indexed": true,
                  "internalType": "uint256",
                  "name": "tokenId",
                  "type": "uint256"
              }
          ],
          "name": "Transfer",
          "type": "event"
        }
			]
		}
    })
}