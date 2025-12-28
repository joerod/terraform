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

resource "scvmm_vm_subnet" "example" {
  name            = "VMSubnet01"
  vm_network_name = "VMNetwork01"
  subnet_vlan_id  = 100
  description     = "Example subnet"
  vm_subnet_id    = 1
  max_ports       = 128
}
