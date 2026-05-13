resource "youtrack_role_assignment" "registered_users" {
  role_id      = youtrack_role.default_observer.id
  holder_login = "Registered Users"
  holder_type  = "group"
}

resource "youtrack_role" "default_observer" {
  description = "Default observer role"
  name        = "Default observer"
  permissions = [
    "Read User Basic"
  ]
}

resource "youtrack_role" "default_read" {
  description = "Default read role"
  name        = "Default Read"
  permissions = [
    "Read Article",
    "Read Article Comment",
    "Read Issue",
    "Read Issue Comment",
    "View Voters",
    "View Watchers",
    "Read Organization",
    "Read Project Basic"
  ]
}

resource "youtrack_role" "helpdesk_read" {
  name = "Helpdesk user"
  permissions = [
    "Read Issue",
    "Read Issue Comment",
    "Create Issue Comment",
    "Read Project Basic"
  ]
}
