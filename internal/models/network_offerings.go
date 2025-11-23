package models

import "github.com/hashicorp/terraform-plugin-framework/types"

type NetworkOfferingsDataSourceModel struct {
	ID        types.String                      `tfsdk:"id"`
	ZoneID    types.String                      `tfsdk:"zone_id"`
	Type      types.String                      `tfsdk:"type"`
	Filter    []NetworkOfferingFilterBlockModel `tfsdk:"filter"`
	Offerings []NetworkOfferingModel            `tfsdk:"offerings"`
}

type NetworkOfferingFilterBlockModel struct {
	Name   types.String   `tfsdk:"name"`
	Values []types.String `tfsdk:"values"`
}

type NetworkOfferingModel struct {
	ID                       types.String  `tfsdk:"id"`
	Name                     types.String  `tfsdk:"name"`
	DisplayName              types.String  `tfsdk:"displayname"`
	DisplayNameFA            types.String  `tfsdk:"displayname_fa"`
	HourlyStartedPrice       types.Float64 `tfsdk:"hourly_started_price"`
	TrafficTransferOverprice types.Float64 `tfsdk:"traffic_transfer_overprice"`
	TrafficTransferPlan      types.Int64   `tfsdk:"traffic_transfer_plan"`
	NetworkRate              types.Int64   `tfsdk:"networkrate"`
	Type                     types.String  `tfsdk:"type"`
	Description              types.String  `tfsdk:"description"`
	InternetProtocol         types.String  `tfsdk:"internet_protocol"`
}
