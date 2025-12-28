terraform {
  required_providers {
    scvmm = {
      source  = "github.com/joerod/scvmm"
      version = "0.1.0"
    }
  }
}

provider "scvmm" {
  server = "vmm01.contoso.local"
}

resource "scvmm_vm_checkpoint" "example" {
  vm_name     = "demo-vm-01"
  name        = "pre-update"
  description = "Before monthly update"
}
