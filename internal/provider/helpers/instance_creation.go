package helpers

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/virak-cloud/cli/pkg/http"
	"github.com/virak-cloud/cli/pkg/http/responses"
	"github.com/virak-cloud/terraform-provider-virak/internal/models"
)

// CreateInstanceCore creates a new instance and returns its ID
func CreateInstanceCore(client *http.Client, data *models.InstanceResourceModel, networkIDs []string, existingIDs map[string]struct{}, diags *diag.Diagnostics) (string, error) {
	_, err := client.CreateInstance(
		data.ZoneID.ValueString(),
		data.ServiceOfferingID.ValueString(),
		data.VMImageID.ValueString(),
		networkIDs,
		data.Name.ValueString(),
	)
	if err != nil {
		diags.AddError(
			"Instance Creation Failed",
			fmt.Sprintf("Failed to create instance '%s' in zone %s. This may be due to invalid service offering, VM image, network IDs, or insufficient resources. Error: %s", data.Name.ValueString(), data.ZoneID.ValueString(), err),
		)
		return "", err
	}

	newInstanceID, err := FindNewInstanceID(client, data.ZoneID.ValueString(), existingIDs, data.Name.ValueString(), MaxInstanceStatusRetries, DefaultPollInterval)
	if err != nil {
		diags.AddError(
			"Instance Creation Timeout",
			fmt.Sprintf("Instance '%s' was submitted for creation but could not be found in the API after timeout. The instance may still be provisioning in the background.", data.Name.ValueString()),
		)
		return "", err
	}

	return newInstanceID, nil
}

// WaitForInstanceReady waits for an instance to reach UP status
func WaitForInstanceReady(client *http.Client, zoneID, instanceID string, diags *diag.Diagnostics) error {
	status, err := WaitForInstanceStatus(client, zoneID, instanceID, []string{InstanceStatusUP}, MaxInstanceStatusRetries, DefaultPollInterval)
	if err != nil {
		diags.AddError(
			"Instance Not Ready",
			fmt.Sprintf("Instance '%s' was created but did not reach 'UP' status within timeout. Current status: %s. The instance may still be provisioning.", instanceID, status),
		)
		return err
	}
	return nil
}

// SetupInstanceNetworks sets up network connections for a newly created instance
func SetupInstanceNetworks(client *http.Client, data *models.InstanceResourceModel, networkIDs []string, diags *diag.Diagnostics) (types.List, string) {
	filtered := make([]responses.InstanceNetwork, 0)
	connectedNetworks := make(map[string]bool)

	for _, networkID := range networkIDs {
		attachment, err := WaitForNetworkConnection(client, data.ZoneID.ValueString(), networkID, data.ID.ValueString(), MaxNetworkConnectionRetries, DefaultNetworkPollInterval)
		if err != nil {
			diags.AddError("Network Connection Timeout", fmt.Sprintf("Instance '%s' was created but network '%s' connection was not established within timeout. The connection may still be provisioning.", data.ID.ValueString(), networkID))
			return types.ListNull(GetNetworkObjectType()), ""
		}

		if attachment != nil {
			connectedNetworks[networkID] = true
			filtered = append(filtered, *attachment)
		}
	}

	if len(connectedNetworks) != len(networkIDs) {
		var missingNetworks []string
		for _, networkID := range networkIDs {
			if !connectedNetworks[networkID] {
				missingNetworks = append(missingNetworks, networkID)
			}
		}
		diags.AddError("Network Connection Verification Failed", fmt.Sprintf("Not all networks were connected. Missing networks: %v", missingNetworks))
		return types.ListNull(GetNetworkObjectType()), ""
	}

	result := BuildNetworkObjects(filtered)
	diags.Append(result.Diags...)
	if diags.HasError() {
		return types.ListNull(GetNetworkObjectType()), ""
	}

	networksList := CreateNetworksList(result.NetworkObjects, diags)
	return networksList, result.InstanceIP
}

// PopulateInstanceState populates computed fields for an instance
func PopulateInstanceState(client *http.Client, data *models.InstanceResourceModel, diags *diag.Diagnostics) {
	readResp, err := GetInstanceDetails(client, data.ZoneID.ValueString(), data.ID.ValueString())
	if err != nil {
		diags.AddError("Client Error", fmt.Sprintf("Unable to read instance after creation, got error: %s", err))
		return
	}

	status := readResp.Data.Status
	if status == "" {
		status = InstanceStatusUP
	}
	data.Status = types.StringValue(status)

	password := readResp.Data.Password
	if password == "" {
		password = "unknown"
	}
	data.Password = types.StringValue(password)

	username := readResp.Data.Username
	if username == "" {
		username = "unknown"
	}
	data.Username = types.StringValue(username)
}
