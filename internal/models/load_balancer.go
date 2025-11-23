package models

import "github.com/hashicorp/terraform-plugin-framework/types"

type LoadBalancerResourceModel struct {
	ID          types.String `tfsdk:"id"`
	ZoneID      types.String `tfsdk:"zone_id"`
	NetworkID   types.String `tfsdk:"network_id"`
	PublicIPID  types.String `tfsdk:"public_ip_id"`
	Name        types.String `tfsdk:"name"`
	Algorithm   types.String `tfsdk:"algorithm"`
	PublicPort  types.Int64  `tfsdk:"public_port"`
	PrivatePort types.Int64  `tfsdk:"private_port"`
	Status      types.String `tfsdk:"status"`
}

type LoadBalancerBackendResourceModel struct {
	ID                types.String `tfsdk:"id"`
	ZoneID            types.String `tfsdk:"zone_id"`
	NetworkID         types.String `tfsdk:"network_id"`
	LoadBalancerID    types.String `tfsdk:"load_balancer_id"`
	InstanceNetworkID types.String `tfsdk:"instance_network_id"`
}
