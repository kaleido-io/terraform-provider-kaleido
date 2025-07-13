# Structured Terraform Resources for Kaleido Platform

## Overview

This implementation adds structured, typed Terraform resources for Besu blockchain networks and services, providing a significantly improved user experience over the generic `kaleido_platform_service` and `kaleido_platform_network` resources.

## New Resources

### `kaleido_platform_besu_network`
- **Purpose**: Creates and manages Besu blockchain networks with structured configuration
- **Benefits**: Type-safe QBFT/IBFT consensus configuration, network parameters, and initialization options
- **File**: `terraform-provider-kaleido/kaleido/platform/besu_network.go`

### `kaleido_platform_besunode_service` 
- **Purpose**: Creates and manages Besu node services with comprehensive typed configuration
- **Benefits**: Structured node configuration, storage settings, API options, and consensus parameters
- **File**: `terraform-provider-kaleido/kaleido/platform/besunode_service.go`

## Key Improvements

### 1. Type Safety & Validation
- **Before**: Unstructured `config_json` with no validation
- **After**: Typed fields with proper validation at plan time

### 2. Documentation & Discoverability
- **Before**: Users had to consult API docs or examples to understand configuration
- **After**: Inline documentation, field descriptions, and valid options

### 3. IDE Support
- **Before**: No autocompletion for configuration fields
- **After**: Full IDE support with autocompletion and type hints

### 4. Default Values
- **Before**: Users had to specify all configuration manually
- **After**: Sensible defaults for optional fields (e.g., `log_level = "INFO"`, `sync_mode = "FULL"`)

### 5. Operational Safety
- **Before**: Configuration errors discovered at apply time
- **After**: Configuration validation at plan time prevents deployment failures

## Example Comparison

### Before (Generic Resources)
```hcl
resource "kaleido_platform_network" "besu_net" {
  type = "BesuNetwork"
  name = "evmchain1"
  environment = kaleido_platform_environment.env_0.id
  config_json = jsonencode({
    bootstrapOptions = {
      qbft = {
        blockperiodseconds = 2
      }
    }
  })
}

resource "kaleido_platform_service" "bns" {
  type = "BesuNode"
  name = "chain_node_1"
  environment = kaleido_platform_environment.env_0.id
  runtime = kaleido_platform_runtime.bnr.id
  config_json = jsonencode({
    network = {
      id = kaleido_platform_network.besu_net.id
    }
  })
  stack_id = kaleido_platform_stack.chain_infra_besu_stack.id
}
```

### After (Structured Resources)
```hcl
resource "kaleido_platform_besu_network" "besu_net" {
  name = "evmchain1"
  environment = kaleido_platform_environment.env_0.id
  
  bootstrap_options = {
    qbft = {
      block_period_seconds = 2
      epoch_length = 30000
      request_timeout = 10000
    }
  }
  
  consensus_type = "qbft"
  init_mode = "automated"
}

resource "kaleido_platform_besunode_service" "besu_node" {
  name = "chain_node_1"
  environment = kaleido_platform_environment.env_0.id
  runtime = kaleido_platform_runtime.bnr.id
  stack_id = kaleido_platform_stack.chain_infra_besu_stack.id
  
  network = {
    id = kaleido_platform_besu_network.besu_net.id
  }
  
  mode = "active"
  signer = true
  log_level = "INFO"
  sync_mode = "FULL"
  data_storage_format = "FOREST"
  
  storage = {
    size = "20Gi"
  }
  
  apis_enabled = ["TRACE", "DEBUG"]
  gas_price = "0"
  target_gas_limit = 30000000
}
```

## Implementation Details

### Architecture
- **Wrapper Pattern**: Structured resources wrap the existing generic `ServiceAPIModel` and `NetworkAPIModel`
- **Translation Layer**: `toServiceAPI()` and `toNetworkAPI()` methods convert structured data to generic API models
- **Backwards Compatibility**: Uses the same underlying API endpoints as generic resources

### Resource Registration
- Added to `Resources()` function in `terraform-provider-kaleido/kaleido/platform/common.go`
- Registered as `BesuNodeServiceResourceFactory` and `BesuNetworkResourceFactory`

### Schema Design
- **Comprehensive Coverage**: All BesuNode and BesuNetwork configuration options supported
- **Proper Defaults**: Default values match service catalog specifications
- **Optional vs Required**: Clear distinction between required and optional fields
- **Nested Objects**: Complex configurations like `storage` and `bootstrap_options` properly structured

## Future Extensibility

This pattern can be extended to create structured resources for other service types:
- `kaleido_platform_firefly_service`
- `kaleido_platform_ipfs_service`
- `kaleido_platform_evmgateway_service`
- `kaleido_platform_ipfs_network`

Each would follow the same pattern:
1. Create structured resource file
2. Define typed schema with proper validation
3. Implement translation to generic API model
4. Register in provider
5. Add comprehensive example and documentation

## Benefits for Operations

### Reduced Configuration Errors
- Type validation prevents common mistakes
- Required field validation ensures complete configuration
- Enum validation prevents invalid option values

### Improved Debugging
- Clear field names make troubleshooting easier
- Structured output makes state inspection clearer
- Better error messages from validation

### Enhanced Team Productivity
- Developers can focus on infrastructure logic rather than deciphering API schemas
- Onboarding new team members is easier with clear documentation
- IDE support accelerates development

### Production Safety
- Configuration validation at plan time prevents deployment failures
- Consistent field naming reduces human error
- Clear documentation prevents misconfiguration

## Migration Strategy

Users can migrate from generic resources to structured resources incrementally:
1. Extract configuration from `config_json` to structured fields
2. Update resource type names
3. Test in non-production environments
4. Apply changes with minimal disruption

The structured resources generate the same underlying API calls, ensuring compatibility with existing infrastructure.

## Example Usage

A complete working example is provided in:
- **Location**: `terraform-provider-kaleido/examples/platform_besu_structured/`
- **Documentation**: `terraform-provider-kaleido/examples/platform_besu_structured/README.md`
- **Features**: Full network and node configuration with best practices

This implementation significantly improves the user experience for managing Besu blockchain infrastructure while maintaining full compatibility with the existing API. 