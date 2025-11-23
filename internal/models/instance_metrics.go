package models

import "github.com/hashicorp/terraform-plugin-framework/types"

type InstanceMetricsDataSourceModel struct {
	ID          types.String  `tfsdk:"id"`
	ZoneID      types.String  `tfsdk:"zone_id"`
	InstanceID  types.String  `tfsdk:"instance_id"`
	CPUUsage    types.Float64 `tfsdk:"cpu_usage"`
	MemoryUsage types.Float64 `tfsdk:"memory_usage"`
	NetworkIn   types.Int64   `tfsdk:"network_in"`
	NetworkOut  types.Int64   `tfsdk:"network_out"`
	DiskRead    types.Int64   `tfsdk:"disk_read"`
	DiskWrite   types.Int64   `tfsdk:"disk_write"`
}
