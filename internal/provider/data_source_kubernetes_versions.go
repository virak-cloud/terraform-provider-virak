package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/virak-cloud/cli/pkg/http"
	"github.com/virak-cloud/terraform-provider-virak/internal/models"
)

var _ datasource.DataSource = &kubernetesVersionsDataSource{}

func NewKubernetesVersionsDataSource() datasource.DataSource {
	return &kubernetesVersionsDataSource{}
}

type kubernetesVersionsDataSource struct {
	client *http.Client
}

func (d *kubernetesVersionsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kubernetes_versions"
}

func (d *kubernetesVersionsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists available Virak Cloud Kubernetes versions.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of this data source.",
			},
			"zone_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the zone to list versions for.",
			},
			"versions": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The ID of the Kubernetes version.",
						},
						"version": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The version string of the Kubernetes version.",
						},
					},
				},
			},
		},
	}
}

func (d *kubernetesVersionsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*http.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	d.client = client
}

func (d *kubernetesVersionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data models.KubernetesVersionsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	versionsResp, err := d.client.GetKubernetesVersions(data.ZoneID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list kubernetes versions, got error: %s", err))
		return
	}

	for _, version := range versionsResp.Data {
		data.Versions = append(data.Versions, models.KubernetesVersionModel{
			ID:      types.StringValue(version.ID),
			Version: types.StringValue(version.Version),
		})
	}

	data.ID = types.StringValue(data.ZoneID.ValueString() + "_versions")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
