terraform {
  required_providers {
    youtrack = {
      source = "registry.opentofu.org/elcait/youtrack"
    }
  }
}

provider "youtrack" {
  base_url = "https://youtrack.example.com"
  token    = "your-api-token"
}

# A template project: every project created from it inherits its role assignments
resource "youtrack_project" "template" {
  name         = "Project template"
  short_name   = "TPL"
  leader_login = "admin"
  template     = true
}

resource "youtrack_role" "reporter" {
  name        = "Issue reporter"
  permissions = ["Read Issue", "Create Issue"]
}

# Assign the role to a user within the project only
resource "youtrack_project_role_assignment" "user" {
  role_id      = youtrack_role.reporter.id
  project_id   = youtrack_project.template.id
  holder_login = "john.doe"
  holder_type  = "user" # Optional: 'user' or 'group'
}

# Assign the role to a group within the project only
resource "youtrack_project_role_assignment" "group" {
  role_id      = youtrack_role.reporter.id
  project_id   = youtrack_project.template.id
  holder_login = "my-group" # Group name
  holder_type  = "group"
}
