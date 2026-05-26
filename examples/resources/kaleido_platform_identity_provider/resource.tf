# See https://docs.kaleido.io/platform/kaleido-platform/access-management/sso/#tested-providers for additional information on the supported providers and their properties.

resource "kaleido_platform_identity_provider" "azure_entra_id" {
  name = "my-azure-entra-id-idp"
  hostname = "my-azure-entra-id-idp" # results in login redirect URL of `https://my-azure-entra-id-idp.${PLATFORM_DOMAIN}/oauth/callback`
  

  client_type = "confidential"
  client_id = "a-client-id"                            # TODO: replace with the actual client ID of your application
  client_secret = "***"                                # TODO: replace with the actual client secret of your application
  id_token_nonce_enabled = true
  scopes = "openid email profile a-client-id/.default" # TODO: replace with the actual client ID of your application

  # omit oidc_config_url and user_info_url - userinfo is not supported for Entra ID v2.0
  # TODO: replace with the actual tenant ID
  issuer = "https://login.microsoftonline.com/your-tenant-id/v2.0/"
  login_url = "https://login.microsoftonline.com/your-tenant-id/oauth2/v2.0/authorize"
  logout_url = "https://login.microsoftonline.com/your-tenant-id/oauth2/v2.0/logout"
  token_url = "https://login.microsoftonline.com/your-tenant-id/oauth2/v2.0/token"
  jwks_url = "https://login.microsoftonline.com/your-tenant-id/discovery/v2.0/keys"
}

resource "kaleido_platform_identity_provider" "google_identity" {
  name = "my-google-idp"
  hostname = "my-google-idp" # results in login redirect URL of `https://my-google-idp.${PLATFORM_DOMAIN}/oauth/callback`
  
  client_type = "confidential"
  client_id = "a-client-id"                            # TODO: replace with the actual client ID of your OAuth client
  client_secret = "***"                                # TODO: replace with the actual client secret of your OAuth client
  id_token_nonce_enabled = true
  // default scopes 'openid profile email' are recommended
  oidc_config_url = "https://accounts.google.com/.well-known/openid-configuration"
}

resource "kaleido_platform_identity_provider" "okta" {
  name = "my-okta-idp"
  hostname = "my-okta-idp" # results in login redirect URL of `https://my-okta-idp.${PLATFORM_DOMAIN}/oauth/callback`

  client_type = "confidential"
  client_id = "a-client-id"                            # TODO: replace with the actual client ID of your OAuth client
  client_secret = "***"                                # TODO: replace with the actual client secret of your OAuth client
  scopes = "openid email profile offline_access" 
  oidc_config_url = "https://your-okta-domain.okta.com/.well-known/openid-configuration"
}