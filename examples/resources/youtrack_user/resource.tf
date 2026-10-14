# A service account used by an integration.
resource "youtrack_user" "ci" {
  login     = "svc-ci"
  full_name = "CI Service Account"
  email     = "svc-ci@example.com"
  password  = var.ci_password
}

variable "ci_password" {
  type      = string
  sensitive = true
}
