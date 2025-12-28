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

data "scvmm_vm_host" "hypervisor" {
  computer_name = "hyperv01.contoso.local"
}

data "scvmm_host_cluster" "cluster" {
  name = "Cluster01.contoso.local"
}

data "scvmm_vm_host" "cluster_node" {
  computer_name = data.scvmm_host_cluster.cluster.host_names[0]
}

resource "scvmm_virtual_machine" "on_hypervisor" {
  name                  = "demo-vm-01"
  template_name         = "Windows-2025-Template"
  # template_id         = "00000000-0000-0000-0000-000000000000" # use either template_name or template_id
  cloud_name            = "DevCloud"
  host_group            = "All Hosts"
  vm_host_name          = data.scvmm_vm_host.hypervisor.computer_name
  owner                 = "CONTOSO\\administrator"
  cpu_count             = 2
  memory_mb             = 4096
  description           = "Example VM on a specific hypervisor"
  power_state           = "Running"
  hardware_profile_name = "HWProfile01"
  guest_os_profile_name = "Windows2025-GuestProfile"
}

resource "scvmm_virtual_machine" "on_cluster" {
  name                  = "demo-vm-02"
  template_name         = "Windows-2025-Template"
  host_group            = "All Hosts"
  vm_host_name          = data.scvmm_vm_host.cluster_node.computer_name
  highly_available      = true
  cpu_count             = 2
  memory_mb             = 4096
  description           = "Example VM on a cluster node"
}
