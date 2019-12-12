
data "gitlab_user" "example" {
  username = "myuser"
}

output "Developer" {
  value = ["${var.dev_users}"]
}
