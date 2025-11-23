package models

import "github.com/hashicorp/terraform-plugin-framework/types"

type PortForwardingRuleResourceModel struct {
	ID          types.String `tfsdk:"id"`
	ZoneID      types.String `tfsdk:"zone_id"`
	NetworkID   types.String `tfsdk:"network_id"`
	PublicIPID  types.String `tfsdk:"public_ip_id"`
	Protocol    types.String `tfsdk:"protocol"`
	PublicPort  types.Int64  `tfsdk:"public_port"`
	PrivatePort types.Int64  `tfsdk:"private_port"`
	InstanceID  types.String `tfsdk:"instance_id"`
	PrivateIP   types.String `tfsdk:"private_ip"`
	Status      types.String `tfsdk:"status"`
	CreatedAt   types.String `tfsdk:"created_at"`
}
