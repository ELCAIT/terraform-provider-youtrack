resource "youtrack_project" "test" {
  name         = "My Awesome Project"
  short_name   = "MYP"
  description  = "A project managed by Terraform"
  leader_login = "admin"
  archived     = false
  template     = false
}

resource "youtrack_enum_bundle" "severity_tpl_values" {
  name = "Severity Template Values"

  values = [
    {
      name = "High"
    },
    {
      name = "Medium"
    },
    {
      name = "Low"
    },
  ]
}

resource "youtrack_project_custom_field" "severity" {
  project_id  = youtrack_project.test.id
  field_name  = youtrack_custom_field.severity.name
  field_type  = "EnumProjectCustomField"
  bundle_name = youtrack_enum_bundle.severity_tpl_values.name
}

resource "youtrack_project_custom_field" "spent_time" {
  project_id = youtrack_project.test.id
  field_name = "Spent time"
  field_type = "PeriodProjectCustomField"
}

resource "youtrack_project_custom_field" "estimation" {
  project_id = youtrack_project.test.id
  field_name = "Estimation"
  field_type = "PeriodProjectCustomField"
}

resource "youtrack_project_time_tracking_settings" "example" {
  project_id            = youtrack_project.test.id
  enabled               = true
  time_spent_field_name = youtrack_project_custom_field.spent_time.field_name
  estimate_field_name   = youtrack_project_custom_field.estimation.field_name
}
