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

var _ datasource.DataSource = &zoneServicesDataSource{}

func NewZoneServicesDataSource() datasource.DataSource {
	return &zoneServicesDataSource{}
}

type zoneServicesDataSource struct {
	client *http.Client
}

func (d *zoneServicesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_zone_services"
}

func (d *zoneServicesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists available services in a Virak Cloud zone.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of this data source.",
			},
			"zone_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the zone to query services for.",
			},
			"services": schema.SingleNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Available services in the zone.",
				Attributes: map[string]schema.Attribute{
					"instance": schema.BoolAttribute{
						Computed:            true,
						MarkdownDescription: "Whether instance service is available.",
					},
					"data_volume": schema.BoolAttribute{
						Computed:            true,
						MarkdownDescription: "Whether data volume service is available.",
					},
					"network": schema.BoolAttribute{
						Computed:            true,
						MarkdownDescription: "Whether network service is available.",
					},
					"object_storage": schema.BoolAttribute{
						Computed:            true,
						MarkdownDescription: "Whether object storage service is available.",
					},
					"k8s": schema.BoolAttribute{
						Computed:            true,
						MarkdownDescription: "Whether Kubernetes service is available.",
					},
				},
			},
		},
	}
}

func (d *zoneServicesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *zoneServicesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg struct {
		ZoneID types.String `tfsdk:"zone_id"`
	}

	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if cfg.ZoneID.IsNull() || cfg.ZoneID.ValueString() == "" {
		resp.Diagnostics.AddError(
			"Missing Required Parameter",
			"zone_id is required and cannot be empty",
		)
		return
	}

	servicesResp, err := d.client.GetZoneActiveServices(cfg.ZoneID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to get zone services for zone %s, got error: %s", cfg.ZoneID.ValueString(), err),
		)
		return
	}

	var data models.ZoneServicesDataSourceModel
	data.ID = types.StringValue(cfg.ZoneID.ValueString() + "_services")
	data.ZoneID = types.StringValue(cfg.ZoneID.ValueString())
	data.Services = models.ZoneServicesModel{
		Instance:      types.BoolValue(servicesResp.Instance),
		DataVolume:    types.BoolValue(servicesResp.DataVolume),
		Network:       types.BoolValue(servicesResp.Network),
		ObjectStorage: types.BoolValue(servicesResp.ObjectStorage),
		K8s:           types.BoolValue(servicesResp.K8s),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
