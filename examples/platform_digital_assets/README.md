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
| besu_node_count | Number of nodes to create | number | 1 | no |

