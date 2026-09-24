resource "youtrack_enum_bundle" "severity_values" {
  name = "Severity Values"

  values = [
    {
      name = "Critical"
    },
    {
      name = "Major"
    },
    {
      name = "Minor"
    },
  ]
}

resource "youtrack_custom_field" "severity" {
  name = "Severity Test"
  # YouTrack field type ID: enum[1] means enum with single-value cardinality.
  field_type_id = "enum[1]"

  is_auto_attached           = false
  is_displayed_in_issue_list = true

  field_defaults = {
    can_be_empty        = true
    empty_field_text    = ""
    is_public           = true
    bundle_name         = youtrack_enum_bundle.severity_values.name
    default_value_names = ["Major"]
  }
}

resource "youtrack_owned_bundle" "components" {
  name = "Components"

  values = [
    {
      name        = "Backend"
      description = "APIs and background jobs"
      owner_login = "admin"
    },
    {
      name        = "Frontend"
      owner_login = "admin"
    },
    {
      name = "Infrastructure"
    },
  ]
}

resource "youtrack_custom_field" "component" {
  name          = "Component"
  field_type_id = "ownedField[1]"

  field_defaults = {
    can_be_empty = true
    bundle_name  = youtrack_owned_bundle.components.name
  }
}
