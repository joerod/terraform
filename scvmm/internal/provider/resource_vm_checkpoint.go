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

func NewVMCheckpointResource() resource.Resource {
	return &vmCheckpointResource{}
}

type vmCheckpointResource struct {
	client *psClient
}

type vmCheckpointResourceModel struct {
	VMName      types.String `tfsdk:"vm_name"`
	VMID        types.String `tfsdk:"vm_id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	ID          types.String `tfsdk:"id"`
}

func (r *vmCheckpointResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm_checkpoint"
}

func (r *vmCheckpointResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"vm_name": schema.StringAttribute{
				Optional:    true,
				Description: "Virtual machine name. Required if vm_id is not set.",
			},
			"vm_id": schema.StringAttribute{
				Optional:    true,
				Description: "Virtual machine ID (GUID). Preferred over vm_name.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Checkpoint name.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Checkpoint description.",
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Synthetic ID based on VM name and checkpoint name.",
			},
		},
	}
}

func (r *vmCheckpointResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*psClient)
}

func (r *vmCheckpointResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data vmCheckpointResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.VMID.IsNull() && data.VMName.IsNull() {
		resp.Diagnostics.AddError("Missing VM reference", "Either vm_id or vm_name must be set.")
		return
	}

	script := buildCheckpointCreateScript(data)
	if err := r.client.runPS(ctx, script); err != nil {
		resp.Diagnostics.AddError("PowerShell error", err.Error())
		return
	}

	readCheckpoint(ctx, r.client, &data, &resp.Diagnostics)
	data.ID = types.StringValue(checkpointID(data))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *vmCheckpointResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data vmCheckpointResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.VMID.IsNull() && data.VMName.IsNull() {
		resp.Diagnostics.AddError("Missing VM reference", "Either vm_id or vm_name must be set.")
		return
	}

	found := readCheckpoint(ctx, r.client, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	data.ID = types.StringValue(checkpointID(data))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *vmCheckpointResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan vmCheckpointResourceModel
	var state vmCheckpointResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.VMID.IsNull() && plan.VMName.IsNull() {
		resp.Diagnostics.AddError("Missing VM reference", "Either vm_id or vm_name must be set.")
		return
	}

	if plan.Name.ValueString() != state.Name.ValueString() {
		resp.Diagnostics.AddError("Unsupported change", "checkpoint name cannot be updated")
		return
	}

	script := buildCheckpointUpdateScript(plan)
	if script != "" {
		if err := r.client.runPS(ctx, script); err != nil {
			resp.Diagnostics.AddError("PowerShell error", err.Error())
			return
		}
	}

	readCheckpoint(ctx, r.client, &plan, &resp.Diagnostics)
	plan.ID = types.StringValue(checkpointID(plan))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *vmCheckpointResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data vmCheckpointResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.VMID.IsNull() && data.VMName.IsNull() {
		resp.Diagnostics.AddError("Missing VM reference", "Either vm_id or vm_name must be set.")
		return
	}

	script := buildCheckpointDeleteScript(data)
	if err := r.client.runPS(ctx, script); err != nil {
		resp.Diagnostics.AddError("PowerShell error", err.Error())
		return
	}
}

func (r *vmCheckpointResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func buildCheckpointCreateScript(data vmCheckpointResourceModel) string {
	script := buildCheckpointVMRef(data)
	args := []string{"-VM $vm", fmt.Sprintf("-Name '%s'", escapeSingleQuotes(data.Name.ValueString()))}
	if !data.Description.IsNull() && data.Description.ValueString() != "" {
		args = append(args, fmt.Sprintf("-Description '%s'", escapeSingleQuotes(data.Description.ValueString())))
	}
	return script + "New-SCVMCheckpoint " + strings.Join(args, " ") + "; "
}

func buildCheckpointDeleteScript(data vmCheckpointResourceModel) string {
	return buildCheckpointVMRef(data) + fmt.Sprintf("$cp = Get-SCVMCheckpoint -VM $vm | Where-Object { $_.Name -eq '%s' } | Select-Object -First 1; if ($cp) { Remove-SCVMCheckpoint -VMCheckpoint $cp -Force }", escapeSingleQuotes(data.Name.ValueString()))
}

func buildCheckpointUpdateScript(data vmCheckpointResourceModel) string {
	if data.Description.IsNull() || data.Description.ValueString() == "" {
		return ""
	}
	return buildCheckpointVMRef(data) + fmt.Sprintf("$cp = Get-SCVMCheckpoint -VM $vm | Where-Object { $_.Name -eq '%s' } | Select-Object -First 1; if ($cp) { Set-SCVMCheckpoint -VMCheckpoint $cp -Description '%s' }", escapeSingleQuotes(data.Name.ValueString()), escapeSingleQuotes(data.Description.ValueString()))
}

func readCheckpoint(ctx context.Context, client *psClient, data *vmCheckpointResourceModel, diags *diag.Diagnostics) bool {
	script := buildCheckpointVMRef(data) + fmt.Sprintf("$cp = Get-SCVMCheckpoint -VM $vm | Where-Object { $_.Name -eq '%s' } | Select-Object -First 1; if ($cp) { $cp | Select-Object Name, Description } else { @{} }", escapeSingleQuotes(data.Name.ValueString()))
	result, err := client.runPSJSON(ctx, script)
	if err != nil {
		diags.AddError("PowerShell error", err.Error())
		return false
	}

	name := stringValue(result, "Name")
	if name == "" {
		return false
	}

	data.Name = types.StringValue(name)
	data.Description = types.StringValue(stringValue(result, "Description"))
	return true
}

func checkpointID(data vmCheckpointResourceModel) string {
	if !data.VMID.IsNull() && data.VMID.ValueString() != "" {
		return fmt.Sprintf("%s:%s", data.VMID.ValueString(), data.Name.ValueString())
	}
	return fmt.Sprintf("%s:%s", data.VMName.ValueString(), data.Name.ValueString())
}

func buildCheckpointVMRef(data vmCheckpointResourceModel) string {
	if !data.VMID.IsNull() && data.VMID.ValueString() != "" {
		return fmt.Sprintf("$vm = Get-SCVirtualMachine -ID '%s'; ", escapeSingleQuotes(data.VMID.ValueString()))
	}
	return fmt.Sprintf("$vm = Get-SCVirtualMachine -Name '%s'; ", escapeSingleQuotes(data.VMName.ValueString()))
}

var _ resource.Resource = (*vmCheckpointResource)(nil)
