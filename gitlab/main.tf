provider "gitlab" {
  token   = "${var.gitlab_token}"
  version = "~> 2.3"
}

resource "gitlab_project" "${var.project_name}" {
  name =  "${var.project_name}"
  only_allow_merge_if_all_discussions_are_resolved = true
  issues_enabled = true
  visibility_level = "public"
  initialize_with_readme = true
  approvals_before_merge = 2
  only_allow_merge_if_all_discussions_are_resolved = true
}

resource "gitlab_project_push_rules" "${var.project_name}" {
  commit_committer_check = true
}

resource "gitlab_project_membership" "${var.project_name}" {
  for_each = toset(var.project_devs)
  project_id   = gitlab_project.project.id
  user_id      = "${each.value}"
  access_level = "Developer"
}

resource "gitlab_project_membership" "${var.project_name}" {
  for_each = toset(var.project_maint)
  project_id   = gitlab_project.project.id
  user_id      = "${each.value}"
  access_level = "Maintainer"
}

resource "gitlab_project_membership" "${var.project_name}" {
  for_each = toset(var.project_own)
  project_id   = gitlab_project.project.id
  user_id      = "${each.value}"
  access_level = "Owner"
}

resource "gitlab_project_hook" "jenkins_hook" {
  project               = "example/hooked"
  url                   = "https://example.com/hook/example"
  merge_requests_events = true
}

