package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// New returns a new provider instance.
func New() provider.Provider {
	return &scvmmProvider{}
}

type scvmmProvider struct{}

type scvmmProviderModel struct {
	Server types.String `tfsdk:"server"`
}

func (p *scvmmProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "scvmm"
}

func (p *scvmmProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"server": schema.StringAttribute{
				Optional:    true,
				Description: "SCVMM server name or FQDN. If unset, PowerShell uses default server context.",
			},
		},
	}
}

func (p *scvmmProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data scvmmProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := &psClient{server: data.Server.ValueString()}

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *scvmmProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewVirtualMachineDataSource,
		NewVirtualDiskDriveDataSource,
		NewVMCheckpointDataSource,
		NewVMHostDataSource,
		NewHostClusterDataSource,
	}
}

func (p *scvmmProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewVirtualMachineResource,
		NewHostGroupResource,
		NewVMHostResource,
		NewHostClusterResource,
		NewVirtualNetworkResource,
		NewVMNetworkAdapterResource,
		NewLogicalSwitchResource,
		NewVirtualDiskDriveResource,
		NewVMCheckpointResource,
		NewVMSubnetResource,
	}
}

// Ensure provider satisfies framework interfaces.
var _ provider.Provider = (*scvmmProvider)(nil)

// Validate that required config is present.
func (p *scvmmProvider) ValidateConfig(ctx context.Context, req provider.ValidateConfigRequest, resp *provider.ValidateConfigResponse) {
	var data scvmmProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// No required fields yet, but keep the hook for future validation.
	_ = path.Root("server")
	_ = ctx
}
