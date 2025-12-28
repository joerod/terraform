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

data "scvmm_vm_host" "example" {
  computer_name = "hyperv01.contoso.local"
}

resource "scvmm_virtual_network" "example" {
  name             = "VMNetwork01"
  vm_host_name     = data.scvmm_vm_host.example.computer_name
  description      = "Example virtual network"
  vlan_id          = 100
  bound_to_vm_host = true
}
