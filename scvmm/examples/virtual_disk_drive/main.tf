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

resource "scvmm_virtual_disk_drive" "example" {
  vm_name   = "demo-vm-01"
  bus_type  = "SCSI"
  bus       = 0
  lun       = 1
  size_gb   = 50
  file_name = "data-disk-01.vhdx"
  path      = "D:\\VMs\\data"
  fixed     = false
  dynamic   = true
  move_path = "E:\\VMs\\data"
}
