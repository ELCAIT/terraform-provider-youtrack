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

# Minimal Entra ID auth module – only required fields.
# Leaving tenant unset allows sign-in from any Microsoft tenant ("common").
resource "youtrack_auth_module_azure" "minimal" {
  name          = "My Entra ID Provider"
  client_id     = "my-client-id"
  client_secret = "my-client-secret"
}

# Full Entra ID auth module – single-tenant configuration.
# tenant is the Entra ID directory (tenant) ID, a GUID – found in the Entra
# admin center under "Tenant ID", not the *.onmicrosoft.com domain name.
resource "youtrack_auth_module_azure" "entra" {
  name                     = "Contoso Entra ID"
  disabled                 = false
  client_id                = "00000000-0000-0000-0000-000000000000"
  client_secret            = var.entra_client_secret
  tenant                   = "11111111-1111-1111-1111-111111111111"
  allowed_create_new_users = true
  background_sync_enabled  = false
  request_group_permission = true
  request_id_token         = false

  # Timeouts (milliseconds)
  connection_timeout = 5000
  read_timeout       = 10000
}

variable "entra_client_secret" {
  description = "Entra ID app registration client secret for the Hub auth module."
  type        = string
  sensitive   = true
}
