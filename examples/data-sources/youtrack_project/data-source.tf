# Look up a project that already exists in YouTrack and is not managed by Terraform.
data "youtrack_project" "support" {
  short_name = "SUP"
}

output "support_project_id" {
  value = data.youtrack_project.support.id
}
