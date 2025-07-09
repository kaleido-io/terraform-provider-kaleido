# Platform Child Accounts Example

This example demonstrates how to create a shared OIDC identity provider and multiple child accounts for the Kaleido platform. This setup is ideal for organizations that want to manage multiple environments (development, staging, production) with a single identity provider.

## Overview

This configuration creates:
- A shared OIDC identity provider with configurable endpoints
- Development, staging, and production accounts
- Each account configured with the shared identity provider
- Hostname bindings for each environment
- Initial admin users for each account

## Prerequisites

- Kaleido platform access with appropriate permissions
- A configured OIDC identity provider (e.g., Auth0, Keycloak, Azure AD, etc.)
- Terraform installed and configured

## Configuration

### Required Variables

The following variables must be provided:

```hcl
kaleido_platform_api      = "https://your-kaleido-platform.com"
kaleido_platform_username = "your-username"
kaleido_platform_password = "your-password"

# OIDC Configuration
oidc_client_id     = "your-oidc-client-id"
oidc_client_secret = "your-oidc-client-secret"
hostname           = "your-hostname.com"

# Admin users for each environment
dev_admin_email        = "dev-admin@example.com"
dev_admin_sub          = "dev-admin-subject-identifier"
staging_admin_email    = "staging-admin@example.com"
staging_admin_sub      = "staging-admin-subject-identifier"
production_admin_email = "prod-admin@example.com"
production_admin_sub   = "prod-admin-subject-identifier"
```

### Optional Variables

You can customize the configuration with these optional variables:

```hcl
# Identity Provider Configuration
identity_provider_name = "my-company-oidc"
oidc_config_url       = "https://auth.example.com/.well-known/openid-configuration"

# Or configure individual endpoints if not using oidc_config_url
issuer       = "https://auth.example.com"
login_url    = "https://auth.example.com/oauth/authorize"
token_url    = "https://auth.example.com/oauth/token"
jwks_url     = "https://auth.example.com/.well-known/jwks.json"
user_info_url = "https://auth.example.com/userinfo"
logout_url   = "https://auth.example.com/logout"

# Account Names
dev_account_name        = "my-dev-environment"
staging_account_name    = "my-staging-environment"
production_account_name = "my-production-environment"

# Hostname Mappings
dev_hostnames = {
  "dev.mycompany.com" = ["rest", "websocket"]
}
staging_hostnames = {
  "staging.mycompany.com" = ["rest"]
}
production_hostnames = {
  "prod.mycompany.com" = ["rest", "websocket"]
}
```

## Usage

1. **Set up variables**: Create a `terraform.tfvars` file with your configuration:

```hcl
kaleido_platform_api      = "https://your-kaleido-platform.com"
kaleido_platform_username = "your-username"
kaleido_platform_password = "your-password"

oidc_client_id     = "your-oidc-client-id"
oidc_client_secret = "your-oidc-client-secret"
hostname           = "your-hostname.com"
oidc_config_url    = "https://auth.example.com/.well-known/openid-configuration"

dev_admin_email        = "dev-admin@example.com"
dev_admin_sub          = "dev-admin-subject-identifier"
staging_admin_email    = "staging-admin@example.com"
staging_admin_sub      = "staging-admin-subject-identifier"
production_admin_email = "prod-admin@example.com"
production_admin_sub   = "prod-admin-subject-identifier"
```

2. **Initialize Terraform**:
```bash
terraform init
```

3. **Plan the deployment**:
```bash
terraform plan
```

4. **Apply the configuration**:
```bash
terraform apply
```

## Outputs

After successful deployment, you'll receive:

- **identity_provider**: Details of the shared OIDC identity provider
- **dev_account**: Development account information
- **staging_account**: Staging account information
- **production_account**: Production account information
- **all_accounts**: Combined information for all accounts
- **setup_summary**: Summary of the entire setup

## OIDC Configuration Options

### Using Discovery Endpoint

The simplest approach is to use the OIDC discovery endpoint:

```hcl
oidc_config_url = "https://auth.example.com/.well-known/openid-configuration"
```

### Manual Endpoint Configuration

If your OIDC provider doesn't support discovery, configure individual endpoints:

```hcl
issuer       = "https://auth.example.com"
login_url    = "https://auth.example.com/oauth/authorize"
token_url    = "https://auth.example.com/oauth/token"
jwks_url     = "https://auth.example.com/.well-known/jwks.json"
user_info_url = "https://auth.example.com/userinfo"
logout_url   = "https://auth.example.com/logout"
```

## Security Considerations

- **Client Secret**: Store sensitive values like `oidc_client_secret` and `kaleido_platform_password` in secure locations (environment variables, HashiCorp Vault, etc.)
- **PKCE**: The example enables PKCE for confidential clients by default
- **Nonce**: ID token nonce validation is enabled by default
- **Validation Policy**: Accounts are configured with a "strict" validation policy by default

## Troubleshooting

### Common Issues

1. **OIDC Configuration**: Ensure your OIDC provider is properly configured with the correct redirect URLs
2. **Admin Users**: Make sure the admin user subjects and emails correspond to actual users in your identity provider
3. **Hostname Bindings**: Verify that hostname mappings are correct for your environment
4. **Permissions**: Ensure your Kaleido platform credentials have sufficient permissions to create accounts and identity providers

### Validation

After deployment, verify:
- Identity provider appears in the Kaleido platform console
- All three accounts are created and properly configured
- Each account references the shared identity provider
- Admin users can authenticate to their respective environments

## Clean Up

To remove all resources:

```bash
terraform destroy
```

## Integration with Other Examples

This example can be used as a foundation for:
- Setting up users and groups within each account (see `platform_user_iam` example)
- Deploying application infrastructure to each environment
- Implementing cross-environment networking and access controls 