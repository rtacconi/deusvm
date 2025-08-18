package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	deusvmproto "github.com/riccardotacconi/deusvm/pkg/proto/gen/github.com/riccardotacconi/deusvm/pkg/proto"
)

type vmResource struct{ clients *GRPCClients }

func NewVMResource() resource.Resource { return &vmResource{} }

type vmModel struct {
	ID     types.String `tfsdk:"id"`
	Name   types.String `tfsdk:"name"`
	Image  types.String `tfsdk:"image"`
	CPU    types.Int64  `tfsdk:"cpu"`
	Memory types.String `tfsdk:"memory"`
	Disk   types.String `tfsdk:"disk"`
}

func (r *vmResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "deusvm_vm"
}

func (r *vmResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id":     schema.StringAttribute{Computed: true},
			"name":   schema.StringAttribute{Required: true},
			"image":  schema.StringAttribute{Required: true},
			"cpu":    schema.Int64Attribute{Required: true},
			"memory": schema.StringAttribute{Required: true},
			"disk":   schema.StringAttribute{Required: true},
		},
	}
}

func (r *vmResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*GRPCClients)
	if ok {
		r.clients = c
	}
}

func (r *vmResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data vmModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	vm, err := r.clients.VM.Create(ctx, &deusvmproto.CreateVMRequest{
		Name: data.Name.ValueString(), Image: data.Image.ValueString(), Cpu: int32(data.CPU.ValueInt64()),
		MemoryBytes: 0, DiskBytes: 0, // for simplicity; convert strings later
	})
	if err != nil {
		resp.Diagnostics.AddError("create vm", err.Error())
		return
	}
	data.ID = types.StringValue(vm.GetId())
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *vmResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data vmModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	vm, err := r.clients.VM.Get(ctx, &deusvmproto.VMIDRequest{Id: data.ID.ValueString()})
	if err != nil {
		return
	}
	data.Name = types.StringValue(vm.GetName())
	data.CPU = types.Int64Value(int64(vm.GetCpu()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *vmResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data vmModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *vmResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data vmModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	_, _ = r.clients.VM.Delete(ctx, &deusvmproto.VMIDRequest{Id: data.ID.ValueString()})
}

func (r *vmResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
