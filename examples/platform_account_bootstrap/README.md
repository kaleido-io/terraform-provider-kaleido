# Platform Account Bootstrap Example

This example demonstrates how to bootstrap a new Kaleido platform account with an OIDC identity provider, validation policies, and Kubernetes Service Account integration. This setup is ideal for organizations that want to create a new account with proper authentication, authorization, and service account integration from the start.

## Overview

This configuration creates:
- An OIDC identity provider for user authentication
- A new bootstrapped account using the OIDC identity provider resource
- A validation policy that supports reader, admin, and owner roles
- A bootstrap OAuth application for Kubernetes service account authentication into the new account
- A default "readers" group in the bootstrapped account
- Just-in-time (JIT) user provisioning enabled

## Prerequisites

Before running this example, ensure you have:
1. OpenTofu installed (version 1.9 or later)
2. Access to a root Kaleido platform account with permissions to create child accounts
3. Credentials for a valid OIDC identity provider (e.g., Auth0, Keycloak, Microsoft Entra ID, etc.)
4. Kubernetes cluster OIDC information (if using the bootstrap application feature)

## Usage

1. **Set up variables**: Create a `input.tfvars` file with your configuration:

```hcl
kaleido_platform_api          = "https://account1.kaleido.dev"
kaleido_platform_bearer_token = "your-bearer-token"

idp_client_id       = "your-oidc-client-id"
idp_client_secret    = "your-oidc-client-secret"
idp_oidc_config_url  = "https://auth.example.com/.well-known/openid-configuration"

first_user_email = "admin@example.com"

# Optional: Customize Kubernetes integration
bootstrap_application_issuer       = "https://kubernetes.default.svc.cluster.local"
bootstrap_application_jwks_endpoint = "https://kubernetes.default.svc.cluster.local/openid/v1/jwks"
bootstrap_application_k8s_namespace = "default"
bootstrap_application_k8s_sa_name   = "kaleidoplatform"
```

2. **Initialize OpenTofu**:
```bash
tofu init
```

3. **Plan the deployment**:
```bash
tofu plan -var-file=input.tfvars 
```

4. **Apply the configuration**:
```bash
tofu apply -var-file=input.tfvars 
```

## Resources Created

### Identity Provider
- **Type**: `kaleido_platform_identity_provider`
- **Name**: `kaleido-id`
- **Purpose**: Provides OIDC authentication for the bootstrapped account
- **Features**:
  - Confidential client type
  - ID token nonce validation enabled
  - Configurable OIDC endpoints

### Bootstrapped Account
- **Type**: `kaleido_platform_account`
- **Name**: `bootstrapped`
- **Purpose**: The new account with full authentication and authorization setup
- **Features**:
  - OIDC integration with the identity provider
  - Just-in-time (JIT) user provisioning enabled
  - Default group assignment for JIT users: "readers"
  - Validation policy supporting reader, admin, and owner roles
  - Bootstrap OAuth application for Kubernetes service account authentication
  - Hostname binding for the account

### Readers Group
- **Type**: `kaleido_platform_group`
- **Name**: `readers`
- **Purpose**: Default group for users provisioned via JIT
- **Location**: Created within the bootstrapped account

## Key Features

### OIDC Authentication
- **Identity Provider**: Fully configured OIDC identity provider
- **Client Configuration**: Confidential client with secure credential handling
- **Nonce Validation**: Enhanced security with ID token nonce validation

### Just-in-Time User Provisioning
- **Automatic Provisioning**: Users are automatically created on first login
- **Default Group**: New users are assigned to the "readers" group by default
- **Email-Based**: Initial admin user configured via email address

### Role-Based Validation Policy
The validation policy supports three role types:
- **Reader**: Basic read access
- **Admin**: Administrative privileges
- **Owner**: Full ownership privileges

The policy validates:
- OIDC audience (client ID) matches
- User roles from the ID token
- Multiple role combinations for flexible access control

### Kubernetes Integration
- **Bootstrap Application**: Pre-configured OAuth application for Kubernetes
- **Service Account Authentication**: Supports Kubernetes service account authentication
- **Configurable Endpoints**: Customizable issuer and JWKS endpoints
- **CA Certificate Support**: Optional CA certificate for secure connections
- **Namespace and Service Account**: Configurable Kubernetes namespace and service account name

## Validation Policy Details

The validation policy uses Rego (Open Policy Agent) to control access:

```rego
package sample_validation

default allow := false

# Validates OIDC audience matches the configured client ID
is_valid_aud(aud) := aud == "${idp_client_id}"

# Role validators
is_valid_reader(roles) := roles == ["reader"]
is_valid_admin(roles) := roles == ["admin"]
is_valid_owner(roles) := roles == ["owner"]

# Access rules
allow {
    is_valid_aud(input.id_token.aud)
    is_valid_reader(input.id_token.roles)
}

allow {
    is_valid_aud(input.id_token.aud)
    is_valid_admin(input.id_token.roles)
}

allow {
    is_valid_aud(input.id_token.aud)
    is_valid_owner(input.id_token.roles)
}
```

### Kubernetes Application Validation Policy

The bootstrap application has its own validation policy for service account authentication:

```rego
package k8s_application_validation

default allow := false

is_valid_sa(sub) := sub == "system:serviceaccount:${namespace}:${service_account_name}"

allow {
    is_valid_sa(input.sub)
}
```

## Security Considerations

- **Client Secret**: Store `idp_client_secret` and `kaleido_platform_bearer_token` securely (environment variables, HashiCorp Vault, etc.)
- **Nonce Validation**: ID token nonce validation is enabled by default for enhanced security
- **Validation Policy**: Strict validation policy ensures only authorized users and service accounts can access the platform
- **Confidential Client**: Identity provider uses confidential client type for secure credential handling
- **CA Certificate**: Use the optional CA certificate for bootstrap application when connecting to Kubernetes clusters with custom certificates

## Account URL

After successful deployment, the bootstrapped account will be accessible at:
```
https://bootstrapped.{your-platform-domain}
```

The platform domain is automatically extracted from the root account's platform API URL.

## Troubleshooting

### Common Issues

1. **OIDC Configuration Errors**:
   - Verify the `idp_oidc_config_url` is accessible and returns valid OIDC configuration
   - Ensure the client ID and secret match your OIDC provider configuration
   - Check that redirect URLs are properly configured in your OIDC provider

2. **Account Creation Failures**:
   - Verify you have permissions to create child accounts in the root account
   - Check that the bearer token is valid and has sufficient privileges
   - Ensure the platform API URL is correct

3. **First User Issues**:
   - Verify the `first_user_email` corresponds to a valid user in your OIDC provider
   - Ensure the user has the appropriate roles (reader, admin, or owner) in the ID token
   - Check that the OIDC provider is configured to include roles in the ID token

4. **Kubernetes Integration Issues**:
   - Verify the Kubernetes issuer URL is correct for your cluster
   - Ensure the JWKS endpoint is accessible from the Kaleido platform
   - Check that the service account exists in the specified namespace
   - Verify the CA certificate if using a custom certificate authority

5. **Validation Policy Errors**:
   - Ensure your OIDC provider includes roles in the ID token
   - Verify the role format matches the validation policy expectations
   - Check that the audience claim matches the configured client ID

### Validation

After deployment, verify:
- Identity provider appears in the root account
- Bootstrapped account is created and accessible
- First user can authenticate and has appropriate access
- Readers group exists in the bootstrapped account
- Bootstrap application is configured correctly (if using Kubernetes integration)

### Testing Authentication

1. Navigate to the bootstrapped account URL
2. Attempt to log in with a user from your OIDC provider
3. Verify the user is automatically provisioned (JIT)
4. Confirm the user is assigned to the "readers" group
5. Test role-based access based on the ID token roles

## Integration with Other Examples

This example can be used as a foundation for:
- Setting up additional users and groups (see `platform_user_iam` example)
- Creating child accounts from the bootstrapped account (see `platform_child_accounts` example)
- Configuring access controls and networking
- Deploying application infrastructure

### Example Integration

```hcl
# Use the bootstrapped account in other configurations
provider "kaleido" {
  platform_api = "https://bootstrapped.${local.kaleido_platform_domain}"
  platform_bearer_token = var.kaleido_platform_bearer_token
}

# Create additional resources in the bootstrapped account
resource "kaleido_platform_group" "developers" {
  name = "developers"
}
```

## Clean Up

To remove all resources:

```bash
tofu destroy
```

**Warning**: This will remove the bootstrapped account, identity provider, and all associated resources. Ensure you have proper backups and that no critical data will be lost.

## Next Steps

After bootstrapping the account:
1. Configure additional groups and users as needed
2. Set up application infrastructure
3. Configure networking and access controls
4. Deploy your applications and services
5. Set up monitoring and logging

This example provides a complete foundation for creating a new Kaleido platform account with proper authentication, authorization, and Kubernetes integration configured from the start.

