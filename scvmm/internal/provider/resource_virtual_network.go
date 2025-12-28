package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewVirtualNetworkResource() resource.Resource {
	return &virtualNetworkResource{}
}

type virtualNetworkResource struct {
	client *psClient
}

type virtualNetworkResourceModel struct {
	Name          types.String `tfsdk:"name"`
	VMHostName    types.String `tfsdk:"vm_host_name"`
	Description   types.String `tfsdk:"description"`
	VLANID        types.Int64  `tfsdk:"vlan_id"`
	BoundToVMHost types.Bool   `tfsdk:"bound_to_vm_host"`
	ID            types.String `tfsdk:"id"`
}

func (r *virtualNetworkResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_virtual_network"
}

func (r *virtualNetworkResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Virtual network name.",
			},
			"vm_host_name": schema.StringAttribute{
				Required:    true,
				Description: "VM host where the virtual network is created.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Virtual network description.",
			},
			"vlan_id": schema.Int64Attribute{
				Optional:    true,
				Description: "Host-bound VLAN ID.",
			},
			"bound_to_vm_host": schema.BoolAttribute{
				Optional:    true,
				Description: "Whether the virtual network is bound to the host.",
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "SCVMM object ID.",
			},
		},
	}
}

func (r *virtualNetworkResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*psClient)
}

func (r *virtualNetworkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data virtualNetworkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	script := buildVirtualNetworkCreateScript(data)
	if err := r.client.runPS(ctx, script); err != nil {
		resp.Diagnostics.AddError("PowerShell error", err.Error())
		return
	}

	readVirtualNetwork(ctx, r.client, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *virtualNetworkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data virtualNetworkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readVirtualNetwork(ctx, r.client, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *virtualNetworkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan virtualNetworkResourceModel
	var state virtualNetworkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	script := buildVirtualNetworkUpdateScript(plan, state)
	if script != "" {
		if err := r.client.runPS(ctx, script); err != nil {
			resp.Diagnostics.AddError("PowerShell error", err.Error())
			return
		}
	}

	readVirtualNetwork(ctx, r.client, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *virtualNetworkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data virtualNetworkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	script := fmt.Sprintf("$host = Get-SCVMHost -Name '%s'; $vnet = Get-SCVirtualNetwork -VMHost $host | Where-Object { $_.Name -eq '%s' } | Select-Object -First 1; if ($vnet) { Remove-SCVirtualNetwork -VirtualNetwork $vnet -Force }", escapeSingleQuotes(data.VMHostName.ValueString()), escapeSingleQuotes(data.Name.ValueString()))
	if err := r.client.runPS(ctx, script); err != nil {
		resp.Diagnostics.AddError("PowerShell error", err.Error())
		return
	}
}

func (r *virtualNetworkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, req, resp, "name")
}

func readVirtualNetwork(ctx context.Context, client *psClient, data *virtualNetworkResourceModel, diags *resource.Diagnostics) {
	script := fmt.Sprintf("$host = Get-SCVMHost -Name '%s'; $vnet = Get-SCVirtualNetwork -VMHost $host | Where-Object { $_.Name -eq '%s' } | Select-Object -First 1", escapeSingleQuotes(data.VMHostName.ValueString()), escapeSingleQuotes(data.Name.ValueString()))
	result, err := client.runPSJSON(ctx, script+"; $vnet | Select-Object Name, ID, Description, HostBoundVlanId, BoundToVMHost")
	if err != nil {
		diags.AddError("PowerShell error", err.Error())
		return
	}

	data.ID = types.StringValue(stringValue(result, "ID"))
	data.Description = types.StringValue(stringValue(result, "Description"))
	data.VLANID = types.Int64Value(int64Value(result, "HostBoundVlanId"))
	data.BoundToVMHost = types.BoolValue(boolValue(result, "BoundToVMHost"))
}

func buildVirtualNetworkCreateScript(data virtualNetworkResourceModel) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("$host = Get-SCVMHost -Name '%s'; ", escapeSingleQuotes(data.VMHostName.ValueString())))

	args := []string{fmt.Sprintf("-Name '%s'", escapeSingleQuotes(data.Name.ValueString())), "-VMHost $host"}
	if !data.Description.IsNull() && data.Description.ValueString() != "" {
		args = append(args, fmt.Sprintf("-Description '%s'", escapeSingleQuotes(data.Description.ValueString())))
	}
	if !data.VLANID.IsNull() && data.VLANID.ValueInt64() > 0 {
		args = append(args, fmt.Sprintf("-HostBoundVlanId %d", data.VLANID.ValueInt64()))
	}
	if !data.BoundToVMHost.IsNull() {
		args = append(args, fmt.Sprintf("-BoundToVMHost $%t", data.BoundToVMHost.ValueBool()))
	}

	b.WriteString("New-SCVirtualNetwork ")
	b.WriteString(strings.Join(args, " "))
	b.WriteString("; ")
	return b.String()
}

func buildVirtualNetworkUpdateScript(plan, state virtualNetworkResourceModel) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("$host = Get-SCVMHost -Name '%s'; $vnet = Get-SCVirtualNetwork -VMHost $host | Where-Object { $_.Name -eq '%s' } | Select-Object -First 1; ", escapeSingleQuotes(state.VMHostName.ValueString()), escapeSingleQuotes(state.Name.ValueString())))

	setArgs := []string{"-VirtualNetwork $vnet"}
	if !plan.Description.IsNull() && plan.Description.ValueString() != state.Description.ValueString() {
		setArgs = append(setArgs, fmt.Sprintf("-Description '%s'", escapeSingleQuotes(plan.Description.ValueString())))
	}
	if !plan.VLANID.IsNull() && plan.VLANID.ValueInt64() != state.VLANID.ValueInt64() {
		setArgs = append(setArgs, fmt.Sprintf("-HostBoundVlanId %d", plan.VLANID.ValueInt64()))
	}
	if !plan.BoundToVMHost.IsNull() && plan.BoundToVMHost.ValueBool() != state.BoundToVMHost.ValueBool() {
		setArgs = append(setArgs, fmt.Sprintf("-BoundToVMHost $%t", plan.BoundToVMHost.ValueBool()))
	}

	if len(setArgs) > 1 {
		b.WriteString("Set-SCVirtualNetwork ")
		b.WriteString(strings.Join(setArgs, " "))
		b.WriteString("; ")
	}

	return b.String()
}

var _ resource.Resource = (*virtualNetworkResource)(nil)
