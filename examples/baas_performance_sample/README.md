## Summary

This Terraform assists Kaleido platform users interested in testing out high throughput scenarios on the Kaleido Blockchain as a Service platform. This script will create 4 large nodes and apply a custom protocol configuration to each node.  

## Inputs

Refer to the file `input.tfvars` for an example input. Rename this file to `terraform.tfvars` if you'd like for it to apply to your environment. Default settings defined in this file can be found below

| Name | Description | Type | Default | Required |
|------|-------------|:----:|:-----:|:-----:|
| kaleido_api_key | Kaleido API Key | string | - | yes |
| kaleido_region | Can be '-ap' for Sydney, or '-eu' for Frankfurt. Defaults to US | string | `` | no |
| provider | Blockchain Provider for the environment. | string | `pantheon (besu)` | no |
| consensus | Consensus methods supported by quorum. | list | `ibft` | no |
| multi_region | Multi region env | string | `false` | no |
| node_size | Size of the nodes to be created | string | `large` | no |
| node_count | Number of nodes to create | string | 4 | no |
| service_count | Number of services to create | string | 1 | no |
| block_period | Block period in seconds | number | 5 | no |
| protocon_config | Custom protocol configuration to apply | json | ```javascript {"restgw_max_inflight": 1000, "restgw_max_tx_wait_time": 60,"restgw_always_manage_nonce": true,"restgw_send_concurrency": 100,"restgw_attempt_gap_fill": true, "restgw_flush_frequency": 0,"restgw_flush_msgs": 0,"restgw_flush_bytes": 0,}``` | no |

Once the Terraform script has run successfully, follow these steps to complete setup of the environment. 

1. Vote system monitor out of the validating set through the Kaleido console
2. Reach out to Kaleido support to resize your system monitor node to a `large` by opening a ticket through the Kaleido platform. Be sure to designate what environment this shouhld be done in
3. Deploy a smart contract, such as SimpleStorage found at this [link](https://github.com/kaleido-io/kaleido-js/blob/master/deploy-transact/contracts/simplestorage_v5.sol)
4. ~Optional~ Use the sample_client_load_test.js as a starting point to test performance of the blockchain network. This sample client code is built to work with simple storage, but could be modified for your own smart contract's generated API on the Kaleido platform. Additional modifications required for custom application development