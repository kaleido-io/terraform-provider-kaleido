# btc-connector

Deploys a Bitcoin connector service (`BTCConnectorStack` + runtime + service) and the
four config types, three config profiles, single submission flow, transaction-events
stream factory, and Bitcoin standard API it ships with.

## Usage

```hcl
module "btc" {
  source = "../../modules/btc-connector"

  environment_id         = kaleido_platform_environment.env.id
  key_manager_service_id = kaleido_platform_service.keymanager.id

  network = { name = "testnet4", displayName = "Bitcoin Testnet 4" }
}
```

Or with a sample `*.tfvars` file:

```
terraform apply -var-file=modules/btc-connector/examples/bitcoin-mainnet.tfvars
```

## Ecosystem presets

| File | Notes |
|------|-------|
| `bitcoin-mainnet.tfvars` | 6 confirmations, RPC-based fee estimation, 100 sat/vB cap |
| `bitcoin-testnet4.tfvars` | 2 confirmations, RPC fee estimation |
| `bitcoin-signet.tfvars` | 1 confirmation, fixed 1 sat/vB fee |

Adding a new network is a `*.tfvars` change — see the
[`tf-sync-connector-module` skill](../../.claude/tf-sync-connector-module/SKILL.md).

## Outputs

- `service_id`, `stack_id`, `runtime_id`
- `submission_flow_name`, `standard_api_name`
- `stream_factories` (map: `transaction_events`)
- `config_profiles` (map: config type → profile ID)
