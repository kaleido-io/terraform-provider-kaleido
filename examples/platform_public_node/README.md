## Summary

Set up a Kaleido Environment connecting to a Public Chain.

Create an environment with:

* Web3 Middleware Stack
    - Transaction Manager connected to an external RPC Endpoint with authentication
    - Hyperledger FireFly Service
* Contract Manager
* Key Manager

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|:----:|:-----:|:-----:|
| kaleido_platform_api | Kaleido API URL | string | - | yes |
| kaleido_platform_username | API Key name | string | `` | yes |
| kaleido_platform_password | API Key value | string | `` | yes |
| environment_name | Environment name | string | `` | yes |
| rpc_url | RPC endpoint for the public chain | string | `` | yes |
| username | Username for public chain access | string | `` | yes |
| password | Password for public chain access | string | `` | yes |

## Output
| Name | Description | Type |
|------|-------------|:----:|
| key_address | Key address  | string |