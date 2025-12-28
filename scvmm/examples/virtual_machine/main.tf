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

resource "scvmm_virtual_machine" "example" {
  name                  = "demo-vm-01"
  template_name         = "Windows-2025-Template"
  # template_id         = "00000000-0000-0000-0000-000000000000" # use either template_name or template_id
  cloud_name            = "DevCloud"
  host_group            = "All Hosts"
  owner                 = "CONTOSO\\administrator"
  cpu_count             = 2
  memory_mb             = 4096
  description           = "Example VM"
  power_state           = "Running"
  highly_available      = true
  hardware_profile_name = "HWProfile01"
  guest_os_profile_name = "Windows2025-GuestProfile"
}
