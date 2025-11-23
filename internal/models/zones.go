package models

import "github.com/hashicorp/terraform-plugin-framework/types"

type ZoneServicesModel struct {
	Instance      types.Bool `tfsdk:"instance"`
	DataVolume    types.Bool `tfsdk:"data_volume"`
	Network       types.Bool `tfsdk:"network"`
	ObjectStorage types.Bool `tfsdk:"object_storage"`
	K8s           types.Bool `tfsdk:"k8s"`
}

type ZoneServicesDataSourceModel struct {
	ID       types.String      `tfsdk:"id"`
	ZoneID   types.String      `tfsdk:"zone_id"`
	Services ZoneServicesModel `tfsdk:"services"`
}

type ZoneResourcesModel struct {
	Memory     ZoneResourceQuota `tfsdk:"memory"`
	CPU        ZoneResourceQuota `tfsdk:"cpu"`
	DataVolume ZoneResourceQuota `tfsdk:"data_volume"`
	VMLimit    ZoneResourceQuota `tfsdk:"vm_limit"`
}

type ZoneResourceQuota struct {
	Collected types.Int64 `tfsdk:"collected"`
	Total     types.Int64 `tfsdk:"total"`
}

type ZoneResourcesDataSourceModel struct {
	ID        types.String       `tfsdk:"id"`
	ZoneID    types.String       `tfsdk:"zone_id"`
	Resources ZoneResourcesModel `tfsdk:"resources"`
}

type ZonesDataSourceModel struct {
	ID    types.String `tfsdk:"id"`
	Zones []ZoneModel  `tfsdk:"zones"`
}

type ZoneModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}
