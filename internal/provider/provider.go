package provider

import (
	"context"
	"net/http"
	"os"
	"sync"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	httppkg "github.com/virak-cloud/cli/pkg/http"
)

// Ensure the implementation satisfies the provider.Provider interface.
// var _ provider.Provider = &virakCloudProvider{}

type virakCloudProvider struct {
	version     string
	createMutex sync.Mutex
}

type virakCloudProviderModel struct {
	Token   types.String `tfsdk:"token"`
	Verbose types.Bool   `tfsdk:"verbose"`
}

func (p *virakCloudProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "virakcloud"
	resp.Version = p.version
}

func (p *virakCloudProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"token": schema.StringAttribute{
				MarkdownDescription: "Virak Cloud API token.",
				Optional:            true,
				Sensitive:           true,
			},
			"verbose": schema.BoolAttribute{
				MarkdownDescription: "Enable verbose logging for debugging.",
				Optional:            true,
			},
		},
	}
}

func (p *virakCloudProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data virakCloudProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	token := os.Getenv("VIRAKCLOUD_TOKEN")

	if data.Token.ValueString() != "" {
		token = data.Token.ValueString()
	}

	if token == "" {
		resp.Diagnostics.AddError(
			"Missing API Token",
			"The provider cannot create the API client as there is a missing API token. Please set the VIRAKCLOUD_TOKEN environment variable or use the 'token' provider attribute.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	baseURL := "https://public-api.virakcloud.com"

	client := &httppkg.Client{
		Token:      token,
		BaseURL:    baseURL,
		HttpClient: &http.Client{},
	}

	resp.ResourceData = client
	resp.DataSourceData = client
}

func (p *virakCloudProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		func() resource.Resource { return NewInstanceResource(&p.createMutex) },
		func() resource.Resource { return NewNetworkResource(&p.createMutex) },
		func() resource.Resource { return NewBucketResource(&p.createMutex) },
		func() resource.Resource { return NewKubernetesClusterResource(&p.createMutex) },
		func() resource.Resource { return NewDnsDomainResource(&p.createMutex) },
		func() resource.Resource { return NewDnsRecordResource(&p.createMutex) },
		func() resource.Resource { return NewVolumeResource(&p.createMutex) },
		func() resource.Resource { return NewSnapshotResource(&p.createMutex) },
		func() resource.Resource { return NewFirewallRuleResource(&p.createMutex) },
		func() resource.Resource { return NewPublicIPResource(&p.createMutex) },
		func() resource.Resource { return NewLoadBalancerResource(&p.createMutex) },
		func() resource.Resource { return NewLoadBalancerBackendResource(&p.createMutex) },
		func() resource.Resource { return NewNetworkVPNResource(&p.createMutex) },
		func() resource.Resource { return NewSSHKeyResource(&p.createMutex) },
		func() resource.Resource { return NewPortForwardingRuleResource(&p.createMutex) },
		// ... other resources
	}
}

func (p *virakCloudProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewZonesDataSource,
		NewInstanceOfferingsDataSource,
		NewInstanceImagesDataSource,
		NewKubernetesVersionsDataSource,
		NewNetworkOfferingsDataSource,
		NewNetworksDataSource,
		NewVolumeOfferingsDataSource,
		NewZoneServicesDataSource,
		// ... other data sources
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &virakCloudProvider{
			version: version,
		}
	}
}
