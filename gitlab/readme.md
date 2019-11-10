# How to use

I had to import my existing projects before I ran `terraform apply` else Terraform may destroy my project

```powershell

terraform import 'gitlab_project.sample_project[\"joes_project\"]' 15255364
terraform import 'gitlab_project.sample_project[\"Test\"]' 12794887

```
