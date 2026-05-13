resource "youtrack_enum_bundle" "priority" {
  name = "Priority"

  values = [
    {
      name = "Critical"
    },
    {
      name = "Major"
    },
    {
      name = "Normal"
    },
    {
      name = "Minor"
    },
  ]
}
