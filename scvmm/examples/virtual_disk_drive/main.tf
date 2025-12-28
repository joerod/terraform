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

resource "scvmm_virtual_disk_drive" "example" {
  vm_id     = data.scvmm_virtual_machine.example.id
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
