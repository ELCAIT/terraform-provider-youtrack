# Register an MCP server as a Hub-authenticated service, restricted to the
# PKCE-protected authorization code flow (the recommended flow for an
# interactive client with a redirect URI).
resource "youtrack_service" "mcp_server" {
  name        = "my-mcp-server"
  description = "MCP server integration for YouTrack"
  trusted     = true

  client_credentials_flow_enabled = false
  auth_code_flow_enabled          = true
  pkce_required                   = true
  implicit_flow_enabled           = false
  resource_owner_flow_enabled     = false
  consent_required                = true
}
