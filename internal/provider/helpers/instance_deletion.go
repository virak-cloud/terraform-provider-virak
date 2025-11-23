package helpers

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/virak-cloud/cli/pkg/http"
)

// DetachAllVolumes detaches all volumes from an instance before deletion
func DetachAllVolumes(client *http.Client, zoneID, instanceID string, diags *diag.Diagnostics) {
	readResp, err := GetInstanceDetails(client, zoneID, instanceID)
	if err != nil {
		diags.AddError("Client Error", fmt.Sprintf("Unable to read instance before volume detachment, got error: %s", err))
		return
	}

	attachedVolumeIDs := GetAttachedVolumeIDs(readResp)

	for _, volID := range attachedVolumeIDs {
		volumesResp, err := client.ListInstanceVolumes(zoneID)
		if err == nil {
			for _, vol := range volumesResp.Data {
				if vol.ID == volID {
					if vol.Status == VolumeStatusAttaching {
						err := WaitForVolumeAttachmentCompletion(client, zoneID, volID, 120, DefaultVolumePollInterval)
						if err != nil {
							return
						}
					}
					break
				}
			}
		}

		err = DetachVolume(client, zoneID, instanceID, volID, diags)
		if err != nil {
			return
		}
	}
}

// DisconnectAllNetworks disconnects all networks from an instance before deletion
func DisconnectAllNetworks(client *http.Client, zoneID, instanceID string, diags *diag.Diagnostics) {
	networks, err := GetInstanceNetworks(client, zoneID, instanceID)
	if err != nil {
		diags.AddWarning(
			"Network Listing Failed",
			fmt.Sprintf("Unable to list networks in zone %s before instance deletion. Error: %s. Proceeding with instance deletion - networks may need manual cleanup.", zoneID, err),
		)
		return
	}

	networkIDs := make([]string, 0, len(networks))
	for _, ni := range networks {
		networkIDs = append(networkIDs, ni.Network.ID)
	}

	DisconnectNetworks(client, zoneID, instanceID, networkIDs, true, diags)
}
