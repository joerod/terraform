terraform {
  required_providers {
    scvmm = {
      source  = "github.com/example/scvmm"
      version = "0.1.0"
    }
  }
}

provider "scvmm" {
  server = "vmm01.contoso.local"
}

resource "scvmm_virtual_machine" "example" {
  name          = "demo-vm-01"
  template_name = "Windows-2025-Template"
  cloud_name    = "DevCloud"
  host_group    = "All Hosts"
  cpu_count     = 2
  memory_mb     = 4096
  power_state   = "Running"
  highly_available = true
  hardware_profile_name = "HWProfile01"
  guest_os_profile_name = "Windows2025-GuestProfile"
}

data "scvmm_virtual_machine" "example" {
  name = scvmm_virtual_machine.example.name
}

resource "scvmm_host_cluster" "example" {
  name          = "Cluster01.contoso.local"
  vm_host_group = "All Hosts"
  run_as_account = "Contoso\\vmm-runas"
  host_nodes    = ["hyperv01.contoso.local", "hyperv02.contoso.local"]
  remove_missing_nodes = true
}

resource "scvmm_logical_switch" "example" {
  name                  = "LogicalSwitch01"
  minimum_bandwidth_mode = "Default"
  virtual_switch_extensions = ["Microsoft NDIS Capture", "Extensibility Example"]
  remove_all_extensions = false
}

resource "scvmm_virtual_network" "example" {
  name         = "VMNetwork01"
  vm_host_name = "hyperv01.contoso.local"
  vlan_id      = 100
  bound_to_vm_host = true
}

resource "scvmm_virtual_disk_drive" "data_disk" {
  vm_name   = scvmm_virtual_machine.example.name
  bus_type  = "SCSI"
  bus       = 0
  lun       = 1
  size_gb   = 50
  file_name = "data-disk-01.vhdx"
  dynamic   = true
  move_path = "D:\\VMs\\data"
}

resource "scvmm_vm_checkpoint" "example" {
  vm_name     = scvmm_virtual_machine.example.name
  name        = "pre-update"
  description = "Before monthly update"
}
