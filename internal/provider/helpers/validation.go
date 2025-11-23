package helpers

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/virak-cloud/cli/pkg/http"
)

func ValidateInstanceName(client *http.Client, zoneID, name string, diags *diag.Diagnostics) bool {
	if client == nil {
		return true
	}

	instances, err := client.ListInstances(zoneID)
	if err != nil {
		return true
	}

	for _, inst := range instances.Data {
		if inst.Name == name {
			diags.AddError(
				"Invalid Configuration",
				fmt.Sprintf("Instance name '%s' already exists in zone '%s'. Please choose a unique name.", name, zoneID),
			)
			return false
		}
	}
	return true
}

func ValidateNetworkName(client *http.Client, zoneID, name string, diags *diag.Diagnostics) bool {
	if client == nil {
		return true
	}

	networks, err := client.ListNetworks(zoneID)
	if err != nil {
		return true
	}

	for _, network := range networks.Data {
		if network.Name == name {
			diags.AddError(
				"Invalid Configuration",
				fmt.Sprintf("Network name '%s' already exists in zone '%s'. Please choose a unique name.", name, zoneID),
			)
			return false
		}
	}
	return true
}

func ValidateNetworkOfferingType(client *http.Client, zoneID, networkOfferingID, networkType string, diags *diag.Diagnostics) bool {
	if client == nil {
		return true
	}

	serviceOfferings, err := client.ListNetworkServiceOfferings(zoneID)
	if err != nil {
		return true
	}

	var targetOffering *struct {
		ID   string
		Name string
		Type string
	}

	for _, offering := range serviceOfferings.Data {
		if offering.ID == networkOfferingID {
			targetOffering = &struct {
				ID   string
				Name string
				Type string
			}{
				ID:   offering.ID,
				Name: offering.Name,
				Type: offering.Type,
			}
			break
		}
	}

	if targetOffering == nil {
		diags.AddError(
			"Invalid Network Offering",
			fmt.Sprintf("network offering ID '%s' not found in zone '%s'", networkOfferingID, zoneID),
		)
		return false
	}

	var expectedType string
	var typeDescription string

	switch networkType {
	case "Isolated", "L3":
		expectedType = "Isolated"
		typeDescription = "L3/Isolated"
	case "L2":
		expectedType = "L2"
		typeDescription = "L2"
	default:
		return true
	}

	if targetOffering.Type != expectedType {
		diags.AddError(
			"Invalid Network Offering Type",
			fmt.Sprintf("network offering '%s' is not of type %s (found type: %s)",
				networkOfferingID, typeDescription, targetOffering.Type),
		)
		return false
	}

	return true
}

func ValidateInstanceNameLength(name string, maxLength int, diags *diag.Diagnostics) bool {
	if len(name) > maxLength {
		diags.AddError(
			"Invalid Configuration",
			fmt.Sprintf("Instance name must be %d characters or less", maxLength),
		)
		return false
	}
	return true
}

func ValidateNetworksForInstanceCreation(client *http.Client, zoneID string, networkIDs []string, diags *diag.Diagnostics) bool {
	if client == nil {
		return true
	}

	if len(networkIDs) == 0 {
		return true
	}

	networks, err := client.ListNetworks(zoneID)
	if err != nil {
		diags.AddError(
			"Network Validation Error",
			fmt.Sprintf("Unable to list networks in zone '%s' for validation. Error: %s", zoneID, err),
		)
		return false
	}

	existingNetworkIDs := make(map[string]bool)
	for _, network := range networks.Data {
		existingNetworkIDs[network.ID] = true
	}

	var missingNetworks []string
	for _, networkID := range networkIDs {
		if networkID == "" {
			continue
		}
		if !existingNetworkIDs[networkID] {
			missingNetworks = append(missingNetworks, networkID)
		}
	}

	if len(missingNetworks) > 0 {
		diags.AddError(
			"Invalid Network Configuration",
			fmt.Sprintf("The following network IDs do not exist in zone '%s': %v. Please ensure all networks exist before creating the instance.", zoneID, missingNetworks),
		)
		return false
	}

	return true
}
