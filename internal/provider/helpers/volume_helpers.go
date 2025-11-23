package helpers

import (
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/virak-cloud/cli/pkg/http"
	"github.com/virak-cloud/cli/pkg/http/responses"
	"github.com/virak-cloud/terraform-provider-virak/internal/models"
)

type VolumeInfo struct {
	ID     string
	Status string
	Found  bool
}

func FindVolumeByName(client *http.Client, zoneID, name string) (VolumeInfo, error) {
	vols, err := client.ListInstanceVolumes(zoneID)
	if err != nil {
		return VolumeInfo{}, err
	}

	for _, vol := range vols.Data {
		if vol.Name == name {
			return VolumeInfo{
				ID:     vol.ID,
				Status: vol.Status,
				Found:  true,
			}, nil
		}
	}

	return VolumeInfo{Found: false}, nil
}

func WaitForVolumeStatus(client *http.Client, zoneID, volumeID, targetStatus string, maxRetries int, interval time.Duration) error {
	checkFunc := func() (bool, error) {
		vols, err := client.ListInstanceVolumes(zoneID)
		if err != nil {
			return false, err
		}

		for _, vol := range vols.Data {
			if vol.ID == volumeID {
				return vol.Status == targetStatus, nil
			}
		}
		return false, nil
	}

	return PollUntilCondition(checkFunc, maxRetries, interval, fmt.Sprintf("Volume '%s' did not reach status '%s' within timeout", volumeID, targetStatus))
}

func IsVolumeAttached(instanceResp *responses.InstanceShowResponse, volumeID string) bool {
	for _, v := range instanceResp.Data.DataVolumes {
		if s, ok := v.(string); ok && s == volumeID {
			return true
		}
	}
	return false
}

func CreateAndAttachVolume(client *http.Client, zoneID, instanceID string, volSpec models.VolumeSpec, existingIDs map[string]struct{}, diags *diag.Diagnostics) (string, error) {
	_, err := client.CreateInstanceVolume(
		zoneID,
		volSpec.ServiceOfferingID.ValueString(),
		int(volSpec.Size.ValueInt64()),
		volSpec.Name.ValueString(),
	)
	if err != nil {
		diags.AddError(
			"Volume Creation Failed",
			fmt.Sprintf("Failed to create volume '%s' in zone '%s' with service offering '%s' and size %d GB. Error: %s", volSpec.Name.ValueString(), zoneID, volSpec.ServiceOfferingID.ValueString(), volSpec.Size.ValueInt64(), err),
		)
		return "", err
	}

	newVolID, err := FindNewVolumeID(client, zoneID, existingIDs, volSpec.Name.ValueString(), MaxVolumeStatusRetries, time.Second)
	if err != nil {
		diags.AddError("Volume ID not found", fmt.Sprintf("Unable to find the created volume ID for '%s': %s", volSpec.Name.ValueString(), err))
		return "", err
	}

	_, err = client.AttachInstanceVolume(zoneID, newVolID, instanceID)
	if err != nil {
		diags.AddError(
			"Volume Attachment Failed",
			fmt.Sprintf("Failed to attach volume '%s' to instance '%s'. Error: %s", newVolID, instanceID, err),
		)
		return "", err
	}

	err = WaitForVolumeAttachment(client, zoneID, instanceID, newVolID, MaxVolumeAttachmentRetries, DefaultVolumePollInterval)
	if err != nil {
		diags.AddError("Volume Attachment Timeout", fmt.Sprintf("Volume '%s' attachment to instance '%s' did not complete within timeout. Error: %s", newVolID, instanceID, err))
		return "", err
	}

	return newVolID, nil
}

func CreateAndAttachVolumes(client *http.Client, zoneID, instanceID string, volumeSpecs []models.VolumeSpec, diags *diag.Diagnostics) ([]types.String, error) {
	volumeIDs := make([]types.String, 0, len(volumeSpecs))

	initialVolumes, err := client.ListInstanceVolumes(zoneID)
	if err != nil {
		diags.AddError(
			"API Communication Error",
			fmt.Sprintf("Failed to list existing volumes before volume creation. Error: %s", err),
		)
		return nil, err
	}

	existingIDs := make(map[string]struct{})
	for _, vol := range initialVolumes.Data {
		existingIDs[vol.ID] = struct{}{}
	}

	for _, volSpec := range volumeSpecs {
		volID, err := CreateAndAttachVolume(client, zoneID, instanceID, volSpec, existingIDs, diags)
		if err != nil {
			return nil, err
		}
		existingIDs[volID] = struct{}{}
		volumeIDs = append(volumeIDs, types.StringValue(volID))
	}

	return volumeIDs, nil
}

func DetachVolume(client *http.Client, zoneID, instanceID, volumeID string, diags *diag.Diagnostics) error {
	_, err := client.DetachInstanceVolume(zoneID, volumeID, instanceID)
	if err != nil {
		diags.AddWarning(
			"Volume Detachment Warning",
			fmt.Sprintf("Failed to detach volume '%s' from instance '%s'. Error: %s", volumeID, instanceID, err),
		)
		return err
	}

	err = WaitForVolumeDetachment(client, zoneID, instanceID, volumeID, 60, DefaultVolumePollInterval)
	if err != nil {
		diags.AddWarning(
			"Volume Detachment Timeout",
			fmt.Sprintf("Volume '%s' detachment from instance '%s' did not complete within timeout. Error: %s", volumeID, instanceID, err),
		)
	}

	return nil
}

func DetachAndDeleteVolume(client *http.Client, zoneID, instanceID, volumeName string, diags *diag.Diagnostics) error {
	volInfo, err := FindVolumeByName(client, zoneID, volumeName)
	if err != nil {
		diags.AddError("Client Error", fmt.Sprintf("Unable to list volumes for removal of '%s', got error: %s", volumeName, err))
		return err
	}

	if !volInfo.Found {
		return nil
	}

	instanceResp, err := client.ShowInstance(zoneID, instanceID)
	if err != nil {
		diags.AddError("Client Error", fmt.Sprintf("Unable to read instance for volume removal, got error: %s", err))
		return err
	}

	if IsVolumeAttached(instanceResp, volInfo.ID) {
		err = DetachVolume(client, zoneID, instanceID, volInfo.ID, diags)
		if err != nil {
			return err
		}
	}

	if volInfo.Status != VolumeStatusAllocated {
		err = WaitForVolumeStatus(client, zoneID, volInfo.ID, VolumeStatusAllocated, 30, time.Second)
		if err != nil {
			diags.AddError("Volume Delete Blocked", fmt.Sprintf("Volume '%s' is not in ALLOCATED status and cannot be deleted. Error: %s", volumeName, err))
			return err
		}
	}

	_, err = client.DeleteInstanceVolume(zoneID, volInfo.ID)
	if err != nil {
		diags.AddError("Volume Delete Failed", fmt.Sprintf("Failed to delete volume '%s' (%s): %s", volumeName, volInfo.ID, err))
		return err
	}

	return nil
}

func DetachAndDeleteVolumes(client *http.Client, zoneID, instanceID string, volumeNames []string, diags *diag.Diagnostics) error {
	for _, name := range volumeNames {
		err := DetachAndDeleteVolume(client, zoneID, instanceID, name, diags)
		if err != nil {
			return err
		}
	}
	return nil
}

func WaitForVolumeAttachmentCompletion(client *http.Client, zoneID, volumeID string, maxRetries int, interval time.Duration) error {
	checkFunc := func() (bool, error) {
		volumesResp, err := client.ListInstanceVolumes(zoneID)
		if err != nil {
			return false, err
		}

		for _, vol := range volumesResp.Data {
			if vol.ID == volumeID {
				return vol.Status != VolumeStatusAttaching, nil
			}
		}
		return false, nil
	}

	return PollUntilCondition(checkFunc, maxRetries, interval, fmt.Sprintf("Volume '%s' remained in ATTACHING status for too long", volumeID))
}

func GetAttachedVolumeIDs(instanceResp *responses.InstanceShowResponse) []string {
	var attachedVolumeIDs []string
	for _, v := range instanceResp.Data.DataVolumes {
		if s, ok := v.(string); ok {
			attachedVolumeIDs = append(attachedVolumeIDs, s)
		}
	}
	return attachedVolumeIDs
}
