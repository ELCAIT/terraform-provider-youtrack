#!/bin/sh
# Import an app attachment using the composite ID: {project_id}/{app_name}
# The app name may itself contain a slash, as in this example.
terraform import youtrack_project_app.subtasks <PROJECT_ID>/@jetbrains/youtrack-workflow-subtasks
