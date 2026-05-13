resource "youtrack_issue_link_type" "depends" {
  name             = "Depend"
  source_to_target = "is required for"
  target_to_source = "depends on"
  directed         = true
  aggregation      = false
}
