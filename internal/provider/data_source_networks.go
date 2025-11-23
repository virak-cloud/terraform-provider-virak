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

var _ datasource.DataSource = &networksDataSource{}

func NewNetworksDataSource() datasource.DataSource {
	return &networksDataSource{}
}

type networksDataSource struct {
	client *http.Client
}

type networkFilterBlock struct {
	Name   types.String   `tfsdk:"name"`
	Values []types.String `tfsdk:"values"`
}

type networksConfig struct {
	ID       types.String          `tfsdk:"id"`
	ZoneID   types.String          `tfsdk:"zone_id"`
	Filter   []networkFilterBlock  `tfsdk:"filter"`
	Networks []models.NetworkModel `tfsdk:"networks"`
}

func (d *networksDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_networks"
}

func (d *networksDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists available Virak Cloud networks in a zone with network offering details.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of this data source.",
			},
			"zone_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the zone to list networks for.",
			},
			"filter": schema.ListNestedAttribute{
				Optional:            true,
				MarkdownDescription: "One or more filter blocks to narrow networks by attribute name and values.",
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
			"networks": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "List of available networks.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The unique identifier of the network.",
						},
						"name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The name of the network.",
						},
						"type": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The type of the network.",
						},
						"status": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The status of the network.",
						},
						"gateway": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The gateway IP address of the network.",
						},
						"netmask": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The netmask of the network.",
						},
						"network_offering": schema.SingleNestedAttribute{
							Computed:            true,
							MarkdownDescription: "The network offering associated with this network.",
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
									MarkdownDescription: "The type of network offering. Valid values include 'Shared' (for public networks), 'L2', or 'Isolated'.",
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
			},
		},
	}
}

func (d *networksDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *networksDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var (
		cfg  networksConfig
		data models.NetworksDataSourceModel
	)

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

	networksResp, err := d.client.ListNetworks(cfg.ZoneID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to list networks for zone %s, got error: %s", cfg.ZoneID.ValueString(), err),
		)
		return
	}

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
			continue
		}
		for _, v := range fb.Values {
			if v.IsNull() || v.IsUnknown() {
				continue
			}
			nameSet[v.ValueString()] = true
		}
	}

	if len(networksResp.Data) == 0 {
		resp.Diagnostics.AddWarning(
			"No Networks Found",
			fmt.Sprintf("No networks found for zone %s", cfg.ZoneID.ValueString()),
		)
	}

	data.Networks = []models.NetworkModel{}

	for _, network := range networksResp.Data {
		if len(nameSet) > 0 {
			if _, ok := nameSet[network.Name]; !ok {
				continue
			}
		}

		gateway := types.StringNull()
		netmask := types.StringNull()
		networkType := types.StringValue(network.NetworkOffering.Type)

		networkOffering := models.NetworkOfferingModel{
			ID:   types.StringValue(network.NetworkOffering.ID),
			Name: types.StringValue(network.NetworkOffering.Name),
			Type: types.StringValue(network.NetworkOffering.Type),
		}

		if network.NetworkOffering.DisplayName != "" {
			networkOffering.DisplayName = types.StringValue(network.NetworkOffering.DisplayName)
		} else {
			networkOffering.DisplayName = types.StringNull()
		}

		if network.NetworkOffering.DisplayNameFA != "" {
			networkOffering.DisplayNameFA = types.StringValue(network.NetworkOffering.DisplayNameFA)
		} else {
			networkOffering.DisplayNameFA = types.StringNull()
		}

		networkOffering.HourlyStartedPrice = types.Float64Null()
		networkOffering.TrafficTransferOverprice = types.Float64Null()
		networkOffering.TrafficTransferPlan = types.Int64Null()
		networkOffering.NetworkRate = types.Int64Null()
		networkOffering.Description = types.StringNull()
		networkOffering.InternetProtocol = types.StringNull()

		networkModel := models.NetworkModel{
			ID:              types.StringValue(network.ID),
			Name:            types.StringValue(network.Name),
			Status:          types.StringValue(network.Status),
			Gateway:         gateway,
			Netmask:         netmask,
			Type:            networkType,
			NetworkOffering: networkOffering,
		}

		data.Networks = append(data.Networks, networkModel)
	}

	data.ZoneID = cfg.ZoneID

	filterBlocks := make([]models.NetworkFilterBlock, 0, len(cfg.Filter))
	for _, fb := range cfg.Filter {
		filterBlocks = append(filterBlocks, models.NetworkFilterBlock{
			Name:   fb.Name,
			Values: fb.Values,
		})
	}
	data.Filter = filterBlocks

	data.ID = types.StringValue(cfg.ZoneID.ValueString() + "_networks")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
