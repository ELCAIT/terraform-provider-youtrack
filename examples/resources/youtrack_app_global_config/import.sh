#!/bin/sh
# Import the configuration of an app using its name, as listed by GET /api/admin/apps.
# settings and permitted_group_ids are not read: the first apply writes the ones
# from the configuration.
terraform import youtrack_app_global_config.agile elca-bundle-values
