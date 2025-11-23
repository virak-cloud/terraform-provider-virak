package helpers

import (
	"fmt"
	"sort"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/virak-cloud/cli/pkg/http"
	"github.com/virak-cloud/cli/pkg/http/responses"
)

// UpdateInstanceNetworks handles network attachment/detachment updates for an instance
func UpdateInstanceNetworks(client *http.Client, zoneID, instanceID string, planNetworkIDs, stateNetworkIDs map[string]bool, diags *diag.Diagnostics) {
	// Find default network ID before making changes
	defaultNetworkID := FindDefaultNetworkID(client, zoneID, instanceID, getKeys(stateNetworkIDs), diags)
	if diags.HasError() {
		return
	}

	// Collect networks to detach (in state but not in plan)
	networksToDetach := make([]string, 0)
	for networkID := range stateNetworkIDs {
		if !planNetworkIDs[networkID] {
			networksToDetach = append(networksToDetach, networkID)
		}
	}

	// Collect networks to attach (in plan but not in state)
	networksToAttach := make([]string, 0)
	for networkID := range planNetworkIDs {
		if !stateNetworkIDs[networkID] {
			networksToAttach = append(networksToAttach, networkID)
		}
	}

	// Verify default network change is valid
	if defaultNetworkID != "" {
		VerifyDefaultNetworkChange(client, zoneID, instanceID, defaultNetworkID, getKeys(planNetworkIDs), diags)
		if diags.HasError() {
			return
		}
	}

	// Detach networks
	DetachNetworksFromInstance(client, zoneID, instanceID, networksToDetach, defaultNetworkID, diags)
	if diags.HasError() {
		return
	}

	// Attach networks
	AttachNetworksToInstance(client, zoneID, instanceID, networksToAttach, diags)
	if diags.HasError() {
		return
	}
}

// DetachNetworksFromInstance detaches the specified networks from an instance
func DetachNetworksFromInstance(client *http.Client, zoneID, instanceID string, networksToDetach []string, defaultNetworkID string, diags *diag.Diagnostics) {
	for _, networkID := range networksToDetach {
		// Find instance_network_id for this network-instance pair
		networkInstancesResp, err := client.ListNetworkInstances(zoneID, networkID, instanceID)
		if err != nil {
			diags.AddError("Client Error", fmt.Sprintf("Unable to list network instances for network %s, got error: %s", networkID, err))
			return
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
			diags.AddError("Client Error", fmt.Sprintf("Unable to find instance network ID for network %s and instance %s", networkID, instanceID))
			return
		}
		if isDefaultNIC {
			continue
		}
		_, err = client.DisconnectInstanceFromNetwork(zoneID, networkID, instanceID, instanceNetworkID)
		if err != nil {
			diags.AddError("Client Error", fmt.Sprintf("Unable to disconnect instance %s from network %s, got error: %s", instanceID, networkID, err))
			return
		}

		// Wait for disconnection to complete
		for i := 0; i < 30; i++ {
			verifyResp, err := client.ListNetworkInstances(zoneID, networkID, instanceID)
			if err != nil {
				if i == 29 {
					diags.AddError("Client Error", fmt.Sprintf("Unable to verify network disconnection for network %s after 30 retries, got error: %s", networkID, err))
					return
				}
				time.Sleep(time.Second)
				continue
			}

			found := false
			for _, ni := range verifyResp.Data {
				if ni.ID == instanceNetworkID && ni.InstanceID == instanceID {
					found = true
					break
				}
			}

			if !found {
				break
			}

			if i == 29 {
				diags.AddError("Network Disconnection Timeout", fmt.Sprintf("Instance '%s' disconnection from network '%s' was not verified within 30 seconds. The disconnection may still be in progress.", instanceID, networkID))
				return
			}

			time.Sleep(time.Second)
		}
	}
}

// AttachNetworksToInstance attaches the specified networks to an instance
func AttachNetworksToInstance(client *http.Client, zoneID, instanceID string, networksToAttach []string, diags *diag.Diagnostics) {
	for _, networkID := range networksToAttach {
		_, err := client.ConnectInstanceToNetwork(zoneID, networkID, instanceID)
		if err != nil {
			diags.AddError("Client Error", fmt.Sprintf("Unable to connect instance %s to network %s, got error: %s", instanceID, networkID, err))
			return
		}

		// Wait for connection to establish
		var attachment responses.InstanceNetwork
		for i := 0; i < 30; i++ {
			networkInstancesResp, err := client.ListNetworkInstances(zoneID, networkID, instanceID)
			if err != nil {
				if i == 29 {
					diags.AddError("Client Error", fmt.Sprintf("Unable to verify network connection for network %s after 30 retries, got error: %s", networkID, err))
					return
				}
				time.Sleep(time.Second)
				continue
			}

			for _, ni := range networkInstancesResp.Data {
				if ni.InstanceID == instanceID && ni.Network.ID == networkID {
					attachment = ni
					break
				}
			}

			if attachment.ID != "" {
				break
			}

			if i == 29 {
				diags.AddError("Network Connection Timeout", fmt.Sprintf("Instance '%s' connection to network '%s' was not established within 30 seconds. The connection may still be provisioning.", instanceID, networkID))
				return
			}

			time.Sleep(time.Second)
		}
	}
}

// FindDefaultNetworkID finds the ID of the default network for an instance
func FindDefaultNetworkID(client *http.Client, zoneID, instanceID string, networkIDs []string, diags *diag.Diagnostics) string {
	for _, networkID := range networkIDs {
		networkInstancesResp, err := client.ListNetworkInstances(zoneID, networkID, instanceID)
		if err != nil {
			continue // Skip on error, we'll check other networks
		}
		for _, ni := range networkInstancesResp.Data {
			if ni.InstanceID == instanceID && ni.Network.ID == networkID && ni.IsDefault {
				return networkID
			}
		}
	}
	return ""
}

// VerifyDefaultNetworkChange validates that disconnecting the default network is safe
func VerifyDefaultNetworkChange(client *http.Client, zoneID, instanceID, defaultNetworkID string, remainingNetworks []string, diags *diag.Diagnostics) {
	hasOtherDefault := false
	for _, networkID := range remainingNetworks {
		if networkID == defaultNetworkID {
			continue
		}
		networkInstancesResp, err := client.ListNetworkInstances(zoneID, networkID, instanceID)
		if err != nil {
			continue
		}
		for _, ni := range networkInstancesResp.Data {
			if ni.InstanceID == instanceID && ni.Network.ID == networkID && ni.IsDefault {
				hasOtherDefault = true
				break
			}
		}
		if hasOtherDefault {
			break
		}
	}

	if !hasOtherDefault && len(remainingNetworks) > 0 {
		firstRemainingNetwork := remainingNetworks[0]
		networkInstancesResp, err := client.ListNetworkInstances(zoneID, firstRemainingNetwork, instanceID)
		if err == nil {
			for _, ni := range networkInstancesResp.Data {
				if ni.InstanceID == instanceID && ni.Network.ID == firstRemainingNetwork {
					if !ni.IsDefault {
						diags.AddWarning(
							"Default Network Change",
							fmt.Sprintf("Disconnecting default network '%s'. The first remaining network '%s' will need to become the default network.", defaultNetworkID, firstRemainingNetwork),
						)
					}
					break
				}
			}
		}
	}
}

// RefreshInstanceNetworks refreshes the network state for an instance and returns the updated networks list and instance IP
func RefreshInstanceNetworks(client *http.Client, zoneID, instanceID string, diags *diag.Diagnostics) (types.List, string) {
	networksResp, err := client.ListNetworks(zoneID)
	if err != nil {
		fmt.Printf("Warning: Unable to list networks in zone %s for instance %s, got error: %s. Keeping existing networks state.\n", zoneID, instanceID, err)
		return types.ListNull(GetNetworkObjectType()), ""
	}

	var networkInstancesResp responses.InstanceNetworkListResponse
	for _, network := range networksResp.Data {
		instanceNetworksResp, err := client.ListNetworkInstances(zoneID, network.ID, instanceID)
		if err != nil {
			continue
		}
		networkInstancesResp.Data = append(networkInstancesResp.Data, instanceNetworksResp.Data...)
	}

	networkObjects := make([]attr.Value, 0)
	var instanceIP string
	filtered := make([]responses.InstanceNetwork, 0, len(networkInstancesResp.Data))
	for _, ni := range networkInstancesResp.Data {
		if ni.InstanceID == instanceID {
			filtered = append(filtered, ni)
		}
	}
	if len(filtered) > 1 {
		sort.Slice(filtered, func(i, j int) bool {
			if filtered[i].IsDefault != filtered[j].IsDefault {
				return filtered[i].IsDefault && !filtered[j].IsDefault
			}
			return filtered[i].Network.ID < filtered[j].Network.ID
		})
	}
	seenNetworks := make(map[string]bool)
	for _, networkInstance := range filtered {
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
			diags.Append(objDiags...)
			continue
		}
		networkObjects = append(networkObjects, networkObj)

		if networkInstance.IsDefault {
			instanceIP = networkInstance.IPAddress
		} else if instanceIP == "" {
			instanceIP = networkInstance.IPAddress
		}
	}

	fmt.Printf("Found %d networks connected to instance %s\n", len(networkObjects), instanceID)

	networksList, listDiags := types.ListValue(
		types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"network_id":    types.StringType,
				"ip_address":    types.StringType,
				"mac_address":   types.StringType,
				"is_default":    types.BoolType,
				"attachment_id": types.StringType,
			},
		},
		networkObjects,
	)
	diags.Append(listDiags...)

	return networksList, instanceIP
}

// getKeys returns the keys of a map[string]bool as []string
func getKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
