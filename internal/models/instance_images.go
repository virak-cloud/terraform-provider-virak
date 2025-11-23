package models

import "github.com/hashicorp/terraform-plugin-framework/types"

type InstanceImagesDataSourceModel struct {
	ID     types.String         `tfsdk:"id"`
	ZoneID types.String         `tfsdk:"zone_id"`
	Images []InstanceImageModel `tfsdk:"images"`
}

type InstanceImageModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}
