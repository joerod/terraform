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

resource "scvmm_vm_network_adapter" "example" {
  vm_id               = data.scvmm_virtual_machine.example.id
  name                = "Network Adapter"
  vm_network_name     = "VMNetwork01"
  vm_subnet_name      = "VMSubnet01"
  mac_address_type    = "Dynamic"
  vlan_enabled        = true
  vlan_id             = 100
  synthetic           = true
  no_connection       = false
  network_location    = "DatacenterA"
  network_tag         = "frontend"
  enable_mac_address_spoofing = false
}
