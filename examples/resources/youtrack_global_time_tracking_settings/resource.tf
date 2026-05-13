# Configure global time tracking settings with a standard 8-hour work day (Mon–Fri)
# and explicitly manage the list of work item types.
resource "youtrack_global_time_tracking_settings" "global" {
  work_time_settings = {
    # 8 hours × 60 minutes = 480 minutes per working day
    minutes_a_day = 480
    # 1 = Monday, 2 = Tuesday, 3 = Wednesday, 4 = Thursday, 5 = Friday
    # 0 = Sunday, 6 = Saturday
    work_days = [1, 2, 3, 4, 5]
  }

  # When set, this list is the source of truth for work item types:
  # types declared here are created/updated, types absent here are deleted.
  # Omit this block entirely to leave work item types unmanaged (read-only).
  work_item_types = [
    { name = "Development", auto_attached = true },
    { name = "Testing", auto_attached = true },
    { name = "Documentation", auto_attached = false },
  ]
}
