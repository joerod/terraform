# Terraform Provider: SCVMM (PowerShell)

This provider shells out to SCVMM PowerShell cmdlets (System Center Virtual Machine Manager 2025) using Windows Integrated Authentication (AD). It is Windows-only and requires the SCVMM PowerShell module installed on the host running Terraform.

## Requirements

- Windows host running Terraform
- SCVMM PowerShell module available in `powershell.exe`
- AD credentials for the SCVMM server (current user session)

## Provider configuration

```hcl
provider "scvmm" {
  server = "vmm01.contoso.local"
}
```

If `server` is not set, PowerShell uses the default SCVMM context.

## Resources

### `scvmm_virtual_machine`

Create, update, and delete a VM. The resource uses:

- `New-SCVirtualMachine`
- `Set-SCVirtualMachine`
- `Remove-SCVirtualMachine`
- `Start-SCVirtualMachine` / `Stop-SCVirtualMachine`

Set `highly_available = true` to mark VMs as highly available when running on a cluster.
Use `hardware_profile_name` or `guest_os_profile_name` to select profiles during VM creation or update.

Required/optional fields depend on your SCVMM environment and template requirements.

### `scvmm_host_cluster`

Adds or updates a host cluster in SCVMM using `Add-SCVMHostCluster` and `Set-SCVMHostCluster`. Optional `host_nodes` will add nodes with `Add-SCVMHost`. Use `remove_missing_nodes` to remove cluster nodes not listed.

### `scvmm_virtual_network`

Creates a host-bound virtual network with `New-SCVirtualNetwork` and updates via `Set-SCVirtualNetwork`.

### `scvmm_logical_switch`

Creates a logical switch with `New-SCLogicalSwitch` and updates via `Set-SCLogicalSwitch`. You can attach extensions by name or remove all extensions.

### `scvmm_virtual_disk_drive`

Adds/removes a virtual disk drive and expands its size using `New-SCVirtualDiskDrive` and `Expand-SCVirtualDiskDrive`. Use `move_path` to move the backing VHD with `Move-SCVirtualHardDisk`.

## Data sources

### `scvmm_virtual_machine`

Looks up a VM by name using `Get-SCVirtualMachine`.

## Limitations

- Windows-only (PowerShell cmdlets are required)
- No validation for template/placement rules; errors come from SCVMM
- Resource schemas are intentionally minimal and will evolve

## Development

```bash
go mod tidy
go build
```

## Example

See `examples/basic`.
