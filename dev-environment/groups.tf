resource "youtrack_nested_group" "youtrack_administrators" {
  name                              = "youtrack-administrators"
  description                       = "Administrator group used to configure Youtrack"
  own_user_logins                   = ["admin"]
  require_two_factor_authentication = false
  auto_join                         = false
}

resource "youtrack_nested_group" "parent-group" {
  name            = "parent-group"
  sub_group_names = ["child-group"]
}

resource "youtrack_nested_group" "child-group" {
  name            = "child-group"
  own_user_logins = ["guest"]
}
