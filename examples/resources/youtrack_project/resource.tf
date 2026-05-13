resource "youtrack_project" "example" {
  name         = "My Project"
  short_name   = "MP"
  description  = "A project managed by Terraform"
  leader_login = "john.doe"
}
