// Sets projects and settings
resource "gitlab_project" "test_project" {
  name         = var.name_test_project
  namespace_id = gitlab_group.joerod.id
  approvals_before_merge  = 1
  only_allow_merge_if_all_discussions_are_resolved = true
  issues_enabled = true
  visibility_level = "public"
  initialize_with_readme = true
}

data "gitlab_user" "test_developer" {
  count = length(var.test_developer)
  username = var.test_developer[count.index]
}

// sets users who will be developers for the project
resource "gitlab_project_membership" "test_developer" {
  count = length(data.gitlab_user.test_developer)
  project_id   = gitlab_project.test_project.id
  user_id      = data.gitlab_user.test_developer[count.index].id
  access_level = "developer"
}

data "gitlab_user" "test_maintainer" {
  count = length(var.test_maintainer)
  username = var.test_maintainer[count.index]
}

// sets users who will be maintainers for the project
resource "gitlab_project_membership" "test_maintainer" {
  count = length(data.gitlab_user.test_maintainer)
  project_id   = gitlab_project.test_project.id
  user_id      = data.gitlab_user.test_maintainer[count.index].id
  access_level = "maintainer"
}