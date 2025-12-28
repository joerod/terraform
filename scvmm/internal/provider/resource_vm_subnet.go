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

func NewVMSubnetResource() resource.Resource {
	return &vmSubnetResource{}
}

type vmSubnetResource struct {
	client *psClient
}

type vmSubnetResourceModel struct {
	Name          types.String `tfsdk:"name"`
	VMNetworkName types.String `tfsdk:"vm_network_name"`
	SubnetVLanID  types.Int64  `tfsdk:"subnet_vlan_id"`
	Description   types.String `tfsdk:"description"`
	VMSubnetID    types.Int64  `tfsdk:"vm_subnet_id"`
	MaxPorts      types.Int64  `tfsdk:"max_ports"`
	ID            types.String `tfsdk:"id"`
}

func (r *vmSubnetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm_subnet"
}

func (r *vmSubnetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:    true,
				Description: "VM subnet name.",
			},
			"vm_network_name": schema.StringAttribute{
				Required:    true,
				Description: "VM network name.",
			},
			"subnet_vlan_id": schema.Int64Attribute{
				Required:    true,
				Description: "Subnet VLAN ID (used to create SubnetVLan).",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Subnet description.",
			},
			"vm_subnet_id": schema.Int64Attribute{
				Optional:    true,
				Description: "VM subnet ID.",
			},
			"max_ports": schema.Int64Attribute{
				Optional:    true,
				Description: "Max number of ports.",
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "SCVMM object ID.",
			},
		},
	}
}

func (r *vmSubnetResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*psClient)
}

func (r *vmSubnetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data vmSubnetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	script := buildVMSubnetCreateScript(data)
	if err := r.client.runPS(ctx, script); err != nil {
		resp.Diagnostics.AddError("PowerShell error", err.Error())
		return
	}

	readVMSubnet(ctx, r.client, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *vmSubnetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data vmSubnetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readVMSubnet(ctx, r.client, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *vmSubnetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan vmSubnetResourceModel
	var state vmSubnetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	script := buildVMSubnetUpdateScript(plan, state)
	if script != "" {
		if err := r.client.runPS(ctx, script); err != nil {
			resp.Diagnostics.AddError("PowerShell error", err.Error())
			return
		}
	}

	readVMSubnet(ctx, r.client, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *vmSubnetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data vmSubnetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	script := buildVMSubnetDeleteScript(data)
	if err := r.client.runPS(ctx, script); err != nil {
		resp.Diagnostics.AddError("PowerShell error", err.Error())
		return
	}
}

func (r *vmSubnetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}

func buildVMSubnetCreateScript(data vmSubnetResourceModel) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("$vmnet = Get-SCVMNetwork -Name '%s'; ", escapeSingleQuotes(data.VMNetworkName.ValueString())))
	b.WriteString(fmt.Sprintf("$vlan = New-SCSubnetVLan -SubnetVLanID %d; ", data.SubnetVLanID.ValueInt64()))

	args := []string{fmt.Sprintf("-Name '%s'", escapeSingleQuotes(data.Name.ValueString())), "-VMNetwork $vmnet", "-SubnetVLan $vlan"}
	if !data.Description.IsNull() && data.Description.ValueString() != "" {
		args = append(args, fmt.Sprintf("-Description '%s'", escapeSingleQuotes(data.Description.ValueString())))
	}
	if !data.VMSubnetID.IsNull() && data.VMSubnetID.ValueInt64() > 0 {
		args = append(args, fmt.Sprintf("-VMSubnetID %d", data.VMSubnetID.ValueInt64()))
	}
	if !data.MaxPorts.IsNull() && data.MaxPorts.ValueInt64() > 0 {
		args = append(args, fmt.Sprintf("-MaxNumberOfPorts %d", data.MaxPorts.ValueInt64()))
	}

	b.WriteString("New-SCVMSubnet ")
	b.WriteString(strings.Join(args, " "))
	b.WriteString("; ")
	return b.String()
}

func buildVMSubnetUpdateScript(plan, state vmSubnetResourceModel) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("$vmnet = Get-SCVMNetwork -Name '%s'; $subnet = Get-SCVMSubnet -VMNetwork $vmnet | Where-Object { $_.Name -eq '%s' } | Select-Object -First 1; ", escapeSingleQuotes(state.VMNetworkName.ValueString()), escapeSingleQuotes(state.Name.ValueString())))

	setArgs := []string{"-VMSubnet $subnet"}
	if !plan.Description.IsNull() && plan.Description.ValueString() != state.Description.ValueString() {
		setArgs = append(setArgs, fmt.Sprintf("-Description '%s'", escapeSingleQuotes(plan.Description.ValueString())))
	}
	if !plan.VMSubnetID.IsNull() && plan.VMSubnetID.ValueInt64() != state.VMSubnetID.ValueInt64() {
		setArgs = append(setArgs, fmt.Sprintf("-VMSubnetID %d", plan.VMSubnetID.ValueInt64()))
	}
	if !plan.MaxPorts.IsNull() && plan.MaxPorts.ValueInt64() != state.MaxPorts.ValueInt64() {
		setArgs = append(setArgs, fmt.Sprintf("-MaxNumberOfPorts %d", plan.MaxPorts.ValueInt64()))
	}

	if len(setArgs) > 1 {
		b.WriteString("Set-SCVMSubnet ")
		b.WriteString(strings.Join(setArgs, " "))
		b.WriteString("; ")
	}

	return b.String()
}

func buildVMSubnetDeleteScript(data vmSubnetResourceModel) string {
	return fmt.Sprintf("$vmnet = Get-SCVMNetwork -Name '%s'; $subnet = Get-SCVMSubnet -VMNetwork $vmnet | Where-Object { $_.Name -eq '%s' } | Select-Object -First 1; if ($subnet) { Remove-SCVMSubnet -VMSubnet $subnet -Force }", escapeSingleQuotes(data.VMNetworkName.ValueString()), escapeSingleQuotes(data.Name.ValueString()))
}

func readVMSubnet(ctx context.Context, client *psClient, data *vmSubnetResourceModel, diags *diag.Diagnostics) {
	script := fmt.Sprintf("$vmnet = Get-SCVMNetwork -Name '%s'; $subnet = Get-SCVMSubnet -VMNetwork $vmnet | Where-Object { $_.Name -eq '%s' } | Select-Object -First 1", escapeSingleQuotes(data.VMNetworkName.ValueString()), escapeSingleQuotes(data.Name.ValueString()))
	result, err := client.runPSJSON(ctx, script+"; $subnet | Select-Object Name, ID, Description, VMSubnetID, MaxNumberOfPorts")
	if err != nil {
		diags.AddError("PowerShell error", err.Error())
		return
	}

	data.ID = types.StringValue(stringValue(result, "ID"))
	data.Description = types.StringValue(stringValue(result, "Description"))
	data.VMSubnetID = types.Int64Value(int64Value(result, "VMSubnetID"))
	data.MaxPorts = types.Int64Value(int64Value(result, "MaxNumberOfPorts"))
}

var _ resource.Resource = (*vmSubnetResource)(nil)
