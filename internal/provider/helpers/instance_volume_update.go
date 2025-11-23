package helpers

import (
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/virak-cloud/cli/pkg/http"
	"github.com/virak-cloud/terraform-provider-virak/internal/models"
)

// UpdateInstanceVolumes handles volume creation, attachment, detachment, and deletion updates for an instance
func UpdateInstanceVolumes(client *http.Client, zoneID, instanceID string, planVolumes, stateVolumes []models.VolumeSpec, diags *diag.Diagnostics) types.List {
	stateVolumeNames := make(map[string]models.VolumeSpec)
	for _, v := range stateVolumes {
		if !v.Name.IsNull() && v.Name.ValueString() != "" {
			stateVolumeNames[v.Name.ValueString()] = v
		}
	}
	planVolumeNames := make(map[string]models.VolumeSpec)
	for _, v := range planVolumes {
		if !v.Name.IsNull() && v.Name.ValueString() != "" {
			planVolumeNames[v.Name.ValueString()] = v
		}
	}

	// Read current instance to verify attachments
	instanceResp, err := client.ShowInstance(zoneID, instanceID)
	if err != nil {
		diags.AddError("Client Error", fmt.Sprintf("Unable to read instance for volume updates, got error: %s", err))
		return types.ListNull(types.StringType)
	}
	isAttached := func(volumeID string) bool {
		for _, v := range instanceResp.Data.DataVolumes {
			if s, ok := v.(string); ok && s == volumeID {
				return true
			}
		}
		return false
	}

	// Detach and delete removed volumes
	removedVolumeNames := make([]string, 0)
	for name := range stateVolumeNames {
		if _, ok := planVolumeNames[name]; !ok {
			removedVolumeNames = append(removedVolumeNames, name)
		}
	}
	DetachAndDeleteRemovedVolumes(client, zoneID, instanceID, removedVolumeNames, isAttached, diags)
	if diags.HasError() {
		return types.ListNull(types.StringType)
	}

	// Create/attach added volumes
	addedVolumes := make([]models.VolumeSpec, 0)
	for name, spec := range planVolumeNames {
		if _, ok := stateVolumeNames[name]; !ok {
			addedVolumes = append(addedVolumes, spec)
		}
	}
	nameToID := CreateAndAttachNewVolumes(client, zoneID, instanceID, addedVolumes, diags)
	if diags.HasError() {
		return types.ListNull(types.StringType)
	}

	// Build VolumeIDs from plan volumes
	return BuildVolumeIDList(planVolumes, nameToID)
}

// DetachAndDeleteRemovedVolumes detaches and deletes volumes that were removed from the plan
func DetachAndDeleteRemovedVolumes(client *http.Client, zoneID, instanceID string, removedVolumeNames []string, isAttached func(string) bool, diags *diag.Diagnostics) {
	// Helper to find volume by name
	findVolumeByName := func(zoneID, name string) (string, string, bool, error) {
		vols, err := client.ListInstanceVolumes(zoneID)
		if err != nil {
			return "", "", false, err
		}
		for _, vol := range vols.Data {
			if vol.Name == name {
				return vol.ID, vol.Status, true, nil
			}
		}
		return "", "", false, nil
	}

	for _, name := range removedVolumeNames {
		volID, volStatus, found, ferr := findVolumeByName(zoneID, name)
		if ferr != nil {
			diags.AddError("Client Error", fmt.Sprintf("Unable to list volumes for removal of '%s', got error: %s", name, ferr))
			return
		}
		if !found {
			continue
		}
		if isAttached(volID) {
			_, derr := client.DetachInstanceVolume(zoneID, volID, instanceID)
			if derr != nil {
				diags.AddError("Volume Detach Failed", fmt.Sprintf("Failed to detach volume '%s' (%s) from instance '%s': %s", name, volID, instanceID, derr))
				return
			}
			detached := false
			for i := 0; i < 60; i++ {
				time.Sleep(5 * time.Second)
				ir, ierr := client.ShowInstance(zoneID, instanceID)
				if ierr != nil {
					diags.AddError("Client Error", fmt.Sprintf("Unable to verify detachment of volume '%s': %s", volID, ierr))
					return
				}
				foundID := false
				for _, v := range ir.Data.DataVolumes {
					if s, ok := v.(string); ok && s == volID {
						foundID = true
						break
					}
				}
				if !foundID {
					detached = true
					break
				}
			}
			if !detached {
				diags.AddError("Volume Detach Timeout", fmt.Sprintf("Detachment of volume '%s' (%s) from instance '%s' did not complete in time", name, volID, instanceID))
				return
			}
		}
		if volStatus != VolumeStatusAllocated {
			statusOk := false
			for i := 0; i < 30; i++ {
				time.Sleep(time.Second)
				_, vs, ok, lerr := findVolumeByName(zoneID, name)
				if lerr != nil {
					diags.AddError("Client Error", fmt.Sprintf("Unable to verify status for volume '%s' before delete: %s", name, lerr))
					return
				}
				if ok && vs == VolumeStatusAllocated {
					statusOk = true
					break
				}
			}
			if !statusOk {
				diags.AddError("Volume Delete Blocked", fmt.Sprintf("Volume '%s' is not in ALLOCATED status and cannot be deleted", name))
				return
			}
		}
		if _, derr := client.DeleteInstanceVolume(zoneID, volID); derr != nil {
			diags.AddError("Volume Delete Failed", fmt.Sprintf("Failed to delete volume '%s' (%s): %s", name, volID, derr))
			return
		}
	}
}

// CreateAndAttachNewVolumes creates and attaches new volumes added to the plan
func CreateAndAttachNewVolumes(client *http.Client, zoneID, instanceID string, addedVolumes []models.VolumeSpec, diags *diag.Diagnostics) map[string]string {
	nameToID := make(map[string]string)
	for _, v := range addedVolumes {
		nameToID[v.Name.ValueString()] = ""
	}
	for _, spec := range addedVolumes {
		name := spec.Name.ValueString()
		initial, ierr := client.ListInstanceVolumes(zoneID)
		if ierr != nil {
			diags.AddError("Client Error", fmt.Sprintf("Unable to list volumes before creating '%s': %s", name, ierr))
			return nil
		}
		existing := make(map[string]struct{})
		for _, vol := range initial.Data {
			existing[vol.ID] = struct{}{}
		}
		_, cerr := client.CreateInstanceVolume(
			zoneID,
			spec.ServiceOfferingID.ValueString(),
			int(spec.Size.ValueInt64()),
			name,
		)
		if cerr != nil {
			diags.AddError("Volume Creation Failed", fmt.Sprintf("Failed to create volume '%s': %s", name, cerr))
			return nil
		}
		newID := ""
		for i := 0; i < 10; i++ {
			time.Sleep(time.Second)
			latest, lerr := client.ListInstanceVolumes(zoneID)
			if lerr != nil {
				diags.AddError("Client Error", fmt.Sprintf("Unable to list volumes after creating '%s': %s", name, lerr))
				return nil
			}
			for _, vol := range latest.Data {
				if _, ok := existing[vol.ID]; !ok && vol.Name == name {
					newID = vol.ID
					break
				}
			}
			if newID != "" {
				break
			}
		}
		if newID == "" {
			diags.AddError("Volume ID not found", fmt.Sprintf("Unable to discover ID for newly created volume '%s'", name))
			return nil
		}
		_, aerr := client.AttachInstanceVolume(zoneID, newID, instanceID)
		if aerr != nil {
			diags.AddError("Volume Attachment Failed", fmt.Sprintf("Failed to attach new volume '%s' (%s) to instance '%s': %s", name, newID, instanceID, aerr))
			return nil
		}
		nameToID[name] = newID
	}
	return nameToID
}

// BuildVolumeIDList builds the volume ID list from plan volumes and name-to-ID mapping
func BuildVolumeIDList(planVolumes []models.VolumeSpec, nameToID map[string]string) types.List {
	// Rebuild VolumeIDs from plan volumes
	volIDs := make([]types.String, 0, len(planVolumes))
	for _, v := range planVolumes {
		n := v.Name.ValueString()
		id := nameToID[n]
		if id == "" {
			// Try to find existing volume ID if not in nameToID
			// This is a fallback for volumes that were already attached
			// In practice, this should be populated by the calling code
			continue
		}
		if id != "" {
			volIDs = append(volIDs, types.StringValue(id))
		}
	}
	values := make([]attr.Value, len(volIDs))
	for i, v := range volIDs {
		values[i] = v
	}
	var vdiags diag.Diagnostics
	volumeList, vdiags := types.ListValue(types.StringType, values)
	// Note: We ignore vdiags here as this is internal and should not fail
	_ = vdiags
	return volumeList
}
