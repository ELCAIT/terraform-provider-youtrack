resource "youtrack_locale_settings" "dev" {
  id        = "en_US"
  locale    = "en_US"
  language  = "en"
  community = false
  name      = "English"
}

resource "youtrack_notification_settings" "dev" {
  is_enabled    = "true"
  mail_protocol = "SMTP"
  host          = "mailpit-smtp"
  port          = 1025
  anonymous     = true
  login         = "test@youtrack.example.com"
  from          = "no-reply@youtrack.example.com"
  reply_to      = "no-reply@youtrack.example.com"
}

resource "youtrack_system_settings" "server" {
  administrator_email         = "test@youtrack.com"
  max_export_items            = 500
  max_upload_file_size        = 10485760
  allow_statistics_collection = false
  is_application_read_only    = false
  base_url                    = "http://localhost:8080"
}

resource "youtrack_appearance_settings" "local" {
  date_format_id = "youtrack.datefieldformat.medium_with_24h"
  time_zone_id   = "Europe/Zurich"
}

resource "youtrack_global_settings" "license" {
  license = "42bd2ce4dbaf8554bf082946a1c039d4384e81a99ed53fb75215c1b6ace8d34cf66889fdeee418cbe93d54f38f6e94a8a05a0108234b1955582d566283e3d084b23a0b8b52b016b0e610abcbbe7573fa3ff8fb3cdc9346dc94ef5f596f5b12719abcb02e6fae23d98f53f22a01788066aac7298b7fe79cc3dcd6f8df2fb41ab4"
}

resource "youtrack_rest_settings" "example" {
  allow_all_origins = false
  allowed_origins   = []
}

resource "youtrack_database_backup_settings" "local" {
  location        = "/opt/youtrack/backups"
  files_to_keep   = 7
  cron_expression = "0 0 2 * * ?"
  archive_format  = "TAR_GZ"
  enabled         = false
  notified_users = [
    "admin"
  ]
}

resource "youtrack_global_time_tracking_settings" "local" {
  work_time_settings = {
    minutes_a_day = 492
    work_days     = [1, 2, 3, 4, 5]
  }
  work_item_types = [
    { name = "Development", auto_attached = true },
    { name = "Documentation", auto_attached = true },
    { name = "Implementation", auto_attached = false },
    { name = "Investigation", auto_attached = false },
    { name = "Testing", auto_attached = true }
  ]
}
