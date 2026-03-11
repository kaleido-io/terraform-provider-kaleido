## Summary

Create an application with an API key and an account access policy that restricts which environments the application can access.

This example demonstrates how to use a [Rego](https://www.openpolicyagent.org/docs/latest/policy-language/) policy to scope an application's access to specific environments by name. The included sample policy (`policy.rego`) allows access only to environments named `production` and `staging`.

## Resources Created

| Resource | Description |
|----------|-------------|
| `kaleido_platform_application` | A non-admin application identity |
| `kaleido_platform_account_access_policy` | A Rego policy attached to the application that controls which account resources it can access |
| `kaleido_platform_api_key` | An API key for authenticating as the application |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|:----:|:-------:|:--------:|
| `kaleido_platform_api` | Kaleido platform API URL | string | - | yes |
| `api_key_name` | API key name used to authenticate with the platform | string | - | yes |
| `api_key_value` | API key secret used to authenticate with the platform | string | - | yes |
| `application_name` | Name of the application to create | string | `env-scoped-app` | no |
| `new_api_key_name` | Name of the API key to generate for the application | string | `env-scoped-api-key` | no |
| `rego_policy` | Rego policy document (if empty, loads `policy.rego`) | string | `""` | no |

## Outputs

| Name | Description |
|------|-------------|
| `api_key_secret` | The generated API key secret (sensitive) |

## Rego Policy

The sample `policy.rego` restricts the application to environments named `production` and `staging`:

```rego
package platform_permission

import rego.v1

default allow := false

is_valid_env(env) := env in ["production", "staging"]

allow if {
    is_valid_env(input.environment.name)
}
```

Edit `policy.rego` to change the allowed environment names, or pass a custom policy via the `rego_policy` variable.

The policy input for `kaleido_platform_account_access_policy` includes `input.environment.name`, which is evaluated against each request to determine whether the application is permitted to access that environment.

## Usage

1. **Set up variables:**
   ```bash
   cp terraform.tfvars.example terraform.tfvars
   # Edit terraform.tfvars with your actual values
   ```

2. **Initialize and apply:**
   ```bash
   tofu init
   tofu plan
   tofu apply
   ```

3. **Retrieve the generated API key:**
   ```bash
   tofu output -raw api_key_secret
   ```

4. **Test the API key** against an allowed environment:
   ```bash
   curl -u "env-scoped-api-key:THE_SECRET" \
     https://your-platform-url/api/v1/environments
   ```
