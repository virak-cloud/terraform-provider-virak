package models

import "github.com/hashicorp/terraform-plugin-framework/types"

type KubernetesVersionsDataSourceModel struct {
	ID       types.String             `tfsdk:"id"`
	ZoneID   types.String             `tfsdk:"zone_id"`
	Versions []KubernetesVersionModel `tfsdk:"versions"`
}

type KubernetesVersionModel struct {
	ID      types.String `tfsdk:"id"`
	Version types.String `tfsdk:"version"`
}
