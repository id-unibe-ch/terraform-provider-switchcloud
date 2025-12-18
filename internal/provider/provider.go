// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"net/http"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure SwitchcloudProvider satisfies various provider interfaces.
var _ provider.Provider = &SwitchcloudProvider{}

// SwitchcloudProvider defines the provider implementation.
type SwitchcloudProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// SwitchcloudProviderModel describes the provider data model.
type SwitchcloudProviderModel struct {
	Endpoint types.String `tfsdk:"endpoint"`
	ApiKey   types.String `tfsdk:"api_key"`
}

func (p *SwitchcloudProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "switchcloud"
	resp.Version = p.version
}

func (p *SwitchcloudProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "SwitchCloud API endpoint",
				Optional:            true,
			},
			"api_key": schema.StringAttribute{
				MarkdownDescription: "SwitchCloud API key",
				Optional:            true,
				Sensitive:           true,
			},
		},
	}
}

func (p *SwitchcloudProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data SwitchcloudProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Default endpoint if not provided
	endpoint := "https://api.switchcloud.com"
	if !data.Endpoint.IsNull() {
		endpoint = data.Endpoint.ValueString()
	}

	if os.Getenv("SWITCHCLOUD_ENDPOINT") != "" {
		endpoint = os.Getenv("SWITCHCLOUD_ENDPOINT")
	}

	// Create HTTP client with authentication if API key is provided
	client := http.DefaultClient

	if os.Getenv("SWITCHCLOUD_API_KEY") != "" {
		data.ApiKey = types.StringValue(os.Getenv("SWITCHCLOUD_API_KEY"))
	}
	if !data.ApiKey.IsNull() {
		// You can customize the HTTP client here to add authentication headers
		// For example, you might create a custom RoundTripper
		client = &http.Client{
			Transport: &authenticatedTransport{
				apiKey:    data.ApiKey.ValueString(),
				transport: http.DefaultTransport,
			},
		}
	}

	// Pass both client and endpoint to resources and data sources
	providerData := map[string]interface{}{
		"client":   client,
		"endpoint": endpoint,
	}

	resp.DataSourceData = providerData
	resp.ResourceData = providerData
}

// authenticatedTransport is a custom HTTP transport that adds authentication headers.
type authenticatedTransport struct {
	apiKey    string
	transport http.RoundTripper
}

func (t *authenticatedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Add authentication header
	req.Header.Set("Authorization", "Bearer "+t.apiKey)
	return t.transport.RoundTrip(req)
}

func (p *SwitchcloudProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewProjectResource,
		NewProjectMemberResource,
	}
}

func (p *SwitchcloudProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewProjectDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &SwitchcloudProvider{
			version: version,
		}
	}
}
