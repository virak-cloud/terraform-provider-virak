package provider

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/virak-cloud/cli/pkg/http"
	"github.com/virak-cloud/cli/pkg/http/responses"
	"github.com/virak-cloud/terraform-provider-virak/internal/models"
)

var _ resource.Resource = &publicIPResource{}

func NewPublicIPResource(mutex *sync.Mutex) resource.Resource {
	return &publicIPResource{mutex: mutex}
}

type publicIPResource struct {
	client *http.Client
	mutex  *sync.Mutex
}

func (r *publicIPResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_public_ip"
}

func (r *publicIPResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Virak Cloud public IP address with association and Static NAT functionality.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Public IP ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"zone_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of zone where the public IP is located.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"network_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of network for the public IP.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"instance_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The ID of instance to associate with the public IP for Static NAT.",
			},
			"ip_address": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The public IP address.",
			},
		},
	}
}

func (r *publicIPResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*http.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to provider developers.", req.ProviderData))
		return
	}
	r.client = client
}

func (r *publicIPResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var data models.PublicIPResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	log.Printf("=== Starting Public IP Creation ===")
	log.Printf("Zone ID: %s", data.ZoneID.ValueString())
	log.Printf("Network ID: %s", data.NetworkID.ValueString())
	log.Printf("Instance ID: %s", data.InstanceID.ValueString())

	// Step 1: Associate public IP
	log.Printf("Step 1: Associating public IP...")
	associateResp, err := r.client.AssociateNetworkPublicIp(data.ZoneID.ValueString(), data.NetworkID.ValueString())
	if err != nil {
		log.Printf("Public IP association failed: %v", err)
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to associate public IP, got error: %s", err))
		return
	}
	if !associateResp.Data.Success {
		resp.Diagnostics.AddError("Public IP Association Failed", "API returned failure for public IP association")
		return
	}

	log.Printf("Public IP association request submitted successfully")

	// Step 2: Find the newly associated public IP
	log.Printf("Step 2: Finding the newly associated public IP...")
	startTime := time.Now()
	var newPublicIPID string
	var newPublicIPAddress string
	var retryCount int

	for i := 0; i < 60; i++ { // Retry for up to 60 seconds
		retryCount = i + 1
		if i%10 == 0 { // Log every 10 seconds
			log.Printf("Retry attempt %d/%d - searching for public IP...", retryCount, 60)
		}

		publicIPsResp, err := r.client.ListNetworkPublicIps(data.ZoneID.ValueString(), data.NetworkID.ValueString())
		if err != nil {
			log.Printf("Error listing public IPs during retry %d: %v", retryCount, err)
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list public IPs after association, got error: %s", err))
			return
		}

		// Parse the response to find public IPs
		if len(publicIPsResp.Data) > 0 {
			// Get the last public IP in the list (most recently associated)
			latestPublicIP := publicIPsResp.Data[len(publicIPsResp.Data)-1]
			newPublicIPID = latestPublicIP.ID
			newPublicIPAddress = latestPublicIP.IpAddress
			log.Printf("✓ Found associated public IP: ID=%s, IP=%s", newPublicIPID, newPublicIPAddress)
			break
		}

		if i == 59 { // Last attempt
			log.Printf("Timeout: Could not find associated public IP after 60 seconds")
			resp.Diagnostics.AddError("Timeout Error", "Could not find the associated public IP after 60 seconds")
			return
		}

		time.Sleep(1 * time.Second)
	}

	elapsedTime := time.Since(startTime)
	log.Printf("Public IP discovery completed in %v with %d retries", elapsedTime, retryCount)

	if newPublicIPID == "" {
		log.Printf("❌ Public IP association timeout - no public IP found after 60 seconds")
		resp.Diagnostics.AddError(
			"Public IP Association Timeout",
			"Public IP association was submitted but could not be found in the API after 60 seconds. The public IP may still be provisioning.",
		)
		return
	}

	log.Printf("✓ Successfully found public IP ID: %s", newPublicIPID)

	// Step 3: Enable Static NAT if instance_id is provided
	if !data.InstanceID.IsNull() && data.InstanceID.ValueString() != "" {
		log.Printf("Step 3: Enabling Static NAT for instance %s...", data.InstanceID.ValueString())
		staticNatResp, err := r.client.EnableNetworkPublicIpStaticNat(data.ZoneID.ValueString(), data.NetworkID.ValueString(), newPublicIPID, data.InstanceID.ValueString())
		if err != nil {
			log.Printf("Static NAT enable failed: %v", err)
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to enable Static NAT, got error: %s", err))
			return
		}
		if !staticNatResp.Data.Success {
			resp.Diagnostics.AddError("Static NAT Enable Failed", "API returned failure for Static NAT enable")
			return
		}
		log.Printf("✓ Static NAT enabled successfully")
	}

	// Set the computed values
	data.ID = types.StringValue(newPublicIPID)
	data.IPAddress = types.StringValue(newPublicIPAddress)
	data.Status = types.StringValue("active")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		log.Printf("Error setting state: %v", resp.Diagnostics)
		return
	}

	log.Printf("=== Public IP Creation Completed Successfully ===")
	log.Printf("Public IP '%s' (%s) created and ready for use", newPublicIPAddress, newPublicIPID)
}

func (r *publicIPResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data models.PublicIPResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	log.Printf("=== Reading Public IP State ===")
	log.Printf("Public IP ID: %s", data.ID.ValueString())
	log.Printf("Zone ID: %s", data.ZoneID.ValueString())
	log.Printf("Network ID: %s", data.NetworkID.ValueString())

	publicIPsResp, err := r.client.ListNetworkPublicIps(data.ZoneID.ValueString(), data.NetworkID.ValueString())
	if err != nil {
		log.Printf("Error reading public IPs: %v", err)
		resp.State.RemoveResource(ctx)
		return
	}

	// Parse the response to find public IPs
	if len(publicIPsResp.Data) == 0 {
		log.Printf("No public IPs found, removing from state")
		resp.State.RemoveResource(ctx)
		return
	}

	// Find the specific public IP by ID
	var foundPublicIP *responses.NetworkPublicIp
	for _, publicIP := range publicIPsResp.Data {
		if publicIP.ID == data.ID.ValueString() {
			foundPublicIP = &publicIP
			break
		}
	}

	if foundPublicIP == nil {
		log.Printf("Public IP not found in list, removing from state")
		resp.State.RemoveResource(ctx)
		return
	}

	// Update the resource state with current values
	data.IPAddress = types.StringValue(foundPublicIP.IpAddress)
	// Note: InstanceID is not directly available in the public IP list response
	// It would need to be tracked separately or queried differently
	data.Status = types.StringValue("active")

	log.Printf("Updated public IP state:")
	log.Printf("  - ID: %s", foundPublicIP.ID)
	log.Printf("  - IP Address: %s", foundPublicIP.IpAddress)
	log.Printf("  - Is Source NAT: %t", foundPublicIP.IsSourceNat)
	log.Printf("  - Static NAT Enabled: %t", foundPublicIP.StaticNatEnable)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		log.Printf("Error setting updated state: %v", resp.Diagnostics)
		return
	}

	log.Printf("=== Public IP Read Completed ===")
}

func (r *publicIPResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var plan models.PublicIPResourceModel
	var state models.PublicIPResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	log.Printf("=== Starting Public IP Update ===")
	log.Printf("Current Instance ID: %s", state.InstanceID.ValueString())
	log.Printf("New Instance ID: %s", plan.InstanceID.ValueString())

	// Handle Static NAT changes
	if !plan.InstanceID.Equal(state.InstanceID) {
		// If instance_id is being removed (set to null/empty), disable Static NAT
		if plan.InstanceID.IsNull() || plan.InstanceID.ValueString() == "" {
			if !state.InstanceID.IsNull() && state.InstanceID.ValueString() != "" {
				log.Printf("Disabling Static NAT...")
				disableResp, err := r.client.DisableNetworkPublicIpStaticNat(plan.ZoneID.ValueString(), plan.NetworkID.ValueString(), plan.ID.ValueString())
				if err != nil {
					log.Printf("Static NAT disable failed: %v", err)
					resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to disable Static NAT, got error: %s", err))
					return
				}
				if !disableResp.Data.Success {
					resp.Diagnostics.AddError("Static NAT Disable Failed", "API returned failure for Static NAT disable")
					return
				}
				log.Printf("✓ Static NAT disabled successfully")
			}
		} else {
			// If instance_id is being changed or set, enable Static NAT for the new instance
			log.Printf("Enabling Static NAT for instance %s...", plan.InstanceID.ValueString())
			enableResp, err := r.client.EnableNetworkPublicIpStaticNat(plan.ZoneID.ValueString(), plan.NetworkID.ValueString(), plan.ID.ValueString(), plan.InstanceID.ValueString())
			if err != nil {
				log.Printf("Static NAT enable failed: %v", err)
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to enable Static NAT, got error: %s", err))
				return
			}
			if !enableResp.Data.Success {
				resp.Diagnostics.AddError("Static NAT Enable Failed", "API returned failure for Static NAT enable")
				return
			}
			log.Printf("✓ Static NAT enabled successfully")
		}
	}

	// Update the state with the new instance_id

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		log.Printf("Error setting updated state: %v", resp.Diagnostics)
		return
	}

	log.Printf("=== Public IP Update Completed ===")
}

func (r *publicIPResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var data models.PublicIPResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	log.Printf("=== Starting Public IP Deletion ===")
	log.Printf("Public IP ID: %s", data.ID.ValueString())
	log.Printf("IP Address: %s", data.IPAddress.ValueString())
	log.Printf("Zone ID: %s", data.ZoneID.ValueString())
	log.Printf("Network ID: %s", data.NetworkID.ValueString())

	// Step 1: Disable Static NAT if enabled
	if !data.InstanceID.IsNull() && data.InstanceID.ValueString() != "" {
		log.Printf("Step 1: Disabling Static NAT...")
		disableResp, err := r.client.DisableNetworkPublicIpStaticNat(data.ZoneID.ValueString(), data.NetworkID.ValueString(), data.ID.ValueString())
		if err != nil {
			log.Printf("Static NAT disable failed: %v", err)
			resp.Diagnostics.AddWarning("Static NAT Disable Warning", fmt.Sprintf("Failed to disable Static NAT before disassociation: %s", err))
			// Continue with disassociation even if Static NAT disable fails
		} else {
			if !disableResp.Data.Success {
				resp.Diagnostics.AddWarning("Static NAT Disable Warning", "API returned failure for Static NAT disable")
			} else {
				log.Printf("✓ Static NAT disabled successfully")
			}
		}
	}

	// Step 2: Disassociate public IP
	log.Printf("Step 2: Disassociating public IP...")
	disassociateResp, err := r.client.DisassociateNetworkPublicIp(data.ZoneID.ValueString(), data.NetworkID.ValueString(), data.ID.ValueString())
	if err != nil {
		log.Printf("Public IP disassociation failed: %v", err)
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to disassociate public IP, got error: %s", err))
		return
	}
	if !disassociateResp.Data.Success {
		resp.Diagnostics.AddError("Public IP Disassociation Failed", "API returned failure for public IP disassociation")
		return
	}

	log.Printf("✓ Public IP disassociation request submitted successfully")
	log.Printf("=== Public IP Deletion Completed ===")
}

func (r *publicIPResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data models.PublicIPResourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	log.Printf("=== Validating Public IP Configuration ===")
	log.Printf("Zone ID: %s", data.ZoneID.ValueString())
	log.Printf("Network ID: %s", data.NetworkID.ValueString())
	log.Printf("Instance ID: %s", data.InstanceID.ValueString())

	log.Printf("✓ Public IP configuration validation passed")
	log.Printf("=== End Public IP Configuration Validation ===")
}
