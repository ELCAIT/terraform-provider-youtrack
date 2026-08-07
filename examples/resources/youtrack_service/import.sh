# Import a Hub service by its ID.
# The ID can be found in the Hub admin UI or via the Hub REST API:
#   GET <hub-url>/hub/api/rest/services?fields=id,name&query=name:MyServiceName
# Note: the service's secret cannot be recovered via the API and is set to
# an empty string on import; re-apply to set a new one if needed.
terraform import youtrack_service.<RESOURCE_NAME> <SERVICE_ID>
