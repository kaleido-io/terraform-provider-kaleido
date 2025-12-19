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
- A configured OIDC identity provider (e.g., Auth0, Keycloak, Microsoft Entra ID, etc.)
- Terraform installed and configured

## Configuration

## Usage

1. **Set up variables**: Create a `input.tfvars` file with your configuration:

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

## Outputs

After successful deployment, you'll receive:

- **identity_provider**: Details of the shared OIDC identity provider
- **dev_account**: Development account information
- **staging_account**: Staging account information
- **production_account**: Production account information
- **all_accounts**: Combined information for all accounts
- **setup_summary**: Summary of the entire setup


## Troubleshooting

### Common Issues

1. **OIDC Configuration**: Ensure your OIDC provider is properly configured with the correct redirect URLs
2. **Admin Users**: Make sure the admin user subjects and emails correspond to actual users in your identity provider
3. **Hostname Bindings**: Verify that hostname mappings are correct for your environment
4. **Permissions**: Ensure your Kaleido platform credentials have sufficient permissions to create accounts and identity providers

### Validation

After deployment, verify:
- Identity provider appears in the root account in the Kaleido platform console
- All three accounts are created and properly configured
- Each account references the shared identity provider
- Admin users can authenticate to their respective environments