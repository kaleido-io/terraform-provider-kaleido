## Summary

Create an environment with:

* Chain Infrastructure Stack 
    - Besu Network 
    - EVM Gateway
* Web3 Middleware Stack
* Digital Assets Stack
* Contract Manager
* Key Manager

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|:----:|:-----:|:-----:|
| kaleido_platform_api | Kaleido API URL | string | - | yes |
| kaleido_platform_username | API Key name | string | `` | yes |
| kaleido_platform_password | API Key value | string | `` | yes |
| environment_name | Environment name | string | `` | yes |
| multi_region | Multi region env | string | `false` | no |
| besu_node_count | Number of nodes to create | string | 1 | no |

