## Summary

Create an environment with initial configuration necessary to optimize for performance and high throughput

## Inputs

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
| protocon_config | Custom protocol configuration to apply | json | ```json {"restgw_max_inflight": 1000, "restgw_max_tx_wait_time": 60,"restgw_always_manage_nonce": true,"restgw_send_concurrency": 100,"restgw_attempt_gap_fill": true, "restgw_flush_frequency": 0,"restgw_flush_msgs": 0,"restgw_flush_bytes": 0,}``` | no |

Once Terraform has been run with the configuration above, users will need to complete a few steps to finalize setup

1. Vote system monitor out of the validating set through the Kaleido console
2. Reach out to Kaleido support to resize your system monitor node to a `large`
3. Deploy a smart contract and write client code to submit transactions at the desired throughput