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

# Create a role
resource "youtrack_role" "developer" {
  name        = "Developer"
  description = "Developer role"
  permissions = ["Read Issue", "Create Issue"]
}

# Assign the role to a user with global scope
resource "youtrack_role_assignment" "example" {
  role_id      = youtrack_role.developer.id
  holder_login = "john.doe" # Username or group name
  holder_type  = "user"     # Optional: 'user' or 'group'
}
