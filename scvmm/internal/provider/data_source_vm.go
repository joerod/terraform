package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewVirtualMachineDataSource() datasource.DataSource {
	return &virtualMachineDataSource{}
}

type virtualMachineDataSource struct {
	client *psClient
}

type virtualMachineDataSourceModel struct {
	Name        types.String `tfsdk:"name"`
	ID          types.String `tfsdk:"id"`
	Status      types.String `tfsdk:"status"`
	HostName    types.String `tfsdk:"host_name"`
	CPUCount    types.Int64  `tfsdk:"cpu_count"`
	MemoryMB    types.Int64  `tfsdk:"memory_mb"`
	VMID        types.String `tfsdk:"vm_id"`
	Owner       types.String `tfsdk:"owner"`
	Description types.String `tfsdk:"description"`
}

func (d *virtualMachineDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_virtual_machine"
}

func (d *virtualMachineDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:    true,
				Description: "SCVMM virtual machine name.",
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
			"cpu_count": schema.Int64Attribute{
				Computed:    true,
				Description: "Number of vCPUs.",
			},
			"memory_mb": schema.Int64Attribute{
				Computed:    true,
				Description: "Memory in MB.",
			},
			"vm_id": schema.StringAttribute{
				Computed:    true,
				Description: "VM ID (GUID) from Hyper-V.",
			},
			"owner": schema.StringAttribute{
				Computed:    true,
				Description: "Owner user.",
			},
			"description": schema.StringAttribute{
				Computed:    true,
				Description: "Description.",
			},
		},
	}
}

func (d *virtualMachineDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*psClient)
}

func (d *virtualMachineDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data virtualMachineDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := data.Name.ValueString()
	script := fmt.Sprintf("$vm = Get-SCVirtualMachine -Name '%s'", escapeSingleQuotes(name))
	result, err := d.client.runPSJSON(ctx, script+"; $vm | Select-Object Name, ID, Status, HostName, CPUCount, MemoryMB, VMId, Owner, Description")
	if err != nil {
		resp.Diagnostics.AddError("PowerShell error", err.Error())
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

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

var _ datasource.DataSource = (*virtualMachineDataSource)(nil)
