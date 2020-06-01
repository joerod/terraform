# Gitlab / Terraform

## Objective

I want to manage my Gitlab projects via Terraform.  I need to import my current projects.

## How to use

I had to import my existing projects before I ran `terraform apply` else Terraform may destroy my projects.  Importing when using a `for_each` in Terraform is a bit different.  See [this](https://www.terraform.io/docs/commands/import.html) at the bottom of the page for more info.

```powershell

terraform import 'gitlab_project.joerod_project[\"joes_project\"]' 15255364
terraform import 'gitlab_project.joerod_project[\"Test\"]' 12794887

```

<<<<<<< HEAD
## Adding members to a Gitlab project
=======
## Adding members to Gitlab group
>>>>>>> b0ffd4a1fa5af0930102b498c19297ff7daffbb9

In the test_project.tf file I provide a list of Gitlab user names in a list.  I then iterate over the list to set developers and maintainers of the Gitlab group.

### Credentials

I used another variable file with this to hold my Gitlab token

```powershell
variable "gitlab_token" {
  description = "Gitlab Token"
  default     = ""
}
```
