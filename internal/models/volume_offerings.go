package models

import "github.com/hashicorp/terraform-plugin-framework/types"

type VolumeOfferingsDataSourceModel struct {
	ID        types.String          `tfsdk:"id"`
	ZoneID    types.String          `tfsdk:"zone_id"`
	Offerings []VolumeOfferingModel `tfsdk:"offerings"`
}

type VolumeOfferingModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}
