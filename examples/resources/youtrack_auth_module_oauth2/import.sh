# Import an OAuth 2.0 auth module by its Hub ID.
# The ID can be found in the Hub admin UI or via the Hub REST API:
#   GET <hub-url>/hub/api/rest/authmodules?fields=id,name&query=name:MyModuleName
terraform import youtrack_auth_module_oauth2.<RESOURCE_NAME> <AUTH_MODULE_ID>
