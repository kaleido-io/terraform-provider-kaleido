## Summary

Create an environment with:

* Nodes and IPFS and Data Exchange Services

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|:----:|:-----:|:-----:|
| kaleido_api_key | Kaleido API Key | string | - | yes |
| kaleido_region | Can be '-ap' for Sydney, or '-eu' for Frankfurt. Defaults to US | string | `` | no |
| provider | Blockchain Provider for the environment. | string | `pantheon` | no |
| consensus | Consensus methods supported by pantheon. | list | `ibft` | no |
| multi_region | Multi region env | string | `false` | no |
| node_size | Size of the nodes to be created | string | `small` | no |
| node_count | Number of nodes to create | string | 4 | no |
| service_count | Number of services to create | string | 1 | no |
