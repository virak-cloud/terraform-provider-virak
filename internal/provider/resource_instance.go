package provider

import (
	"context"
	"fmt"
	"sync"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/virak-cloud/cli/pkg/http/responses"
	"github.com/virak-cloud/terraform-provider-virak/internal/models"
	"github.com/virak-cloud/terraform-provider-virak/internal/provider/helpers"
)

// Ensure the implementation satisfies the resource interfaces.
var _ resource.Resource = &instanceResource{}
var _ resource.ResourceWithModifyPlan = &instanceResource{}

func NewInstanceResource(mutex *sync.Mutex) resource.Resource {
	r := &instanceResource{}
	r.setMutex(mutex)
	return r
}

type instanceResource struct {
	baseResource
}

func (r *instanceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance"
}

func (r *instanceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Virak Cloud instance.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Instance ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"zone_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the zone to create the instance in.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the instance.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"service_offering_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the service offering for the instance.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"vm_image_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the VM image to use for the instance.",
			},
			"network_ids": schema.ListAttribute{
				Required:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "List of network IDs to attach to the instance.",
			},
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The status of the instance.",
			},
			"password": schema.StringAttribute{
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "The password for the instance.",
			},
			"username": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The username for accessing the instance.",
			},
			"ip": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The IP address of the instance.",
			},
			"networks": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "List of networks connected to this instance.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"network_id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The ID of the network.",
						},
						"ip_address": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The IP address assigned to the instance in this network.",
						},
						"mac_address": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The MAC address of the instance in this network.",
						},
						"is_default": schema.BoolAttribute{
							Computed:            true,
							MarkdownDescription: "Whether this is the default network for the instance.",
						},
						"attachment_id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The ID of the instance network attachment.",
						},
					},
				},
			},
			"desired_state": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Desired state of the instance. Valid values: 'running', 'stopped'. Setting this will trigger start/stop operations. Use 'reboot' to restart a running instance.",
			},
		},
	}
}

func (r *instanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var data models.InstanceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	networkIDs := make([]string, 0)
	if !data.NetworkIDs.IsNull() && !data.NetworkIDs.IsUnknown() {
		for _, elem := range data.NetworkIDs.Elements() {
			if str, ok := elem.(types.String); ok && !str.IsNull() && !str.IsUnknown() && str.ValueString() != "" {
				networkIDs = append(networkIDs, str.ValueString())
			}
		}
	}

	if !helpers.ValidateNetworksForInstanceCreation(r.client, data.ZoneID.ValueString(), networkIDs, &resp.Diagnostics) {
		return
	}

	initialInstances, err := r.client.ListInstances(data.ZoneID.ValueString())
	if err != nil {
		helpers.HandleAPIError(&resp.Diagnostics, "API Communication Error", fmt.Errorf("failed to list existing instances in zone %s before creation: %w", data.ZoneID.ValueString(), err))
		return
	}
	existingIDs := make(map[string]struct{})
	for _, instance := range initialInstances.Data {
		existingIDs[instance.ID] = struct{}{}
	}

	newInstanceID, err := helpers.CreateInstanceCore(r.client, &data, networkIDs, existingIDs, &resp.Diagnostics)
	if err != nil {
		return
	}

	data.ID = types.StringValue(newInstanceID)

	if err := helpers.WaitForInstanceReady(r.client, data.ZoneID.ValueString(), data.ID.ValueString(), &resp.Diagnostics); err != nil {
		return
	}

	helpers.PopulateInstanceState(r.client, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	networksList, instanceIP := helpers.SetupInstanceNetworks(r.client, &data, networkIDs, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Networks = networksList

	if instanceIP != "" {
		data.IP = types.StringValue(instanceIP)
	} else {
		data.IP = types.StringValue("")
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *instanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data models.InstanceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readResp, err := helpers.GetInstanceDetails(r.client, data.ZoneID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data.Name = types.StringValue(readResp.Data.Name)
	data.Status = types.StringValue(readResp.Data.Status)
	data.Username = types.StringValue(readResp.Data.Username)

	networks, err := helpers.GetInstanceNetworks(r.client, data.ZoneID.ValueString(), data.ID.ValueString())
	if err != nil {
		helpers.LogWarning("Unable to list networks in zone %s for instance %s, got error: %s. Keeping existing networks state.", data.ZoneID.ValueString(), data.ID.ValueString(), err)
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	result := helpers.BuildNetworkObjects(networks)
	resp.Diagnostics.Append(result.Diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Networks = helpers.CreateNetworksList(result.NetworkObjects, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if result.InstanceIP != "" {
		data.IP = types.StringValue(result.InstanceIP)
	} else {
		data.IP = types.StringValue("")
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *instanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var plan models.InstanceResourceModel
	var state models.InstanceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.Password = state.Password
	plan.Username = state.Username
	imageChanged := !plan.VMImageID.Equal(state.VMImageID)
	desiredStateChanged := !plan.DesiredState.IsNull() && !plan.DesiredState.Equal(state.DesiredState)

	if imageChanged {
		if err := helpers.EnsureInstanceStopped(r.client, plan.ZoneID.ValueString(), plan.ID.ValueString(), &resp.Diagnostics); err != nil {
			return
		}
		_, err := r.client.RebuildInstance(
			plan.ZoneID.ValueString(),
			plan.ID.ValueString(),
			plan.VMImageID.ValueString(),
		)
		if err != nil {
			helpers.HandleAPIError(&resp.Diagnostics, "Client Error", fmt.Errorf("unable to rebuild instance: %w", err))
			return
		}
		status, err := helpers.WaitForInstanceStatus(
			r.client,
			plan.ZoneID.ValueString(),
			plan.ID.ValueString(),
			[]string{helpers.InstanceStatusUP},
			helpers.MaxInstanceStatusRetries,
			helpers.DefaultPollInterval,
		)
		if err != nil {
			resp.Diagnostics.AddError("Instance Rebuild Incomplete", fmt.Sprintf("Instance '%s' did not reach 'UP' status after rebuild. Current status: %s. Error: %s", plan.ID.ValueString(), status, err))
			return
		}
		helpers.PopulateInstanceState(r.client, &plan, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	if desiredStateChanged {
		currentStatus := ""
		if !plan.Status.IsNull() && !plan.Status.IsUnknown() && plan.Status.ValueString() != "" {
			currentStatus = plan.Status.ValueString()
		} else if !state.Status.IsNull() && !state.Status.IsUnknown() {
			currentStatus = state.Status.ValueString()
		}
		result := helpers.HandleInstanceLifecycle(r.client, plan.ZoneID.ValueString(), plan.ID.ValueString(), plan.DesiredState.ValueString(), currentStatus)
		resp.Diagnostics.Append(result.Diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		if !result.Status.IsNull() {
			plan.Status = result.Status
		}
		readResp, err := r.client.ShowInstance(plan.ZoneID.ValueString(), plan.ID.ValueString())
		if err != nil {
			helpers.HandleAPIError(&resp.Diagnostics, "Client Error", fmt.Errorf("unable to read instance after lifecycle operation: %w", err))
			return
		}
		plan.Status = types.StringValue(readResp.Data.Status)
		if readResp.Data.Username != "" {
			plan.Username = types.StringValue(readResp.Data.Username)
		}
		if readResp.Data.Password != "" {
			plan.Password = types.StringValue(readResp.Data.Password)
		}
	}

	// Handle network changes
	// Build network ID maps from plan and state
	// IMPORTANT: stateNetworkIDs should ONLY include explicitly managed networks from state.NetworkIDs
	// NOT networks from state.Networks, which contains ALL actual networks (including unmanaged ones)
	stateNetworkIDs := make(map[string]bool)
	if !state.NetworkIDs.IsNull() && !state.NetworkIDs.IsUnknown() {
		for _, elem := range state.NetworkIDs.Elements() {
			if str, ok := elem.(types.String); ok && !str.IsNull() && !str.IsUnknown() && str.ValueString() != "" {
				stateNetworkIDs[str.ValueString()] = true
			}
		}
	}

	// actualNetworkIDs contains ALL networks currently connected to the instance in the infrastructure
	actualNetworkIDs := make(map[string]bool)
	if !state.Networks.IsNull() && !state.Networks.IsUnknown() {
		for _, networkElem := range state.Networks.Elements() {
			if networkObj, ok := networkElem.(types.Object); ok && !networkObj.IsNull() {
				attrs := networkObj.Attributes()
				if networkIDAttr, exists := attrs["network_id"]; exists {
					if networkID, ok := networkIDAttr.(types.String); ok && !networkID.IsNull() && networkID.ValueString() != "" {
						actualNetworkIDs[networkID.ValueString()] = true
					}
				}
			}
		}
	}

	planNetworkIDs := make(map[string]bool)
	if !plan.NetworkIDs.IsNull() && !plan.NetworkIDs.IsUnknown() {
		for _, elem := range plan.NetworkIDs.Elements() {
			if str, ok := elem.(types.String); ok && !str.IsNull() && !str.IsUnknown() && str.ValueString() != "" {
				planNetworkIDs[str.ValueString()] = true
			}
		}
	}

	if len(planNetworkIDs) == 0 {
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			"At least one network must remain connected to the instance. Cannot remove all networks.",
		)
		return
	}

	// Detect unmanaged networks (in actual state but not in managed configuration)
	unmanagedNetworks := make([]string, 0)
	for networkID := range actualNetworkIDs {
		if !stateNetworkIDs[networkID] && !planNetworkIDs[networkID] {
			unmanagedNetworks = append(unmanagedNetworks, networkID)
		}
	}

	// If unmanaged networks are detected, detach them automatically
	if len(unmanagedNetworks) > 0 {
		resp.Diagnostics.AddWarning(
			"Unmanaged Networks Detected",
			fmt.Sprintf("Found %d unmanaged network(s) attached to instance '%s' that are not in the configuration. These networks will be automatically detached: %v", len(unmanagedNetworks), plan.ID.ValueString(), unmanagedNetworks),
		)
		helpers.DetachNetworksFromInstance(r.client, plan.ZoneID.ValueString(), plan.ID.ValueString(), unmanagedNetworks, "", &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
		for _, networkID := range unmanagedNetworks {
			stateNetworkIDs[networkID] = true
		}
	}

	// Update instance networks using helper
	helpers.UpdateInstanceNetworks(r.client, plan.ZoneID.ValueString(), plan.ID.ValueString(), planNetworkIDs, stateNetworkIDs, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Refresh network state
	networksList, instanceIP := helpers.RefreshInstanceNetworks(r.client, plan.ZoneID.ValueString(), plan.ID.ValueString(), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.Networks = networksList
	if instanceIP != "" {
		plan.IP = types.StringValue(instanceIP)
	} else {
		plan.IP = types.StringValue("")
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *instanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var data models.InstanceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := helpers.EnsureInstanceRunning(r.client, data.ZoneID.ValueString(), data.ID.ValueString(), &resp.Diagnostics); err != nil {
		return
	}

	helpers.DetachAllVolumes(r.client, data.ZoneID.ValueString(), data.ID.ValueString(), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	helpers.DisconnectAllNetworks(r.client, data.ZoneID.ValueString(), data.ID.ValueString(), &resp.Diagnostics)

	initialInstances, err := r.client.ListInstances(data.ZoneID.ValueString())
	if err != nil {
		helpers.HandleAPIError(&resp.Diagnostics, "Client Error", fmt.Errorf("unable to list instances before deletion: %w", err))
		return
	}
	existingIDs := make(map[string]struct{})
	for _, instance := range initialInstances.Data {
		existingIDs[instance.ID] = struct{}{}
	}

	_, err = r.client.DeleteInstance(data.ZoneID.ValueString(), data.ID.ValueString(), data.Name.ValueString())
	if err != nil {
		helpers.HandleAPIError(&resp.Diagnostics, "Client Error", fmt.Errorf("unable to delete instance: %w", err))
		return
	}

	listFunc := func(zoneID string) (interface{}, error) {
		return r.client.ListInstances(zoneID)
	}
	checkFunc := func(listResp interface{}, resourceID string) bool {
		instancesResp, ok := listResp.(*responses.InstanceListResponse)
		if !ok {
			return false
		}
		for _, instance := range instancesResp.Data {
			if instance.ID == resourceID {
				return false
			}
		}
		return true
	}

	err = helpers.WaitForResourceDeletion(r.client, data.ZoneID.ValueString(), data.ID.ValueString(), listFunc, checkFunc, helpers.MaxInstanceDeletionRetries, helpers.DefaultPollInterval)
	if err != nil {
		helpers.HandleAPIError(&resp.Diagnostics, "Client Error", fmt.Errorf("instance was not deleted successfully: %w", err))
		return
	}
}

func (r *instanceResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data models.InstanceResourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	helpers.ValidateInstanceNameLength(data.Name.ValueString(), 63, &resp.Diagnostics)
}

func (r *instanceResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan models.InstanceResourceModel
	var state models.InstanceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	isNewResource := req.State.Raw.IsNull()
	if !isNewResource {
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	currentStatus := ""
	if isNewResource {
		currentStatus = helpers.InstanceStatusUP
	} else if !state.Status.IsNull() && !state.Status.IsUnknown() {
		currentStatus = state.Status.ValueString()
	}

	desiredStatus := ""
	if !plan.DesiredState.IsNull() && !plan.DesiredState.IsUnknown() {
		desiredStatus = plan.DesiredState.ValueString()
	}

	statusAlreadyMatches := false
	if desiredStatus == "running" && currentStatus != "" && helpers.IsInstanceRunning(currentStatus) {
		statusAlreadyMatches = true
	} else if desiredStatus == "stopped" && currentStatus != "" && helpers.IsInstanceStopped(currentStatus) {
		statusAlreadyMatches = true
	}

	operations := make([]string, 0)

	if desiredStatus == "running" {
		if !isNewResource && currentStatus != "" && !helpers.IsInstanceRunning(currentStatus) {
			operations = append(operations, "start")
		}
	} else if desiredStatus == "stopped" {
		if isNewResource || (currentStatus != "" && helpers.IsInstanceRunning(currentStatus)) {
			operations = append(operations, "stop")
		}
	} else if desiredStatus == "reboot" {
		if !isNewResource && currentStatus != "" && helpers.IsInstanceRunning(currentStatus) {
			operations = append(operations, "reboot")
		}
	}

	rebuildNeeded := false
	if !isNewResource &&
		!plan.VMImageID.IsNull() && !plan.VMImageID.IsUnknown() &&
		!state.VMImageID.IsNull() && !state.VMImageID.IsUnknown() &&
		!plan.VMImageID.Equal(state.VMImageID) {
		operations = append(operations, "rebuild")
		rebuildNeeded = true
	}

	if len(operations) == 0 {
		return
	}

	if statusAlreadyMatches && !rebuildNeeded {
		return
	}

	message := "Instance lifecycle information:\n"
	if currentStatus != "" {
		message += fmt.Sprintf("  Current status: %s\n", currentStatus)
	} else {
		message += "  Current status: (unknown)\n"
	}

	if desiredStatus != "" {
		message += fmt.Sprintf("  Desired status: %s\n", desiredStatus)
	} else {
		message += "  Desired status: (not set)\n"
	}

	message += fmt.Sprintf("  Operations to perform: %v", operations)
	resp.Diagnostics.AddWarning("Instance Lifecycle Operations", message)
}
