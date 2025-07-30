# Structured Besu Resources Example

This example demonstrates how to use the new structured Terraform resources for Besu blockchain networks:

- `kaleido_platform_besu_network` - Creates a Besu blockchain network with typed configuration
- `kaleido_platform_besunode_service` - Creates Besu node services with structured schema

## Benefits of Structured Resources

### Instead of Generic Resources

The traditional approach using generic resources requires working with unstructured `config_json`:

```hcl
# OLD WAY - Generic resource with unstructured config
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

### With Structured Resources

The new structured resources provide typed, documented configuration:

```hcl
# NEW WAY - Structured resources with typed configuration
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

## Key Benefits

1. **Type Safety**: Fields are properly typed (string, int, bool) with validation
2. **Documentation**: Each field has clear descriptions and valid options
3. **Autocomplete**: IDE support for field names and values
4. **Validation**: Terraform validates configuration at plan time
5. **Defaults**: Sensible defaults for optional fields
6. **Discoverability**: No need to consult API docs or examples to understand schema

## Running the Example

1. Set your Kaleido platform credentials:
   ```bash
   export TF_VAR_kaleido_platform_api="https://your-platform-api.kaleido.io"
   export TF_VAR_kaleido_platform_username="your-username"
   export TF_VAR_kaleido_platform_password="your-password"
   ```

2. Initialize and apply:
   ```bash
   terraform init
   terraform plan
   terraform apply
   ```

3. View the outputs to see network information and node endpoints:
   ```bash
   terraform output besu_network_id
   terraform output besu_network_info
   terraform output besu_node_endpoints
   ```

## Configuration Options

### Besu Network Configuration

- **consensus_type**: `qbft` or `ibft` (default: `qbft`)
- **chain_id**: Custom chain ID (auto-generated if not provided)
- **init_mode**: `automated` or `manual` (default: `automated`)
- **bootstrap_options**: Consensus-specific configuration
  - **qbft**: QBFT consensus parameters
    - `block_period_seconds`: Block time in seconds (default: 2)
    - `epoch_length`: Epoch length in blocks (default: 30000)
    - `request_timeout`: Request timeout in milliseconds (default: 10000)
  - **ibft**: IBFT consensus parameters (same structure as QBFT)

### Besu Node Configuration

- **mode**: `active` or `standby` (default: `active`)
- **signer**: Whether node participates in consensus (default: `true`)
- **log_level**: `INFO`, `DEBUG`, or `TRACE` (default: `INFO`)
- **sync_mode**: `FAST`, `FULL`, or `SNAP` (default: `FULL`)
- **data_storage_format**: `FOREST` or `BONSAI` (default: `FOREST`)
- **storage**: Storage configuration
  - `size`: Storage size in Kubernetes format (default: `10Gi`)
- **apis_enabled**: Additional APIs to enable beyond the defaults
- **gas_price**: Gas price for transactions (default: `0`)
- **target_gas_limit**: Target gas limit for blocks (default: `0`)
- **custom_besu_args**: Additional command-line arguments for Besu

## Advanced Usage

### Using File Sets and Credential Sets

For advanced configurations, you can reference file sets and credential sets:

```hcl
resource "kaleido_platform_besu_network" "besu_net" {
  name = "custom-genesis-network"
  environment = kaleido_platform_environment.env_0.id
  
  # Reference a custom genesis file
  genesis = {
    file_set_ref = "my-genesis-fileset"
  }
  
  # Reference a custom network key
  node_key = {
    cred_set_ref = "my-network-key"
  }
}

resource "kaleido_platform_besunode_service" "besu_node" {
  name = "custom-node"
  environment = kaleido_platform_environment.env_0.id
  runtime = kaleido_platform_runtime.bnr.id
  stack_id = kaleido_platform_stack.chain_infra_besu_stack.id
  
  network = {
    id = kaleido_platform_besu_network.besu_net.id
  }
  
  # Reference a custom node key
  node_key = {
    cred_set_ref = "my-node-key"
  }
}
```

### Force Deletion

For protected networks and services, you can enable force deletion:

```hcl
resource "kaleido_platform_besu_network" "besu_net" {
  # ... other configuration
  force_delete = true
}

resource "kaleido_platform_besunode_service" "besu_node" {
  # ... other configuration
  force_delete = true
}
```

Remember to apply the `force_delete = true` change before running `terraform destroy`.

## Migration from Generic Resources

To migrate from generic `kaleido_platform_service` and `kaleido_platform_network` resources:

1. **Extract configuration**: Take the values from your `config_json` and map them to the structured fields
2. **Update resource types**: Change the resource type names
3. **Remove config_json**: Replace with structured field configurations
4. **Test thoroughly**: Ensure all configuration options are preserved

The structured resources translate to the same underlying API calls, so your existing infrastructure should be compatible. 