package helpers

import (
	"fmt"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/virak-cloud/cli/pkg/http"
	"github.com/virak-cloud/cli/pkg/http/responses"
)

type NetworkObjectResult struct {
	NetworkObjects []attr.Value
	InstanceIP     string
	Diags          diag.Diagnostics
}

func BuildNetworkObjects(networks []responses.InstanceNetwork) NetworkObjectResult {
	result := NetworkObjectResult{
		NetworkObjects: make([]attr.Value, 0),
	}

	if len(networks) > 1 {
		sort.Slice(networks, func(i, j int) bool {
			if networks[i].IsDefault != networks[j].IsDefault {
				return networks[i].IsDefault && !networks[j].IsDefault
			}
			return networks[i].Network.ID < networks[j].Network.ID
		})
	}

	seenNetworks := make(map[string]bool)
	for _, networkInstance := range networks {
		networkID := networkInstance.Network.ID
		if seenNetworks[networkID] {
			continue
		}
		seenNetworks[networkID] = true

		networkObj, objDiags := types.ObjectValue(
			map[string]attr.Type{
				"network_id":    types.StringType,
				"ip_address":    types.StringType,
				"mac_address":   types.StringType,
				"is_default":    types.BoolType,
				"attachment_id": types.StringType,
			},
			map[string]attr.Value{
				"network_id":    types.StringValue(networkInstance.Network.ID),
				"ip_address":    types.StringValue(networkInstance.IPAddress),
				"mac_address":   types.StringValue(networkInstance.MACAddress),
				"is_default":    types.BoolValue(networkInstance.IsDefault),
				"attachment_id": types.StringValue(networkInstance.ID),
			},
		)
		if objDiags.HasError() {
			result.Diags.Append(objDiags...)
			continue
		}
		result.NetworkObjects = append(result.NetworkObjects, networkObj)

		if networkInstance.IsDefault {
			result.InstanceIP = networkInstance.IPAddress
		} else if result.InstanceIP == "" {
			result.InstanceIP = networkInstance.IPAddress
		}
	}

	return result
}

func GetInstanceNetworks(client *http.Client, zoneID, instanceID string) ([]responses.InstanceNetwork, error) {
	networksResp, err := client.ListNetworks(zoneID)
	if err != nil {
		return nil, err
	}

	var networkInstancesResp responses.InstanceNetworkListResponse
	for _, network := range networksResp.Data {
		instanceNetworksResp, err := client.ListNetworkInstances(zoneID, network.ID, instanceID)
		if err != nil {
			continue
		}
		networkInstancesResp.Data = append(networkInstancesResp.Data, instanceNetworksResp.Data...)
	}

	filtered := make([]responses.InstanceNetwork, 0, len(networkInstancesResp.Data))
	for _, ni := range networkInstancesResp.Data {
		if ni.InstanceID == instanceID {
			filtered = append(filtered, ni)
		}
	}

	return filtered, nil
}

func GetNetworkInstances(client *http.Client, zoneID, networkID string) ([]responses.InstanceNetwork, error) {
	instancesResp, err := client.ListInstances(zoneID)
	if err != nil {
		return nil, err
	}

	var networkInstancesResp responses.InstanceNetworkListResponse
	for _, instance := range instancesResp.Data {
		instanceNetworksResp, err := client.ListNetworkInstances(zoneID, networkID, instance.ID)
		if err != nil {
			continue
		}
		networkInstancesResp.Data = append(networkInstancesResp.Data, instanceNetworksResp.Data...)
	}

	filtered := make([]responses.InstanceNetwork, 0, len(networkInstancesResp.Data))
	for _, ni := range networkInstancesResp.Data {
		if ni.Network.ID == networkID {
			filtered = append(filtered, ni)
		}
	}

	return filtered, nil
}

func ConnectNetworks(client *http.Client, zoneID, instanceID string, networkIDs []string, diags *diag.Diagnostics) ([]responses.InstanceNetwork, string) {
	connectedNetworks := make(map[string]bool)
	var instanceIP string
	var allAttachments []responses.InstanceNetwork

	for _, networkID := range networkIDs {
		if networkID == "" {
			continue
		}

		_, err := client.ConnectInstanceToNetwork(zoneID, networkID, instanceID)
		if err != nil {
			diags.AddError("Client Error", fmt.Sprintf("Unable to connect instance %s to network %s, got error: %s", instanceID, networkID, err))
			return nil, ""
		}

		attachment, err := WaitForNetworkConnection(client, zoneID, networkID, instanceID, MaxNetworkConnectionRetries, DefaultNetworkPollInterval)
		if err != nil {
			diags.AddError("Network Connection Timeout", fmt.Sprintf("Instance '%s' connection to network '%s' was not established within timeout. Error: %s", instanceID, networkID, err))
			return nil, ""
		}

		if attachment != nil {
			connectedNetworks[networkID] = true
			allAttachments = append(allAttachments, *attachment)

			if attachment.IsDefault {
				instanceIP = attachment.IPAddress
			} else if instanceIP == "" {
				instanceIP = attachment.IPAddress
			}
		}
	}

	return allAttachments, instanceIP
}

func DisconnectNetworks(client *http.Client, zoneID, instanceID string, networkIDs []string, skipDefault bool, diags *diag.Diagnostics) {
	for _, networkID := range networkIDs {
		if networkID == "" {
			continue
		}

		networkInstancesResp, err := client.ListNetworkInstances(zoneID, networkID, instanceID)
		if err != nil {
			diags.AddError("Client Error", fmt.Sprintf("Unable to list network instances for network %s, got error: %s", networkID, err))
			continue
		}

		var instanceNetworkID string
		var isDefaultNIC bool
		for _, ni := range networkInstancesResp.Data {
			if ni.InstanceID == instanceID && ni.Network.ID == networkID {
				instanceNetworkID = ni.ID
				isDefaultNIC = ni.IsDefault
				break
			}
		}

		if instanceNetworkID == "" {
			diags.AddWarning("Network Not Found", fmt.Sprintf("Unable to find instance network ID for network %s and instance %s", networkID, instanceID))
			continue
		}

		if skipDefault && isDefaultNIC {
			continue
		}

		_, err = client.DisconnectInstanceFromNetwork(zoneID, networkID, instanceID, instanceNetworkID)
		if err != nil {
			diags.AddError("Client Error", fmt.Sprintf("Unable to disconnect instance %s from network %s, got error: %s", instanceID, networkID, err))
			continue
		}

		err = WaitForNetworkDisconnection(client, zoneID, networkID, instanceID, instanceNetworkID, MaxNetworkConnectionRetries, DefaultNetworkPollInterval)
		if err != nil {
			diags.AddWarning("Network Disconnection Timeout", fmt.Sprintf("Instance '%s' disconnection from network '%s' was not verified within timeout. Error: %s", instanceID, networkID, err))
		}
	}
}

func FindDefaultNetwork(networks []responses.InstanceNetwork) *responses.InstanceNetwork {
	for i := range networks {
		if networks[i].IsDefault {
			return &networks[i]
		}
	}
	return nil
}

func GetNetworkObjectType() types.ObjectType {
	return types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"network_id":    types.StringType,
			"ip_address":    types.StringType,
			"mac_address":   types.StringType,
			"is_default":    types.BoolType,
			"attachment_id": types.StringType,
		},
	}
}

func CreateNetworksList(networkObjects []attr.Value, diags *diag.Diagnostics) types.List {
	networksList, listDiags := types.ListValue(
		GetNetworkObjectType(),
		networkObjects,
	)
	diags.Append(listDiags...)
	return networksList
}

func VerifyNetworkDisconnected(client *http.Client, zoneID, networkID string) (bool, error) {
	instances, err := GetNetworkInstances(client, zoneID, networkID)
	if err != nil {
		return false, err
	}

	for _, networkInstance := range instances {
		if networkInstance.Network.ID != networkID {
			continue
		}
		if !networkInstance.IsDefault {
			return false, nil
		}
	}

	return true, nil
}
