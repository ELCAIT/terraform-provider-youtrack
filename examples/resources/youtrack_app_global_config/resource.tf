variable "youtrack_app_token" {
  type      = string
  sensitive = true
}

# The IDs of the groups come from the youtrack_group data source, not from
# values written by hand.
data "youtrack_group" "admins" {
  name = "youtrack-administrators"
}

data "youtrack_group" "registered_users" {
  name = "Registered Users"
}

# Configure an installed app. app_name is the package name listed by
# GET /api/admin/apps, not the title shown in the UI.
resource "youtrack_app_global_config" "agile" {
  app_name = "elca-bundle-values"
  enabled  = true

  # The keys depend on the app: each one declares its own in settings.json.
  # A user group is written as { id = "..." }. The whole document is written
  # whenever this value changes, so keep every setting here, secrets included.
  settings = jsonencode({
    valueManagers   = [{ id = data.youtrack_group.admins.id }]
    apiToken        = var.youtrack_app_token
    youtrackUrl     = "http://youtrack:8080"
    archiveDirectly = false
  })

  permitted_group_ids = [data.youtrack_group.registered_users.id]
}
