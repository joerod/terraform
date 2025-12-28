package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewVirtualDiskDriveResource() resource.Resource {
	return &virtualDiskDriveResource{}
}

type virtualDiskDriveResource struct {
	client *psClient
}

type virtualDiskDriveResourceModel struct {
	VMName     types.String `tfsdk:"vm_name"`
	BusType    types.String `tfsdk:"bus_type"`
	Bus        types.Int64  `tfsdk:"bus"`
	LUN        types.Int64  `tfsdk:"lun"`
	SizeGB     types.Int64  `tfsdk:"size_gb"`
	FileName   types.String `tfsdk:"file_name"`
	Path       types.String `tfsdk:"path"`
	Fixed      types.Bool   `tfsdk:"fixed"`
	Dynamic    types.Bool   `tfsdk:"dynamic"`
	ID         types.String `tfsdk:"id"`
}

func (r *virtualDiskDriveResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_virtual_disk_drive"
}

func (r *virtualDiskDriveResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"vm_name": schema.StringAttribute{
				Required:    true,
				Description: "Virtual machine name.",
			},
			"bus_type": schema.StringAttribute{
				Optional:    true,
				Description: "Bus type: IDE or SCSI (default SCSI).",
			},
			"bus": schema.Int64Attribute{
				Required:    true,
				Description: "Bus number.",
			},
			"lun": schema.Int64Attribute{
				Required:    true,
				Description: "LUN number.",
			},
			"size_gb": schema.Int64Attribute{
				Required:    true,
				Description: "Virtual hard disk size in GB.",
			},
			"file_name": schema.StringAttribute{
				Required:    true,
				Description: "VHD/VHDX file name (without path).",
			},
			"path": schema.StringAttribute{
				Optional:    true,
				Description: "Optional storage path for the VHD/VHDX.",
			},
			"fixed": schema.BoolAttribute{
				Optional:    true,
				Description: "Create a fixed-size disk.",
			},
			"dynamic": schema.BoolAttribute{
				Optional:    true,
				Description: "Create a dynamically expanding disk.",
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Synthetic ID based on VM name, bus, and LUN.",
			},
		},
	}
}

func (r *virtualDiskDriveResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*psClient)
}

func (r *virtualDiskDriveResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data virtualDiskDriveResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	script := buildVirtualDiskDriveCreateScript(data)
	if err := r.client.runPS(ctx, script); err != nil {
		resp.Diagnostics.AddError("PowerShell error", err.Error())
		return
	}

	readVirtualDiskDrive(ctx, r.client, &data, &resp.Diagnostics)
	data.ID = types.StringValue(diskID(data))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *virtualDiskDriveResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data virtualDiskDriveResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readVirtualDiskDrive(ctx, r.client, &data, &resp.Diagnostics)
	data.ID = types.StringValue(diskID(data))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *virtualDiskDriveResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan virtualDiskDriveResourceModel
	var state virtualDiskDriveResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.SizeGB.ValueInt64() < state.SizeGB.ValueInt64() {
		resp.Diagnostics.AddError("Unsupported change", "disk size can only be increased")
		return
	}

	if plan.SizeGB.ValueInt64() > state.SizeGB.ValueInt64() {
		script := buildVirtualDiskDriveExpandScript(plan)
		if err := r.client.runPS(ctx, script); err != nil {
			resp.Diagnostics.AddError("PowerShell error", err.Error())
			return
		}
	}

	readVirtualDiskDrive(ctx, r.client, &plan, &resp.Diagnostics)
	plan.ID = types.StringValue(diskID(plan))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *virtualDiskDriveResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data virtualDiskDriveResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	script := buildVirtualDiskDriveDeleteScript(data)
	if err := r.client.runPS(ctx, script); err != nil {
		resp.Diagnostics.AddError("PowerShell error", err.Error())
		return
	}
}

func (r *virtualDiskDriveResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, req, resp, "id")
}

func buildVirtualDiskDriveCreateScript(data virtualDiskDriveResourceModel) string {
	busType := strings.ToUpper(data.BusType.ValueString())
	if busType == "" {
		busType = "SCSI"
	}

	sizeMB := data.SizeGB.ValueInt64() * 1024
	args := []string{
		fmt.Sprintf("-VM $vm"),
		fmt.Sprintf("-Bus %d", data.Bus.ValueInt64()),
		fmt.Sprintf("-LUN %d", data.LUN.ValueInt64()),
		fmt.Sprintf("-VirtualHardDiskSizeMB %d", sizeMB),
		fmt.Sprintf("-FileName '%s'", escapeSingleQuotes(data.FileName.ValueString())),
	}

	if busType == "IDE" {
		args = append(args, "-IDE")
	} else {
		args = append(args, "-SCSI")
	}

	if !data.Path.IsNull() && data.Path.ValueString() != "" {
		args = append(args, fmt.Sprintf("-Path '%s'", escapeSingleQuotes(data.Path.ValueString())))
	}

	if !data.Fixed.IsNull() && data.Fixed.ValueBool() {
		args = append(args, "-Fixed")
	} else if !data.Dynamic.IsNull() && data.Dynamic.ValueBool() {
		args = append(args, "-Dynamic")
	}

	return fmt.Sprintf("$vm = Get-SCVirtualMachine -Name '%s'; New-SCVirtualDiskDrive %s; ", escapeSingleQuotes(data.VMName.ValueString()), strings.Join(args, " "))
}

func buildVirtualDiskDriveExpandScript(data virtualDiskDriveResourceModel) string {
	return fmt.Sprintf("$vm = Get-SCVirtualMachine -Name '%s'; $disk = Get-SCVirtualDiskDrive -VM $vm | Where-Object { $_.Bus -eq %d -and $_.LUN -eq %d } | Select-Object -First 1; if ($disk) { Expand-SCVirtualDiskDrive -VirtualDiskDrive $disk -VirtualHardDiskSizeGB %d }", escapeSingleQuotes(data.VMName.ValueString()), data.Bus.ValueInt64(), data.LUN.ValueInt64(), data.SizeGB.ValueInt64())
}

func buildVirtualDiskDriveDeleteScript(data virtualDiskDriveResourceModel) string {
	return fmt.Sprintf("$vm = Get-SCVirtualMachine -Name '%s'; $disk = Get-SCVirtualDiskDrive -VM $vm | Where-Object { $_.Bus -eq %d -and $_.LUN -eq %d } | Select-Object -First 1; if ($disk) { Remove-SCVirtualDiskDrive -VirtualDiskDrive $disk -Force }", escapeSingleQuotes(data.VMName.ValueString()), data.Bus.ValueInt64(), data.LUN.ValueInt64())
}

func readVirtualDiskDrive(ctx context.Context, client *psClient, data *virtualDiskDriveResourceModel, diags *resource.Diagnostics) {
	script := fmt.Sprintf("$vm = Get-SCVirtualMachine -Name '%s'; $disk = Get-SCVirtualDiskDrive -VM $vm | Where-Object { $_.Bus -eq %d -and $_.LUN -eq %d } | Select-Object -First 1", escapeSingleQuotes(data.VMName.ValueString()), data.Bus.ValueInt64(), data.LUN.ValueInt64())
	result, err := client.runPSJSON(ctx, script+"; $disk | Select-Object Bus, LUN, VirtualHardDiskSize, FileName")
	if err != nil {
		diags.AddError("PowerShell error", err.Error())
		return
	}

	if v := stringValue(result, "FileName"); v != "" {
		data.FileName = types.StringValue(v)
	}

	sizeRaw := stringValue(result, "VirtualHardDiskSize")
	if sizeRaw != "" {
		if sizeBytes, err := strconv.ParseFloat(sizeRaw, 64); err == nil {
			data.SizeGB = types.Int64Value(int64(sizeBytes / (1024 * 1024 * 1024)))
		}
	}
}

func diskID(data virtualDiskDriveResourceModel) string {
	return fmt.Sprintf("%s:%d:%d", data.VMName.ValueString(), data.Bus.ValueInt64(), data.LUN.ValueInt64())
}

var _ resource.Resource = (*virtualDiskDriveResource)(nil)
