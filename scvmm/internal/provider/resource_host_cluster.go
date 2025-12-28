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

func NewHostClusterResource() resource.Resource {
	return &hostClusterResource{}
}

type hostClusterResource struct {
	client *psClient
}

type hostClusterResourceModel struct {
	Name                 types.String `tfsdk:"name"`
	VMHostGroup          types.String `tfsdk:"vm_host_group"`
	RunAsAccount         types.String `tfsdk:"run_as_account"`
	Description          types.String `tfsdk:"description"`
	ClusterReserve       types.Int64  `tfsdk:"cluster_reserve"`
	RemoteConnectEnabled types.Bool   `tfsdk:"remote_connect_enabled"`
	RemoteConnectPort    types.Int64  `tfsdk:"remote_connect_port"`
	EnableLiveMigration  types.Bool   `tfsdk:"enable_live_migration"`
	HostNodes            types.List   `tfsdk:"host_nodes"`
	RemoveMissingNodes   types.Bool   `tfsdk:"remove_missing_nodes"`
	ID                   types.String `tfsdk:"id"`
}

func (r *hostClusterResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_host_cluster"
}

func (r *hostClusterResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Cluster name (FQDN or short name).",
			},
			"vm_host_group": schema.StringAttribute{
				Required:    true,
				Description: "VM host group to place the cluster in.",
			},
			"run_as_account": schema.StringAttribute{
				Required:    true,
				Description: "SCVMM Run As account name for cluster access.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Cluster description.",
			},
			"cluster_reserve": schema.Int64Attribute{
				Optional:    true,
				Description: "Cluster reserve percentage.",
			},
			"remote_connect_enabled": schema.BoolAttribute{
				Optional:    true,
				Description: "Enable remote connections to hosts in the cluster.",
			},
			"remote_connect_port": schema.Int64Attribute{
				Optional:    true,
				Description: "Remote connection port.",
			},
			"enable_live_migration": schema.BoolAttribute{
				Optional:    true,
				Description: "Enable live migration for the cluster.",
			},
			"host_nodes": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Host nodes to add to the cluster.",
			},
			"remove_missing_nodes": schema.BoolAttribute{
				Optional:    true,
				Description: "Remove cluster nodes not listed in host_nodes.",
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "SCVMM object ID.",
			},
		},
	}
}

func (r *hostClusterResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*psClient)
}

func (r *hostClusterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data hostClusterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	script := buildHostClusterCreateScript(data)
	if err := r.client.runPS(ctx, script); err != nil {
		resp.Diagnostics.AddError("PowerShell error", err.Error())
		return
	}

	addNodesToCluster(ctx, r.client, data, &resp.Diagnostics)

	readHostCluster(ctx, r.client, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *hostClusterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data hostClusterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readHostCluster(ctx, r.client, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *hostClusterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan hostClusterResourceModel
	var state hostClusterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	script := buildHostClusterUpdateScript(plan, state)
	if script != "" {
		if err := r.client.runPS(ctx, script); err != nil {
			resp.Diagnostics.AddError("PowerShell error", err.Error())
			return
		}
	}

	addNodesToClusterUpdate(ctx, r.client, plan, state, &resp.Diagnostics)
	removeNodesFromClusterUpdate(ctx, r.client, plan, &resp.Diagnostics)

	readHostCluster(ctx, r.client, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *hostClusterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data hostClusterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	script := fmt.Sprintf("$cluster = Get-SCVMHostCluster -Name '%s'; if ($cluster) { Remove-SCVMHostCluster -VMHostCluster $cluster -Force }", escapeSingleQuotes(data.Name.ValueString()))
	if err := r.client.runPS(ctx, script); err != nil {
		resp.Diagnostics.AddError("PowerShell error", err.Error())
		return
	}
}

func (r *hostClusterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}

func readHostCluster(ctx context.Context, client *psClient, data *hostClusterResourceModel, diags *diag.Diagnostics) {
	name := data.Name.ValueString()
	script := fmt.Sprintf("$cluster = Get-SCVMHostCluster -Name '%s'", escapeSingleQuotes(name))
	result, err := client.runPSJSON(ctx, script+"; $cluster | Select-Object Name, ID, Description")
	if err != nil {
		diags.AddError("PowerShell error", err.Error())
		return
	}

	data.ID = types.StringValue(stringValue(result, "ID"))
	data.Description = types.StringValue(stringValue(result, "Description"))
}

func buildHostClusterCreateScript(data hostClusterResourceModel) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("$group = Get-SCVMHostGroup -Name '%s'; ", escapeSingleQuotes(data.VMHostGroup.ValueString())))
	b.WriteString(fmt.Sprintf("$cred = Get-SCRunAsAccount -Name '%s'; ", escapeSingleQuotes(data.RunAsAccount.ValueString())))
	b.WriteString("if (-not $cred) { throw 'Run As account not found.' }; ")

	args := []string{fmt.Sprintf("-Name '%s'", escapeSingleQuotes(data.Name.ValueString())), "-VMHostGroup $group", "-Credential $cred"}
	if !data.Description.IsNull() && data.Description.ValueString() != "" {
		args = append(args, fmt.Sprintf("-Description '%s'", escapeSingleQuotes(data.Description.ValueString())))
	}
	if !data.ClusterReserve.IsNull() && data.ClusterReserve.ValueInt64() > 0 {
		args = append(args, fmt.Sprintf("-ClusterReserve %d", data.ClusterReserve.ValueInt64()))
	}
	if !data.RemoteConnectEnabled.IsNull() {
		args = append(args, fmt.Sprintf("-RemoteConnectEnabled $%t", data.RemoteConnectEnabled.ValueBool()))
	}
	if !data.RemoteConnectPort.IsNull() && data.RemoteConnectPort.ValueInt64() > 0 {
		args = append(args, fmt.Sprintf("-RemoteConnectPort %d", data.RemoteConnectPort.ValueInt64()))
	}
	if !data.EnableLiveMigration.IsNull() {
		args = append(args, fmt.Sprintf("-EnableLiveMigration $%t", data.EnableLiveMigration.ValueBool()))
	}

	b.WriteString("Add-SCVMHostCluster ")
	b.WriteString(strings.Join(args, " "))
	b.WriteString("; ")

	return b.String()
}

func buildHostClusterUpdateScript(plan, state hostClusterResourceModel) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("$cluster = Get-SCVMHostCluster -Name '%s'; ", escapeSingleQuotes(state.Name.ValueString())))

	setArgs := []string{"-VMHostCluster $cluster"}
	if !plan.Description.IsNull() && plan.Description.ValueString() != state.Description.ValueString() {
		setArgs = append(setArgs, fmt.Sprintf("-Description '%s'", escapeSingleQuotes(plan.Description.ValueString())))
	}
	if !plan.ClusterReserve.IsNull() && plan.ClusterReserve.ValueInt64() != state.ClusterReserve.ValueInt64() {
		setArgs = append(setArgs, fmt.Sprintf("-ClusterReserve %d", plan.ClusterReserve.ValueInt64()))
	}
	if !plan.RemoteConnectEnabled.IsNull() && plan.RemoteConnectEnabled.ValueBool() != state.RemoteConnectEnabled.ValueBool() {
		setArgs = append(setArgs, fmt.Sprintf("-RemoteConnectEnabled $%t", plan.RemoteConnectEnabled.ValueBool()))
	}
	if !plan.RemoteConnectPort.IsNull() && plan.RemoteConnectPort.ValueInt64() != state.RemoteConnectPort.ValueInt64() {
		setArgs = append(setArgs, fmt.Sprintf("-RemoteConnectPort %d", plan.RemoteConnectPort.ValueInt64()))
	}
	if !plan.EnableLiveMigration.IsNull() && plan.EnableLiveMigration.ValueBool() != state.EnableLiveMigration.ValueBool() {
		setArgs = append(setArgs, fmt.Sprintf("-EnableLiveMigration $%t", plan.EnableLiveMigration.ValueBool()))
	}

	if len(setArgs) > 1 {
		b.WriteString("Set-SCVMHostCluster ")
		b.WriteString(strings.Join(setArgs, " "))
		b.WriteString("; ")
	}

	return b.String()
}

func addNodesToCluster(ctx context.Context, client *psClient, data hostClusterResourceModel, diags *diag.Diagnostics) {
	nodes := listStrings(ctx, data.HostNodes, diags)
	if diags.HasError() || len(nodes) == 0 {
		return
	}

	script := buildAddClusterNodesScript(data, nodes)
	if err := client.runPS(ctx, script); err != nil {
		diags.AddError("PowerShell error", err.Error())
	}
}

func addNodesToClusterUpdate(ctx context.Context, client *psClient, plan, state hostClusterResourceModel, diags *diag.Diagnostics) {
	planNodes := listStrings(ctx, plan.HostNodes, diags)
	stateNodes := listStrings(ctx, state.HostNodes, diags)
	if diags.HasError() || len(planNodes) == 0 {
		return
	}

	if stringSliceEqual(planNodes, stateNodes) {
		return
	}

	missing := make([]string, 0, len(planNodes))
	existing := make(map[string]struct{}, len(stateNodes))
	for _, n := range stateNodes {
		existing[strings.ToLower(n)] = struct{}{}
	}
	for _, n := range planNodes {
		if _, ok := existing[strings.ToLower(n)]; !ok {
			missing = append(missing, n)
		}
	}
	if len(missing) == 0 {
		return
	}

	script := buildAddClusterNodesScript(plan, missing)
	if err := client.runPS(ctx, script); err != nil {
		diags.AddError("PowerShell error", err.Error())
	}
}

func buildAddClusterNodesScript(data hostClusterResourceModel, nodes []string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("$cluster = Get-SCVMHostCluster -Name '%s'; ", escapeSingleQuotes(data.Name.ValueString())))
	b.WriteString(fmt.Sprintf("$cred = Get-SCRunAsAccount -Name '%s'; ", escapeSingleQuotes(data.RunAsAccount.ValueString())))
	b.WriteString("if (-not $cred) { throw 'Run As account not found.' }; ")
	for _, node := range nodes {
		b.WriteString(fmt.Sprintf("Add-SCVMHost -ComputerName '%s' -VMHostCluster $cluster -Credential $cred; ", escapeSingleQuotes(node)))
	}
	return b.String()
}

func removeNodesFromClusterUpdate(ctx context.Context, client *psClient, plan hostClusterResourceModel, diags *diag.Diagnostics) {
	if plan.RemoveMissingNodes.IsNull() || !plan.RemoveMissingNodes.ValueBool() {
		return
	}

	planNodes := listStrings(ctx, plan.HostNodes, diags)
	if diags.HasError() || len(planNodes) == 0 {
		return
	}

	currentNodes, err := fetchClusterNodes(ctx, client, plan.Name.ValueString())
	if err != nil {
		diags.AddError("PowerShell error", err.Error())
		return
	}

	keep := make(map[string]struct{}, len(planNodes))
	for _, n := range planNodes {
		keep[strings.ToLower(n)] = struct{}{}
	}

	var remove []string
	for _, n := range currentNodes {
		if _, ok := keep[strings.ToLower(n)]; !ok {
			remove = append(remove, n)
		}
	}

	if len(remove) == 0 {
		return
	}

	script := buildRemoveClusterNodesScript(remove)
	if err := client.runPS(ctx, script); err != nil {
		diags.AddError("PowerShell error", err.Error())
	}
}

func fetchClusterNodes(ctx context.Context, client *psClient, clusterName string) ([]string, error) {
	script := fmt.Sprintf("$cluster = Get-SCVMHostCluster -Name '%s'; $nodes = Get-SCVMHost -VMHostCluster $cluster | Select-Object -ExpandProperty ComputerName; @{Names=$nodes} | ConvertTo-Json -Depth 3", escapeSingleQuotes(clusterName))
	result, err := client.runPSJSON(ctx, script)
	if err != nil {
		return nil, err
	}

	return stringSliceValue(result, "Names"), nil
}

func buildRemoveClusterNodesScript(nodes []string) string {
	var b strings.Builder
	for _, node := range nodes {
		b.WriteString(fmt.Sprintf("$host = Get-SCVMHost -ComputerName '%s'; if ($host) { Remove-SCVMHost -VMHost $host -Force }; ", escapeSingleQuotes(node)))
	}
	return b.String()
}

var _ resource.Resource = (*hostClusterResource)(nil)
