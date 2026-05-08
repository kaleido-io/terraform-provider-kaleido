## Summary

Provisions a single Kaleido Platform environment with all the infrastructure needed for custody and tokenization workflows. Deploys stacks and services via Terraform, leaving connector setup, policy creation, and asset/wallet management to be completed interactively through the UI.

### Stacks
| Stack | Type | Services | Purpose |
|-------|------|----------|---------|
| **testnet** | `chain_infrastructure` / `BesuStack` | BesuNode, EVMGateway, BlockIndexer | Single-node QBFT Besu chain with zero gas fees |
| **evm** | `web3_middleware` / `EVMConnectorStack` | EVMConnector | Transaction submission, event indexing, and contract interaction via WFE |
| **tokenization** | `digital_assets` / `TokenizationStack` | AssetManager | Token lifecycle management (define, deploy, mint, transfer) |
| **custody** | `digital_assets` / `CustodyStack` | WalletManager, PolicyManager | Wallet orchestration with policy-gated signing |
Shared services (KeyManager, ContractManager, WorkflowEngine) are created outside of stacks.

### Post-Apply (via UI)
- **Connector Setup** -- configure confirmation thresholds, gas pricing, and event polling on the EVMConnector
- **Policies** -- create approval policies on the PolicyManager
- **Assets & Wallets** -- register tokens, create wallets, and link signing keys

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|:----:|:-------:|:--------:|
| kaleido_platform_api | Kaleido API URL | string | - | yes |
| kaleido_platform_username | API Key name | string | `` | yes |
| kaleido_platform_password | API Key value | string | `` | yes |
| environment_name | Environment name | string | `` | yes |
| databases | Database names per service | object | `null` | no |

### `databases`

Required when `database-management-disabled` is set (Kaleido Software with externally managed databases). When `null`, the platform auto-provisions databases.

| Field | Service |
|-------|---------|
| `kms_db` | KeyManager |
| `cms_db` | ContractManager |
| `wfe_db` | WorkflowEngine |
| `bis_db` | BlockIndexer |
| `ecs_db` | EVMConnector |
| `ams_db` | AssetManager |
| `wms_db` | WalletManager |
| `pms_db` | PolicyManager |
