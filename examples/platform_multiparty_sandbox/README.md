# Platform Multiparty Sandbox

Create a complete Hyperledger FireFly multiparty blockchain environment with Web3 middleware and digital assets infrastructure.

## What This Creates

* **Chain Infrastructure Stack - Besu**
    - Besu Network with QBFT consensus
    - Besu Nodes (configurable count)
    - EVM Gateway
    - Block Indexer with UI
* **Chain Infrastructure Stack - IPFS**
    - IPFS Network 
    - IPFS Node for off-chain storage
* **3 Web3 Middleware Stacks** (1 per member)
    - FireFly Service (multiparty orchestration)
    - Private Data Manager (encrypted data exchange)
    - Transaction Manager (blockchain transactions)
* **3 Digital Assets Stacks**
    - Asset Manager Service (per member)
* **Environment Tools**
    - Key Manager (1 per member with HD wallets)
    - Contract Manager (shared)
    - Firefly smart contract deployment

## Prerequisites

Before running this Terraform example, ensure you have:

- **Kaleido Platform deployed** on Kubernetes
- **User account created** in the Kaleido Platform UI
- **Terraform installed** (v1.0+)
- **kubectl configured** with access to your cluster
- **Network access** to the platform API endpoint

### Getting Your Credentials

**Important:** Use your Kaleido Platform UI user credentials, NOT Keycloak admin or Kubernetes secrets.

1. **Find Your Platform URL:**
   ```bash
   kubectl get ingress kaleidoplatform -n default
   ```
   > e.g. [`https://account1.kaleido.dev`](https://account1.kaleido.dev)
   
2. **Get Initial Login Credentials:**
   
   For first-time setup, retrieve the Keycloak admin credentials:
   ```bash
   kubectl get secret keycloak-kaleido-admin -n default -o yaml
   ```
   
   Decode the username and password:
   ```bash
   kubectl get secret keycloak-kaleido-admin -n default -o jsonpath='{.data.username}' | base64 -d
   kubectl get secret keycloak-kaleido-admin -n default -o jsonpath='{.data.password}' | base64 -d
   ```

3. **Access the UI and Create User:**
   - Open the platform URL in your browser (e.g., `https://account1.kaleido.dev`)
   - Login with the Keycloak admin credentials from step 2
   - Create a new user account or use the admin account
   - **Important:** Use these UI user credentials in `terraform.tfvars`, not the K8s secret values

## Quick Start

### 1. Configure Your Variables

Copy the example file and fill in your details:

```bash
cp terraform.tfvars.example terraform.tfvars
```

Edit `terraform.tfvars` with your configuration. See [terraform.tfvars.example](terraform.tfvars.example) for a complete template.

**Minimum required configuration:**
```hcl
kaleido_platform_api      = "https://account1.kaleido.dev"
kaleido_platform_username = "your-username"                  # Created in the UI
kaleido_platform_password = "your-password"                  # Created in the UI
environment_name          = "my-multiparty-env"              # This could be any name
members                   = ["org1", "org2", "org3"]
```

### 2. Initialize Terraform

```bash
terraform init
```

### 3. Deploy the Environment

For local K8s deployments with self-signed certificates:

```bash
export KALEIDO_PLATFORM_INSECURE=true
terraform plan
terraform apply
```

**Note:** The deployment takes approximately 5 minutes as all services initialize.

**Verify Deployment:**

Run:
```bash
terraform state list
```

The output should look like this:
```bash
kaleido_platform_environment.env_0
kaleido_platform_network.besu_net
kaleido_platform_runtime.asset_managers["org1"]
kaleido_platform_runtime.asset_managers["org2"]
kaleido_platform_runtime.asset_managers["org3"]
kaleido_platform_runtime.bir_0
kaleido_platform_runtime.bnr[0]
kaleido_platform_runtime.cmr_0
kaleido_platform_runtime.gwr_0
kaleido_platform_runtime.kmr_0["org1"]
kaleido_platform_runtime.kmr_0["org2"]
kaleido_platform_runtime.kmr_0["org3"]
kaleido_platform_runtime.tmr_0["org1"]
kaleido_platform_runtime.tmr_0["org2"]
kaleido_platform_runtime.tmr_0["org3"]
kaleido_platform_service.asset_managers["org1"]
kaleido_platform_service.asset_managers["org2"]
kaleido_platform_service.cms_0
kaleido_platform_service.gws_0
kaleido_platform_service.kms_0["org1"]
kaleido_platform_service.kms_0["org2"]
kaleido_platform_service.kms_0["org3"]
kaleido_platform_stack.chain_infra_besu_stack
kaleido_platform_stack.digital_assets_stack["org1"]
kaleido_platform_stack.digital_assets_stack["org2"]
kaleido_platform_stack.digital_assets_stack["org3"]
kaleido_platform_stack.web3_middleware_stack["org1"]
kaleido_platform_stack.web3_middleware_stack["org2"]
kaleido_platform_stack.web3_middleware_stack["org3"]
```

### 4. Access Your Environment

Check the environment in the UI
Navigate to: https://account1.kaleido.dev and view your environment

## Configuration Options

| Name | Description | Type | Default | Required |
|------|-------------|:----:|:-------:|:--------:|
| `kaleido_platform_api` | Kaleido Platform API URL | string | - | yes |
| `kaleido_platform_username` | Platform user username | string | - | yes |
| `kaleido_platform_password` | Platform user password | string | - | yes |
| `environment_name` | Environment name | string | - | yes |
| `members` | List of organization names | list(string) | - | yes |
| `besu_node_count` | Number of Besu nodes | number | 1 | no |
| `runtime_size` | Size for all runtimes (Small/Medium/Large) | string | Small | no |
| `pdm_manage_p2p_tls` | Manage P2P TLS for Private Data Manager | bool | false | no |
 
## Cleanup

```bash
export KALEIDO_PLATFORM_INSECURE=true

# Step 1: Enable force deletion for Besu nodes
# Edit main.tf and uncomment the force_delete = true lines in:
#   - kaleido_platform_runtime.bnr (around line 73)
#   - kaleido_platform_service.bns (around line 89)

# Step 2: Apply the force_delete changes
terraform apply
# Note: You may see some errors as Terraform attempts to update running services.
# This is expected and safe - the important part is setting the force_delete flag.

# Step 3: Destroy all resources
terraform destroy
# WARNING: This may take 5-10 minutes or hang. If it hangs, use Ctrl+C and delete via UI instead.
```

## Troubleshooting

### Authentication Errors (401 Unauthorized)

**Error:** `KA011007: Unauthorized`

**Solutions:**
- ✓ Verify you're using Platform UI user credentials (not K8s secrets)
- ✓ Ensure you can login to the UI at the same URL
- ✓ Check username and password are correct
- ✓ Confirm user account exists and is active

### Certificate Errors (Local Software Users Only)

**Error:** `x509: certificate is not trusted`

> **Note:** This only applies to users running the Kaleido Platform locally using the Quick Start guide.

**Solution:**
```bash
export KALEIDO_PLATFORM_INSECURE=true
terraform apply
```

## Need Help?

- Ensure all prerequisites are met before deploying
- Contact Kaleido support if issues persist
