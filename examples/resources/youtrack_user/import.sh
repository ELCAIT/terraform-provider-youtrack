# The ID is the YouTrack user ID (for example "1-42"), not the login.
# The password is never read back, so the first apply after an import sets it.
terraform import youtrack_user.ci <USER_ID>
