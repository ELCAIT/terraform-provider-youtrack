variable "support_mailbox_password" {
  type      = string
  sensitive = true
}

variable "exchange_client_secret" {
  type      = string
  sensitive = true
}

# An IMAP mailbox.
resource "youtrack_mailbox" "support" {
  protocol = "IMAPS"
  host     = "imap.example.com"
  port     = 993
  login    = "support@example.com"
  password = var.support_mailbox_password
}

# A Microsoft Exchange Online mailbox, read through a Microsoft Entra ID app.
resource "youtrack_mailbox" "helpdesk" {
  protocol                  = "OFFICE365"
  microsoft_mailbox_address = "helpdesk@example.com"
  microsoft_client_id       = "00000000-0000-0000-0000-000000000000"
  microsoft_tenant_id       = "11111111-1111-1111-1111-111111111111"
  microsoft_client_secret   = var.exchange_client_secret
}
