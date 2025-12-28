package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewVMCheckpointDataSource() datasource.DataSource {
	return &vmCheckpointDataSource{}
}

type vmCheckpointDataSource struct {
	client *psClient
}

type vmCheckpointDataSourceModel struct {
	VMName      types.String `tfsdk:"vm_name"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	ID          types.String `tfsdk:"id"`
}

func (d *vmCheckpointDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm_checkpoint"
}

func (d *vmCheckpointDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"vm_name": schema.StringAttribute{
				Required:    true,
				Description: "Virtual machine name.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Checkpoint name.",
			},
			"description": schema.StringAttribute{
				Computed:    true,
				Description: "Checkpoint description.",
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Synthetic ID based on VM name and checkpoint name.",
			},
		},
	}
}

func (d *vmCheckpointDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*psClient)
}

func (d *vmCheckpointDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data vmCheckpointDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	script := fmt.Sprintf("$vm = Get-SCVirtualMachine -Name '%s'; $cp = Get-SCVMCheckpoint -VM $vm | Where-Object { $_.Name -eq '%s' } | Select-Object -First 1", escapeSingleQuotes(data.VMName.ValueString()), escapeSingleQuotes(data.Name.ValueString()))
	result, err := d.client.runPSJSON(ctx, script+"; $cp | Select-Object Name, Description")
	if err != nil {
		resp.Diagnostics.AddError("PowerShell error", err.Error())
		return
	}

	name := stringValue(result, "Name")
	if name == "" {
		resp.Diagnostics.AddError("Checkpoint not found", "No checkpoint with the given name exists for the VM.")
		return
	}

	data.Name = types.StringValue(name)
	data.Description = types.StringValue(stringValue(result, "Description"))
	data.ID = types.StringValue(fmt.Sprintf("%s:%s", data.VMName.ValueString(), data.Name.ValueString()))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

var _ datasource.DataSource = (*vmCheckpointDataSource)(nil)
