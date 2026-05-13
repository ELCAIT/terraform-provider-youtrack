resource "youtrack_state_bundle" "workflow" {
  name = "Workflow States"

  values = [
    {
      name        = "Open"
      is_resolved = false
    },
    {
      name        = "In Progress"
      is_resolved = false
    },
    {
      name        = "Done"
      is_resolved = true
    },
  ]
}
