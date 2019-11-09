provider "gitlab" {
  token   = "${var.gitlab_token}"
  version = "~> 2.3"
}


resource "gitlab_project" "sample_project" {
  for_each = toset(var.project_name)
  name =  "${each.value}"
  approvals_before_merge  = 1
  only_allow_merge_if_all_discussions_are_resolved = true
  issues_enabled = true
  visibility_level = "public"
  initialize_with_readme = true
}