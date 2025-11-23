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
	"github.com/virak-cloud/cli/pkg/http"
	"github.com/virak-cloud/terraform-provider-virak/internal/models"
)

// Ensure the implementation satisfies the resource.Resource interface.
var _ resource.Resource = &firewallRuleResource{}

func NewFirewallRuleResource(mutex *sync.Mutex) resource.Resource {
	return &firewallRuleResource{mutex: mutex}
}

type firewallRuleResource struct {
	client *http.Client
	mutex  *sync.Mutex
}

func (r *firewallRuleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_firewall_rule"
}

func (r *firewallRuleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Virak Cloud firewall rule.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Firewall rule ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"zone_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the zone.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"network_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the network.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"ip_version": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "IP version (ipv4 or ipv6).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"traffic_type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Traffic type (Ingress or Egress).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"protocol": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Protocol (TCP, UDP, or ICMP).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"ip_source": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Source IP address or CIDR.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"ip_destination": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Destination IP address or CIDR.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"start_port": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Start port (for TCP/UDP protocols).",
				PlanModifiers:       []planmodifier.Int64{
					// TODO: Add RequiresReplace if needed
				},
			},
			"end_port": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "End port (for TCP/UDP protocols).",
				PlanModifiers:       []planmodifier.Int64{
					// TODO: Add RequiresReplace if needed
				},
			},
			"icmp_code": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "ICMP code (for ICMP protocol).",
				PlanModifiers:       []planmodifier.Int64{
					// TODO: Add RequiresReplace if needed
				},
			},
			"icmp_type": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "ICMP type (for ICMP protocol).",
				PlanModifiers:       []planmodifier.Int64{
					// TODO: Add RequiresReplace if needed
				},
			},
			"public_ip_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Public IP ID (for IPv4 ingress rules).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The status of the firewall rule.",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The creation timestamp of the firewall rule.",
			},
		},
	}
}

func (r *firewallRuleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *firewallRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var data models.FirewallRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ipVersion := data.IPVersion.ValueString()
	networkID := data.NetworkID.ValueString()
	zoneID := data.ZoneID.ValueString()

	// Build the request body
	body := map[string]interface{}{
		"traffic_type":   data.TrafficType.ValueString(),
		"protocol_type":  data.Protocol.ValueString(),
		"ip_source":      data.IPSource.ValueString(),
		"ip_destination": data.IPDestination.ValueString(),
	}

	// Add protocol-specific fields
	protocol := data.Protocol.ValueString()
	if protocol == "ICMP" {
		if !data.ICMPCode.IsNull() {
			body["icmp_code"] = int(data.ICMPCode.ValueInt64())
		}
		if !data.ICMPType.IsNull() {
			body["icmp_type"] = int(data.ICMPType.ValueInt64())
		}
	} else {
		if !data.StartPort.IsNull() {
			body["port_start"] = int(data.StartPort.ValueInt64())
		}
		if !data.EndPort.IsNull() {
			body["port_end"] = int(data.EndPort.ValueInt64())
		}
	}

	// Add public IP ID for IPv4 ingress rules
	if ipVersion == "ipv4" && !data.PublicIPID.IsNull() {
		body["public_ip_id"] = data.PublicIPID.ValueString()
	}

	// Create the firewall rule

	if ipVersion == "ipv4" {
		createResp, err := r.client.CreateIPv4FirewallRule(zoneID, networkID, body)
		if err != nil {
			resp.Diagnostics.AddError(
				"Firewall Rule Creation Failed",
				fmt.Sprintf("Failed to create IPv4 firewall rule in network %s. Error: %s", networkID, err),
			)
			return
		}
		if !createResp.Data.Success {
			resp.Diagnostics.AddError(
				"Firewall Rule Creation Failed",
				"API returned failure for firewall rule creation",
			)
			return
		}
	} else if ipVersion == "ipv6" {
		createResp, err := r.client.CreateIPv6FirewallRule(zoneID, networkID, body)
		if err != nil {
			resp.Diagnostics.AddError(
				"Firewall Rule Creation Failed",
				fmt.Sprintf("Failed to create IPv6 firewall rule in network %s. Error: %s", networkID, err),
			)
			return
		}
		if !createResp.Data.Success {
			resp.Diagnostics.AddError(
				"Firewall Rule Creation Failed",
				"API returned failure for firewall rule creation",
			)
			return
		}
	} else {
		resp.Diagnostics.AddError(
			"Invalid IP Version",
			fmt.Sprintf("IP version must be 'ipv4' or 'ipv6', got '%s'", ipVersion),
		)
		return
	}

	// For now, we need to list the rules to get the ID since the create response doesn't return the ID
	// This is a limitation of the current API
	var ruleID string
	var ruleStatus string
	var createdAt string

	if ipVersion == "ipv4" {
		listResp, err := r.client.ListIPv4FirewallRules(zoneID, networkID)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list firewall rules after creation, got error: %s", err))
			return
		}
		// Find the most recently created rule (this is approximate)
		for _, rule := range listResp.Data {
			// For now, we'll assume the last rule in the list is the one we just created
			// This is not ideal but works for basic functionality
			ruleID = rule.ID
			ruleStatus = rule.Status
			createdAt = fmt.Sprintf("%d", rule.CreatedAt)
		}
	} else {
		listResp, err := r.client.ListIPv6FirewallRules(zoneID, networkID)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list firewall rules after creation, got error: %s", err))
			return
		}
		for _, rule := range listResp.Data {
			ruleID = rule.ID
			ruleStatus = rule.Status
			createdAt = fmt.Sprintf("%d", rule.CreatedAt)
		}
	}

	if ruleID == "" {
		resp.Diagnostics.AddError(
			"Firewall Rule Creation Error",
			"Firewall rule was created but could not retrieve the rule ID",
		)
		return
	}

	data.ID = types.StringValue(ruleID)
	data.Status = types.StringValue(ruleStatus)
	data.CreatedAt = types.StringValue(createdAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *firewallRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data models.FirewallRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ipVersion := data.IPVersion.ValueString()
	networkID := data.NetworkID.ValueString()
	zoneID := data.ZoneID.ValueString()
	ruleID := data.ID.ValueString()

	// Find the rule in the list
	if ipVersion == "ipv4" {
		listResp, err := r.client.ListIPv4FirewallRules(zoneID, networkID)
		if err != nil {
			resp.State.RemoveResource(ctx)
			return
		}
		for _, rule := range listResp.Data {
			if rule.ID == ruleID {
				data.Status = types.StringValue(rule.Status)
				data.CreatedAt = types.StringValue(fmt.Sprintf("%d", rule.CreatedAt))
				break
			}
		}
	} else {
		listResp, err := r.client.ListIPv6FirewallRules(zoneID, networkID)
		if err != nil {
			resp.State.RemoveResource(ctx)
			return
		}
		for _, rule := range listResp.Data {
			if rule.ID == ruleID {
				data.Status = types.StringValue(rule.Status)
				data.CreatedAt = types.StringValue(fmt.Sprintf("%d", rule.CreatedAt))
				break
			}
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *firewallRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Firewall rules don't support updates - they must be recreated
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"Firewall rule resources do not support updates. To change a firewall rule, delete and recreate it.",
	)
}

func (r *firewallRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var data models.FirewallRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ipVersion := data.IPVersion.ValueString()
	networkID := data.NetworkID.ValueString()
	zoneID := data.ZoneID.ValueString()
	ruleID := data.ID.ValueString()

	// Delete the firewall rule
	if ipVersion == "ipv4" {
		deleteResp, err := r.client.DeleteIPv4FirewallRule(zoneID, networkID, ruleID)
		if err != nil {
			resp.Diagnostics.AddError(
				"Firewall Rule Deletion Failed",
				fmt.Sprintf("Failed to delete IPv4 firewall rule %s. Error: %s", ruleID, err),
			)
			return
		}
		if !deleteResp.Data.Success {
			resp.Diagnostics.AddError(
				"Firewall Rule Deletion Failed",
				fmt.Sprintf("API returned failure for IPv4 firewall rule deletion %s", ruleID),
			)
			return
		}
	} else {
		deleteResp, err := r.client.DeleteIPv6FirewallRule(zoneID, networkID, ruleID)
		if err != nil {
			resp.Diagnostics.AddError(
				"Firewall Rule Deletion Failed",
				fmt.Sprintf("Failed to delete IPv6 firewall rule %s. Error: %s", ruleID, err),
			)
			return
		}
		if !deleteResp.Data.Success {
			resp.Diagnostics.AddError(
				"Firewall Rule Deletion Failed",
				fmt.Sprintf("API returned failure for IPv6 firewall rule deletion %s", ruleID),
			)
			return
		}
	}
}
