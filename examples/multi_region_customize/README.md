## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|:----:|:-----:|:-----:|
| env_types | List of environment types you want to deploy. Options are 'quorum' and 'geth'. | list | `<list>` | no |
| kaleido_api_key | Kaleido API Key | string | - | yes |
| kaleido_api_url | API RUL https://docs.kaleido.io/developers/automation/regions/ for regional URLs (defines metadata location) | string | `` | no |
| cloud | `aws` or `azure` | string | `` | no |
| region | `us-east-2`, `westus2`, `ap-southeast-2` etc. | string | `` | no |
| quorum_consensus | Consensus methods supported by quorum. | list | `<list>` | no |

