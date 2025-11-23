package models

import "github.com/hashicorp/terraform-plugin-framework/types"

type PublicIPResourceModel struct {
	ID         types.String `tfsdk:"id"`
	ZoneID     types.String `tfsdk:"zone_id"`
	NetworkID  types.String `tfsdk:"network_id"`
	InstanceID types.String `tfsdk:"instance_id"`
	IPAddress  types.String `tfsdk:"ip_address"`
	Status     types.String `tfsdk:"status"`
}
