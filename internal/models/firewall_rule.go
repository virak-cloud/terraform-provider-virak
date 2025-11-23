package models

import "github.com/hashicorp/terraform-plugin-framework/types"

type FirewallRuleResourceModel struct {
	ID            types.String `tfsdk:"id"`
	ZoneID        types.String `tfsdk:"zone_id"`
	NetworkID     types.String `tfsdk:"network_id"`
	IPVersion     types.String `tfsdk:"ip_version"`
	TrafficType   types.String `tfsdk:"traffic_type"`
	Protocol      types.String `tfsdk:"protocol"`
	IPSource      types.String `tfsdk:"ip_source"`
	IPDestination types.String `tfsdk:"ip_destination"`
	StartPort     types.Int64  `tfsdk:"start_port"`
	EndPort       types.Int64  `tfsdk:"end_port"`
	ICMPCode      types.Int64  `tfsdk:"icmp_code"`
	ICMPType      types.Int64  `tfsdk:"icmp_type"`
	PublicIPID    types.String `tfsdk:"public_ip_id"`
	Status        types.String `tfsdk:"status"`
	CreatedAt     types.String `tfsdk:"created_at"`
}
