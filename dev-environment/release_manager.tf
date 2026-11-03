# Release Manager, configured as documented in elca-youtrack
# (docs/operation/manual-configuration.md). The app must be installed first:
# app_name is its package name, as listed by GET /api/admin/apps.

# The group is looked up by name, so the ID is the one YouTrack knows it by.
data "youtrack_group" "release_managers" {
  name       = "youtrack-administrators"
  depends_on = [youtrack_nested_group.youtrack_administrators]
}

data "youtrack_group" "registered_users" {
  name = "Registered Users"
}

resource "youtrack_app_global_config" "release_manager" {
  app_name = "release-manager"
  enabled  = true

  settings = jsonencode({
    releaseManagers       = [{ id = data.youtrack_group.release_managers.id }]
    lightManagers         = [{ id = data.youtrack_group.release_managers.id }]
    manualIssueManagement = true
    metaIssuesEnabled     = true
    customFieldsMapping   = true
    autoAttachByFilter    = false
    outboundWebhooks      = false
  })

  # Visible to everyone, like the other apps in manual-configuration.md.
  permitted_group_ids = [data.youtrack_group.registered_users.id]
}
