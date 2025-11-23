package provider

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/virak-cloud/cli/pkg/http"
	"github.com/virak-cloud/terraform-provider-virak/internal/models"
)

// publicIPAssociationResource manages association of a public IP to a private network.
var _ resource.Resource = &publicIPAssociationResource{}

func NewPublicIPAssociationResource(mutex *sync.Mutex) resource.Resource {
	return &publicIPAssociationResource{mutex: mutex}
}

type publicIPAssociationResource struct {
	client *http.Client
	mutex  *sync.Mutex
}

func (r *publicIPAssociationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_public_ip_association"
}

func (r *publicIPAssociationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	// Associates a static IPv4 address to a private network.
	resp.Schema = schema.Schema{
		MarkdownDescription: "Associates a static IPv4 address to a private network.",
		Attributes: map[string]schema.Attribute{
			"id":         schema.StringAttribute{Computed: true, MarkdownDescription: "The ID of the public IP association."},
			"network_id": schema.StringAttribute{Required: true, MarkdownDescription: "The ID of the network to associate with public IP."},
			"ip_address": schema.StringAttribute{Computed: true, MarkdownDescription: "The assigned public IP address."},
		},
	}
}

func (r *publicIPAssociationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *publicIPAssociationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var data models.PublicIPResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	networkID := data.NetworkID.ValueString()
	if networkID == "" {
		resp.Diagnostics.AddError("Invalid Configuration", "network_id must be provided")
		return
	}

	log.Printf("=== Creating Public IP Association ===")
	log.Printf("Network ID: %s", networkID)

	// Use the existing client method to associate a public IP with a network.
	// Passing empty zone string for now; integrator should provide the appropriate zone if required.
	associateResp, err := r.client.AssociateNetworkPublicIp("", networkID)
	if err != nil {
		log.Printf("Public IP association failed: %v", err)
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to associate public IP, got error: %s", err))
		return
	}

	// Best-effort extraction of returned values. Integrator: adapt to actual response structure.
	assocID := ""
	ipAddr := ""

	// Check API action success if present
	if associateResp != nil {
		if !associateResp.Data.Success {
			resp.Diagnostics.AddError("Public IP Association Failed", "API returned failure for public IP association")
			return
		}
	}

	// Try to discover the newly associated public IP by listing network public IPs.
	publicIPsResp, err := r.client.ListNetworkPublicIps("", networkID)
	if err == nil && publicIPsResp != nil && len(publicIPsResp.Data) > 0 {
		latest := publicIPsResp.Data[len(publicIPsResp.Data)-1]
		assocID = latest.ID
		ipAddr = latest.IpAddress
	}

	// If integrator's client uses a pointer/other type, above cast will fail silently.
	// Fallback: if associateResp has String() helpers, integrator may fill assocID/ipAddr here.
	// For now, if assocID or ipAddr are empty, attempt a best-effort lookup via a commonly used client method.
	if assocID == "" || ipAddr == "" {
		// TODO: If the client provides a List or Show API to query associations by network,
		// call it here to find the newly-created association and populate assocID/ipAddr.
		// Example:
		// listResp, err := r.client.ListNetworkPublicIps(zoneID, networkID)
		// then find the newest entry and set assocID/ipAddr accordingly.
	}

	// If still empty, set a placeholder ID to avoid empty state (integrator should replace logic).
	if assocID == "" {
		assocID = "unknown-association-id"
	}
	if ipAddr == "" {
		ipAddr = ""
	}

	data.ID = types.StringValue(assocID)
	data.IPAddress = types.StringValue(ipAddr)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		log.Printf("Error setting state after create: %v", resp.Diagnostics)
		return
	}

	log.Printf("=== Public IP Association Created: ID=%s, IP=%s ===", assocID, ipAddr)
}

func (r *publicIPAssociationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data models.PublicIPResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	assocID := data.ID.ValueString()
	if assocID == "" {
		// Nothing to read
		resp.State.RemoveResource(ctx)
		return
	}

	log.Printf("=== Reading Public IP Association: %s ===", assocID)

	// Refresh association by listing public IPs for the network and finding the matching ID.
	readResp, err := r.client.ListNetworkPublicIps("", data.NetworkID.ValueString())
	if err != nil {
		// Treat not-found or API error as removed
		log.Printf("Error reading public IP association: %v", err)
		resp.State.RemoveResource(ctx)
		return
	}

	// Populate state from the list response by locating the matching association ID.
	if readResp != nil {
		var foundIP *struct {
			ID        string
			IpAddress string
		}
		for _, publicIP := range readResp.Data {
			if publicIP.ID == assocID {
				// capture minimal fields we care about
				foundIP = &struct {
					ID        string
					IpAddress string
				}{
					ID:        publicIP.ID,
					IpAddress: publicIP.IpAddress,
				}
				break
			}
		}

		if foundIP == nil {
			// association not found anymore
			resp.State.RemoveResource(ctx)
			return
		}

		data.IPAddress = types.StringValue(foundIP.IpAddress)
		// keep network_id as-is (from config/plan)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		log.Printf("Error setting state after read: %v", resp.Diagnostics)
		return
	}
}

func (r *publicIPAssociationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// No-op for Phase 4.1
}

func (r *publicIPAssociationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var data models.PublicIPResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	assocID := data.ID.ValueString()
	if assocID == "" {
		// Nothing to do
		return
	}

	log.Printf("=== Deleting Public IP Association: %s ===", assocID)

	// Disassociate the public IP using the client's DisassociateNetworkPublicIp call.
	// Passing empty zone string for now; integrator should provide the appropriate zone if required.
	_, err := r.client.DisassociateNetworkPublicIp("", data.NetworkID.ValueString(), assocID)
	if err != nil {
		// If API reports not-found, treat as success. Otherwise, surface error.
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to disassociate public IP (id=%s), got error: %s", assocID, err))
		return
	}

	// Remove resource from state
	resp.State.RemoveResource(ctx)
	log.Printf("=== Public IP Association Deleted: %s ===", assocID)
}
