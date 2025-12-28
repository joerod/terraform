package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewVMHostDataSource() datasource.DataSource {
	return &vmHostDataSource{}
}

type vmHostDataSource struct {
	client *psClient
}

type vmHostDataSourceModel struct {
	ComputerName  types.String `tfsdk:"computer_name"`
	HostGroupName types.String `tfsdk:"host_group_name"`
	ID            types.String `tfsdk:"id"`
}

func (d *vmHostDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm_host"
}

func (d *vmHostDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"computer_name": schema.StringAttribute{
				Required:    true,
				Description: "Host computer name (FQDN or short name).",
			},
			"host_group_name": schema.StringAttribute{
				Computed:    true,
				Description: "Host group name.",
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "SCVMM host ID.",
			},
		},
	}
}

func (d *vmHostDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*psClient)
}

func (d *vmHostDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data vmHostDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	script := fmt.Sprintf("$host = Get-SCVMHost -ComputerName '%s'", escapeSingleQuotes(data.ComputerName.ValueString()))
	result, err := d.client.runPSJSON(ctx, script+"; $host | Select-Object Name, ID, @{Name='HostGroupName';Expression={$_.VMHostGroup.Name}}")
	if err != nil {
		resp.Diagnostics.AddError("PowerShell error", err.Error())
		return
	}

	data.ID = types.StringValue(stringValue(result, "ID"))
	data.HostGroupName = types.StringValue(stringValue(result, "HostGroupName"))
	data.ComputerName = types.StringValue(stringValue(result, "Name"))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

var _ datasource.DataSource = (*vmHostDataSource)(nil)
