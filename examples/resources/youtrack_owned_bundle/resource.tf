resource "youtrack_owned_bundle" "subsystems" {
  name = "Subsystems"

  values = [
    {
      name        = "Backend"
      description = "APIs and background jobs"
      owner_login = "jane.doe"
    },
    {
      name        = "Frontend"
      owner_login = "john.smith"
    },
    {
      name = "Infrastructure"
    },
  ]
}

resource "youtrack_custom_field" "subsystem" {
  name          = "Subsystem"
  field_type_id = "ownedField[1]"

  field_defaults = {
    can_be_empty = true
    bundle_name  = youtrack_owned_bundle.subsystems.name
  }
}
