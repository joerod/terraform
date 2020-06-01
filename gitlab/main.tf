terraform {
  required_version = ">= 0.12"
}

provider "gitlab" {
  token   = var.gitlab_token
  version = "~> 2.3"
}

// Sets main group
resource "gitlab_group" "group" {
  name        = "JoeRod"
  path        = var.path
  request_access_enabled = true
  visibility_level = "public"
}
