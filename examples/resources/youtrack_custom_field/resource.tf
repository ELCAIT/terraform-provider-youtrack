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
  name = "Severity"
  # YouTrack field type ID: enum[1] means enum with single-value cardinality.
  field_type_id = "enum[1]"

  is_auto_attached           = false
  is_displayed_in_issue_list = true

  field_defaults = {
    can_be_empty     = true
    empty_field_text = "No severity"
    is_public        = true
    # Only one default bundle reference is supported for this custom field.
    bundle_id = youtrack_enum_bundle.severity_values.id
  }
}
