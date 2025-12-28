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

data "scvmm_virtual_machine" "example" {
  name = "demo-vm-01"
}

resource "scvmm_vm_checkpoint" "example" {
  vm_id       = data.scvmm_virtual_machine.example.id
  name        = "pre-update"
  description = "Before monthly update"
}

data "scvmm_vm_checkpoint" "example" {
  vm_name = data.scvmm_virtual_machine.example.name
  name    = "pre-update"
}
