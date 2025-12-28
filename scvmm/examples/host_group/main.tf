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

resource "scvmm_host_group" "example" {
  name                           = "Compute"
  parent_host_group_name         = "All Hosts"
  description                    = "Compute hosts"
  enable_unencrypted_file_transfer = false
  inherit_network_settings        = true
}
