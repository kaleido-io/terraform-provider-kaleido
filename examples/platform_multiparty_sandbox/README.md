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
* 3 Web3 Middleware Stacks (1 per member)
    - 1 Firefly Service
    - 1 Private Data Manager 
    - 1 Transaction Manager
* 3 Digital Assets Stacks
    - Digital Asset Service
* Environment Tools
    - Key Manager 
        - 1 per member
    - Contract Manager

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|:----:|:-----:|:-----:|
| kaleido_platform_api | Kaleido API URL | string | - | yes |
| kaleido_platform_username | API Key name | string | `` | yes |
| kaleido_platform_password | API Key value | string | `` | yes |
| environment_name | Environment name | string | `` | yes |
| besu_node_count | Number of nodes to create | number | 1 | no |
| members | Number of members in the multiparty network to create | list | `<list>` | yes |
| runtime_size| Size to set for every runtime | string | `Small` | no |


