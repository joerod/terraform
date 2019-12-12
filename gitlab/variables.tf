variable "project_name" {
  description = "Project Name"
  default     = "joes_project"
}

variable "dev_users" {
  description = "Developer access"
  type    = list("joerod","otheruser")
}