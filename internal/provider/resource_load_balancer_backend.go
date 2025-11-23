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
var _ resource.Resource = &loadBalancerBackendResource{}

func NewLoadBalancerBackendResource(mutex *sync.Mutex) resource.Resource {
	return &loadBalancerBackendResource{mutex: mutex}
}

type loadBalancerBackendResource struct {
	client *http.Client
	mutex  *sync.Mutex
}

func (r *loadBalancerBackendResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_load_balancer_backend"
}

func (r *loadBalancerBackendResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Virak Cloud load balancer backend instance assignment.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Load balancer backend assignment ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"zone_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of zone where the load balancer is located.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"network_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of network for the load balancer.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"load_balancer_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the load balancer rule.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"instance_network_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the instance network interface to assign to the load balancer.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *loadBalancerBackendResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *loadBalancerBackendResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var data models.LoadBalancerBackendResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	log.Printf("=== Starting Load Balancer Backend Assignment ===")
	log.Printf("Zone ID: %s", data.ZoneID.ValueString())
	log.Printf("Network ID: %s", data.NetworkID.ValueString())
	log.Printf("Load Balancer ID: %s", data.LoadBalancerID.ValueString())
	log.Printf("Instance Network ID: %s", data.InstanceNetworkID.ValueString())

	// Assign the instance to the load balancer
	assignResp, err := r.client.AssignLoadBalancerRule(
		data.ZoneID.ValueString(),
		data.NetworkID.ValueString(),
		data.LoadBalancerID.ValueString(),
		[]string{data.InstanceNetworkID.ValueString()},
	)
	if err != nil {
		log.Printf("Load balancer backend assignment failed: %v", err)
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to assign instance to load balancer, got error: %s", err))
		return
	}
	if !assignResp.Data.Success {
		resp.Diagnostics.AddError("Load Balancer Backend Assignment Failed", "API returned failure for load balancer backend assignment")
		return
	}

	log.Printf("✓ Load balancer backend assignment request submitted successfully")

	// Generate a unique ID for this assignment
	assignmentID := fmt.Sprintf("%s-%s", data.LoadBalancerID.ValueString(), data.InstanceNetworkID.ValueString())
	data.ID = types.StringValue(assignmentID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		log.Printf("Error setting state: %v", resp.Diagnostics)
		return
	}

	log.Printf("=== Load Balancer Backend Assignment Completed Successfully ===")
	log.Printf("Instance network '%s' assigned to load balancer '%s'", data.InstanceNetworkID.ValueString(), data.LoadBalancerID.ValueString())
}

func (r *loadBalancerBackendResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data models.LoadBalancerBackendResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	log.Printf("=== Reading Load Balancer Backend State ===")
	log.Printf("Backend ID: %s", data.ID.ValueString())
	log.Printf("Load Balancer ID: %s", data.LoadBalancerID.ValueString())
	log.Printf("Instance Network ID: %s", data.InstanceNetworkID.ValueString())

	// For now, we assume the assignment exists if the resource exists
	// In a more complete implementation, we might query the load balancer to verify assignments
	// But the API doesn't seem to provide a way to list assignments directly

	log.Printf("Load balancer backend assignment exists")
	log.Printf("=== Load Balancer Backend Read Completed ===")
}

func (r *loadBalancerBackendResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var plan models.LoadBalancerBackendResourceModel
	var state models.LoadBalancerBackendResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	log.Printf("=== Starting Load Balancer Backend Update ===")
	log.Printf("Current Instance Network ID: %s", state.InstanceNetworkID.ValueString())
	log.Printf("New Instance Network ID: %s", plan.InstanceNetworkID.ValueString())

	// Load balancer backend assignments are immutable - require replacement for changes
	resp.Diagnostics.AddError("Load Balancer Backend Update Not Supported", "Load balancer backend assignments cannot be updated. Please create a new assignment with the desired changes.")
}

func (r *loadBalancerBackendResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var data models.LoadBalancerBackendResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	log.Printf("=== Starting Load Balancer Backend Removal ===")
	log.Printf("Backend ID: %s", data.ID.ValueString())
	log.Printf("Load Balancer ID: %s", data.LoadBalancerID.ValueString())
	log.Printf("Instance Network ID: %s", data.InstanceNetworkID.ValueString())

	// Remove the instance from the load balancer
	deassignResp, err := r.client.DeassignLoadBalancerRule(
		data.ZoneID.ValueString(),
		data.NetworkID.ValueString(),
		data.LoadBalancerID.ValueString(),
		data.InstanceNetworkID.ValueString(),
	)
	if err != nil {
		log.Printf("Load balancer backend deassignment failed: %v", err)
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to remove instance from load balancer, got error: %s", err))
		return
	}
	if !deassignResp.Data.Success {
		resp.Diagnostics.AddError("Load Balancer Backend Deassignment Failed", "API returned failure for load balancer backend deassignment")
		return
	}

	log.Printf("✓ Load balancer backend deassignment request submitted successfully")
	log.Printf("=== Load Balancer Backend Removal Completed ===")
}
