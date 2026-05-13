resource "youtrack_nested_group" "devops" {
  name                              = "DevOps"
  description                       = "DevOps team"
  own_user_logins                   = ["alice", "bob"]
  sub_group_names                   = ["Platform", "SRE"]
  require_two_factor_authentication = true
  viewers                           = ["security-team"]
  updaters                          = ["platform-admins"]
  auto_join                         = true
  auto_join_domain                  = "example.com"
}
