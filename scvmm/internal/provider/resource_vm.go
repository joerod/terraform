package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewVirtualMachineResource() resource.Resource {
	return &virtualMachineResource{}
}

type virtualMachineResource struct {
	client *psClient
}

type virtualMachineResourceModel struct {
	Name         types.String `tfsdk:"name"`
	TemplateName types.String `tfsdk:"template_name"`
	TemplateID   types.String `tfsdk:"template_id"`
	CloudName    types.String `tfsdk:"cloud_name"`
	HostGroup    types.String `tfsdk:"host_group"`
	Owner        types.String `tfsdk:"owner"`
	CPUCount     types.Int64  `tfsdk:"cpu_count"`
	MemoryMB     types.Int64  `tfsdk:"memory_mb"`
	Description  types.String `tfsdk:"description"`
	PowerState   types.String `tfsdk:"power_state"`
	HighlyAvailable types.Bool `tfsdk:"highly_available"`
	ID           types.String `tfsdk:"id"`
	Status       types.String `tfsdk:"status"`
	HostName     types.String `tfsdk:"host_name"`
	VMID         types.String `tfsdk:"vm_id"`
}

func (r *virtualMachineResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_virtual_machine"
}

func (r *virtualMachineResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:    true,
				Description: "SCVMM virtual machine name.",
			},
			"template_name": schema.StringAttribute{
				Optional:    true,
				Description: "VM template name used for creation.",
			},
			"template_id": schema.StringAttribute{
				Optional:    true,
				Description: "VM template ID (GUID) used for creation.",
			},
			"cloud_name": schema.StringAttribute{
				Optional:    true,
				Description: "Cloud name for placement.",
			},
			"host_group": schema.StringAttribute{
				Optional:    true,
				Description: "Host group name for placement.",
			},
			"owner": schema.StringAttribute{
				Optional:    true,
				Description: "Owner user.",
			},
			"cpu_count": schema.Int64Attribute{
				Optional:    true,
				Description: "Number of vCPUs.",
			},
			"memory_mb": schema.Int64Attribute{
				Optional:    true,
				Description: "Memory in MB.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Description.",
			},
			"power_state": schema.StringAttribute{
				Optional:    true,
				Description: "Desired power state: Running or Stopped.",
			},
			"highly_available": schema.BoolAttribute{
				Optional:    true,
				Description: "Make the VM highly available when hosted on a cluster.",
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "SCVMM object ID.",
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "VM status.",
			},
			"host_name": schema.StringAttribute{
				Computed:    true,
				Description: "Host name.",
			},
			"vm_id": schema.StringAttribute{
				Computed:    true,
				Description: "VM ID (GUID) from Hyper-V.",
			},
		},
	}
}

func (r *virtualMachineResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*psClient)
}

func (r *virtualMachineResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data virtualMachineResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	script := buildCreateScript(data)
	if err := r.client.runPS(ctx, script); err != nil {
		resp.Diagnostics.AddError("PowerShell error", err.Error())
		return
	}

	readVM(ctx, r.client, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *virtualMachineResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data virtualMachineResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readVM(ctx, r.client, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *virtualMachineResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan virtualMachineResourceModel
	var state virtualMachineResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	script := buildUpdateScript(plan, state)
	if script != "" {
		if err := r.client.runPS(ctx, script); err != nil {
			resp.Diagnostics.AddError("PowerShell error", err.Error())
			return
		}
	}

	readVM(ctx, r.client, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *virtualMachineResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data virtualMachineResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	script := fmt.Sprintf("$vm = Get-SCVirtualMachine -Name '%s'; if ($vm) { Remove-SCVirtualMachine -VM $vm -Force }", escapeSingleQuotes(data.Name.ValueString()))
	if err := r.client.runPS(ctx, script); err != nil {
		resp.Diagnostics.AddError("PowerShell error", err.Error())
		return
	}
}

func (r *virtualMachineResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, req, resp, "name")
}

func readVM(ctx context.Context, client *psClient, data *virtualMachineResourceModel, diags *diag.Diagnostics) {
	name := data.Name.ValueString()
	script := fmt.Sprintf("$vm = Get-SCVirtualMachine -Name '%s'", escapeSingleQuotes(name))
	result, err := client.runPSJSON(ctx, script+"; $vm | Select-Object Name, ID, Status, HostName, CPUCount, MemoryMB, VMId, Owner, Description, HighlyAvailable")
	if err != nil {
		diags.AddError("PowerShell error", err.Error())
		return
	}

	data.ID = types.StringValue(stringValue(result, "ID"))
	data.Status = types.StringValue(stringValue(result, "Status"))
	data.HostName = types.StringValue(stringValue(result, "HostName"))
	data.CPUCount = types.Int64Value(int64Value(result, "CPUCount"))
	data.MemoryMB = types.Int64Value(int64Value(result, "MemoryMB"))
	data.VMID = types.StringValue(stringValue(result, "VMId"))
	data.Owner = types.StringValue(stringValue(result, "Owner"))
	data.Description = types.StringValue(stringValue(result, "Description"))
	data.HighlyAvailable = types.BoolValue(boolValue(result, "HighlyAvailable"))
}

func buildCreateScript(data virtualMachineResourceModel) string {
	name := escapeSingleQuotes(data.Name.ValueString())

	var b strings.Builder
	b.WriteString("$vmTemplate = $null; ")
	if !data.TemplateName.IsNull() && data.TemplateName.ValueString() != "" {
		b.WriteString(fmt.Sprintf("$vmTemplate = Get-SCVMTemplate -Name '%s'; ", escapeSingleQuotes(data.TemplateName.ValueString())))
	} else if !data.TemplateID.IsNull() && data.TemplateID.ValueString() != "" {
		b.WriteString(fmt.Sprintf("$vmTemplate = Get-SCVMTemplate -ID '%s'; ", escapeSingleQuotes(data.TemplateID.ValueString())))
	}

	b.WriteString("$cloud = $null; ")
	if !data.CloudName.IsNull() && data.CloudName.ValueString() != "" {
		b.WriteString(fmt.Sprintf("$cloud = Get-SCCloud -Name '%s'; ", escapeSingleQuotes(data.CloudName.ValueString())))
	}

	b.WriteString("$hostGroup = $null; ")
	if !data.HostGroup.IsNull() && data.HostGroup.ValueString() != "" {
		b.WriteString(fmt.Sprintf("$hostGroup = Get-SCVMHostGroup -Name '%s'; ", escapeSingleQuotes(data.HostGroup.ValueString())))
	}

	args := []string{fmt.Sprintf("-Name '%s'", name)}
	if !data.Owner.IsNull() && data.Owner.ValueString() != "" {
		args = append(args, fmt.Sprintf("-Owner '%s'", escapeSingleQuotes(data.Owner.ValueString())))
	}
	if !data.Description.IsNull() && data.Description.ValueString() != "" {
		args = append(args, fmt.Sprintf("-Description '%s'", escapeSingleQuotes(data.Description.ValueString())))
	}
	if !data.CPUCount.IsNull() && data.CPUCount.ValueInt64() > 0 {
		args = append(args, fmt.Sprintf("-CPUCount %d", data.CPUCount.ValueInt64()))
	}
	if !data.MemoryMB.IsNull() && data.MemoryMB.ValueInt64() > 0 {
		args = append(args, fmt.Sprintf("-MemoryMB %d", data.MemoryMB.ValueInt64()))
	}
	if !data.HighlyAvailable.IsNull() {
		args = append(args, fmt.Sprintf("-HighlyAvailable $%t", data.HighlyAvailable.ValueBool()))
	}
	args = append(args, "-VMTemplate $vmTemplate")
	if !data.CloudName.IsNull() && data.CloudName.ValueString() != "" {
		args = append(args, "-Cloud $cloud")
	}
	if !data.HostGroup.IsNull() && data.HostGroup.ValueString() != "" {
		args = append(args, "-VMHostGroup $hostGroup")
	}

	b.WriteString("$vm = New-SCVirtualMachine ")
	b.WriteString(strings.Join(args, " "))
	b.WriteString("; ")

	if !data.PowerState.IsNull() && data.PowerState.ValueString() != "" {
		desired := strings.ToLower(data.PowerState.ValueString())
		switch desired {
		case "running":
			b.WriteString("Start-SCVirtualMachine -VM $vm; ")
		case "stopped":
			b.WriteString("Stop-SCVirtualMachine -VM $vm -Force; ")
		}
	}

	return b.String()
}

func buildUpdateScript(plan, state virtualMachineResourceModel) string {
	name := escapeSingleQuotes(plan.Name.ValueString())
	var b strings.Builder
	b.WriteString(fmt.Sprintf("$vm = Get-SCVirtualMachine -Name '%s'; ", name))

	setArgs := []string{"-VM $vm"}

	if !plan.Owner.IsNull() && plan.Owner.ValueString() != state.Owner.ValueString() {
		setArgs = append(setArgs, fmt.Sprintf("-Owner '%s'", escapeSingleQuotes(plan.Owner.ValueString())))
	}
	if !plan.Description.IsNull() && plan.Description.ValueString() != state.Description.ValueString() {
		setArgs = append(setArgs, fmt.Sprintf("-Description '%s'", escapeSingleQuotes(plan.Description.ValueString())))
	}
	if !plan.CPUCount.IsNull() && plan.CPUCount.ValueInt64() != state.CPUCount.ValueInt64() {
		setArgs = append(setArgs, fmt.Sprintf("-CPUCount %d", plan.CPUCount.ValueInt64()))
	}
	if !plan.MemoryMB.IsNull() && plan.MemoryMB.ValueInt64() != state.MemoryMB.ValueInt64() {
		setArgs = append(setArgs, fmt.Sprintf("-MemoryMB %d", plan.MemoryMB.ValueInt64()))
	}
	if !plan.HighlyAvailable.IsNull() && plan.HighlyAvailable.ValueBool() != state.HighlyAvailable.ValueBool() {
		setArgs = append(setArgs, fmt.Sprintf("-HighlyAvailable $%t", plan.HighlyAvailable.ValueBool()))
	}

	if len(setArgs) > 1 {
		b.WriteString("Set-SCVirtualMachine ")
		b.WriteString(strings.Join(setArgs, " "))
		b.WriteString("; ")
	}

	if !plan.PowerState.IsNull() && plan.PowerState.ValueString() != "" {
		desired := strings.ToLower(plan.PowerState.ValueString())
		switch desired {
		case "running":
			b.WriteString("Start-SCVirtualMachine -VM $vm; ")
		case "stopped":
			b.WriteString("Stop-SCVirtualMachine -VM $vm -Force; ")
		}
	}

	if b.Len() == 0 {
		return ""
	}
	return b.String()
}

var _ resource.Resource = (*virtualMachineResource)(nil)
