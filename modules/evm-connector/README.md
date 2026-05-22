# evm-connector

Deploys an EVM connector service (`EVMConnectorStack` + runtime + service) and the
full set of config types, config profiles, connector flows, stream factories, and
the standard API/stream it ships with. The resource graph is hardcoded; per-ecosystem
behaviour is controlled by overriding the typed config-profile variables.

## Usage

```hcl
module "evm" {
  source = "https://github.com/kaleido-io/terraform-provider-kaleido/modules/evm-connector?ref=v1.3.0"

  environment_id         = kaleido_platform_environment.env.id
  key_manager_service_id = kaleido_platform_service.keymanager.id

  ecosystem = { name = "besu", displayName = "Besu" }
  network   = { name = "besu-private", chainId = "3333" }

  confirmations = { count = 0 }
}
```

Or with one of the sample `*.tfvars` files:

```
terraform apply -var-file=modules/evm-connector/examples/ethereum-mainnet.tfvars
```

## Ecosystem presets

Drop-in `*.tfvars` files under `examples/`:

| File | Notes |
|------|-------|
| `besu.tfvars` | Private Besu, count=0, fixed zero-fee gas |
| `ethereum-mainnet.tfvars` | 12 confirmations, resubmission on |
| `ethereum-sepolia.tfvars` | 6 confirmations, resubmission on |
| `base-mainnet.tfvars` | 20 confirmations |
| `base-sepolia.tfvars` | 6 confirmations |
| `polygon-mainnet.tfvars` | 50 confirmations (reorg risk) |
| `polygon-amoy.tfvars` | 50 confirmations (reorg risk) |
| `arbitrum-sepolia.tfvars` | 6 confirmations |

Adding a new ecosystem is a `*.tfvars` change — see the
[`tf-sync-connector-module` skill](../../.claude/tf-sync-connector-module/SKILL.md)
for the methodology.

## Outputs

- `service_id`, `stack_id`, `runtime_id`
- `submission_flow_name`, `query_flow_name`, `standard_api_name`
- `stream_factories` (map: `block_events`, `transaction_events`)
- `config_profiles` (map: config type → profile ID)
