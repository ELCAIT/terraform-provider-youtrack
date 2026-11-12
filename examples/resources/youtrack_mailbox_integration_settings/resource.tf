# Poll every mailbox every five minutes.
resource "youtrack_mailbox_integration_settings" "example" {
  cron_expression = "0 0/5 * * * ?"
}
