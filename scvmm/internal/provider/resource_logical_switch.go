package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewLogicalSwitchResource() resource.Resource {
	return &logicalSwitchResource{}
}

type logicalSwitchResource struct {
	client *psClient
}

type logicalSwitchResourceModel struct {
	Name                 types.String `tfsdk:"name"`
	Description          types.String `tfsdk:"description"`
	EnableSriov          types.Bool   `tfsdk:"enable_sriov"`
	EnablePacketDirect   types.Bool   `tfsdk:"enable_packet_direct"`
	SwitchUplinkMode     types.String `tfsdk:"switch_uplink_mode"`
	MinimumBandwidthMode types.String `tfsdk:"minimum_bandwidth_mode"`
	VirtualSwitchExtensions types.List `tfsdk:"virtual_switch_extensions"`
	RemoveAllExtensions  types.Bool   `tfsdk:"remove_all_extensions"`
	ID                   types.String `tfsdk:"id"`
}

func (r *logicalSwitchResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_logical_switch"
}

func (r *logicalSwitchResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Logical switch name.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Logical switch description.",
			},
			"enable_sriov": schema.BoolAttribute{
				Optional:    true,
				Description: "Enable SR-IOV.",
			},
			"enable_packet_direct": schema.BoolAttribute{
				Optional:    true,
				Description: "Enable Packet Direct.",
			},
			"switch_uplink_mode": schema.StringAttribute{
				Optional:    true,
				Description: "Switch uplink mode (e.g. Team, SwitchIndependent).",
			},
			"minimum_bandwidth_mode": schema.StringAttribute{
				Optional:    true,
				Description: "Minimum bandwidth mode (e.g. Default, Weight, Absolute).",
			},
			"virtual_switch_extensions": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Virtual switch extension names to attach.",
			},
			"remove_all_extensions": schema.BoolAttribute{
				Optional:    true,
				Description: "Remove all virtual switch extensions from the logical switch.",
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "SCVMM object ID.",
			},
		},
	}
}

func (r *logicalSwitchResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*psClient)
}

func (r *logicalSwitchResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data logicalSwitchResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	exts := listStrings(ctx, data.VirtualSwitchExtensions, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	script := buildLogicalSwitchCreateScript(data, exts)
	if err := r.client.runPS(ctx, script); err != nil {
		resp.Diagnostics.AddError("PowerShell error", err.Error())
		return
	}

	readLogicalSwitch(ctx, r.client, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *logicalSwitchResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data logicalSwitchResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readLogicalSwitch(ctx, r.client, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *logicalSwitchResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan logicalSwitchResourceModel
	var state logicalSwitchResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	planExts := listStrings(ctx, plan.VirtualSwitchExtensions, &resp.Diagnostics)
	stateExts := listStrings(ctx, state.VirtualSwitchExtensions, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	script := buildLogicalSwitchUpdateScript(plan, state, planExts, stateExts)
	if script != "" {
		if err := r.client.runPS(ctx, script); err != nil {
			resp.Diagnostics.AddError("PowerShell error", err.Error())
			return
		}
	}

	readLogicalSwitch(ctx, r.client, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *logicalSwitchResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data logicalSwitchResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	script := fmt.Sprintf("$ls = Get-SCLogicalSwitch -Name '%s'; if ($ls) { Remove-SCLogicalSwitch -LogicalSwitch $ls -Force }", escapeSingleQuotes(data.Name.ValueString()))
	if err := r.client.runPS(ctx, script); err != nil {
		resp.Diagnostics.AddError("PowerShell error", err.Error())
		return
	}
}

func (r *logicalSwitchResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, req, resp, "name")
}

func readLogicalSwitch(ctx context.Context, client *psClient, data *logicalSwitchResourceModel, diags *resource.Diagnostics) {
	script := fmt.Sprintf("$ls = Get-SCLogicalSwitch -Name '%s'", escapeSingleQuotes(data.Name.ValueString()))
	result, err := client.runPSJSON(ctx, script+"; $ls | Select-Object Name, ID, Description, EnableSriov, EnablePacketDirect, SwitchUplinkMode, MinimumBandwidthMode")
	if err != nil {
		diags.AddError("PowerShell error", err.Error())
		return
	}

	data.ID = types.StringValue(stringValue(result, "ID"))
	data.Description = types.StringValue(stringValue(result, "Description"))
	data.EnableSriov = types.BoolValue(boolValue(result, "EnableSriov"))
	data.EnablePacketDirect = types.BoolValue(boolValue(result, "EnablePacketDirect"))
	data.SwitchUplinkMode = types.StringValue(stringValue(result, "SwitchUplinkMode"))
	data.MinimumBandwidthMode = types.StringValue(stringValue(result, "MinimumBandwidthMode"))
}

func buildLogicalSwitchCreateScript(data logicalSwitchResourceModel, exts []string) string {
	args := []string{fmt.Sprintf("-Name '%s'", escapeSingleQuotes(data.Name.ValueString()))}
	if !data.Description.IsNull() && data.Description.ValueString() != "" {
		args = append(args, fmt.Sprintf("-Description '%s'", escapeSingleQuotes(data.Description.ValueString())))
	}
	if !data.EnableSriov.IsNull() {
		args = append(args, fmt.Sprintf("-EnableSriov $%t", data.EnableSriov.ValueBool()))
	}
	if !data.EnablePacketDirect.IsNull() {
		args = append(args, fmt.Sprintf("-EnablePacketDirect $%t", data.EnablePacketDirect.ValueBool()))
	}
	if !data.SwitchUplinkMode.IsNull() && data.SwitchUplinkMode.ValueString() != "" {
		args = append(args, fmt.Sprintf("-SwitchUplinkMode '%s'", escapeSingleQuotes(data.SwitchUplinkMode.ValueString())))
	}
	if !data.MinimumBandwidthMode.IsNull() && data.MinimumBandwidthMode.ValueString() != "" {
		args = append(args, fmt.Sprintf("-MinimumBandwidthMode '%s'", escapeSingleQuotes(data.MinimumBandwidthMode.ValueString())))
	}

	var b strings.Builder
	b.WriteString("New-SCLogicalSwitch ")
	b.WriteString(strings.Join(args, " "))
	b.WriteString("; ")

	if len(exts) > 0 && (data.RemoveAllExtensions.IsNull() || !data.RemoveAllExtensions.ValueBool()) {
		b.WriteString("$exts = @(); ")
		for _, name := range exts {
			b.WriteString(fmt.Sprintf("$ext = Get-SCVirtualSwitchExtension -Name '%s'; if ($ext) { $exts += $ext }; ", escapeSingleQuotes(name)))
		}
		b.WriteString("if ($exts.Count -gt 0) { Set-SCLogicalSwitch -LogicalSwitch (Get-SCLogicalSwitch -Name '")
		b.WriteString(escapeSingleQuotes(data.Name.ValueString()))
		b.WriteString("') -VirtualSwitchExtensions $exts; }; ")
	}

	if !data.RemoveAllExtensions.IsNull() && data.RemoveAllExtensions.ValueBool() {
		b.WriteString("Set-SCLogicalSwitch -LogicalSwitch (Get-SCLogicalSwitch -Name '")
		b.WriteString(escapeSingleQuotes(data.Name.ValueString()))
		b.WriteString("') -RemoveAllExtensions; ")
	}

	return b.String()
}

func buildLogicalSwitchUpdateScript(plan, state logicalSwitchResourceModel, planExts, stateExts []string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("$ls = Get-SCLogicalSwitch -Name '%s'; ", escapeSingleQuotes(state.Name.ValueString())))

	setArgs := []string{"-LogicalSwitch $ls"}
	if !plan.Description.IsNull() && plan.Description.ValueString() != state.Description.ValueString() {
		setArgs = append(setArgs, fmt.Sprintf("-Description '%s'", escapeSingleQuotes(plan.Description.ValueString())))
	}
	if !plan.EnableSriov.IsNull() && plan.EnableSriov.ValueBool() != state.EnableSriov.ValueBool() {
		setArgs = append(setArgs, fmt.Sprintf("-EnableSriov $%t", plan.EnableSriov.ValueBool()))
	}
	if !plan.EnablePacketDirect.IsNull() && plan.EnablePacketDirect.ValueBool() != state.EnablePacketDirect.ValueBool() {
		setArgs = append(setArgs, fmt.Sprintf("-EnablePacketDirect $%t", plan.EnablePacketDirect.ValueBool()))
	}
	if !plan.SwitchUplinkMode.IsNull() && plan.SwitchUplinkMode.ValueString() != state.SwitchUplinkMode.ValueString() {
		setArgs = append(setArgs, fmt.Sprintf("-SwitchUplinkMode '%s'", escapeSingleQuotes(plan.SwitchUplinkMode.ValueString())))
	}
	if !plan.MinimumBandwidthMode.IsNull() && plan.MinimumBandwidthMode.ValueString() != state.MinimumBandwidthMode.ValueString() {
		setArgs = append(setArgs, fmt.Sprintf("-MinimumBandwidthMode '%s'", escapeSingleQuotes(plan.MinimumBandwidthMode.ValueString())))
	}

	if len(setArgs) > 1 {
		b.WriteString("Set-SCLogicalSwitch ")
		b.WriteString(strings.Join(setArgs, " "))
		b.WriteString("; ")
	}

	if len(planExts) > 0 && !stringSliceEqual(planExts, stateExts) && (plan.RemoveAllExtensions.IsNull() || !plan.RemoveAllExtensions.ValueBool()) {
		b.WriteString("$exts = @(); ")
		for _, name := range planExts {
			b.WriteString(fmt.Sprintf("$ext = Get-SCVirtualSwitchExtension -Name '%s'; if ($ext) { $exts += $ext }; ", escapeSingleQuotes(name)))
		}
		b.WriteString("if ($exts.Count -gt 0) { Set-SCLogicalSwitch -LogicalSwitch $ls -VirtualSwitchExtensions $exts; }; ")
	}

	if !plan.RemoveAllExtensions.IsNull() && plan.RemoveAllExtensions.ValueBool() {
		b.WriteString("Set-SCLogicalSwitch -LogicalSwitch $ls -RemoveAllExtensions; ")
	}

	return b.String()
}

var _ resource.Resource = (*logicalSwitchResource)(nil)
