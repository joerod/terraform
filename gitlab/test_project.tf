// Configure sub-project settings

resource "gitlab_project" "test_project" {
  name        = "test2"
}
data "gitlab_user" "test_user" {
  username = "tom_cxdr"
}
resource "gitlab_project_membership" "test" {
  project_id   = gitlab_project.test_project.id
  user_id      = data.gitlab_user.test_user.id
  access_level = "developer"
}