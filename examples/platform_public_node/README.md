## Summary

Set up a Kaleido Environment that can connect to a Public Node RPC endpoint and submit transactions. 

Create an environment with:

* Web3 Middleware Stack
    - Transaction Manager connected to an external RPC Endpoint with authentication
* Contract Manager
* Key Manager

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|:----:|:-----:|:-----:|
| kaleido_platform_api | Kaleido API URL | string | - | yes |
| kaleido_platform_username | API Key name | string | `` | yes |
| kaleido_platform_password | API Key value | string | `` | yes |
| environment_name | Environment name | string | `` | yes |
| rpc_url | RPC URl of the Public Node | string | `` | yes |
| username | Username for Public Node access | string | `` | yes |
| password | Password for Public Node access | string | `` | yes |

## Output
| Name | Description | Type |
|------|-------------|:----:|
| key_address | Key address created by the module  | string |