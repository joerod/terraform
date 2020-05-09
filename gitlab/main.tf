provider "gitlab" {
  token   = var.gitlab_token
  version = "~> 2.3"
}

// Sets main group
resource "gitlab_group" "joerod" {
  name        = "JoeRod"
  path        = "joerod2"
  request_access_enabled = true
}

// Sets projects and settings
resource "gitlab_project" "joerod_projects" {
  for_each     = toset(var.project_name)
  name         = each.value
  namespace_id = gitlab_group.joerod.id
  approvals_before_merge  = 1
  only_allow_merge_if_all_discussions_are_resolved = true
  issues_enabled = true
  visibility_level = "public"
  initialize_with_readme = true
}

