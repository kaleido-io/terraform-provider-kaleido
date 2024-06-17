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
    - name: naming
      options:
        template: >-
          {
              "pool": "pool",
              "poolQualified": input.blockchainEvent.info.address & "/pool",
              "activity": "pool-" & input.blockchainEvent.info.address,
              "protocolIdSafe": $replace(input.blockchainEvent.protocolId, "/", "_")
          }
      stopCondition: >-
        $not(
            input.blockchainEvent.info.signature = "Transfer(address,address,uint256)" and
            $exists(input.blockchainEvent.output.value)
        )
      type: jsonata_template
    - dynamicOptions:
        activities: |-
          [{
              "updateType": "create_or_ignore",
              "name": steps.naming.data.activity
          }]
        addresses: |-
          [
            {
              "updateType": "create_or_update",
              "address": input.blockchainEvent.info.address,
              "displayName": input.blockchainEvent.info.address,
              "contract": true 
            },
            {
              "updateType": "create_or_update",
              "address": input.blockchainEvent.output.from,
              "displayName": input.blockchainEvent.output.from
            },
            {
              "updateType": "create_or_update",
              "address": input.blockchainEvent.output.to,
              "displayName": input.blockchainEvent.output.to
            }
          ]
        events: |-
          [{
              "updateType": "create_or_replace",
              "name": "transfer-" & steps.naming.data.protocolIdSafe,
              "activity": steps.naming.data.activity,
              "parent": {
                  "type": "pool",
                  "ref": steps.naming.data.poolQualified
              },
              "info": {
                "address": input.blockchainEvent.info.address,
                "blockNumber": input.blockchainEvent.info.blockNumber,
                "protocolId": input.blockchainEvent.protocolId,
                "transactionHash": input.blockchainEvent.info.transactionHash
              }
          }]
        pools: |-
          [{
              "updateType": "create_or_ignore",
              "address": input.blockchainEvent.info.address,
              "name": steps.naming.data.pool,
              "standard": "ERC-20",
              "firefly": {
                "namespace": input.blockchainEvent.namespace
              }
          }]
        transfers: |-
          [{
              "updateType": "create_or_replace",
              "protocolId": input.blockchainEvent.protocolId,
              "from": input.blockchainEvent.output.from,
              "to": input.blockchainEvent.output.to,
              "amount": input.blockchainEvent.output.value,
              "transactionHash": input.blockchainEvent.info.transactionHash,
              "parent": {
                  "type": "pool",
                  "ref": steps.naming.data.poolQualified
              },
              "firefly": {
                "namespace": input.blockchainEvent.namespace
              },
              "info": {
                  "blockNumber": input.blockchainEvent.info.blockNumber
              }
          }]
      name: transfer_upsert
      type: data_model_update
    - dynamicOptions:
        assets: |-
          [{
              "updateType": "create_or_ignore",
              "name": "pool_" & input.blockchainEvent.info.address
          }]
        pools: |-
          [{
              "updateType": "update_only",
              "address": input.blockchainEvent.info.address,
              "name": steps.naming.data.pool,
              "asset": "pool_" & input.blockchainEvent.info.address
          }]
      name: link_asset
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
    - name: naming
      options:
        template: >-
          (
            $tokenId := $exists(input.blockchainEvent.output._tokenId) ?
                $string(input.blockchainEvent.output._tokenId) :
                $string(input.blockchainEvent.output.tokenId);

            {
              "tokenId": $tokenId,
              "nft": $tokenId,
              "nftQualified": input.blockchainEvent.info.address & "/" & $tokenId,
              "activity": "nft-" & input.blockchainEvent.info.address & "-" & $tokenId,
              "protocolIdSafe": $replace(input.blockchainEvent.protocolId, "/", "_")
            }
          )
      stopCondition: >-
        $not(
          (
            input.blockchainEvent.info.signature = "Transfer(address,address,uint256)" and
            $exists(input.blockchainEvent.output.tokenId)
          ) or
          (
            input.blockchainEvent.info.signature = "Approval(address,address,uint256)" and
            $exists(input.blockchainEvent.output.tokenId)
          ) or
          (
            input.blockchainEvent.info.signature = "MetadataUpdate(uint256)" and
            $exists(input.blockchainEvent.output._tokenId)
          )
        )
      type: jsonata_template
    - name: common_upsert
      options:
        template: |-
          {
            "addresses": [{
              "updateType": "create_or_ignore",
              "address": input.blockchainEvent.info.address,
              "contract": true
            }],
            "activities": [{
              "updateType": "create_or_ignore",
              "name": steps.naming.data.activity
            }],
            "nfts": [{
              "updateType": "create_or_ignore",
              "name": steps.naming.data.nft,
              "standard": "ERC-721",
              "tokenIndex": steps.naming.data.tokenId,
              "address": input.blockchainEvent.info.address,
              "firefly": {
                "namespace": input.blockchainEvent.namespace
              }
            }]
          }
      type: jsonata_template
    - dynamicOptions:
        activities: steps.common_upsert.data.activities
        addresses: steps.common_upsert.data.addresses
        events: |-
          [{
              "updateType": "create_or_replace",
              "name": "approval-" & steps.naming.data.protocolIdSafe,
              "activity": steps.naming.data.activity,
              "parent": {
                "type": "nft",
                "ref": steps.naming.data.nftQualified
              },
              "info": {
                "address": input.blockchainEvent.info.address,
                "blockNumber": input.blockchainEvent.info.blockNumber,
                "protocolId": input.blockchainEvent.protocolId,
                "transactionHash": input.blockchainEvent.info.transactionHash,
                "owner": input.blockchainEvent.output.owner,
                "approved": input.blockchainEvent.output.approved,
                "tokenIndex": steps.naming.data.tokenId
              }
          }]
        nfts: steps.common_upsert.data.nfts
      name: approval_upsert
      skip: input.blockchainEvent.name != "Approval"
      type: data_model_update
    - dynamicOptions:
        activities: steps.common_upsert.data.activities
        addresses: steps.common_upsert.data.addresses
        events: |-
          [{
              "updateType": "create_or_replace",
              "name": "metadata-" & steps.naming.data.protocolIdSafe,
              "activity": steps.naming.data.activity,
              "parent": {
                "type": "nft",
                "ref": steps.naming.data.nftQualified
              },
              "info": {
                "address": input.blockchainEvent.info.address,
                "blockNumber": input.blockchainEvent.info.blockNumber,
                "protocolId": input.blockchainEvent.protocolId,
                "transactionHash": input.blockchainEvent.info.transactionHash,
                "tokenIndex": steps.naming.data.tokenId
              }
          }]
        nfts: steps.common_upsert.data.nfts
      name: metadata_upsert
      skip: input.blockchainEvent.name != "MetadataUpdate"
      type: data_model_update
    - dynamicOptions:
        activity: steps.naming.data.activity
        idempotencyKey: input.id
        input: |-
          {
            "nft": steps.common_upsert.data.nfts[0],
            "blockNumber": input.blockchainEvent.info.blockNumber
          }
      name: query_uri
      options:
        taskName: indexer_erc721_query_uri
      skip: input.blockchainEvent.name != "MetadataUpdate"
      type: invoke_task
    - dynamicOptions:
        activities: steps.common_upsert.data.activities
        addresses: |-
          $append(
            steps.common_upsert.data.addresses,
            [
              {
                "updateType": "create_or_ignore",
                "address": input.blockchainEvent.output.from
              },
              {
                "updateType": "create_or_ignore",
                "address": input.blockchainEvent.output.to
              }
            ]
          )
        events: |-
          [{
              "updateType": "create_or_replace",
              "name": "transfer-" & steps.naming.data.protocolIdSafe,
              "activity": steps.naming.data.activity,
              "parent": {
                "type": "nft",
                "ref": steps.naming.data.nftQualified
              },
              "info": {
                "address": input.blockchainEvent.info.address,
                "blockNumber": input.blockchainEvent.info.blockNumber,
                "protocolId": input.blockchainEvent.protocolId,
                "transactionHash": input.blockchainEvent.info.transactionHash,
                "tokenIndex": steps.naming.data.tokenId
              }
          }]
        nfts: steps.common_upsert.data.nfts
        transfers: |-
          [{
              "updateType": "create_or_replace",
              "protocolId": input.blockchainEvent.protocolId,
              "from": input.blockchainEvent.output.from,
              "to": input.blockchainEvent.output.to,
              "amount": "1",
              "transactionHash": input.blockchainEvent.info.transactionHash,
              "parent": {
                "type": "nft",
                "ref": steps.naming.data.nftQualified
              },
              "firefly": {
                "namespace": input.blockchainEvent.namespace
              },
              "info": {
                "blockNumber": input.blockchainEvent.info.blockNumber
              }
          }]
      name: transfer_upsert
      skip: input.blockchainEvent.name != "Transfer"
      type: data_model_update
    - name: asset_naming
      options:
        template: >-
          {
              "collection": input.blockchainEvent.info.address,
              "asset": "nft_" & input.blockchainEvent.info.address & "_" & steps.naming.data.tokenId
          }
      type: jsonata_template
    - dynamicOptions:
        assets: |-
          [{
              "updateType": "create_or_ignore",
              "name": steps.asset_naming.data.asset,
              "collection": steps.asset_naming.data.collection
          }]
        collections: |-
          [{
              "updateType": "create_or_ignore",
              "name": steps.asset_naming.data.collection
          }]
        nfts: |-
          [{
              "updateType": "update_only",
              "name": steps.naming.data.nft,
              "address": input.blockchainEvent.info.address,
              "asset": steps.asset_naming.data.asset
          }]
      name: link_asset
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