package provider

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/virak-cloud/cli/pkg/http/responses"
	"github.com/virak-cloud/terraform-provider-virak/internal/models"
	"github.com/virak-cloud/terraform-provider-virak/internal/provider/helpers"
)

var _ resource.Resource = &networkResource{}

func NewNetworkResource(mutex *sync.Mutex) resource.Resource {
	r := &networkResource{}
	r.setMutex(mutex)
	return r
}

type networkResource struct {
	baseResource
}

func (r *networkResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network"
}

func (r *networkResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Virak Cloud network.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Network ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the network.",
			},
			"zone_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the zone where the network is located.",
			},
			"network_offering_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the network offering.",
			},
			"type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The type of the network.",
			},
			"gateway": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The gateway IP address of the network.",
			},
			"netmask": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The netmask of the network.",
			},
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The status of the network.",
			},
			"instances": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "List of instances connected to this network.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"instance_id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The ID of the instance.",
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
		},
	}
}

func (r *networkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var data models.NetworkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	log.Printf("=== Starting Network Creation ===")
	log.Printf("Network Name: %s", data.Name.ValueString())
	log.Printf("Zone ID: %s", data.ZoneID.ValueString())
	log.Printf("Network Offering ID: %s", data.NetworkOfferingID.ValueString())
	log.Printf("Network Type: %s", data.Type.ValueString())
	log.Printf("Gateway: %s", data.Gateway.ValueString())
	log.Printf("Netmask: %s", data.Netmask.ValueString())

	// Step 1: Get current networks before creation
	log.Printf("Step 1: Listing current networks before creation...")
	initialNetworks, err := r.client.ListNetworks(data.ZoneID.ValueString())
	if err != nil {
		log.Printf("Error listing networks before creation: %v", err)
		helpers.HandleAPIError(&resp.Diagnostics, "Client Error", fmt.Errorf("unable to list networks before creation: %w", err))
		return
	}

	if !helpers.ValidateNetworkName(r.client, data.ZoneID.ValueString(), data.Name.ValueString(), &resp.Diagnostics) {
		return
	}

	helpers.LogNetworks("Pre-Creation Networks", initialNetworks)

	// Store existing IDs for comparison
	existingIDs := make(map[string]struct{})
	for _, network := range initialNetworks.Data {
		existingIDs[network.ID] = struct{}{}
		log.Printf("Found existing network: %s (ID: %s)", network.Name, network.ID)
	}

	// Step 2: Create the network
	log.Printf("Step 2: Creating network...")
	var createResponse interface{}
	var createErr error

	if data.Type.ValueString() == "Isolated" || data.Type.ValueString() == "L3" {
		if data.Gateway.IsNull() || data.Netmask.IsNull() {
			helpers.HandleValidationError(&resp.Diagnostics, "Validation Error", "gateway and netmask are required for Isolated networks")
			return
		}
		log.Printf("Creating L3/Isolated network with gateway: %s, netmask: %s",
			data.Gateway.ValueString(), data.Netmask.ValueString())
		createResponse, createErr = r.client.CreateL3Network(
			data.ZoneID.ValueString(),
			data.NetworkOfferingID.ValueString(),
			data.Name.ValueString(),
			data.Gateway.ValueString(),
			data.Netmask.ValueString(),
		)
	} else {
		log.Printf("Creating L2 network")
		createResponse, createErr = r.client.CreateL2Network(
			data.ZoneID.ValueString(),
			data.NetworkOfferingID.ValueString(),
			data.Name.ValueString(),
		)
	}

	// Log the creation response
	helpers.LogAPIResponse("Network Creation", createResponse, createErr)

	if createErr != nil {
		log.Printf("Network creation failed: %v", createErr)
		helpers.HandleAPIError(&resp.Diagnostics, "Client Error", fmt.Errorf("unable to create network: %w", createErr))
		return
	}

	log.Printf("Network creation request submitted successfully")

	log.Printf("Step 3: Searching for newly created network...")
	newNetworkID, err := helpers.FindNewNetworkID(r.client, data.ZoneID.ValueString(), existingIDs, data.Name.ValueString(), 60, helpers.DefaultPollInterval)
	if err != nil {
		log.Printf("❌ Network creation timeout - network '%s' not found after timeout", data.Name.ValueString())
		resp.Diagnostics.AddError(
			"Network Creation Timeout",
			fmt.Sprintf("Network '%s' was submitted for creation but could not be found in the API after timeout. The network may still be provisioning in the background.", data.Name.ValueString()),
		)
		return
	}

	log.Printf("✓ Successfully found new network ID: %s", newNetworkID)

	// Step 4: Update the resource with new network ID
	log.Printf("Step 4: Updating resource state...")
	data.ID = types.StringValue(newNetworkID)
	data.Status = types.StringValue("Active")

	instances, err := helpers.GetNetworkInstances(r.client, data.ZoneID.ValueString(), data.ID.ValueString())
	if err != nil {
		log.Printf("Warning: Unable to list instances in zone %s, got error: %s. Setting instances to empty list.", data.ZoneID.ValueString(), err)
		instances = []responses.InstanceNetwork{}
	}

	instanceObjects := make([]attr.Value, 0)
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
			resp.Diagnostics.Append(objDiags...)
			continue
		}
		instanceObjects = append(instanceObjects, instanceObj)
	}
	log.Printf("Found %d instances connected to network %s", len(instanceObjects), data.ID.ValueString())

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
	resp.Diagnostics.Append(listDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Instances = instancesList

	// Log final state
	log.Printf("Final network state:")
	log.Printf("  - ID: %s", newNetworkID)
	log.Printf("  - Name: %s", data.Name.ValueString())
	log.Printf("  - Zone ID: %s", data.ZoneID.ValueString())
	log.Printf("  - Status: %s", data.Status.ValueString())

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		log.Printf("Error setting resource state: %v", resp.Diagnostics)
		return
	}

	log.Printf("=== Network Creation Completed Successfully ===")
	log.Printf("Network '%s' (ID: %s) created and ready for use", data.Name.ValueString(), newNetworkID)
}

func (r *networkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data models.NetworkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	log.Printf("=== Reading Network State ===")
	log.Printf("Network ID: %s", data.ID.ValueString())
	log.Printf("Zone ID: %s", data.ZoneID.ValueString())

	readResp, err := r.client.ShowNetwork(data.ZoneID.ValueString(), data.ID.ValueString())
	if err != nil {
		log.Printf("Error reading network: %v", err)
		resp.State.RemoveResource(ctx)
		return
	}

	// Log the read response
	helpers.LogAPIResponse("Network Read", readResp, nil)

	instances, err := helpers.GetNetworkInstances(r.client, data.ZoneID.ValueString(), data.ID.ValueString())
	if err != nil {
		log.Printf("Warning: Unable to list instances in zone %s, got error: %s. Setting instances to empty list.", data.ZoneID.ValueString(), err)
		instances = []responses.InstanceNetwork{}
	}

	instanceObjects := make([]attr.Value, 0)
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
			resp.Diagnostics.Append(objDiags...)
			continue
		}
		instanceObjects = append(instanceObjects, instanceObj)
	}
	log.Printf("Found %d instances connected to network %s", len(instanceObjects), data.ID.ValueString())

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
	resp.Diagnostics.Append(listDiags...)

	// Preserve all existing values and only update fields from API
	// Create a new data object with preserved values
	updatedData := models.NetworkResourceModel{
		ID:                data.ID,
		Name:              types.StringValue(readResp.Data.Name),
		ZoneID:            data.ZoneID,
		NetworkOfferingID: types.StringValue(readResp.Data.NetworkOffering.ID),
		Type:              data.Type,
		Gateway:           data.Gateway,
		Netmask:           data.Netmask,
		Status:            types.StringValue(readResp.Data.Status),
		Instances:         instancesList,
	}

	log.Printf("Updated network state:")
	log.Printf("  - Name: %s", updatedData.Name.ValueString())
	log.Printf("  - Status: %s", updatedData.Status.ValueString())
	log.Printf("  - Type: %s", updatedData.Type.ValueString())
	log.Printf("  - Instances: %d", len(instanceObjects))

	resp.Diagnostics.Append(resp.State.Set(ctx, &updatedData)...)
	if resp.Diagnostics.HasError() {
		log.Printf("Error setting updated state: %v", resp.Diagnostics)
		return
	}

	log.Printf("=== Network Read Completed ===")
}

func (r *networkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var plan models.NetworkResourceModel
	var state models.NetworkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	log.Printf("=== Network Update Requested ===")
	log.Printf("Network ID: %s", plan.ID.ValueString())

	configurableAttributesChanged := false
	changedAttributes := make([]string, 0)

	if !plan.Name.Equal(state.Name) {
		log.Printf("Name changed: '%s' -> '%s' (state: null=%v, unknown=%v | plan: null=%v, unknown=%v)",
			state.Name.ValueString(), plan.Name.ValueString(),
			state.Name.IsNull(), state.Name.IsUnknown(),
			plan.Name.IsNull(), plan.Name.IsUnknown())
		configurableAttributesChanged = true
		changedAttributes = append(changedAttributes, "name")
	}
	if !plan.ZoneID.Equal(state.ZoneID) {
		log.Printf("Zone ID changed: '%s' -> '%s' (state: null=%v, unknown=%v | plan: null=%v, unknown=%v)",
			state.ZoneID.ValueString(), plan.ZoneID.ValueString(),
			state.ZoneID.IsNull(), state.ZoneID.IsUnknown(),
			plan.ZoneID.IsNull(), plan.ZoneID.IsUnknown())
		configurableAttributesChanged = true
		changedAttributes = append(changedAttributes, "zone_id")
	}
	if !plan.NetworkOfferingID.Equal(state.NetworkOfferingID) {
		log.Printf("Network Offering ID changed: '%s' -> '%s' (state: null=%v, unknown=%v | plan: null=%v, unknown=%v)",
			state.NetworkOfferingID.ValueString(), plan.NetworkOfferingID.ValueString(),
			state.NetworkOfferingID.IsNull(), state.NetworkOfferingID.IsUnknown(),
			plan.NetworkOfferingID.IsNull(), plan.NetworkOfferingID.IsUnknown())
		configurableAttributesChanged = true
		changedAttributes = append(changedAttributes, "network_offering_id")
	}
	if !plan.Type.Equal(state.Type) {
		log.Printf("Type changed: '%s' -> '%s' (state: null=%v, unknown=%v | plan: null=%v, unknown=%v)",
			state.Type.ValueString(), plan.Type.ValueString(),
			state.Type.IsNull(), state.Type.IsUnknown(),
			plan.Type.IsNull(), plan.Type.IsUnknown())
		configurableAttributesChanged = true
		changedAttributes = append(changedAttributes, "type")
	}
	if !plan.Gateway.Equal(state.Gateway) {
		log.Printf("Gateway changed: '%s' -> '%s' (state: null=%v, unknown=%v | plan: null=%v, unknown=%v)",
			state.Gateway.ValueString(), plan.Gateway.ValueString(),
			state.Gateway.IsNull(), state.Gateway.IsUnknown(),
			plan.Gateway.IsNull(), plan.Gateway.IsUnknown())
		if !plan.Gateway.IsUnknown() && !state.Gateway.IsUnknown() {
			if plan.Gateway.ValueString() != state.Gateway.ValueString() {
				configurableAttributesChanged = true
				changedAttributes = append(changedAttributes, "gateway")
			}
		}
	}
	if !plan.Netmask.Equal(state.Netmask) {
		log.Printf("Netmask changed: '%s' -> '%s' (state: null=%v, unknown=%v | plan: null=%v, unknown=%v)",
			state.Netmask.ValueString(), plan.Netmask.ValueString(),
			state.Netmask.IsNull(), state.Netmask.IsUnknown(),
			plan.Netmask.IsNull(), plan.Netmask.IsUnknown())
		if !plan.Netmask.IsUnknown() && !state.Netmask.IsUnknown() {
			if plan.Netmask.ValueString() != state.Netmask.ValueString() {
				configurableAttributesChanged = true
				changedAttributes = append(changedAttributes, "netmask")
			}
		}
	}

	if configurableAttributesChanged {
		log.Printf("=== Network Update Rejected (Configurable Attributes Changed) ===")
		log.Printf("Changed attributes: %v", changedAttributes)
		resp.Diagnostics.AddError("Update Not Supported", fmt.Sprintf("Network resources cannot be updated after creation. Changed attributes: %v. Please destroy and recreate the network if changes are needed.", changedAttributes))
		return
	}

	log.Printf("Only computed attributes changed, refreshing state from API...")

	readResp, err := r.client.ShowNetwork(plan.ZoneID.ValueString(), plan.ID.ValueString())
	if err != nil {
		log.Printf("Error reading network during update: %v", err)
		helpers.HandleAPIError(&resp.Diagnostics, "Client Error", fmt.Errorf("unable to read network during update: %w", err))
		return
	}

	instances, err := helpers.GetNetworkInstances(r.client, plan.ZoneID.ValueString(), plan.ID.ValueString())
	if err != nil {
		log.Printf("Warning: Unable to list instances in zone %s, got error: %s. Setting instances to empty list.", plan.ZoneID.ValueString(), err)
		instances = []responses.InstanceNetwork{}
	}

	instanceObjects := make([]attr.Value, 0)
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
			resp.Diagnostics.Append(objDiags...)
			continue
		}
		instanceObjects = append(instanceObjects, instanceObj)
	}
	log.Printf("Found %d instances connected to network %s", len(instanceObjects), plan.ID.ValueString())

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
	resp.Diagnostics.Append(listDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updatedData := models.NetworkResourceModel{
		ID:                plan.ID,
		Name:              plan.Name,
		ZoneID:            plan.ZoneID,
		NetworkOfferingID: plan.NetworkOfferingID,
		Type:              plan.Type,
		Gateway:           plan.Gateway,
		Netmask:           plan.Netmask,
		Status:            types.StringValue(readResp.Data.Status),
		Instances:         instancesList,
	}

	log.Printf("Network state refreshed successfully")
	log.Printf("  - Status: %s", updatedData.Status.ValueString())
	log.Printf("  - Instances: %d", len(instanceObjects))

	resp.Diagnostics.Append(resp.State.Set(ctx, &updatedData)...)
	if resp.Diagnostics.HasError() {
		log.Printf("Error setting updated state: %v", resp.Diagnostics)
		return
	}

	log.Printf("=== Network Update Completed (State Refreshed) ===")
}

func (r *networkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var data models.NetworkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	log.Printf("=== Starting Network Deletion ===")
	log.Printf("Network ID: %s", data.ID.ValueString())
	log.Printf("Network Name: %s", data.Name.ValueString())
	log.Printf("Zone ID: %s", data.ZoneID.ValueString())

	log.Printf("Step 1: Checking for instances connected to this network...")
	instances, err := helpers.GetNetworkInstances(r.client, data.ZoneID.ValueString(), data.ID.ValueString())
	if err != nil {
		log.Printf("Error listing instances in zone %s: %v", data.ZoneID.ValueString(), err)
		helpers.HandleAPIError(&resp.Diagnostics, "Client Error", fmt.Errorf("unable to list instances in zone %s: %w", data.ZoneID.ValueString(), err))
		return
	}

	log.Printf("Found %d instance connections to this network", len(instances))

	disconnectedCount := 0
	for _, networkInstance := range instances {
		if networkInstance.Network.ID != data.ID.ValueString() {
			continue
		}
		if networkInstance.IsDefault {
			log.Printf("Skipping disconnection from default NIC for instance %s (attachment ID: %s)", networkInstance.InstanceID, networkInstance.ID)
			continue
		}
		log.Printf("Disconnecting instance %s from network %s (attachment ID: %s)", networkInstance.InstanceID, data.ID.ValueString(), networkInstance.ID)
		_, err = r.client.DisconnectInstanceFromNetwork(data.ZoneID.ValueString(), data.ID.ValueString(), networkInstance.InstanceID, networkInstance.ID)
		if err != nil {
			log.Printf("Error disconnecting instance %s: %v", networkInstance.InstanceID, err)
			helpers.HandleAPIError(&resp.Diagnostics, "Client Error", fmt.Errorf("unable to disconnect instance %s from network %s: %w", networkInstance.InstanceID, data.ID.ValueString(), err))
			return
		}

		err = helpers.WaitForNetworkDisconnection(r.client, data.ZoneID.ValueString(), data.ID.ValueString(), networkInstance.InstanceID, networkInstance.ID, helpers.MaxNetworkConnectionRetries, helpers.DefaultNetworkPollInterval)
		if err != nil {
			log.Printf("Warning: Disconnection verification timeout for instance %s, but proceeding", networkInstance.InstanceID)
		} else {
			disconnectedCount++
		}
	}

	log.Printf("Disconnected %d instances from the network", disconnectedCount)

	log.Printf("Step 2: Verifying network has no remaining connections...")
	maxVerificationRetries := 10
	verificationInterval := 2 * time.Second
	verified := false
	for i := 0; i < maxVerificationRetries; i++ {
		isDisconnected, err := helpers.VerifyNetworkDisconnected(r.client, data.ZoneID.ValueString(), data.ID.ValueString())
		if err != nil {
			log.Printf("Warning: Error verifying network disconnection (attempt %d/%d): %v", i+1, maxVerificationRetries, err)
			if i < maxVerificationRetries-1 {
				time.Sleep(verificationInterval)
				continue
			}
		} else if isDisconnected {
			verified = true
			log.Printf("✓ Network verified as disconnected from all instances (excluding default NICs)")
			break
		} else {
			log.Printf("Network still has connections, waiting... (attempt %d/%d)", i+1, maxVerificationRetries)
			if i < maxVerificationRetries-1 {
				time.Sleep(verificationInterval)
			}
		}
	}

	if !verified {
		log.Printf("Warning: Could not verify network is fully disconnected, proceeding with deletion attempt")
	}

	log.Printf("Step 3: Deleting network...")
	const maxRetries = 5
	baseDelay := 2 * time.Second
	var deleteResponse interface{}

	for attempt := 0; attempt < maxRetries; attempt++ {
		deleteResponse, err = r.client.DeleteNetwork(data.ZoneID.ValueString(), data.ID.ValueString())

		helpers.LogAPIResponse("Network Deletion", deleteResponse, err)

		if err == nil {
			log.Printf("✓ Network deletion request submitted successfully")
			log.Printf("=== Network Deletion Completed ===")
			return
		}

		errStr := err.Error()
		isNetworkConnectedError := strings.Contains(errStr, "network is connected") || strings.Contains(errStr, "The network is connected")

		if !isNetworkConnectedError || attempt == maxRetries-1 {
			log.Printf("Error deleting network: %v", err)
			helpers.HandleAPIError(&resp.Diagnostics, "Client Error", fmt.Errorf("unable to delete network: %w", err))
			return
		}

		delay := baseDelay * time.Duration(1<<uint(attempt))
		log.Printf("Network deletion failed (attempt %d/%d): %v. Retrying in %v...", attempt+1, maxRetries, err, delay)
		time.Sleep(delay)
	}
}

func (r *networkResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data models.NetworkResourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	log.Printf("=== Validating Network Configuration ===")
	log.Printf("Network Name: %s", data.Name.ValueString())
	log.Printf("Zone ID: %s", data.ZoneID.ValueString())
	log.Printf("Network Offering ID: %s", data.NetworkOfferingID.ValueString())
	log.Printf("Network Type: %s", data.Type.ValueString())

	// Validate network offering type
	if !data.NetworkOfferingID.IsNull() && !data.NetworkOfferingID.IsUnknown() && !data.Type.IsNull() && !data.Type.IsUnknown() {
		if err := r.validateNetworkOfferingType(data.ZoneID.ValueString(), data.NetworkOfferingID.ValueString(), data.Type.ValueString(), resp); err != nil {
			log.Printf("❌ Network offering validation failed: %v", err)
			return
		}
	}

	log.Printf("✓ Network configuration validation passed")
	log.Printf("=== End Network Configuration Validation ===")
}

// validateNetworkOfferingType validates that the network offering matches the expected network type
func (r *networkResource) validateNetworkOfferingType(zoneID, networkOfferingID, networkType string, diagnostics *resource.ValidateConfigResponse) error {
	log.Printf("Validating network offering '%s' for network type '%s' in zone '%s'", networkOfferingID, networkType, zoneID)

	// Get all network service offerings for the zone
	serviceOfferings, err := r.client.ListNetworkServiceOfferings(zoneID)
	if err != nil {
		log.Printf("Error getting network service offerings: %v", err)
		// Don't fail validation if API is unavailable, but log the issue
		return nil
	}

	// Find the specific network offering
	var targetOffering *responses.NetworkOffering

	found := false
	for _, offering := range serviceOfferings.Data {
		if offering.ID == networkOfferingID {
			targetOffering = &offering
			found = true
			break
		}
	}

	if !found {
		errMsg := fmt.Sprintf("network offering ID '%s' not found in zone '%s'", networkOfferingID, zoneID)
		diagnostics.Diagnostics.AddError("Invalid Network Offering", errMsg)
		return fmt.Errorf("%s", errMsg)
	}

	log.Printf("Found network offering: %s (ID: %s, Type: %s)", targetOffering.Name, targetOffering.ID, targetOffering.Type)

	// Determine expected type based on network type
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
		// For unknown network types, we'll skip validation but log a warning
		log.Printf("Warning: Unknown network type '%s', skipping offering type validation", networkType)
		return nil
	}

	// Validate that the offering type matches the expected type
	if targetOffering.Type != expectedType {
		errMsg := fmt.Sprintf("network offering '%s' is not of type %s (found type: %s)",
			networkOfferingID, typeDescription, targetOffering.Type)
		diagnostics.Diagnostics.AddError("Invalid Network Offering Type", errMsg)
		return fmt.Errorf("%s", errMsg)
	}

	log.Printf("✓ Network offering type validation passed: offering '%s' has correct type '%s'",
		targetOffering.Name, targetOffering.Type)

	return nil
}
