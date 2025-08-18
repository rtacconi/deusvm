package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type DeusProvider struct{}

type DeusProviderModel struct {
	Endpoint types.String `tfsdk:"endpoint"`
	Token    types.String `tfsdk:"token"`
}

func (p *DeusProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "deusvm"
}

func (p *DeusProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{Optional: true},
			"token":    schema.StringAttribute{Optional: true, Validators: []validator.String{}},
		},
	}
}

func (p *DeusProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var cfg DeusProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	endpoint := cfg.Endpoint.ValueString()
	if endpoint == "" {
		endpoint = "127.0.0.1:9090"
	}
	clients, err := NewGRPCClients(ctx, endpoint, false)
	if err != nil {
		resp.Diagnostics.AddError("configure grpc", err.Error())
		return
	}
	resp.ResourceData = clients
	resp.DataSourceData = clients
}

func (p *DeusProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		func() resource.Resource { return NewImageResource() },
		func() resource.Resource { return NewVMResource() },
	}
}

func (p *DeusProvider) DataSources(ctx context.Context) []func() datasource.DataSource { return nil }
