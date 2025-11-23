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

var _ datasource.DataSource = &volumeOfferingsDataSource{}

func NewVolumeOfferingsDataSource() datasource.DataSource {
	return &volumeOfferingsDataSource{}
}

type volumeOfferingsDataSource struct {
	client *http.Client
}

func (d *volumeOfferingsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_volume_service_offerings"
}

func (d *volumeOfferingsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists available Virak Cloud volume service offerings.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of this data source.",
			},
			"zone_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the zone to list offerings for.",
			},
			"offerings": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The ID of the volume offering.",
						},
						"name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The name of the volume offering.",
						},
					},
				},
			},
		},
	}
}

func (d *volumeOfferingsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *volumeOfferingsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data models.VolumeOfferingsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	offeringsResp, err := d.client.ListInstanceVolumeServiceOfferings(data.ZoneID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list volume service offerings, got error: %s", err))
		return
	}

	for _, offering := range offeringsResp.Data {
		data.Offerings = append(data.Offerings, models.VolumeOfferingModel{
			ID:   types.StringValue(offering.ID),
			Name: types.StringValue(offering.Name),
		})
	}

	data.ID = types.StringValue(data.ZoneID.ValueString() + "_volume_offerings")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
