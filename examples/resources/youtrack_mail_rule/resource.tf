variable "support_mailbox_password" {
  type      = string
  sensitive = true
}

# The project already exists in YouTrack: look it up rather than create it.
data "youtrack_project" "support" {
  short_name = "SUP"
}

resource "youtrack_mailbox" "support" {
  protocol = "IMAPS"
  host     = "imap.example.com"
  port     = 993
  login    = "support@example.com"
  password = var.support_mailbox_password
}

# Turn the emails sent to support@example.com into issues of the Support project,
# with the sender as reporter, ignoring newsletters.
resource "youtrack_mail_rule" "support" {
  mailbox_id        = youtrack_mailbox.support.id
  project_id        = data.youtrack_project.support.id
  maintainer_login  = "john.doe"
  to                = "support@example.com"
  spam_filter_mode  = "BLOCK_TICKETS"
  new_issue_command = "Type Support"

  filters = [
    {
      scope          = "SUBJECT"
      patterns       = ["newsletter", "unsubscribe"]
      accept_matched = false
    },
  ]
}
