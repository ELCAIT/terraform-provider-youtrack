variable "exchange_client_secret" {
  type      = string
  sensitive = true
}

resource "youtrack_mailbox" "helpdesk" {
  protocol                  = "OFFICE365"
  microsoft_mailbox_address = "helpdesk@example.com"
  microsoft_client_id       = "00000000-0000-0000-0000-000000000000"
  microsoft_tenant_id       = "11111111-1111-1111-1111-111111111111"
  microsoft_client_secret   = var.exchange_client_secret
}

data "youtrack_project" "demo" {
  short_name = "DEMO"
}

resource "youtrack_mail_rule" "support" {
  mailbox_id       = youtrack_mailbox.helpdesk.id
  project_id       = data.youtrack_project.demo.id
  maintainer_login = "admin"
  to               = "helpdesk@example.com"
  folder           = "Inbox"
  spam_filter_mode = "BLOCK_TICKETS"

}
