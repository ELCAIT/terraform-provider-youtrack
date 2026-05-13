resource "youtrack_issue_link_type" "raised" {
  name             = "Raising"
  source_to_target = "is raised by"
  target_to_source = "raises"
  directed         = true
  aggregation      = false
}
