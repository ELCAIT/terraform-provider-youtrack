resource "youtrack_role" "org_reader" {
  name        = "ORG Read"
  description = "Org Read"
  permissions = [
    "Read Article",
    "Read Article Comment",
    "Read Issue",
    "Read Issue Comment",
    "View Voters",
    "View Watchers",
    "Read Organization",
    "Read project basic",
    "Read Report"
  ]
}
