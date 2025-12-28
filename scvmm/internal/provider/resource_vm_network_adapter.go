package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewVMNetworkAdapterResource() resource.Resource {
	return &vmNetworkAdapterResource{}
}

type vmNetworkAdapterResource struct {
	client *psClient
}

type vmNetworkAdapterResourceModel struct {
	VMID                    types.String `tfsdk:"vm_id"`
	VMName                  types.String `tfsdk:"vm_name"`
	Name                    types.String `tfsdk:"name"`
	VMNetworkName           types.String `tfsdk:"vm_network_name"`
	VirtualNetworkName      types.String `tfsdk:"virtual_network_name"`
	VMSubnetName            types.String `tfsdk:"vm_subnet_name"`
	LogicalNetworkName      types.String `tfsdk:"logical_network_name"`
	MACAddress              types.String `tfsdk:"mac_address"`
	MACAddressType          types.String `tfsdk:"mac_address_type"`
	VLANEnabled             types.Bool   `tfsdk:"vlan_enabled"`
	VLANID                  types.Int64  `tfsdk:"vlan_id"`
	Synthetic               types.Bool   `tfsdk:"synthetic"`
	NoConnection            types.Bool   `tfsdk:"no_connection"`
	NetworkLocation         types.String `tfsdk:"network_location"`
	NetworkTag              types.String `tfsdk:"network_tag"`
	EnableMACAddressSpoofing types.Bool  `tfsdk:"enable_mac_address_spoofing"`
	ID                      types.String `tfsdk:"id"`
}

func (r *vmNetworkAdapterResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm_network_adapter"
}

func (r *vmNetworkAdapterResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"vm_id": schema.StringAttribute{
				Optional:    true,
				Description: "Virtual machine ID (GUID). Preferred over vm_name.",
			},
			"vm_name": schema.StringAttribute{
				Optional:    true,
				Description: "Virtual machine name. Required if vm_id is not set.",
			},
			"name": schema.StringAttribute{
				Optional:    true,
				Description: "Adapter name (if supported by host).",
			},
			"vm_network_name": schema.StringAttribute{
				Optional:    true,
				Description: "VM network name to connect.",
			},
			"virtual_network_name": schema.StringAttribute{
				Optional:    true,
				Description: "Virtual network name to connect.",
			},
			"vm_subnet_name": schema.StringAttribute{
				Optional:    true,
				Description: "VM subnet name to connect.",
			},
			"logical_network_name": schema.StringAttribute{
				Optional:    true,
				Description: "Logical network name.",
			},
			"mac_address": schema.StringAttribute{
				Optional:    true,
				Description: "Static MAC address.",
			},
			"mac_address_type": schema.StringAttribute{
				Optional:    true,
				Description: "MAC address type (Static or Dynamic).",
			},
			"vlan_enabled": schema.BoolAttribute{
				Optional:    true,
				Description: "Enable VLAN on the adapter.",
			},
			"vlan_id": schema.Int64Attribute{
				Optional:    true,
				Description: "VLAN ID.",
			},
			"synthetic": schema.BoolAttribute{
				Optional:    true,
				Description: "Use synthetic adapter.",
			},
			"no_connection": schema.BoolAttribute{
				Optional:    true,
				Description: "Create adapter with no connection.",
			},
			"network_location": schema.StringAttribute{
				Optional:    true,
				Description: "Network location label.",
			},
			"network_tag": schema.StringAttribute{
				Optional:    true,
				Description: "Network tag label.",
			},
			"enable_mac_address_spoofing": schema.BoolAttribute{
				Optional:    true,
				Description: "Enable MAC address spoofing.",
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Synthetic ID based on VM and adapter name.",
			},
		},
	}
}

func (r *vmNetworkAdapterResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*psClient)
}

func (r *vmNetworkAdapterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data vmNetworkAdapterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.VMID.IsNull() && data.VMName.IsNull() {
		resp.Diagnostics.AddError("Missing VM reference", "Either vm_id or vm_name must be set.")
		return
	}

	script := buildVMNetworkAdapterCreateScript(data)
	if err := r.client.runPS(ctx, script); err != nil {
		resp.Diagnostics.AddError("PowerShell error", err.Error())
		return
	}

	readVMNetworkAdapter(ctx, r.client, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *vmNetworkAdapterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data vmNetworkAdapterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readVMNetworkAdapter(ctx, r.client, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *vmNetworkAdapterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan vmNetworkAdapterResourceModel
	var state vmNetworkAdapterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.VMID.IsNull() && plan.VMName.IsNull() {
		resp.Diagnostics.AddError("Missing VM reference", "Either vm_id or vm_name must be set.")
		return
	}

	script := buildVMNetworkAdapterUpdateScript(plan, state)
	if script != "" {
		if err := r.client.runPS(ctx, script); err != nil {
			resp.Diagnostics.AddError("PowerShell error", err.Error())
			return
		}
	}

	readVMNetworkAdapter(ctx, r.client, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *vmNetworkAdapterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data vmNetworkAdapterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	script := buildVMNetworkAdapterDeleteScript(data)
	if err := r.client.runPS(ctx, script); err != nil {
		resp.Diagnostics.AddError("PowerShell error", err.Error())
		return
	}
}

func (r *vmNetworkAdapterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func buildVMNetworkAdapterCreateScript(data vmNetworkAdapterResourceModel) string {
	var b strings.Builder
	b.WriteString(buildVMRefScriptVMAdapter(data))

	args := []string{"-VM $vm"}
	if !data.VirtualNetworkName.IsNull() && data.VirtualNetworkName.ValueString() != "" {
		args = append(args, fmt.Sprintf("-VirtualNetwork '%s'", escapeSingleQuotes(data.VirtualNetworkName.ValueString())))
	}
	if !data.VMNetworkName.IsNull() && data.VMNetworkName.ValueString() != "" {
		b.WriteString(fmt.Sprintf("$vmNetwork = Get-SCVMNetwork -Name '%s'; ", escapeSingleQuotes(data.VMNetworkName.ValueString())))
		args = append(args, "-VMNetwork $vmNetwork")
	}
	if !data.VMSubnetName.IsNull() && data.VMSubnetName.ValueString() != "" {
		b.WriteString(fmt.Sprintf("$vmSubnet = Get-SCVMSubnet -Name '%s'; ", escapeSingleQuotes(data.VMSubnetName.ValueString())))
		args = append(args, "-VMSubnet $vmSubnet")
	}
	if !data.LogicalNetworkName.IsNull() && data.LogicalNetworkName.ValueString() != "" {
		b.WriteString(fmt.Sprintf("$logicalNetwork = Get-SCLogicalNetwork -Name '%s'; ", escapeSingleQuotes(data.LogicalNetworkName.ValueString())))
		args = append(args, "-LogicalNetwork $logicalNetwork")
	}
	if !data.MACAddress.IsNull() && data.MACAddress.ValueString() != "" {
		args = append(args, fmt.Sprintf("-MACAddress '%s'", escapeSingleQuotes(data.MACAddress.ValueString())))
	}
	if !data.MACAddressType.IsNull() && data.MACAddressType.ValueString() != "" {
		args = append(args, fmt.Sprintf("-MACAddressType '%s'", escapeSingleQuotes(data.MACAddressType.ValueString())))
	}
	if !data.VLANEnabled.IsNull() {
		args = append(args, fmt.Sprintf("-VLanEnabled $%t", data.VLANEnabled.ValueBool()))
	}
	if !data.VLANID.IsNull() && data.VLANID.ValueInt64() > 0 {
		args = append(args, fmt.Sprintf("-VLanID %d", data.VLANID.ValueInt64()))
	}
	if !data.Synthetic.IsNull() && data.Synthetic.ValueBool() {
		args = append(args, "-Synthetic")
	}
	if !data.NoConnection.IsNull() && data.NoConnection.ValueBool() {
		args = append(args, "-NoConnection")
	}
	if !data.NetworkLocation.IsNull() && data.NetworkLocation.ValueString() != "" {
		args = append(args, fmt.Sprintf("-NetworkLocation '%s'", escapeSingleQuotes(data.NetworkLocation.ValueString())))
	}
	if !data.NetworkTag.IsNull() && data.NetworkTag.ValueString() != "" {
		args = append(args, fmt.Sprintf("-NetworkTag '%s'", escapeSingleQuotes(data.NetworkTag.ValueString())))
	}
	if !data.EnableMACAddressSpoofing.IsNull() {
		args = append(args, fmt.Sprintf("-EnableMACAddressSpoofing $%t", data.EnableMACAddressSpoofing.ValueBool()))
	}

	b.WriteString("New-SCVirtualNetworkAdapter ")
	b.WriteString(strings.Join(args, " "))
	b.WriteString("; ")
	return b.String()
}

func buildVMNetworkAdapterUpdateScript(plan, state vmNetworkAdapterResourceModel) string {
	var b strings.Builder
	b.WriteString(buildVMRefScriptVMAdapter(plan))
	b.WriteString("$adapter = Get-SCVirtualNetworkAdapter -VM $vm | ")
	b.WriteString(fmt.Sprintf("Where-Object { $_.Name -eq '%s' } | Select-Object -First 1; ", escapeSingleQuotes(adapterName(plan))))

	setArgs := []string{"-VirtualNetworkAdapter $adapter"}
	if !plan.MACAddress.IsNull() && plan.MACAddress.ValueString() != state.MACAddress.ValueString() {
		setArgs = append(setArgs, fmt.Sprintf("-MACAddress '%s'", escapeSingleQuotes(plan.MACAddress.ValueString())))
	}
	if !plan.MACAddressType.IsNull() && plan.MACAddressType.ValueString() != state.MACAddressType.ValueString() {
		setArgs = append(setArgs, fmt.Sprintf("-MACAddressType '%s'", escapeSingleQuotes(plan.MACAddressType.ValueString())))
	}
	if !plan.VLANEnabled.IsNull() && plan.VLANEnabled.ValueBool() != state.VLANEnabled.ValueBool() {
		setArgs = append(setArgs, fmt.Sprintf("-VLanEnabled $%t", plan.VLANEnabled.ValueBool()))
	}
	if !plan.VLANID.IsNull() && plan.VLANID.ValueInt64() != state.VLANID.ValueInt64() {
		setArgs = append(setArgs, fmt.Sprintf("-VLanID %d", plan.VLANID.ValueInt64()))
	}
	if !plan.EnableMACAddressSpoofing.IsNull() && plan.EnableMACAddressSpoofing.ValueBool() != state.EnableMACAddressSpoofing.ValueBool() {
		setArgs = append(setArgs, fmt.Sprintf("-EnableMACAddressSpoofing $%t", plan.EnableMACAddressSpoofing.ValueBool()))
	}

	if len(setArgs) > 1 {
		b.WriteString("Set-SCVirtualNetworkAdapter ")
		b.WriteString(strings.Join(setArgs, " "))
		b.WriteString("; ")
	}

	return b.String()
}

func buildVMNetworkAdapterDeleteScript(data vmNetworkAdapterResourceModel) string {
	return buildVMRefScriptVMAdapter(data) + fmt.Sprintf("$adapter = Get-SCVirtualNetworkAdapter -VM $vm | Where-Object { $_.Name -eq '%s' } | Select-Object -First 1; if ($adapter) { Remove-SCVirtualNetworkAdapter -VirtualNetworkAdapter $adapter -Force }", escapeSingleQuotes(adapterName(data)))
}

func readVMNetworkAdapter(ctx context.Context, client *psClient, data *vmNetworkAdapterResourceModel, diags *diag.Diagnostics) {
	script := buildVMRefScriptVMAdapter(data) + fmt.Sprintf("$adapter = Get-SCVirtualNetworkAdapter -VM $vm | Where-Object { $_.Name -eq '%s' } | Select-Object -First 1", escapeSingleQuotes(adapterName(data)))
	result, err := client.runPSJSON(ctx, script+"; $adapter | Select-Object Name, ID")
	if err != nil {
		diags.AddError("PowerShell error", err.Error())
		return
	}

	data.Name = types.StringValue(stringValue(result, "Name"))
	data.ID = types.StringValue(stringValue(result, "ID"))
}

func adapterName(data vmNetworkAdapterResourceModel) string {
	if !data.Name.IsNull() && data.Name.ValueString() != "" {
		return data.Name.ValueString()
	}
	return "Network Adapter"
}

func buildVMRefScriptVMAdapter(data vmNetworkAdapterResourceModel) string {
	if !data.VMID.IsNull() && data.VMID.ValueString() != "" {
		return fmt.Sprintf("$vm = Get-SCVirtualMachine -ID '%s'; ", escapeSingleQuotes(data.VMID.ValueString()))
	}
	return fmt.Sprintf("$vm = Get-SCVirtualMachine -Name '%s'; ", escapeSingleQuotes(data.VMName.ValueString()))
}

var _ resource.Resource = (*vmNetworkAdapterResource)(nil)
