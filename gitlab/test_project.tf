// Sets projects and settings
resource "gitlab_project" "test_project" {
  name         = "test"
  namespace_id = gitlab_group.joerod.id
  approvals_before_merge  = 1
  only_allow_merge_if_all_discussions_are_resolved = true
  issues_enabled = true
  visibility_level = "public"
  initialize_with_readme = true
}

data "gitlab_user" "test_user" {
  username = "dwrfg4wgtgh"
}

resource "gitlab_project_membership" "test" {
  project_id   = gitlab_project.test_project.id
  user_id      = data.gitlab_user.test_user.id
  access_level = "developer"
}
