## Summary

Create an environment with:

* Chain Infrastructure Stack 
    - Besu Network 
    - EVM Gateway
    - Block Indexer
* Web3 Middleware Stack
    - Transaction Manager
    - FireFly
* Contract Manager
* Key Manager
* ERC20 Token Contract (deployed and configured)
* FireFly Event Listener and Webhook Subscription

This example creates a chain infrastructure setup similar to the digital assets example but without the Asset Manager components. It deploys an ERC20 contract and configures FireFly to listen for blockchain events and forward them to a webhook endpoint.

## Features

- **Chain Infrastructure**: Besu network with EVM gateway and block indexer
- **Web3 Middleware**: Transaction manager and FireFly runtime
- **ERC20 Contract**: Deploys ERC20WithData contract from Hyperledger FireFly tokens repository
- **FireFly Integration**: 
  - Event subscription for blockchain events
  - Contract listener for ERC20 Transfer events
  - Webhook delivery to `http://localhost:9090/webhook`

## Important Notes

- **Webhook Endpoint**: The FireFly subscription is configured to send events to `http://localhost:9090/webhook`. Ensure this endpoint is available, accesible from the location where firefly is running and listening before events are generated.
  
  **Note for Kubernetes deployments**: If FireFly is running in a Kubernetes cluster, `localhost:9090` will not reach your local machine. You'll need to either:
  - Expose your webhook server using a tool like `ngrok` or `localtunnel` and update the URL
  - Deploy the webhook server inside the cluster
  - Use a service accessible from the cluster (LoadBalancer, NodePort, or Ingress)
- **Webhook Configuration**: Webhook-specific fields like `url`, `method`, `tlsConfigName`, `retry`, and `httpOptions` should be set directly in the `options` object. See the Webhook Configuration section below for details.

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|:----:|:-----:|:-----:|
| kaleido_platform_api | Kaleido API URL | string | - | yes |
| kaleido_platform_username | API Key name | string | `` | yes |
| kaleido_platform_password | API Key value | string | `` | yes |
| environment_name | Environment name | string | `` | yes |
| besu_node_count | Number of nodes to create | number | 1 | no |

## Outputs

The example deploys:
- ERC20 contract at address: `kaleido_platform_cms_action_deploy.demotoken_erc20.contract_address`
- FireFly subscription: `kaleido_platform_firefly_subscription.erc20_transfer_webhook`
- FireFly contract listener: `kaleido_platform_firefly_contract_listener.erc20_transfer`
- FireFly namespace: `kaleido_platform_service.ffs_0.name`

## Usage

1. **Set up your Terraform variables:**
   ```bash
   cp terraform.tfvars.example terraform.tfvars
   # Edit terraform.tfvars with your actual values
   ```

   Or provide variables via command line:
   ```bash
   terraform plan -var="kaleido_platform_api=https://..." -var="kaleido_platform_username=..." -var="kaleido_platform_password=..." -var="environment_name=..."
   ```

   Or disable interactive prompts:
   ```bash
   terraform plan -input=false  # Will fail if variables are not provided
   ```

2. Run `terraform init` to initialize the provider (skip if using dev_overrides)
3. Run `terraform plan` to review the changes
4. Run `terraform apply` to create the infrastructure
5. Start the webhook echo server(
6. Run `terraform apply` to create the infrastructure
7. The FireFly subscription will automatically forward blockchain events to your webhook


## FireFly Event Flow

1. ERC20 Transfer events are emitted on the blockchain
2. FireFly contract listener captures these events
3. Events are processed by FireFly and sent to the configured subscription
4. Subscription forwards events to the webhook endpoint at `http://localhost:9090/webhook`

## Webhook Configuration

As documented in the [FireFly Subscription API schema](https://hyperledger.github.io/firefly/latest/reference/types/subscription/), webhook-specific fields should be set directly in the `options` object, not nested in a `webhook` object:

```hcl
options = {
  withData = true
  url = "https://example.com/webhook"
  method = "POST"  # Optional, defaults to POST
  tlsConfigName = "my-tls-config"  # Optional, name of TLS config associated with namespace
  retry = {  # Optional retry configuration
    enabled = true
    count = 3
    initialDelay = "1s"
    maxDelay = "30s"
  }
  httpOptions = {  # Optional HTTP connection options
    requestTimeout = "30s"
    connectionTimeout = "10s"
  }
}
```

**Note:** For HTTPS webhooks, use `tlsConfigName` to reference a TLS configuration associated with the namespace, rather than inline TLS config. See the FireFly documentation for details on configuring TLS.
