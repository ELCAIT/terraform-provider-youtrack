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

# Minimal service registration – only the display name.
resource "youtrack_service" "minimal" {
  name = "My Service"
}

# Register an MCP server as a Hub-authenticated service, restricted to the
# PKCE-protected authorization code flow (the recommended flow for an
# interactive client with a redirect URI).
resource "youtrack_service" "mcp_server" {
  name             = "My MCP Server"
  application_name = "my-mcp-server"
  home_url         = "https://mcp.example.com"
  description      = "MCP server integration for YouTrack"
  redirect_uris    = ["https://mcp.example.com/oauth/callback"]
  base_urls        = ["https://mcp.example.com"]
  trusted          = false
  consent_required = true

  client_credentials_flow_enabled = false
  auth_code_flow_enabled          = true
  pkce_required                   = true
  implicit_flow_enabled           = false
  resource_owner_flow_enabled     = false
}

output "mcp_server_secret" {
  value     = youtrack_service.mcp_server.secret
  sensitive = true
}

# Register a machine-to-machine integration that authenticates on its own
# behalf (no interactive user), restricted to the client credentials flow.
resource "youtrack_service" "backend_integration" {
  name        = "My Backend Integration"
  description = "Server-to-server integration with YouTrack"
  trusted     = true

  client_credentials_flow_enabled = true
  auth_code_flow_enabled          = false
  pkce_required                   = false
  implicit_flow_enabled           = false
  resource_owner_flow_enabled     = false
}

output "backend_integration_secret" {
  value     = youtrack_service.backend_integration.secret
  sensitive = true
}
