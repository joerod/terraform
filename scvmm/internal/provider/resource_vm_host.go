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

func NewVMHostResource() resource.Resource {
	return &vmHostResource{}
}

type vmHostResource struct {
	client *psClient
}

type vmHostResourceModel struct {
	ComputerName     types.String `tfsdk:"computer_name"`
	HostGroupName    types.String `tfsdk:"host_group_name"`
	RemoveOnDelete   types.Bool   `tfsdk:"remove_on_delete"`
	ID               types.String `tfsdk:"id"`
}

func (r *vmHostResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm_host"
}

func (r *vmHostResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"computer_name": schema.StringAttribute{
				Required:    true,
				Description: "Host computer name (FQDN or short name).",
			},
			"host_group_name": schema.StringAttribute{
				Required:    true,
				Description: "Target host group name.",
			},
			"remove_on_delete": schema.BoolAttribute{
				Optional:    true,
				Description: "Remove the host from VMM when the resource is destroyed.",
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "SCVMM host ID.",
			},
		},
	}
}

func (r *vmHostResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*psClient)
}

func (r *vmHostResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data vmHostResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	script := buildVMHostMoveScript(data)
	if err := r.client.runPS(ctx, script); err != nil {
		resp.Diagnostics.AddError("PowerShell error", err.Error())
		return
	}

	readVMHost(ctx, r.client, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *vmHostResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data vmHostResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readVMHost(ctx, r.client, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *vmHostResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan vmHostResourceModel
	var state vmHostResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.HostGroupName.ValueString() != state.HostGroupName.ValueString() {
		script := buildVMHostMoveScript(plan)
		if err := r.client.runPS(ctx, script); err != nil {
			resp.Diagnostics.AddError("PowerShell error", err.Error())
			return
		}
	}

	readVMHost(ctx, r.client, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *vmHostResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data vmHostResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !data.RemoveOnDelete.IsNull() && data.RemoveOnDelete.ValueBool() {
		script := fmt.Sprintf("$host = Get-SCVMHost -ComputerName '%s'; if ($host) { Remove-SCVMHost -VMHost $host -Force }", escapeSingleQuotes(data.ComputerName.ValueString()))
		if err := r.client.runPS(ctx, script); err != nil {
			resp.Diagnostics.AddError("PowerShell error", err.Error())
			return
		}
	}
}

func (r *vmHostResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("computer_name"), req, resp)
}

func readVMHost(ctx context.Context, client *psClient, data *vmHostResourceModel, diags *diag.Diagnostics) {
	script := fmt.Sprintf("$host = Get-SCVMHost -ComputerName '%s'", escapeSingleQuotes(data.ComputerName.ValueString()))
	result, err := client.runPSJSON(ctx, script+"; $host | Select-Object Name, ID, @{Name='HostGroupName';Expression={$_.VMHostGroup.Name}}")
	if err != nil {
		diags.AddError("PowerShell error", err.Error())
		return
	}

	data.ID = types.StringValue(stringValue(result, "ID"))
	data.HostGroupName = types.StringValue(stringValue(result, "HostGroupName"))
	data.ComputerName = types.StringValue(stringValue(result, "Name"))
}

func buildVMHostMoveScript(data vmHostResourceModel) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("$host = Get-SCVMHost -ComputerName '%s'; ", escapeSingleQuotes(data.ComputerName.ValueString())))
	b.WriteString(fmt.Sprintf("$group = Get-SCVMHostGroup -Name '%s'; ", escapeSingleQuotes(data.HostGroupName.ValueString())))
	b.WriteString("if (-not $host) { throw 'Host not found in VMM.' }; ")
	b.WriteString("if (-not $group) { throw 'Host group not found in VMM.' }; ")
	b.WriteString("Set-SCVMHost -VMHost $host -VMHostGroup $group; ")
	return b.String()
}

var _ resource.Resource = (*vmHostResource)(nil)
