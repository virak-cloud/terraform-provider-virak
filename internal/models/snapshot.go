package models

import "github.com/hashicorp/terraform-plugin-framework/types"

type SnapshotResourceModel struct {
	ID         types.String `tfsdk:"id"`
	ZoneID     types.String `tfsdk:"zone_id"`
	InstanceID types.String `tfsdk:"instance_id"`
	Name       types.String `tfsdk:"name"`
	Status     types.String `tfsdk:"status"`
	CreatedAt  types.String `tfsdk:"created_at"`
	Revert     types.Bool   `tfsdk:"revert"`
}
