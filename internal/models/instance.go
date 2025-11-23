package models

import "github.com/hashicorp/terraform-plugin-framework/types"

type Volume struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	Size              types.Int64  `tfsdk:"size"`
	ServiceOfferingID types.String `tfsdk:"service_offering_id"`
}

type VolumeSpec struct {
	Name              types.String `tfsdk:"name"`
	Size              types.Int64  `tfsdk:"size"`
	ServiceOfferingID types.String `tfsdk:"service_offering_id"`
}

type InstanceNetwork struct {
	NetworkID    types.String `tfsdk:"network_id"`
	IPAddress    types.String `tfsdk:"ip_address"`
	MACAddress   types.String `tfsdk:"mac_address"`
	IsDefault    types.Bool   `tfsdk:"is_default"`
	AttachmentID types.String `tfsdk:"attachment_id"`
}

type InstanceResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	ZoneID            types.String `tfsdk:"zone_id"`
	ServiceOfferingID types.String `tfsdk:"service_offering_id"`
	VMImageID         types.String `tfsdk:"vm_image_id"`
	NetworkIDs        types.List   `tfsdk:"network_ids"`
	Status            types.String `tfsdk:"status"`
	Password          types.String `tfsdk:"password"`
	Username          types.String `tfsdk:"username"`
	IP                types.String `tfsdk:"ip"`
	Networks          types.List   `tfsdk:"networks"`
	DesiredState      types.String `tfsdk:"desired_state"`
}

type InstanceOfferingsDataSourceModel struct {
	ID          types.String                       `tfsdk:"id"`
	ZoneID      types.String                       `tfsdk:"zone_id"`
	Filter      []InstanceOfferingFilterBlockModel `tfsdk:"filter"`
	FilterLogic types.String                       `tfsdk:"filter_logic"`
	Offerings   []InstanceOfferingModel            `tfsdk:"offerings"`
}

type InstanceOfferingFilterBlockModel struct {
	Name          types.String   `tfsdk:"name"`
	MatchType     types.String   `tfsdk:"match_type"`
	Values        []types.String `tfsdk:"values"`
	CaseSensitive types.Bool     `tfsdk:"case_sensitive"`
}

type InstanceOfferingModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	CPUCore         types.Int64  `tfsdk:"cpu_core"`
	MemoryMB        types.Int64  `tfsdk:"memory_mb"`
	CPUSpeedMHz     types.Int64  `tfsdk:"cpu_speed_mhz"`
	RootDiskSizeGB  types.Int64  `tfsdk:"root_disk_size_gb"`
	NetworkRate     types.Int64  `tfsdk:"network_rate"`
	DiskIOPS        types.Int64  `tfsdk:"disk_iops"`
	HourlyPriceUp   types.Int64  `tfsdk:"hourly_price_up"`
	HourlyPriceDown types.Int64  `tfsdk:"hourly_price_down"`
	IsAvailable     types.Bool   `tfsdk:"is_available"`
	IsPublic        types.Bool   `tfsdk:"is_public"`
	Description     types.String `tfsdk:"description"`
}
