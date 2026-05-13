# Each project has exactly one time tracking settings object - this resource configures it.
# Destroying this resource disables time tracking for the project (the settings object is not deleted).
#
# estimate_field_name and time_spent_field_name refer to the names of custom fields already attached
# to the project via youtrack_project_custom_field.

resource "youtrack_project" "parent" {
  name         = "My Project"
  short_name   = "MP"
  leader_login = "john.doe"
}

# Attach the "Spent time" field first so it can be referenced in time tracking settings
resource "youtrack_project_custom_field" "spent_time" {
  project_id = youtrack_project.parent.id
  field_name = "Spent time"
  field_type = "PeriodProjectCustomField"
}

resource "youtrack_project_time_tracking_settings" "example" {
  project_id            = youtrack_project.parent.id
  enabled               = true
  time_spent_field_name = youtrack_project_custom_field.spent_time.field_name
}
