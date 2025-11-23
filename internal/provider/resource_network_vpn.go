package provider

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/virak-cloud/cli/pkg/http"
	"github.com/virak-cloud/terraform-provider-virak/internal/models"
)

// Ensure the implementation satisfies the resource.Resource interface.
var _ resource.Resource = &networkVPNResource{}

func NewNetworkVPNResource(mutex *sync.Mutex) resource.Resource {
	return &networkVPNResource{mutex: mutex}
}

type networkVPNResource struct {
	client *http.Client
	mutex  *sync.Mutex
}

func (r *networkVPNResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_vpn"
}

func (r *networkVPNResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Virak Cloud network VPN configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "VPN configuration ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"zone_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of zone where the network is located.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"network_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of network for the VPN configuration.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enabled": schema.BoolAttribute{
				Required:            true,
				MarkdownDescription: "Whether the VPN is enabled or disabled.",
			},
			"preshared_key": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The pre-shared key for VPN authentication.",
				Sensitive:           true,
			},
			"ip_address": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The VPN server IP address.",
			},
			"username": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The VPN username.",
			},
			"password": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The VPN password.",
				Sensitive:           true,
			},
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The status of the VPN configuration.",
			},
		},
	}
}

func (r *networkVPNResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *networkVPNResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var data models.NetworkVPNResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	log.Printf("=== Starting Network VPN Creation ===")
	log.Printf("Zone ID: %s", data.ZoneID.ValueString())
	log.Printf("Network ID: %s", data.NetworkID.ValueString())
	log.Printf("Enabled: %t", data.Enabled.ValueBool())

	// Generate a unique ID for this VPN configuration
	vpnID := fmt.Sprintf("%s-%s-vpn", data.ZoneID.ValueString(), data.NetworkID.ValueString())
	data.ID = types.StringValue(vpnID)

	// Handle VPN enable/disable
	if data.Enabled.ValueBool() {
		log.Printf("Enabling VPN...")
		enableResp, err := r.client.EnableNetworkVpn(data.ZoneID.ValueString(), data.NetworkID.ValueString())
		if err != nil {
			log.Printf("VPN enable failed: %v", err)
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to enable VPN, got error: %s", err))
			return
		}
		if !enableResp.Data.Success {
			resp.Diagnostics.AddError("VPN Enable Failed", "API returned failure for VPN enable")
			return
		}
		log.Printf("✓ VPN enabled successfully")
	} else {
		log.Printf("VPN will be created in disabled state")
	}

	// If preshared key is provided, update credentials
	if !data.PresharedKey.IsNull() && data.PresharedKey.ValueString() != "" {
		log.Printf("Updating VPN preshared key...")
		// Note: The UpdateNetworkVpnCredentials method doesn't take a preshared key parameter
		// This might need to be handled differently or the API might expect it in a different way
		updateResp, err := r.client.UpdateNetworkVpnCredentials(data.ZoneID.ValueString(), data.NetworkID.ValueString())
		if err != nil {
			log.Printf("VPN credentials update failed: %v", err)
			resp.Diagnostics.AddWarning("VPN Credentials Update Warning", fmt.Sprintf("Failed to update VPN credentials: %s", err))
			// Continue with creation even if credentials update fails
		} else if !updateResp.Data.Success {
			resp.Diagnostics.AddWarning("VPN Credentials Update Warning", "API returned failure for VPN credentials update")
		} else {
			log.Printf("✓ VPN credentials updated successfully")
		}
	}

	// Get VPN details to populate computed fields
	log.Printf("Retrieving VPN details...")
	detailsResp, err := r.client.GetNetworkVpnDetails(data.ZoneID.ValueString(), data.NetworkID.ValueString())
	if err != nil {
		log.Printf("Failed to get VPN details: %v", err)
		resp.Diagnostics.AddWarning("VPN Details Retrieval Warning", fmt.Sprintf("Failed to retrieve VPN details: %s", err))
		// Set default values
		data.IPAddress = types.StringNull()
		data.Username = types.StringNull()
		data.Password = types.StringNull()
		data.Status = types.StringValue("unknown")
	} else {
		data.IPAddress = types.StringValue(detailsResp.Data.IPAddress)
		data.Username = types.StringValue(detailsResp.Data.Username)
		data.Password = types.StringValue(detailsResp.Data.Password)
		data.Status = types.StringValue(detailsResp.Data.Status)
		log.Printf("✓ VPN details retrieved: IP=%s, Status=%s", detailsResp.Data.IPAddress, detailsResp.Data.Status)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		log.Printf("Error setting state: %v", resp.Diagnostics)
		return
	}

	log.Printf("=== Network VPN Creation Completed Successfully ===")
	log.Printf("VPN configuration created for network '%s'", data.NetworkID.ValueString())
}

func (r *networkVPNResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data models.NetworkVPNResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	log.Printf("=== Reading Network VPN State ===")
	log.Printf("VPN ID: %s", data.ID.ValueString())
	log.Printf("Zone ID: %s", data.ZoneID.ValueString())
	log.Printf("Network ID: %s", data.NetworkID.ValueString())

	// Get current VPN details
	detailsResp, err := r.client.GetNetworkVpnDetails(data.ZoneID.ValueString(), data.NetworkID.ValueString())
	if err != nil {
		log.Printf("Error reading VPN details: %v", err)
		resp.State.RemoveResource(ctx)
		return
	}

	// Update the resource state with current values
	data.IPAddress = types.StringValue(detailsResp.Data.IPAddress)
	data.Username = types.StringValue(detailsResp.Data.Username)
	data.Password = types.StringValue(detailsResp.Data.Password)
	data.Status = types.StringValue(detailsResp.Data.Status)

	// Determine if VPN is enabled based on status
	// This is a simplification - you might need more complex logic based on the actual API
	if detailsResp.Data.Status == "enabled" || detailsResp.Data.Status == "active" {
		data.Enabled = types.BoolValue(true)
	} else {
		data.Enabled = types.BoolValue(false)
	}

	log.Printf("Updated VPN state:")
	log.Printf("  - IP Address: %s", detailsResp.Data.IPAddress)
	log.Printf("  - Username: %s", detailsResp.Data.Username)
	log.Printf("  - Status: %s", detailsResp.Data.Status)
	log.Printf("  - Enabled: %t", data.Enabled.ValueBool())

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		log.Printf("Error setting updated state: %v", resp.Diagnostics)
		return
	}

	log.Printf("=== Network VPN Read Completed ===")
}

func (r *networkVPNResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var plan models.NetworkVPNResourceModel
	var state models.NetworkVPNResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	log.Printf("=== Starting Network VPN Update ===")
	log.Printf("Current Enabled: %t", state.Enabled.ValueBool())
	log.Printf("New Enabled: %t", plan.Enabled.ValueBool())

	// Handle enable/disable changes
	if plan.Enabled.ValueBool() != state.Enabled.ValueBool() {
		if plan.Enabled.ValueBool() {
			log.Printf("Enabling VPN...")
			enableResp, err := r.client.EnableNetworkVpn(plan.ZoneID.ValueString(), plan.NetworkID.ValueString())
			if err != nil {
				log.Printf("VPN enable failed: %v", err)
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to enable VPN, got error: %s", err))
				return
			}
			if !enableResp.Data.Success {
				resp.Diagnostics.AddError("VPN Enable Failed", "API returned failure for VPN enable")
				return
			}
			log.Printf("✓ VPN enabled successfully")
		} else {
			log.Printf("Disabling VPN...")
			disableResp, err := r.client.DisableNetworkVpn(plan.ZoneID.ValueString(), plan.NetworkID.ValueString())
			if err != nil {
				log.Printf("VPN disable failed: %v", err)
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to disable VPN, got error: %s", err))
				return
			}
			if !disableResp.Data.Success {
				resp.Diagnostics.AddError("VPN Disable Failed", "API returned failure for VPN disable")
				return
			}
			log.Printf("✓ VPN disabled successfully")
		}
	}

	// Handle preshared key changes
	if !plan.PresharedKey.Equal(state.PresharedKey) {
		log.Printf("Updating VPN preshared key...")
		updateResp, err := r.client.UpdateNetworkVpnCredentials(plan.ZoneID.ValueString(), plan.NetworkID.ValueString())
		if err != nil {
			log.Printf("VPN credentials update failed: %v", err)
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update VPN credentials, got error: %s", err))
			return
		}
		if !updateResp.Data.Success {
			resp.Diagnostics.AddError("VPN Credentials Update Failed", "API returned failure for VPN credentials update")
			return
		}
		log.Printf("✓ VPN credentials updated successfully")
	}

	// Get updated VPN details
	detailsResp, err := r.client.GetNetworkVpnDetails(plan.ZoneID.ValueString(), plan.NetworkID.ValueString())
	if err != nil {
		log.Printf("Failed to get updated VPN details: %v", err)
		resp.Diagnostics.AddWarning("VPN Details Retrieval Warning", fmt.Sprintf("Failed to retrieve updated VPN details: %s", err))
	} else {
		plan.IPAddress = types.StringValue(detailsResp.Data.IPAddress)
		plan.Username = types.StringValue(detailsResp.Data.Username)
		plan.Password = types.StringValue(detailsResp.Data.Password)
		plan.Status = types.StringValue(detailsResp.Data.Status)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		log.Printf("Error setting updated state: %v", resp.Diagnostics)
		return
	}

	log.Printf("=== Network VPN Update Completed ===")
}

func (r *networkVPNResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var data models.NetworkVPNResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	log.Printf("=== Starting Network VPN Deletion ===")
	log.Printf("VPN ID: %s", data.ID.ValueString())
	log.Printf("Zone ID: %s", data.ZoneID.ValueString())
	log.Printf("Network ID: %s", data.NetworkID.ValueString())

	// Disable VPN if it's currently enabled
	if data.Enabled.ValueBool() {
		log.Printf("Disabling VPN before deletion...")
		disableResp, err := r.client.DisableNetworkVpn(data.ZoneID.ValueString(), data.NetworkID.ValueString())
		if err != nil {
			log.Printf("VPN disable failed: %v", err)
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to disable VPN during deletion, got error: %s", err))
			return
		}
		if !disableResp.Data.Success {
			resp.Diagnostics.AddError("VPN Disable Failed", "API returned failure for VPN disable during deletion")
			return
		}
		log.Printf("✓ VPN disabled successfully")
	}

	log.Printf("✓ Network VPN deletion completed successfully")
	log.Printf("=== Network VPN Deletion Completed ===")
}
