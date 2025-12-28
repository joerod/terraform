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

resource "scvmm_vm_host" "example" {
  computer_name   = "hyperv01.contoso.local"
  host_group_name = "All Hosts\\Compute"
  remove_on_delete = false
}

data "scvmm_vm_host" "example" {
  computer_name = "hyperv01.contoso.local"
}
