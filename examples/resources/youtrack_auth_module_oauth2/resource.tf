terraform {
  required_providers {
    youtrack = {
      source = "registry.opentofu.org/elcait/youtrack"
    }
  }
}

provider "youtrack" {
  base_url = "https://youtrack.example.com"
  token    = "your-api-token"
}

# Minimal OAuth 2.0 auth module – only required fields
resource "youtrack_auth_module_oauth2" "minimal" {
  name          = "My OAuth2 Provider"
  client_id     = "my-client-id"
  client_secret = "my-client-secret"
  server_url    = "https://idp.example.com"
  token_url     = "https://idp.example.com/oauth2/token"
}

# Full OAuth 2.0 auth module – typical Keycloak / generic IdP configuration
resource "youtrack_auth_module_oauth2" "keycloak" {
  name                      = "Keycloak"
  disabled                  = false
  client_id                 = "hub"
  client_secret             = var.keycloak_client_secret
  server_url                = "https://keycloak.example.com/realms/myrealm"
  token_url                 = "https://keycloak.example.com/realms/myrealm/protocol/openid-connect/token"
  scope                     = "openid email profile"
  user_info_url             = "https://keycloak.example.com/realms/myrealm/protocol/openid-connect/userinfo"
  idp_logout_url            = "https://keycloak.example.com/realms/myrealm/protocol/openid-connect/logout"
  form_client_auth          = false
  email_verified_by_default = true
  allowed_create_new_users  = true
  background_sync_enabled   = false

  # Claim mappings
  user_id_path     = "sub"
  user_email_path  = "email"
  user_name_path   = "preferred_username"
  full_name_path   = "name"
  user_groups_path = "groups"

  # Timeouts (milliseconds)
  connection_timeout = 5000
  read_timeout       = 10000
}

variable "keycloak_client_secret" {
  description = "Keycloak client secret for the Hub OAuth 2.0 auth module."
  type        = string
  sensitive   = true
}
