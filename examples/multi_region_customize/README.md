## Inputs

Demonstrates how you deploy resources to a different deployment region to the one
implicit in the API URL used to create the consortium.

Provides a starting point for creating multi-region blockchain environments on Kaleido.

> Note the API URL determines where the metadata for your business network is located, so should be chosen with care.

| Name | Description | Type | Default | Required |
|------|-------------|:----:|:-----:|:-----:|
| env_types | List of environment types you want to deploy. Options are 'quorum' and 'geth'. | list | `<list>` | no |
| kaleido_api_key | Kaleido API Key | string | - | yes |
| kaleido_api_url | API URL https://docs.kaleido.io/developers/automation/regions/ for regional URLs (defines metadata location) | string | `` | no |
| cloud | `aws` or `azure` | string | `` | no |
| region | `us-east-2`, `westus2`, `ap-southeast-2` etc. | string | `` | no |
| quorum_consensus | Consensus methods supported by quorum. | list | `<list>` | no |
