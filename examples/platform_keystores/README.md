## Summary

Examples of how to create different types of Keystores in a Key Manager service. 

Create an environment with:

* Key Manager Service
    Keystores: 
        - Azure Key Vault Keystore

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|:----:|:-----:|:-----:|
| kaleido_platform_api | Kaleido API URL | string | - | yes |
| kaleido_platform_username | API Key name | string | - | yes |
| kaleido_platform_password | API Key value | string | - | yes |
| environment_name | Environment name | string | `keystores-example` | no |
| azure_key_vault_name | The Name of the Azure Key Vault where you want to fetch keys | string | - | yes |
| azure_key_vault_base_url | The base URL of the Azure Key Vault where you want to fetch your keys | string | `https://vault.azure.net` | no |
| azure_app_registration_client_id | The client ID for the Azure App Registration | string | - | yes |
| azure_app_registration_client_secret | The client secret of the Azure App Registration | string | - | yes |
| azure_app_registration_tenant_id | The tenant ID of the Azure App Registration | string | - | yes |


