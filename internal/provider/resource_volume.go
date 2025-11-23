package provider

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/virak-cloud/cli/pkg/http"
	"github.com/virak-cloud/terraform-provider-virak/internal/models"
	"github.com/virak-cloud/terraform-provider-virak/internal/provider/helpers"
)

// Ensure the implementation satisfies the resource.Resource interface.
var _ resource.Resource = &volumeResource{}

func NewVolumeResource(mutex *sync.Mutex) resource.Resource {
	return &volumeResource{mutex: mutex}
}

type volumeResource struct {
	client *http.Client
	mutex  *sync.Mutex
}

func (r *volumeResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_volume"
}

func (r *volumeResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Virak Cloud volume.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Volume ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"zone_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the zone where the volume resides.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"service_offering_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the service offering for the volume type.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"size": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "Volume size in GB.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the volume.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"instance_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "ID of the instance to attach the volume to.",
			},
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Volume status (e.g., 'ready', 'attached').",
			},
			"attached_instance_id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The ID of the instance the volume is currently attached to.",
			},
		},
	}
}

func (r *volumeResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*http.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	r.client = client
}

type volumeInfo struct {
	ID                 string
	Name               string
	Size               int
	ServiceOfferingID  string
	Status             string
	AttachedInstanceID string
}

func (r *volumeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var data models.VolumeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.validateVolumeServiceOfferingForCreate(data.ZoneID.ValueString(), data.ServiceOfferingID.ValueString(), resp); err != nil {
		return
	}

	if !data.InstanceID.IsNull() && data.InstanceID.ValueString() != "" {
		if err := r.validateInstanceForVolumeAttachment(data.ZoneID.ValueString(), data.InstanceID.ValueString(), resp); err != nil {
			return
		}
	}

	initialVolumes, err := r.client.ListInstanceVolumes(data.ZoneID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"API Communication Error",
			fmt.Sprintf("Failed to list existing volumes in zone %s before creation. Error: %s", data.ZoneID.ValueString(), err),
		)
		return
	}
	existingIDs := make(map[string]struct{})
	for _, vol := range initialVolumes.Data {
		existingIDs[vol.ID] = struct{}{}
	}

	_, err = r.client.CreateInstanceVolume(
		data.ZoneID.ValueString(),
		data.ServiceOfferingID.ValueString(),
		int(data.Size.ValueInt64()),
		data.Name.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Volume Creation Failed",
			fmt.Sprintf("Failed to create volume '%s' in zone %s with service offering %s and size %d GB. Error: %s", data.Name.ValueString(), data.ZoneID.ValueString(), data.ServiceOfferingID.ValueString(), data.Size.ValueInt64(), err),
		)
		return
	}

	newVolumeID, err := helpers.FindNewVolumeID(r.client, data.ZoneID.ValueString(), existingIDs, data.Name.ValueString(), helpers.MaxVolumeStatusRetries, time.Second)
	if err != nil {
		resp.Diagnostics.AddError("Volume ID not found", fmt.Sprintf("Unable to find the created volume ID for '%s': %s", data.Name.ValueString(), err))
		return
	}

	data.ID = types.StringValue(newVolumeID)

	err = helpers.WaitForVolumeStatus(r.client, data.ZoneID.ValueString(), newVolumeID, helpers.VolumeStatusAllocated, 30, time.Second)
	if err != nil {
		resp.Diagnostics.AddError("Volume Status Timeout", fmt.Sprintf("Volume '%s' did not reach ALLOCATED status within timeout. Error: %s", newVolumeID, err))
		return
	}

	data.Status = types.StringValue(helpers.VolumeStatusAllocated)

	if !data.InstanceID.IsNull() && data.InstanceID.ValueString() != "" {
		_, err = r.client.AttachInstanceVolume(
			data.ZoneID.ValueString(),
			newVolumeID,
			data.InstanceID.ValueString(),
		)
		if err != nil {
			resp.Diagnostics.AddError(
				"Volume Attachment Failed",
				fmt.Sprintf("Failed to attach volume '%s' to instance '%s'. Error: %s", newVolumeID, data.InstanceID.ValueString(), err),
			)
			return
		}

		err = helpers.WaitForVolumeAttachmentCompletion(r.client, data.ZoneID.ValueString(), newVolumeID, helpers.MaxVolumeAttachmentRetries, helpers.DefaultVolumePollInterval)
		if err != nil {
			resp.Diagnostics.AddWarning(
				"Volume Attachment Timeout",
				fmt.Sprintf("Volume '%s' attachment to instance '%s' did not complete within timeout. Volume may still be attaching in the background. Error: %s", newVolumeID, data.InstanceID.ValueString(), err),
			)
		}

		volumesResp, err := r.client.ListInstanceVolumes(data.ZoneID.ValueString())
		if err == nil {
			for _, vol := range volumesResp.Data {
				if vol.ID == newVolumeID {
					data.Status = types.StringValue(vol.Status)
					break
				}
			}
		}

		data.AttachedInstanceID = data.InstanceID
	} else {
		data.AttachedInstanceID = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *volumeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data models.VolumeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// List volumes and find by ID
	volumes, err := r.client.ListInstanceVolumes(data.ZoneID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"API Communication Error",
			fmt.Sprintf("Failed to list volumes in zone %s. Error: %s", data.ZoneID.ValueString(), err),
		)
		return
	}

	// Find the volume by ID
	var foundVolume *volumeInfo
	for _, vol := range volumes.Data {
		if vol.ID == data.ID.ValueString() {
			foundVolume = &volumeInfo{
				ID:                 vol.ID,
				Name:               vol.Name,
				Size:               vol.Size,
				ServiceOfferingID:  data.ServiceOfferingID.ValueString(), // Keep existing value since API doesn't return it
				Status:             vol.Status,
				AttachedInstanceID: "",
			}
			break
		}
	}

	if foundVolume == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// Always check attachment status regardless of API status field (to be safe)
	instances, err := r.client.ListInstances(data.ZoneID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"API Communication Error",
			fmt.Sprintf("Failed to list instances in zone %s to determine volume attachment. Error: %s", data.ZoneID.ValueString(), err),
		)
		return
	}

	// Save the current attached instance ID from state
	oldAttachedInstanceID := data.AttachedInstanceID

	for _, instance := range instances.Data {
		for _, volID := range instance.DataVolumes {
			if volIDStr, ok := volID.(string); ok && volIDStr == foundVolume.ID {
				foundVolume.AttachedInstanceID = instance.ID
				break
			}
		}
		if foundVolume.AttachedInstanceID != "" {
			break
		}
	}

	if foundVolume.AttachedInstanceID == "" && foundVolume.Status != helpers.VolumeStatusAllocated && !oldAttachedInstanceID.IsNull() {
		foundVolume.AttachedInstanceID = oldAttachedInstanceID.ValueString()
	}

	data.Name = types.StringValue(foundVolume.Name)
	data.Size = types.Int64Value(int64(foundVolume.Size))
	data.ServiceOfferingID = types.StringValue(foundVolume.ServiceOfferingID)
	data.Status = types.StringValue(foundVolume.Status)
	if foundVolume.AttachedInstanceID == "" {
		data.AttachedInstanceID = types.StringNull()
	} else {
		data.AttachedInstanceID = types.StringValue(foundVolume.AttachedInstanceID)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *volumeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var plan models.VolumeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state models.VolumeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zoneID := plan.ZoneID.ValueString()
	volumeID := plan.ID.ValueString()
	targetInstanceID := plan.InstanceID.ValueString()

	instances, err := r.client.ListInstances(zoneID)
	if err != nil {
		resp.Diagnostics.AddError(
			"API Communication Error",
			fmt.Sprintf("Failed to list instances in zone %s to determine volume attachment. Error: %s", zoneID, err),
		)
		return
	}

	var currentAttachedInstanceID string
	for _, instance := range instances.Data {
		for _, volID := range instance.DataVolumes {
			if volIDStr, ok := volID.(string); ok && volIDStr == volumeID {
				currentAttachedInstanceID = instance.ID
				break
			}
		}
		if currentAttachedInstanceID != "" {
			break
		}
	}

	if currentAttachedInstanceID == "" && !state.AttachedInstanceID.IsNull() {
		currentAttachedInstanceID = state.AttachedInstanceID.ValueString()
	}

	if currentAttachedInstanceID == targetInstanceID {
		volumes, err := r.client.ListInstanceVolumes(zoneID)
		if err != nil {
			resp.Diagnostics.AddError(
				"API Communication Error",
				fmt.Sprintf("Failed to list volumes in zone %s to get volume status. Error: %s", zoneID, err),
			)
			return
		}

		var volumeStatus string
		for _, vol := range volumes.Data {
			if vol.ID == volumeID {
				volumeStatus = vol.Status
				break
			}
		}

		if volumeStatus == "" {
			resp.Diagnostics.AddError(
				"Volume Not Found",
				fmt.Sprintf("Volume '%s' not found in zone %s.", volumeID, zoneID),
			)
			return
		}

		plan.AttachedInstanceID = plan.InstanceID
		plan.Status = types.StringValue(volumeStatus)
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		return
	}

	if currentAttachedInstanceID != "" {
		instanceResp, err := r.client.ShowInstance(zoneID, currentAttachedInstanceID)
		if err != nil {
			resp.Diagnostics.AddError(
				"API Communication Error",
				fmt.Sprintf("Failed to get instance '%s' status. Error: %s", currentAttachedInstanceID, err),
			)
			return
		}

		instanceStatus := instanceResp.Data.Status
		if instanceStatus == "on_process" || !r.isInstanceStateStable(instanceStatus) {
			stableStatuses := []string{
				helpers.InstanceStatusRunning,
				helpers.InstanceStatusUP,
				helpers.InstanceStatusStopped,
				helpers.InstanceStatusSTOPPED,
				helpers.InstanceStatusDown,
			}
			_, err = helpers.WaitForInstanceStatus(
				r.client,
				zoneID,
				currentAttachedInstanceID,
				stableStatuses,
				helpers.MaxInstanceStatusRetries,
				helpers.DefaultPollInterval,
			)
			if err != nil {
				resp.Diagnostics.AddError(
					"Instance State Timeout",
					fmt.Sprintf("Instance '%s' did not reach a stable state within timeout. Current status: %s. Error: %s", currentAttachedInstanceID, instanceStatus, err),
				)
				return
			}
		}

		_, err = r.client.DetachInstanceVolume(zoneID, volumeID, currentAttachedInstanceID)
		if err != nil {
			resp.Diagnostics.AddError(
				"Volume Detachment Failed",
				fmt.Sprintf("Failed to detach volume '%s' from instance '%s'. Error: %s", volumeID, currentAttachedInstanceID, err),
			)
			return
		}

		err = helpers.WaitForVolumeStatus(r.client, zoneID, volumeID, helpers.VolumeStatusAllocated, helpers.MaxVolumeAttachmentRetries, helpers.DefaultVolumePollInterval)
		if err != nil {
			resp.Diagnostics.AddWarning(
				"Volume Detachment Timeout",
				fmt.Sprintf("Volume '%s' detachment from instance '%s' did not complete within timeout. Volume may still be detaching in the background. Error: %s", volumeID, currentAttachedInstanceID, err),
			)
		}
	}

	if targetInstanceID != "" {
		instanceResp, err := r.client.ShowInstance(zoneID, targetInstanceID)
		if err != nil {
			resp.Diagnostics.AddError(
				"API Communication Error",
				fmt.Sprintf("Failed to get instance '%s' status. Error: %s", targetInstanceID, err),
			)
			return
		}

		instanceStatus := instanceResp.Data.Status
		if instanceStatus == "on_process" || !r.isInstanceStateStable(instanceStatus) {
			stableStatuses := []string{
				helpers.InstanceStatusRunning,
				helpers.InstanceStatusUP,
				helpers.InstanceStatusStopped,
				helpers.InstanceStatusSTOPPED,
				helpers.InstanceStatusDown,
			}
			_, err = helpers.WaitForInstanceStatus(
				r.client,
				zoneID,
				targetInstanceID,
				stableStatuses,
				helpers.MaxInstanceStatusRetries,
				helpers.DefaultPollInterval,
			)
			if err != nil {
				resp.Diagnostics.AddError(
					"Instance State Timeout",
					fmt.Sprintf("Instance '%s' did not reach a stable state within timeout. Current status: %s. Error: %s", targetInstanceID, instanceStatus, err),
				)
				return
			}
		}

		_, err = r.client.AttachInstanceVolume(zoneID, volumeID, targetInstanceID)
		if err != nil {
			resp.Diagnostics.AddError(
				"Volume Attachment Failed",
				fmt.Sprintf("Failed to attach volume '%s' to instance '%s'. Error: %s", volumeID, targetInstanceID, err),
			)
			return
		}

		err = helpers.WaitForVolumeAttachmentCompletion(r.client, zoneID, volumeID, helpers.MaxVolumeAttachmentRetries, helpers.DefaultVolumePollInterval)
		if err != nil {
			resp.Diagnostics.AddWarning(
				"Volume Attachment Timeout",
				fmt.Sprintf("Volume '%s' attachment to instance '%s' did not complete within timeout. Volume may still be attaching in the background. Error: %s", volumeID, targetInstanceID, err),
			)
		}
	}

	volumes, err := r.client.ListInstanceVolumes(zoneID)
	if err != nil {
		resp.Diagnostics.AddError(
			"API Communication Error",
			fmt.Sprintf("Failed to list volumes in zone %s to get volume status. Error: %s", zoneID, err),
		)
		return
	}

	var volumeStatus string
	for _, vol := range volumes.Data {
		if vol.ID == volumeID {
			volumeStatus = vol.Status
			break
		}
	}

	if volumeStatus == "" {
		resp.Diagnostics.AddError(
			"Volume Not Found",
			fmt.Sprintf("Volume '%s' not found in zone %s after update operation.", volumeID, zoneID),
		)
		return
	}

	plan.AttachedInstanceID = plan.InstanceID
	plan.Status = types.StringValue(volumeStatus)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *volumeResource) isInstanceStateStable(status string) bool {
	stableStates := []string{
		helpers.InstanceStatusRunning,
		helpers.InstanceStatusUP,
		helpers.InstanceStatusStopped,
		helpers.InstanceStatusSTOPPED,
		helpers.InstanceStatusDown,
	}
	for _, stableState := range stableStates {
		if status == stableState {
			return true
		}
	}
	return false
}

func (r *volumeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var data models.VolumeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Extract volume info for better error messages
	volumeID := data.ID.ValueString()
	volumeName := data.Name.ValueString()
	zoneID := data.ZoneID.ValueString()

	// 1. Read volume details like Read function and update status
	volumes, err := r.client.ListInstanceVolumes(zoneID)
	if err != nil {
		resp.Diagnostics.AddError(
			"API Communication Error",
			fmt.Sprintf("Failed to list volumes in zone %s. Error: %s", zoneID, err),
		)
		return
	}

	// Find the volume by ID
	var foundVolume *volumeInfo
	for _, vol := range volumes.Data {
		if vol.ID == volumeID {
			foundVolume = &volumeInfo{
				ID:                 vol.ID,
				Name:               vol.Name,
				Size:               vol.Size,
				ServiceOfferingID:  data.ServiceOfferingID.ValueString(), // Keep existing value since API doesn't return it
				Status:             vol.Status,
				AttachedInstanceID: "",
			}
			break
		}
	}

	if foundVolume == nil {
		return
	}

	// Always check attachment status regardless of API status field (to be safe)
	instances, err := r.client.ListInstances(zoneID)
	if err != nil {
		resp.Diagnostics.AddError(
			"API Communication Error",
			fmt.Sprintf("Failed to list instances in zone %s to determine volume attachment. Error: %s", zoneID, err),
		)
		return
	}

	for _, instance := range instances.Data {
		for _, volID := range instance.DataVolumes {
			if volIDStr, ok := volID.(string); ok && volIDStr == foundVolume.ID {
				foundVolume.AttachedInstanceID = instance.ID
				break
			}
		}
		if foundVolume.AttachedInstanceID != "" {
			break
		}
	}

	// Update the data model with latest info
	data.Name = types.StringValue(foundVolume.Name)
	data.Size = types.Int64Value(int64(foundVolume.Size))
	data.ServiceOfferingID = types.StringValue(foundVolume.ServiceOfferingID)
	data.Status = types.StringValue(foundVolume.Status)
	if foundVolume.AttachedInstanceID == "" {
		data.AttachedInstanceID = types.StringNull()
	} else {
		data.AttachedInstanceID = types.StringValue(foundVolume.AttachedInstanceID)
	}

	if foundVolume.AttachedInstanceID == "" && foundVolume.Status != helpers.VolumeStatusAllocated && !data.AttachedInstanceID.IsNull() {
		foundVolume.AttachedInstanceID = data.AttachedInstanceID.ValueString()
	}

	currentAttachedInstanceID := foundVolume.AttachedInstanceID

	if currentAttachedInstanceID != "" {
		_, err := r.client.DetachInstanceVolume(zoneID, volumeID, currentAttachedInstanceID)
		if err != nil {
			errMsg := strings.ToLower(err.Error())
			// If instance is not found, it might already be destroyed - warn and proceed with delete
			if strings.Contains(errMsg, "404") || strings.Contains(errMsg, "not found") ||
				strings.Contains(errMsg, "does not exist") {
				resp.Diagnostics.AddWarning(
					"Volume Detachment Skipped",
					fmt.Sprintf("Volume '%s' (%s) in zone '%s' is attached to instance '%s' but the instance appears to be destroyed. Proceeding with volume deletion. Error: %s", volumeName, volumeID, zoneID, currentAttachedInstanceID, err),
				)
			} else if strings.Contains(errMsg, "403") || strings.Contains(errMsg, "unauthorized") ||
				strings.Contains(errMsg, "forbidden") {
				// Permission issues - cannot proceed safely
				resp.Diagnostics.AddError(
					"Volume Deletion Blocked - Permission Denied",
					fmt.Sprintf("Cannot detach volume '%s' (%s) in zone '%s' from instance '%s' - insufficient permissions. Please ensure you have permission to manage volume attachments, or manually detach the volume before deletion. Error: %s", volumeName, volumeID, zoneID, currentAttachedInstanceID, err),
				)
				return
			} else if strings.Contains(errMsg, "409") || strings.Contains(errMsg, "attached") ||
				strings.Contains(errMsg, "in use") {
				// Volume is still actively attached and can't be detached
				resp.Diagnostics.AddError(
					"Volume Deletion Blocked - Still Attached",
					fmt.Sprintf("Volume '%s' (%s) in zone '%s' is currently attached to instance '%s' and cannot be automatically detached. Please stop the instance or manually detach the volume before deleting it. Error: %s", volumeName, volumeID, zoneID, currentAttachedInstanceID, err),
				)
				return
			} else {
				// Other detachment errors - warn but proceed (might still be detachable)
				resp.Diagnostics.AddWarning(
					"Volume Detachment Failed",
					fmt.Sprintf("Failed to detach volume '%s' (%s) in zone '%s' from instance '%s'. Proceeding with volume deletion attempt. Error: %s", volumeName, volumeID, zoneID, currentAttachedInstanceID, err),
				)
			}
		} else {
			err = helpers.WaitForVolumeStatus(r.client, zoneID, volumeID, helpers.VolumeStatusAllocated, helpers.MaxVolumeAttachmentRetries, helpers.DefaultVolumePollInterval)
			if err != nil {
				resp.Diagnostics.AddError(
					"Volume Detachment Timeout",
					fmt.Sprintf("Volume '%s' (%s) in zone '%s' detachment from instance '%s' did not complete within timeout. Cannot proceed with deletion. Error: %s", volumeName, volumeID, zoneID, currentAttachedInstanceID, err),
				)
				return
			}
		}
	}

	finalVolumes, err := r.client.ListInstanceVolumes(zoneID)
	if err != nil {
		resp.Diagnostics.AddError(
			"API Communication Error",
			fmt.Sprintf("Failed to verify volume '%s' (%s) status before deletion in zone '%s'. Error: %s", volumeName, volumeID, zoneID, err),
		)
		return
	}

	finalVolumeStatus := ""
	for _, vol := range finalVolumes.Data {
		if vol.ID == volumeID {
			finalVolumeStatus = vol.Status
			break
		}
	}

	if finalVolumeStatus != helpers.VolumeStatusAllocated {
		resp.Diagnostics.AddError(
			"Volume Deletion Blocked - Invalid Status",
			fmt.Sprintf("Volume '%s' (%s) in zone '%s' has status '%s' and cannot be deleted. Expected status: 'ALLOCATED'. Please ensure the volume is properly detached before deletion.", volumeName, volumeID, zoneID, finalVolumeStatus),
		)
		return
	}
	_, err = r.client.DeleteInstanceVolume(zoneID, volumeID)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			return
		}
		if strings.Contains(strings.ToLower(err.Error()), "409") || strings.Contains(strings.ToLower(err.Error()), "attached") ||
			strings.Contains(strings.ToLower(err.Error()), "in use") {
			resp.Diagnostics.AddError(
				"Volume Deletion Blocked",
				fmt.Sprintf("Cannot delete volume '%s' (%s) in zone '%s' - volume is still attached or in use. Please ensure the volume is fully detached before deleting. Error: %s", volumeName, volumeID, zoneID, err),
			)
			return
		}
		resp.Diagnostics.AddError(
			"Volume Deletion Failed",
			fmt.Sprintf("Failed to delete volume '%s' (%s) in zone '%s'. Error: %s. This may be due to API issues or volume state problems.", volumeName, volumeID, zoneID, err),
		)
		return
	}

}

func (r *volumeResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data models.VolumeResourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !data.ZoneID.IsNull() && !data.ZoneID.IsUnknown() && !data.ServiceOfferingID.IsNull() && !data.ServiceOfferingID.IsUnknown() {
		if err := r.validateVolumeServiceOfferingForConfig(data.ZoneID.ValueString(), data.ServiceOfferingID.ValueString(), resp); err != nil {
			return
		}
	}
}

func (r *volumeResource) validateVolumeServiceOfferingForConfig(zoneID, serviceOfferingID string, diagnostics *resource.ValidateConfigResponse) error {
	serviceOfferings, err := r.client.ListInstanceVolumeServiceOfferings(zoneID)
	if err != nil {
		return nil
	}

	found := false
	for _, offering := range serviceOfferings.Data {
		if offering.ID == serviceOfferingID {
			found = true
			break
		}
	}

	if !found {
		errMsg := fmt.Sprintf("volume service offering ID '%s' not found in zone '%s'", serviceOfferingID, zoneID)
		diagnostics.Diagnostics.AddError("Invalid Volume Service Offering", errMsg)
		return fmt.Errorf("%s", errMsg)
	}

	return nil
}

func (r *volumeResource) validateVolumeServiceOfferingForCreate(zoneID, serviceOfferingID string, diagnostics *resource.CreateResponse) error {
	serviceOfferings, err := r.client.ListInstanceVolumeServiceOfferings(zoneID)
	if err != nil {
		diagnostics.Diagnostics.AddError(
			"API Communication Error",
			fmt.Sprintf("Failed to list volume service offerings in zone %s. Error: %s", zoneID, err),
		)
		return err
	}

	found := false
	for _, offering := range serviceOfferings.Data {
		if offering.ID == serviceOfferingID {
			found = true
			break
		}
	}

	if !found {
		errMsg := fmt.Sprintf("volume service offering ID '%s' not found in zone '%s'", serviceOfferingID, zoneID)
		diagnostics.Diagnostics.AddError("Invalid Volume Service Offering", errMsg)
		return fmt.Errorf("%s", errMsg)
	}

	return nil
}

func (r *volumeResource) validateInstanceForVolumeAttachment(zoneID, instanceID string, diagnostics *resource.CreateResponse) error {
	instanceResp, err := r.client.ShowInstance(zoneID, instanceID)
	if err != nil {
		diagnostics.Diagnostics.AddError(
			"Instance Not Found",
			fmt.Sprintf("Cannot create and attach volume: instance '%s' does not exist in zone '%s'. Please ensure the instance is created before creating the volume. Error: %s", instanceID, zoneID, err),
		)
		return err
	}

	instanceStatus := instanceResp.Data.Status
	if instanceStatus == "on_process" || !r.isInstanceStateStable(instanceStatus) {
		stableStatuses := []string{
			helpers.InstanceStatusRunning,
			helpers.InstanceStatusUP,
			helpers.InstanceStatusStopped,
			helpers.InstanceStatusSTOPPED,
			helpers.InstanceStatusDown,
		}
		_, err = helpers.WaitForInstanceStatus(
			r.client,
			zoneID,
			instanceID,
			stableStatuses,
			helpers.MaxInstanceStatusRetries,
			helpers.DefaultPollInterval,
		)
		if err != nil {
			diagnostics.Diagnostics.AddError(
				"Instance Not Ready",
				fmt.Sprintf("Cannot create and attach volume: instance '%s' in zone '%s' is not in a stable state. Current status: '%s'. Please wait for the instance to be ready before creating the volume. Error: %s", instanceID, zoneID, instanceStatus, err),
			)
			return err
		}
	}

	return nil
}
