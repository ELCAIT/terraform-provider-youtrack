# Full OAuth 2.0 auth module – typical Keycloak / generic IdP configuration
resource "youtrack_auth_module_oauth2" "keycloak" {
  name                      = "Keycloak PP"
  disabled                  = false
  is_default                = false
  client_id                 = "youtrack-local"
  client_secret             = var.keycloak_client_secret
  server_url                = "https://keycloak.example.com/realms/demo/protocol/openid-connect/auth"
  token_url                 = "https://keycloak.example.com/realms/demo/protocol/openid-connect/token"
  scope                     = "openid email profile"
  user_info_url             = "https://keycloak.example.com/realms/demo/protocol/openid-connect/userinfo"
  form_client_auth          = false
  email_verified_by_default = false
  allowed_create_new_users  = true
  background_sync_enabled   = false
  sync_interval             = "0 0 0/3 * * ?"

  # Claim mappings
  user_id_path             = "sub"
  user_email_path          = "email"
  user_email_verified_path = "email_verified"
  user_name_path           = "visa"
  full_name_path           = "name"
  user_picture_id_path     = "picture"
  user_groups_path         = "groups"

  # Timeouts (milliseconds)
  connection_timeout = 5000
  read_timeout       = 5000
}

variable "keycloak_client_secret" {
  description = "Keycloak client secret for the Hub OAuth 2.0 auth module."
  type        = string
  sensitive   = true
}
