package models

import "github.com/hashicorp/terraform-plugin-framework/types"

type NetworkVPNResourceModel struct {
	ID           types.String `tfsdk:"id"`
	ZoneID       types.String `tfsdk:"zone_id"`
	NetworkID    types.String `tfsdk:"network_id"`
	Enabled      types.Bool   `tfsdk:"enabled"`
	PresharedKey types.String `tfsdk:"preshared_key"`
	IPAddress    types.String `tfsdk:"ip_address"`
	Username     types.String `tfsdk:"username"`
	Password     types.String `tfsdk:"password"`
	Status       types.String `tfsdk:"status"`
}
