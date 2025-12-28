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

resource "scvmm_logical_switch" "example" {
  name                    = "LogicalSwitch01"
  description             = "Example logical switch"
  enable_sriov            = true
  enable_packet_direct    = false
  switch_uplink_mode      = "Team"
  minimum_bandwidth_mode  = "Default"
  virtual_switch_extensions = ["Microsoft NDIS Capture", "Extensibility Example"]
  remove_all_extensions   = false
}
