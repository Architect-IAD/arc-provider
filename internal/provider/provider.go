// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// Ensure ArchitectProvider satisfies various provider interfaces.
var _ provider.Provider = &ArchitectProvider{}
var _ provider.ProviderWithFunctions = &ArchitectProvider{}
var _ provider.ProviderWithEphemeralResources = &ArchitectProvider{}

// ArchitectProvider defines the provider implementation.
type ArchitectProvider struct {
	orgs *organizations.Client
	version string
}


func (p *ArchitectProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "architect"
	resp.Version = p.version
}

func (p *ArchitectProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{}
}

func (p *ArchitectProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	cfg, err := config.LoadDefaultConfig(ctx)

	if err != nil {
		resp.Diagnostics.AddError("AWS configuration error", err.Error())
		return
	}

	p.orgs = organizations.NewFromConfig(cfg)

	resp.DataSourceData = p.orgs
	resp.ResourceData = p.orgs
}

func (p *ArchitectProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewAccountResource,
	}
}

func (p *ArchitectProvider) EphemeralResources(ctx context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{}
}

func (p *ArchitectProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func (p *ArchitectProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &ArchitectProvider{
			version: version,
		}
	}
}
