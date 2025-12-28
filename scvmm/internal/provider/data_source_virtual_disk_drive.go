package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewVirtualDiskDriveDataSource() datasource.DataSource {
	return &virtualDiskDriveDataSource{}
}

type virtualDiskDriveDataSource struct {
	client *psClient
}

type virtualDiskDriveDataSourceModel struct {
	VMName   types.String `tfsdk:"vm_name"`
	Bus      types.Int64  `tfsdk:"bus"`
	LUN      types.Int64  `tfsdk:"lun"`
	BusType  types.String `tfsdk:"bus_type"`
	SizeGB   types.Int64  `tfsdk:"size_gb"`
	FileName types.String `tfsdk:"file_name"`
	MovePath types.String `tfsdk:"move_path"`
	ID       types.String `tfsdk:"id"`
}

func (d *virtualDiskDriveDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_virtual_disk_drive"
}

func (d *virtualDiskDriveDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"vm_name": schema.StringAttribute{
				Required:    true,
				Description: "Virtual machine name.",
			},
			"bus": schema.Int64Attribute{
				Required:    true,
				Description: "Bus number.",
			},
			"lun": schema.Int64Attribute{
				Required:    true,
				Description: "LUN number.",
			},
			"bus_type": schema.StringAttribute{
				Computed:    true,
				Description: "Bus type (IDE or SCSI).",
			},
			"size_gb": schema.Int64Attribute{
				Computed:    true,
				Description: "Virtual hard disk size in GB.",
			},
			"file_name": schema.StringAttribute{
				Computed:    true,
				Description: "VHD/VHDX file name.",
			},
			"move_path": schema.StringAttribute{
				Computed:    true,
				Description: "Location of the backing virtual hard disk.",
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Synthetic ID based on VM name, bus, and LUN.",
			},
		},
	}
}

func (d *virtualDiskDriveDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*psClient)
}

func (d *virtualDiskDriveDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data virtualDiskDriveDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	script := fmt.Sprintf("$vm = Get-SCVirtualMachine -Name '%s'; $disk = Get-SCVirtualDiskDrive -VM $vm | Where-Object { $_.Bus -eq %d -and $_.LUN -eq %d } | Select-Object -First 1", escapeSingleQuotes(data.VMName.ValueString()), data.Bus.ValueInt64(), data.LUN.ValueInt64())
	result, err := d.client.runPSJSON(ctx, script+"; $disk | Select-Object BusType, VirtualHardDiskSize, FileName, @{Name='VHDLocation';Expression={$_.VirtualHardDisk.Location}}")
	if err != nil {
		resp.Diagnostics.AddError("PowerShell error", err.Error())
		return
	}

	data.BusType = types.StringValue(stringValue(result, "BusType"))
	data.FileName = types.StringValue(stringValue(result, "FileName"))
	data.MovePath = types.StringValue(stringValue(result, "VHDLocation"))

	sizeRaw := stringValue(result, "VirtualHardDiskSize")
	if sizeRaw != "" {
		if sizeBytes, err := strconv.ParseFloat(sizeRaw, 64); err == nil {
			data.SizeGB = types.Int64Value(int64(sizeBytes / (1024 * 1024 * 1024)))
		}
	}

	data.ID = types.StringValue(fmt.Sprintf("%s:%d:%d", data.VMName.ValueString(), data.Bus.ValueInt64(), data.LUN.ValueInt64()))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

var _ datasource.DataSource = (*virtualDiskDriveDataSource)(nil)
