package models

import "github.com/hashicorp/terraform-plugin-framework/types"

type VolumeResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	ZoneID             types.String `tfsdk:"zone_id"`
	ServiceOfferingID  types.String `tfsdk:"service_offering_id"`
	Size               types.Int64  `tfsdk:"size"`
	Name               types.String `tfsdk:"name"`
	InstanceID         types.String `tfsdk:"instance_id"`
	Status             types.String `tfsdk:"status"`
	AttachedInstanceID types.String `tfsdk:"attached_instance_id"`
}
