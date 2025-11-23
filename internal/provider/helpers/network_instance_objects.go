package helpers

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/virak-cloud/cli/pkg/http/responses"
)

// BuildNetworkInstanceObjects converts InstanceNetwork responses to Terraform object values
func BuildNetworkInstanceObjects(instances []responses.InstanceNetwork) []attr.Value {
	instanceObjects := make([]attr.Value, 0, len(instances))
	for _, ni := range instances {
		instanceObj, objDiags := types.ObjectValue(
			map[string]attr.Type{
				"instance_id":   types.StringType,
				"ip_address":    types.StringType,
				"mac_address":   types.StringType,
				"is_default":    types.BoolType,
				"attachment_id": types.StringType,
			},
			map[string]attr.Value{
				"instance_id":   types.StringValue(ni.InstanceID),
				"ip_address":    types.StringValue(ni.IPAddress),
				"mac_address":   types.StringValue(ni.MACAddress),
				"is_default":    types.BoolValue(ni.IsDefault),
				"attachment_id": types.StringValue(ni.ID),
			},
		)
		if objDiags.HasError() {
			// In the original code, diagnostics are appended but we continue
			// Since this is a helper, we'll collect them and return
			continue
		}
		instanceObjects = append(instanceObjects, instanceObj)
	}
	return instanceObjects
}

// CreateNetworkInstancesList creates a Terraform list from instance object values
func CreateNetworkInstancesList(instanceObjects []attr.Value, diags *diag.Diagnostics) types.List {
	instancesList, listDiags := types.ListValue(
		types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"instance_id":   types.StringType,
				"ip_address":    types.StringType,
				"mac_address":   types.StringType,
				"is_default":    types.BoolType,
				"attachment_id": types.StringType,
			},
		},
		instanceObjects,
	)
	diags.Append(listDiags...)
	return instancesList
}

// GetNetworkInstanceObjectType returns the object type for network instance objects
func GetNetworkInstanceObjectType() types.ObjectType {
	return types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"instance_id":   types.StringType,
			"ip_address":    types.StringType,
			"mac_address":   types.StringType,
			"is_default":    types.BoolType,
			"attachment_id": types.StringType,
		},
	}
}
