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

# Register an MCP server as a Hub-authenticated service.
resource "youtrack_service" "mcp_server" {
  name             = "My MCP Server"
  application_name = "my-mcp-server"
  home_url         = "https://mcp.example.com"
  description      = "MCP server integration for YouTrack"
  redirect_uris    = ["https://mcp.example.com/oauth/callback"]
  base_urls        = ["https://mcp.example.com"]
  trusted          = false
  consent_required = true
}

output "mcp_server_secret" {
  value     = youtrack_service.mcp_server.secret
  sensitive = true
}
