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

func NewHostGroupResource() resource.Resource {
	return &hostGroupResource{}
}

type hostGroupResource struct {
	client *psClient
}

type hostGroupResourceModel struct {
	Name                          types.String `tfsdk:"name"`
	Description                   types.String `tfsdk:"description"`
	ParentHostGroupName           types.String `tfsdk:"parent_host_group_name"`
	EnableUnencryptedFileTransfer types.Bool   `tfsdk:"enable_unencrypted_file_transfer"`
	InheritNetworkSettings        types.Bool   `tfsdk:"inherit_network_settings"`
	ID                            types.String `tfsdk:"id"`
}

func (r *hostGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_host_group"
}

func (r *hostGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Host group name.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Host group description.",
			},
			"parent_host_group_name": schema.StringAttribute{
				Optional:    true,
				Description: "Parent host group name (defaults to All Hosts).",
			},
			"enable_unencrypted_file_transfer": schema.BoolAttribute{
				Optional:    true,
				Description: "Enable unencrypted file transfer for this host group.",
			},
			"inherit_network_settings": schema.BoolAttribute{
				Optional:    true,
				Description: "Inherit network settings from parent host group.",
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "SCVMM object ID.",
			},
		},
	}
}

func (r *hostGroupResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*psClient)
}

func (r *hostGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data hostGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	script := buildHostGroupCreateScript(data)
	if err := r.client.runPS(ctx, script); err != nil {
		resp.Diagnostics.AddError("PowerShell error", err.Error())
		return
	}

	readHostGroup(ctx, r.client, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *hostGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data hostGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readHostGroup(ctx, r.client, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *hostGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan hostGroupResourceModel
	var state hostGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	script := buildHostGroupUpdateScript(plan, state)
	if script != "" {
		if err := r.client.runPS(ctx, script); err != nil {
			resp.Diagnostics.AddError("PowerShell error", err.Error())
			return
		}
	}

	readHostGroup(ctx, r.client, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *hostGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data hostGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	script := fmt.Sprintf("$group = Get-SCVMHostGroup -Name '%s'; if ($group) { Remove-SCVMHostGroup -VMHostGroup $group -Force }", escapeSingleQuotes(data.Name.ValueString()))
	if err := r.client.runPS(ctx, script); err != nil {
		resp.Diagnostics.AddError("PowerShell error", err.Error())
		return
	}
}

func (r *hostGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}

func readHostGroup(ctx context.Context, client *psClient, data *hostGroupResourceModel, diags *diag.Diagnostics) {
	name := data.Name.ValueString()
	script := fmt.Sprintf("$group = Get-SCVMHostGroup -Name '%s'", escapeSingleQuotes(name))
	result, err := client.runPSJSON(ctx, script+"; $group | Select-Object Name, ID, Description, @{Name='ParentHostGroupName';Expression={$_.ParentHostGroup.Name}}, EnableUnencryptedFileTransfer, InheritNetworkSettings")
	if err != nil {
		diags.AddError("PowerShell error", err.Error())
		return
	}

	data.ID = types.StringValue(stringValue(result, "ID"))
	data.Description = types.StringValue(stringValue(result, "Description"))
	data.ParentHostGroupName = types.StringValue(stringValue(result, "ParentHostGroupName"))
	data.EnableUnencryptedFileTransfer = types.BoolValue(boolValue(result, "EnableUnencryptedFileTransfer"))
	data.InheritNetworkSettings = types.BoolValue(boolValue(result, "InheritNetworkSettings"))
}

func buildHostGroupCreateScript(data hostGroupResourceModel) string {
	var b strings.Builder
	if !data.ParentHostGroupName.IsNull() && data.ParentHostGroupName.ValueString() != "" {
		b.WriteString(fmt.Sprintf("$parent = Get-SCVMHostGroup -Name '%s'; ", escapeSingleQuotes(data.ParentHostGroupName.ValueString())))
	}

	args := []string{fmt.Sprintf("-Name '%s'", escapeSingleQuotes(data.Name.ValueString()))}
	if !data.Description.IsNull() && data.Description.ValueString() != "" {
		args = append(args, fmt.Sprintf("-Description '%s'", escapeSingleQuotes(data.Description.ValueString())))
	}
	if !data.ParentHostGroupName.IsNull() && data.ParentHostGroupName.ValueString() != "" {
		args = append(args, "-ParentHostGroup $parent")
	}
	if !data.EnableUnencryptedFileTransfer.IsNull() {
		args = append(args, fmt.Sprintf("-EnableUnencryptedFileTransfer $%t", data.EnableUnencryptedFileTransfer.ValueBool()))
	}
	if !data.InheritNetworkSettings.IsNull() {
		args = append(args, fmt.Sprintf("-InheritNetworkSettings $%t", data.InheritNetworkSettings.ValueBool()))
	}

	b.WriteString("New-SCVMHostGroup ")
	b.WriteString(strings.Join(args, " "))
	b.WriteString("; ")
	return b.String()
}

func buildHostGroupUpdateScript(plan, state hostGroupResourceModel) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("$group = Get-SCVMHostGroup -Name '%s'; ", escapeSingleQuotes(state.Name.ValueString())))

	setArgs := []string{"-VMHostGroup $group"}
	if !plan.Description.IsNull() && plan.Description.ValueString() != state.Description.ValueString() {
		setArgs = append(setArgs, fmt.Sprintf("-Description '%s'", escapeSingleQuotes(plan.Description.ValueString())))
	}
	if !plan.ParentHostGroupName.IsNull() && plan.ParentHostGroupName.ValueString() != state.ParentHostGroupName.ValueString() {
		b.WriteString(fmt.Sprintf("$parent = Get-SCVMHostGroup -Name '%s'; ", escapeSingleQuotes(plan.ParentHostGroupName.ValueString())))
		setArgs = append(setArgs, "-ParentHostGroup $parent")
	}
	if !plan.EnableUnencryptedFileTransfer.IsNull() && plan.EnableUnencryptedFileTransfer.ValueBool() != state.EnableUnencryptedFileTransfer.ValueBool() {
		setArgs = append(setArgs, fmt.Sprintf("-EnableUnencryptedFileTransfer $%t", plan.EnableUnencryptedFileTransfer.ValueBool()))
	}
	if !plan.InheritNetworkSettings.IsNull() && plan.InheritNetworkSettings.ValueBool() != state.InheritNetworkSettings.ValueBool() {
		setArgs = append(setArgs, fmt.Sprintf("-InheritNetworkSettings $%t", plan.InheritNetworkSettings.ValueBool()))
	}

	if len(setArgs) > 1 {
		b.WriteString("Set-SCVMHostGroup ")
		b.WriteString(strings.Join(setArgs, " "))
		b.WriteString("; ")
	}

	return b.String()
}

var _ resource.Resource = (*hostGroupResource)(nil)
