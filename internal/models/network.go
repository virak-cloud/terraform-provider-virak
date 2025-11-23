package models

import "github.com/hashicorp/terraform-plugin-framework/types"

type NetworkInstance struct {
	InstanceID   types.String `tfsdk:"instance_id"`
	IPAddress    types.String `tfsdk:"ip_address"`
	MACAddress   types.String `tfsdk:"mac_address"`
	IsDefault    types.Bool   `tfsdk:"is_default"`
	AttachmentID types.String `tfsdk:"attachment_id"`
}

type NetworkResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	ZoneID            types.String `tfsdk:"zone_id"`
	NetworkOfferingID types.String `tfsdk:"network_offering_id"`
	Type              types.String `tfsdk:"type"`
	Gateway           types.String `tfsdk:"gateway"`
	Netmask           types.String `tfsdk:"netmask"`
	Status            types.String `tfsdk:"status"`
	Instances         types.List   `tfsdk:"instances"`
}

type NetworkFilterBlock struct {
	Name   types.String   `tfsdk:"name"`
	Values []types.String `tfsdk:"values"`
}

type NetworksDataSourceModel struct {
	ID       types.String         `tfsdk:"id"`
	ZoneID   types.String         `tfsdk:"zone_id"`
	Filter   []NetworkFilterBlock `tfsdk:"filter"`
	Networks []NetworkModel       `tfsdk:"networks"`
}

type NetworkModel struct {
	ID              types.String         `tfsdk:"id"`
	Name            types.String         `tfsdk:"name"`
	Type            types.String         `tfsdk:"type"`
	Status          types.String         `tfsdk:"status"`
	Gateway         types.String         `tfsdk:"gateway"`
	Netmask         types.String         `tfsdk:"netmask"`
	NetworkOffering NetworkOfferingModel `tfsdk:"network_offering"`
}
