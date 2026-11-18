resource "youtrack_enum_bundle" "priority" {
  name = "Priority"

  values = [
    {
      name     = "Critical"
      color_id = "20"
    },
    {
      name     = "Major"
      color_id = "10"
    },
    {
      name = "Normal"
    },
    {
      name = "Minor"
    },
  ]
}
