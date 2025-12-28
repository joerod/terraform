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

resource "scvmm_host_cluster" "example" {
  name                 = "Cluster01.contoso.local"
  vm_host_group        = "All Hosts"
  run_as_account       = "Contoso\\vmm-runas"
  description          = "Example host cluster"
  cluster_reserve      = 20
  remote_connect_enabled = true
  remote_connect_port  = 5900
  enable_live_migration = true
  host_nodes           = ["hyperv01.contoso.local", "hyperv02.contoso.local"]
  remove_missing_nodes = false
}
