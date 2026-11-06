resource "youtrack_project" "scrum_template" {
  name         = "Scrum template"
  short_name   = "SCRUMTPL"
  leader_login = "john.doe"
  template     = true
}

# Attach a workflow to the template, so every project created from it inherits
# the workflow. Workflows are apps: app_name is the package name listed by
# GET /api/admin/apps, not the title shown in the UI.
resource "youtrack_project_app" "subtasks" {
  project_id = youtrack_project.scrum_template.id
  app_name   = "@jetbrains/youtrack-workflow-subtasks"
  enabled    = true
}
