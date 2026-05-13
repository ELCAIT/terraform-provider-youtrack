resource "youtrack_notification_settings" "example" {
  is_enabled    = "true"
  mail_protocol = "SMTP"
  host          = "mailpit-smtp"
  port          = 1025
  anonymous     = true
  login         = "test@youtrack.com"
  from          = "no-reply@youtrack.com"
  reply_to      = "no-reply@youtrack.com"
}
