resource "youtrack_project" "parent" {
  name         = "My Project"
  short_name   = "MP"
  leader_login = "john.doe"
}

# Attach the global "Priority" custom field to the project, using a project-specific bundle
resource "youtrack_project_custom_field" "priority" {
  project_id   = youtrack_project.parent.id
  field_name   = "Priority"
  field_type   = "EnumProjectCustomField"
  bundle_name  = "My Project Priorities"
  can_be_empty = true
  is_public    = true
}
