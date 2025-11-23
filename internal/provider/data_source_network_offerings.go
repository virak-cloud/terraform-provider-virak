package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/virak-cloud/cli/pkg/http"
	"github.com/virak-cloud/terraform-provider-virak/internal/models"
)

var _ datasource.DataSource = &networkOfferingsDataSource{}

func NewNetworkOfferingsDataSource() datasource.DataSource {
	return &networkOfferingsDataSource{}
}

type networkOfferingsDataSource struct {
	client *http.Client
}

// local config type used to read zone_id, type and filter blocks from the data source config
type networkOfferingsConfig struct {
	ID        types.String                             `tfsdk:"id"`
	ZoneID    types.String                             `tfsdk:"zone_id"`
	Type      types.String                             `tfsdk:"type"`
	Filter    []models.NetworkOfferingFilterBlockModel `tfsdk:"filter"`
	Offerings []models.NetworkOfferingModel            `tfsdk:"offerings"`
}

func (d *networkOfferingsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_service_offerings"
}

func (d *networkOfferingsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists available Virak Cloud network service offerings with detailed pricing and configuration information.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of this data source.",
			},
			"zone_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the zone to list offerings for.",
			},
			"type": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The type of network offerings to filter by. Must be one of: 'L2', 'Isolated'.",
				Validators: []validator.String{
					stringvalidator.OneOf("L2", "Isolated"),
				},
			},
			// filter blocks allow narrowing the results by attribute name and values (e.g., name)
			"filter": schema.ListNestedAttribute{
				Optional:            true,
				MarkdownDescription: "One or more filter blocks to narrow offerings by attribute name and values.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The attribute name to filter on. Supported: 'name'.",
						},
						"values": schema.ListAttribute{
							Required:            true,
							ElementType:         types.StringType,
							MarkdownDescription: "List of string values to match for the given attribute.",
						},
					},
				},
			},
			"offerings": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "List of available network service offerings.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The unique identifier of the network offering.",
						},
						"name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The technical name of the network offering.",
						},
						"displayname": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The human-readable display name of the network offering.",
						},
						"displayname_fa": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The Persian (Farsi) display name of the network offering.",
						},
						"hourly_started_price": schema.Float64Attribute{
							Computed:            true,
							MarkdownDescription: "The hourly price in the smallest currency unit.",
						},
						"traffic_transfer_overprice": schema.Float64Attribute{
							Computed:            true,
							MarkdownDescription: "The price per unit of traffic transfer beyond the included plan.",
						},
						"traffic_transfer_plan": schema.Int64Attribute{
							Computed:            true,
							MarkdownDescription: "The amount of traffic transfer included in the plan (in GB).",
						},
						"networkrate": schema.Int64Attribute{
							Computed:            true,
							MarkdownDescription: "The network bandwidth rate in Mbps (megabits per second).",
						},
						"type": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The type of network offering. Valid values are 'L2' (Layer 2) or 'Isolated' (Layer 3 with routing).",
						},
						"description": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "A detailed description of the network offering and its features.",
						},
						"internet_protocol": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The supported internet protocol. Valid values are 'IPv4' or 'DualStack' (IPv4 + IPv6).",
						},
					},
				},
			},
		},
	}
}

func (d *networkOfferingsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *networkOfferingsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var (
		cfg  networkOfferingsConfig
		data models.NetworkOfferingsDataSourceModel
	)

	// Read config (zone_id + optional type and filter blocks)
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate zone_id is provided
	if cfg.ZoneID.IsNull() || cfg.ZoneID.ValueString() == "" {
		resp.Diagnostics.AddError(
			"Missing Required Parameter",
			"zone_id is required and cannot be empty",
		)
		return
	}

	offeringsResp, err := d.client.ListNetworkServiceOfferings(cfg.ZoneID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to list network service offerings for zone %s, got error: %s", cfg.ZoneID.ValueString(), err),
		)
		return
	}

	// Build name filter set from provided filter blocks (exact match)
	nameSet := map[string]bool{}
	for _, fb := range cfg.Filter {
		if fb.Name.IsNull() || fb.Name.IsUnknown() {
			continue
		}
		if len(fb.Values) == 0 {
			continue
		}
		nameKey := fb.Name.ValueString()
		if nameKey != "name" {
			// ignore unknown filter names
			continue
		}
		for _, v := range fb.Values {
			if v.IsNull() || v.IsUnknown() {
				continue
			}
			nameSet[v.ValueString()] = true
		}
	}

	// Check if any offerings were returned
	if len(offeringsResp.Data) == 0 {
		resp.Diagnostics.AddWarning(
			"No Offerings Found",
			fmt.Sprintf("No network service offerings found for zone %s", cfg.ZoneID.ValueString()),
		)
	}

	// Initialize the offerings slice
	data.Offerings = []models.NetworkOfferingModel{}

	for _, offering := range offeringsResp.Data {
		// Apply existing type filter if specified (preserve behavior)
		if !cfg.Type.IsNull() && !cfg.Type.IsUnknown() && offering.Type != cfg.Type.ValueString() {
			continue
		}

		// Apply name filter (exact match) if provided
		if len(nameSet) > 0 {
			if _, ok := nameSet[offering.Name]; !ok {
				continue
			}
		}

		offeringModel := models.NetworkOfferingModel{
			ID:                       types.StringValue(offering.ID),
			Name:                     types.StringValue(offering.Name),
			DisplayName:              types.StringValue(offering.DisplayName),
			DisplayNameFA:            types.StringValue(offering.DisplayNameFA),
			HourlyStartedPrice:       types.Float64Value(offering.HourlyStartedPrice),
			TrafficTransferOverprice: types.Float64Value(offering.TrafficTransferOverprice),
			TrafficTransferPlan:      types.Int64Value(int64(offering.TrafficTransferPlan)),
			NetworkRate:              types.Int64Value(int64(offering.NetworkRate)),
			Type:                     types.StringValue(offering.Type),
			Description:              types.StringValue(offering.Description),
			InternetProtocol:         types.StringValue(offering.InternetProtocol),
		}

		data.Offerings = append(data.Offerings, offeringModel)
	}

	// Set zone and type in state and generate ID
	data.ZoneID = types.StringValue(cfg.ZoneID.ValueString())
	data.Type = cfg.Type
	data.ID = types.StringValue(cfg.ZoneID.ValueString() + "_network_offerings")
	if !cfg.Type.IsNull() && !cfg.Type.IsUnknown() {
		data.ID = types.StringValue(cfg.ZoneID.ValueString() + "_network_offerings_" + cfg.Type.ValueString())
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
