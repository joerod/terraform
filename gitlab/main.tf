provider "gitlab" {
  token   = var.gitlab_token
  version = "~> 2.3"
}

// Sets main group
resource "gitlab_group" "joerod" {
  name        = "JoeRod"
  path        = "joerod2"
  request_access_enabled = true
  visibility_level = "public"
}
