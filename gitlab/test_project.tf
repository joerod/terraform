// Sets projects and settings
resource "gitlab_project" "project" {
  name         = var.name_project
  namespace_id = gitlab_group.group.id
  approvals_before_merge  = 1
  only_allow_merge_if_all_discussions_are_resolved = true
  issues_enabled = true
  visibility_level = "public"
  initialize_with_readme = true
}

data "gitlab_user" "developer" {
  count = length(var.developer)
  username = var.developer[count.index]
}

// sets users who will be developers for the project
resource "gitlab_project_membership" "developer" {
  count = length(data.gitlab_user.developer)
  project_id   = gitlab_project.project.id
  user_id      = data.gitlab_user.developer[count.index].id
  access_level = "developer"
}

data "gitlab_user" "maintainer" {
  count = length(var.maintainer)
  username = var.maintainer[count.index]
}

// sets users who will be maintainers for the project
resource "gitlab_project_membership" "maintainer" {
  count = length(data.gitlab_user.maintainer)
  project_id   = gitlab_project.project.id
  user_id      = data.gitlab_user.maintainer[count.index].id
  access_level = "maintainer"
}