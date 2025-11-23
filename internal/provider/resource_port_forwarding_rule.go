package provider

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/virak-cloud/cli/pkg/http"
	"github.com/virak-cloud/terraform-provider-virak/internal/models"
)

var _ resource.Resource = &portForwardingRuleResource{}

func NewPortForwardingRuleResource(mutex *sync.Mutex) resource.Resource {
	return &portForwardingRuleResource{mutex: mutex}
}

type portForwardingRuleResource struct {
	client *http.Client
	mutex  *sync.Mutex
}

func (r *portForwardingRuleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_port_forwarding_rule"
}

func (r *portForwardingRuleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Virak Cloud port forwarding rule.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Port forwarding rule ID",
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
			"public_ip_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the public IP address.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"protocol": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The protocol (TCP or UDP).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"public_port": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "The public port number.",
				PlanModifiers:       []planmodifier.Int64{
					// RequiresReplace if needed
				},
			},
			"private_port": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "The private port number.",
				PlanModifiers:       []planmodifier.Int64{
					// RequiresReplace if needed
				},
			},
			"instance_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the instance.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"private_ip": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The private IP address of the instance.",
			},
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The status of the port forwarding rule.",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The creation timestamp of the port forwarding rule.",
			},
		},
	}
}

func (r *portForwardingRuleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *portForwardingRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var data models.PortForwardingRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// First get port forward list
	_, err := r.client.ListPortForwards(
		data.ZoneID.ValueString(),
		data.NetworkID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to List Port Forwards",
			fmt.Sprintf("Failed to list port forwards before creation. Error: %s", err),
		)
		return
	}

	body := map[string]interface{}{
		"network_id":   data.NetworkID.ValueString(),
		"protocol":     data.Protocol.ValueString(),
		"public_port":  int(data.PublicPort.ValueInt64()),
		"private_port": int(data.PrivatePort.ValueInt64()),
		"private_ip":   data.PrivateIP.ValueString(),
	}

	_, err = r.client.CreatePortForward(
		data.ZoneID.ValueString(),
		body,
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Port Forwarding Rule Creation Failed",
			fmt.Sprintf("Failed to create port forwarding rule. Error: %s", err),
		)
		return
	}

	// Poll to ensure creation happened, with 5 second backoff for up to 600 seconds
	for i := 0; i < 120; i++ {
		listResp, err := r.client.ListPortForwards(
			data.ZoneID.ValueString(),
			data.NetworkID.ValueString(),
		)
		if err == nil {
			for _, rule := range listResp.Data {
				if rule.ID == data.ID.ValueString() {
					data.Status = types.StringValue(rule.Status)
					data.CreatedAt = types.StringValue(fmt.Sprintf("%d", rule.CreatedAt))
					if rule.PrivateIP != "" {
						data.PrivateIP = types.StringValue(rule.PrivateIP)
					}
					break
				}
			}
			if data.Status.ValueString() != "" {
				break
			}
		}
		time.Sleep(5 * time.Second)
	}

	if data.Status.ValueString() == "" {
		resp.Diagnostics.AddError(
			"Port Forwarding Rule Creation Verification Failed",
			"The port forwarding rule was not found in the list after creation.",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *portForwardingRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data models.PortForwardingRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	listResp, err := r.client.ListPortForwards(
		data.ZoneID.ValueString(),
		data.NetworkID.ValueString(),
	)
	if err != nil {
		resp.State.RemoveResource(ctx)
		return
	}

	var foundRule *models.PortForwardingRuleResourceModel
	for _, rule := range listResp.Data {
		if rule.ID == data.ID.ValueString() {
			foundRule = &models.PortForwardingRuleResourceModel{
				ID:          types.StringValue(rule.ID),
				ZoneID:      data.ZoneID,
				NetworkID:   data.NetworkID,
				PublicIPID:  data.PublicIPID,
				Protocol:    types.StringValue(rule.Protocol),
				PublicPort:  types.Int64Value(int64(rule.PublicPort)),
				PrivatePort: types.Int64Value(int64(rule.PrivatePort)),
				InstanceID:  data.InstanceID,
				PrivateIP:   types.StringValue(rule.PrivateIP),
				Status:      types.StringValue(rule.Status),
				CreatedAt:   types.StringValue(fmt.Sprintf("%d", rule.CreatedAt)),
			}
			break
		}
	}

	if foundRule == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data = *foundRule
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *portForwardingRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"Port forwarding rule resources do not support updates. To change a port forwarding rule, delete and recreate it.",
	)
}

func (r *portForwardingRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var data models.PortForwardingRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// First get port forward list
	_, err := r.client.ListPortForwards(
		data.ZoneID.ValueString(),
		data.NetworkID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to List Port Forwards",
			fmt.Sprintf("Failed to list port forwards before deletion. Error: %s", err),
		)
		return
	}

	_, err = r.client.DeletePortForward(
		data.ZoneID.ValueString(),
		data.ID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Port Forwarding Rule Deletion Failed",
			fmt.Sprintf("Failed to delete port forwarding rule '%s'. Error: %s", data.ID.ValueString(), err),
		)
		return
	}

	// Poll to ensure deletion happened, with 5 second backoff for up to 600 seconds
	for i := 0; i < 120; i++ {
		listResp, err := r.client.ListPortForwards(
			data.ZoneID.ValueString(),
			data.NetworkID.ValueString(),
		)
		if err == nil {
			found := false
			for _, rule := range listResp.Data {
				if rule.ID == data.ID.ValueString() {
					found = true
					break
				}
			}
			if !found {
				return // successfully deleted
			}
		}
		time.Sleep(5 * time.Second)
	}

	// If here, deletion not verified
	resp.Diagnostics.AddError(
		"Port Forwarding Rule Deletion Verification Failed",
		"The port forwarding rule was still found in the list after deletion.",
	)
}
