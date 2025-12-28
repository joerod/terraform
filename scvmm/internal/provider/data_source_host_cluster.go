package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewHostClusterDataSource() datasource.DataSource {
	return &hostClusterDataSource{}
}

type hostClusterDataSource struct {
	client *psClient
}

type hostClusterDataSourceModel struct {
	Name      types.String `tfsdk:"name"`
	ID        types.String `tfsdk:"id"`
	HostNames types.List   `tfsdk:"host_names"`
}

func (d *hostClusterDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_host_cluster"
}

func (d *hostClusterDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Host cluster name.",
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "SCVMM host cluster ID.",
			},
			"host_names": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Host names in the cluster.",
			},
		},
	}
}

func (d *hostClusterDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*psClient)
}

func (d *hostClusterDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data hostClusterDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	script := fmt.Sprintf("$cluster = Get-SCVMHostCluster -Name '%s'; $hosts = Get-SCVMHost -VMHostCluster $cluster | Select-Object -ExpandProperty ComputerName; @{ID=$cluster.ID; Names=$hosts} | ConvertTo-Json -Depth 3", escapeSingleQuotes(data.Name.ValueString()))
	result, err := d.client.runPSJSON(ctx, script)
	if err != nil {
		resp.Diagnostics.AddError("PowerShell error", err.Error())
		return
	}

	data.ID = types.StringValue(stringValue(result, "ID"))

	names := stringSliceValue(result, "Names")
	list, diags := types.ListValueFrom(ctx, types.StringType, names)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.HostNames = list

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

var _ datasource.DataSource = (*hostClusterDataSource)(nil)
