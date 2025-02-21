## Summary

Create an environment with:

* Chain Infrastructure Stack - Besu
    - Besu Network 
    - Besu Nodes
    - EVM Gateway
    - Block Indexer
* Chain Infrastructure Stack - IPFS
    - IPFS Network 
    - IPFS Node
* Web3 Middleware Stack
    - 1 Firefly Service per member 
    - Private Data Manager
    - Transaction Manager
* Digital Assets Stack
    - Digital Asset Service per member 
* Environment Tools
    - Key Manager 
        - 1 Wallet per member
    - Contract Manager 

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|:----:|:-----:|:-----:|
| kaleido_platform_api | Kaleido API URL | string | - | yes |
| kaleido_platform_username | API Key name | string | `` | yes |
| kaleido_platform_password | API Key value | string | `` | yes |
| environment_name | Environment name | string | `` | yes |
| multi_region | Multi region env | string | `false` | no |
| besu_node_count | Number of nodes to create | string | 1 | no |
| members | Number of members in the multiparty network to create | list | `<list>` | yes |


