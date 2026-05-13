resource "youtrack_database_backup_settings" "example" {
  location        = "/opt/youtrack/backups"
  files_to_keep   = 7
  cron_expression = "0 0 2 * * ?"
  archive_format  = "ZIP"
  enabled         = true
  notified_users = [
    "admin",
    "ops.user"
  ]
}
