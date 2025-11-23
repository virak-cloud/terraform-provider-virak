package models

import "github.com/hashicorp/terraform-plugin-framework/types"

type KubernetesClusterResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Name                types.String `tfsdk:"name"`
	ZoneID              types.String `tfsdk:"zone_id"`
	KubernetesVersionID types.String `tfsdk:"kubernetes_version_id"`
	ServiceOfferingID   types.String `tfsdk:"service_offering_id"`
	SshKeyID            types.String `tfsdk:"ssh_key_id"`
	NetworkID           types.String `tfsdk:"network_id"`
	Description         types.String `tfsdk:"description"`
	HaEnabled           types.Bool   `tfsdk:"ha_enabled"`
	ClusterSize         types.Int64  `tfsdk:"cluster_size"`
	Status              types.String `tfsdk:"status"`
	CreatedAt           types.String `tfsdk:"created_at"`
	UpdatedAt           types.String `tfsdk:"updated_at"`
	Kubeconfig          types.String `tfsdk:"kubeconfig"`
	DesiredState        types.String `tfsdk:"desired_state"`
	DesiredSize         types.Int64  `tfsdk:"desired_size"`
	DesiredVersionID    types.String `tfsdk:"desired_version_id"`
}
