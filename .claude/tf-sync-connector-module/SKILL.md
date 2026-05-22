---
name: tf-sync-connector-module
description: Sync the `modules/evm-connector` and `modules/btc-connector` Terraform modules with the upstream connector definitions source when new config types, flows, stream factories, standard APIs, or standard streams are added (or existing ones change schema or version).
---

# tf-sync-connector-module

Use this skill when:

- The user says "sync the connector modules" or "update evm-connector / btc-connector"
- A new config type, connector flow, stream factory, standard API, or standard stream
  needs to be reflected in the Terraform modules
- A `config_type` schema or `version:` changed upstream and the module variable type or
  default needs to follow

This skill **does not** know where the upstream connector definitions live.
**Always ask the user** for the local filesystem path on every invocation. The path
must not be persisted, hard-coded, or referenced in any file this skill writes —
the terraform-provider-kaleido repository is open-source and must not name or describe
the upstream source repository.

## Step 1 — Get the upstream path from the user

Ask:

> "What's the local filesystem path to the upstream connector definitions source for
> this run? I expect a directory containing subdirectories `evm/` and `btc/`, each with
> `config_types/`, `connector_flows/`, `connector_stream_factories/`, `standard_apis/`,
> `standard_streams/`, `ecosystems/`, and `protocol.yaml`."

Do not proceed without a path. Do not guess. Do not reuse a path from a previous
conversation. The agent that invokes this skill in another repo or another checkout
will likely have a different path.

## Step 2 — Compare upstream against the modules

For each protocol (`evm`, `btc`):

1. Read `<upstream>/<proto>/protocol.yaml` to confirm the `initialSetup:` list. This is
   the authoritative set of `connectorFlows`, `connectorStreamFactories`, `standardAPIs`,
   and `standardStreams` the module must declare.
2. List `<upstream>/<proto>/config_types/*.yaml`. The `name:` and `version:` of each
   become the keys of `local.config_types` and the type-bindings inside flow resources.
3. For each config type, the `schema:` block in the YAML is the JSON Schema the module's
   typed variable must mirror.

Then look at `modules/<proto>-connector/`:

- `main.tf` — must declare `kaleido_platform_connector_config_type.this` for each
  upstream config type, `kaleido_platform_connector_config_profile.this` for each,
  `kaleido_platform_connector_flow` per upstream flow, `kaleido_platform_connector_stream_factory`
  per upstream factory, `kaleido_platform_connector_standard_api` per upstream API,
  `kaleido_platform_connector_standard_stream` per upstream stream.
- `variables.tf` — must have one variable per config type, with `type = object({...})`
  matching the upstream `schema:` and `default =` reflecting any `default:` keys.
- `examples/*.tfvars` — must reflect the upstream `ecosystems/*.yaml` config profile
  overrides for each ecosystem & network.

## Step 3 — Derive Terraform variable types from JSON Schema

Translation rules (apply per `schema.properties.<field>`):

| Upstream YAML | Terraform variable type |
|---------------|--------------------------|
| `type: string` (no `enum`, no `format: duration`) | `optional(string, <default>)` |
| `type: string, format: duration` | `optional(string, "<default>")` — durations are strings |
| `type: string, enum: [...]` | `optional(string, "<default>")` — enums are still strings; document allowed values |
| `type: boolean` | `optional(bool, <default>)` |
| `type: integer` or `type: number` | `optional(number, <default>)` |
| `type: object` with concrete `properties` | `optional(object({ ... nested ... }))` |
| `type: object` with `additionalProperties: true` and no concrete properties | `optional(map(any))` |
| `type: array, items: <prim>` | `optional(list(<prim>))` |
| `type: array, items: <object>` | `optional(list(object({...})))` |
| Tagged union (multiple sibling sub-objects, only one set at a time — e.g. `gasPricing.source` has `fixedGasPrice`/`gasOracleAPI`/`rpcEndpoint`) | Fall back to `type = any` — Terraform's type system can't express tagged unions cleanly. Document the shape in the variable description. |

Always wrap top-level fields in `optional(...)` and provide a default empty object
`default = {}` on the variable itself, so users only override what they need.

The variable's `description` must mention the upstream config type name (e.g.
`"evm.confirmations — ..."`) so users can search for the type in any documentation.

## Step 4 — Derive ecosystem `*.tfvars` from `ecosystems/`

For each `<upstream>/<proto>/ecosystems/*.yaml`:

- `nativeCurrency`, `identifiers`, `catalogs` — informational only, do **not** turn into
  tfvars.
- `configProfiles:` block at the ecosystem level — these become tfvars values at the
  root of the ecosystem's `mainnet`/`default` tfvars file.
- Each entry under `networks:` may carry a per-network `configProfiles:` override —
  each network with overrides gets its own tfvars file (e.g. `ethereum-mainnet.tfvars`
  vs `ethereum-sepolia.tfvars`).
- The `ecosystem` and `network` block in the tfvars uses the `name` (and optionally
  `displayName`) from the upstream entry. `chainId` comes from `networks[].chainId`
  for EVM.

## Step 5 — Bindings (the part most likely to drift)

The module's flow / standard API resources bind to config profiles and other flows.
Re-derive these from upstream every sync:

- **Connector flow `config_type_bindings`:** look at the flow's `operations[].configProfile.type`
  and `events[].configProfile.type` and `stages[].configProfile.type` references in
  `<upstream>/<proto>/connector_flows/<name>.yaml`. The deduplicated set of `type:`
  values is what the module binds.
- **Standard API `flow_type_bindings`:** read `subflowBindingTypes:` in
  `<upstream>/<proto>/standard_apis/<name>.yaml`. The keys are the local subflow
  binding names, the values are the flow types (e.g. `submission`, `query`) which must
  resolve to deployed `kaleido_platform_connector_flow` resources.
- **Standard stream `config_profile_name_or_id`:** look at
  `<upstream>/<proto>/connector_stream_factories/<factoryName>.yaml`'s `configType:`
  field — the standard stream binds to the config profile of that config type.

## Step 6 — Update `.ai/plan.md` if the resource graph changed

The plan file lists the exact resource counts for each protocol. If a sync adds or
removes a resource, update that file too.

## Step 7 — Do not leak the upstream path

After the sync, double-check none of these files contains the upstream path or repo
name:

- Any file in `modules/`
- Any file in `examples/`
- Any file in `kaleido/`
- `.ai/plan.md` and `.ai/research.md`
- `README.md` (root)

Use `grep -r "<repo-or-org-name>"` against the working tree before reporting done.

## Output

Report what changed: new variables added, new resources added in `main.tf`, new
`*.tfvars` files, any defaults that shifted because the upstream `default:` changed.
Highlight any **breaking** changes (variable type tightening, removed config types)
so the human can write a CHANGELOG entry.
