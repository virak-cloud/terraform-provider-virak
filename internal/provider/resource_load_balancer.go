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
	"github.com/virak-cloud/cli/pkg/http/responses"
	"github.com/virak-cloud/terraform-provider-virak/internal/models"
)

// Ensure the implementation satisfies the resource.Resource interface.
var _ resource.Resource = &loadBalancerResource{}

func NewLoadBalancerResource(mutex *sync.Mutex) resource.Resource {
	return &loadBalancerResource{mutex: mutex}
}

type loadBalancerResource struct {
	client *http.Client
	mutex  *sync.Mutex
}

func (r *loadBalancerResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_load_balancer"
}

func (r *loadBalancerResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Virak Cloud load balancer rule.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Load balancer rule ID",
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
			"public_ip_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the public IP address for the load balancer.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the load balancer rule.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"algorithm": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The load balancing algorithm (e.g., 'roundrobin', 'leastconn').",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"public_port": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "The public port for the load balancer.",
				PlanModifiers:       []planmodifier.Int64{
					// TODO: Add RequiresReplace if needed
				},
			},
			"private_port": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "The private port for backend instances.",
				PlanModifiers:       []planmodifier.Int64{
					// TODO: Add RequiresReplace if needed
				},
			},
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The status of the load balancer rule.",
			},
		},
	}
}

func (r *loadBalancerResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *loadBalancerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var data models.LoadBalancerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	log.Printf("=== Starting Load Balancer Creation ===")
	log.Printf("Zone ID: %s", data.ZoneID.ValueString())
	log.Printf("Network ID: %s", data.NetworkID.ValueString())
	log.Printf("Public IP ID: %s", data.PublicIPID.ValueString())
	log.Printf("Name: %s", data.Name.ValueString())
	log.Printf("Algorithm: %s", data.Algorithm.ValueString())
	log.Printf("Public Port: %d", data.PublicPort.ValueInt64())
	log.Printf("Private Port: %d", data.PrivatePort.ValueInt64())

	// Create the load balancer rule
	createResp, err := r.client.CreateLoadBalancerRule(
		data.ZoneID.ValueString(),
		data.NetworkID.ValueString(),
		data.PublicIPID.ValueString(),
		data.Name.ValueString(),
		data.Algorithm.ValueString(),
		int(data.PublicPort.ValueInt64()),
		int(data.PrivatePort.ValueInt64()),
	)
	if err != nil {
		log.Printf("Load balancer creation failed: %v", err)
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create load balancer, got error: %s", err))
		return
	}
	if !createResp.Data.Success {
		resp.Diagnostics.AddError("Load Balancer Creation Failed", "API returned failure for load balancer creation")
		return
	}

	log.Printf("Load balancer creation request submitted successfully")

	// Find the newly created load balancer rule
	log.Printf("Finding the newly created load balancer rule...")
	listResp, err := r.client.ListLoadBalancerRules(data.ZoneID.ValueString(), data.NetworkID.ValueString())
	if err != nil {
		log.Printf("Error listing load balancer rules: %v", err)
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list load balancer rules after creation, got error: %s", err))
		return
	}

	// Find the load balancer rule by name (most recently created)
	var newLoadBalancerRule *responses.LoadBalancerRule
	if len(listResp.Data) > 0 {
		// Get the last load balancer rule in the list (most recently created)
		newLoadBalancerRule = &listResp.Data[len(listResp.Data)-1]
		log.Printf("✓ Found load balancer rule: ID=%s, Name=%s", newLoadBalancerRule.ID, newLoadBalancerRule.Name)
	} else {
		log.Printf("No load balancer rules found after creation")
		resp.Diagnostics.AddError("Load Balancer Not Found", "Could not find the created load balancer rule")
		return
	}

	// Set the computed values
	data.ID = types.StringValue(newLoadBalancerRule.ID)
	data.Status = types.StringValue(newLoadBalancerRule.Status)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		log.Printf("Error setting state: %v", resp.Diagnostics)
		return
	}

	log.Printf("=== Load Balancer Creation Completed Successfully ===")
	log.Printf("Load balancer '%s' (%s) created and ready for use", newLoadBalancerRule.Name, newLoadBalancerRule.ID)
}

func (r *loadBalancerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data models.LoadBalancerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	log.Printf("=== Reading Load Balancer State ===")
	log.Printf("Load Balancer ID: %s", data.ID.ValueString())
	log.Printf("Zone ID: %s", data.ZoneID.ValueString())
	log.Printf("Network ID: %s", data.NetworkID.ValueString())

	listResp, err := r.client.ListLoadBalancerRules(data.ZoneID.ValueString(), data.NetworkID.ValueString())
	if err != nil {
		log.Printf("Error reading load balancer rules: %v", err)
		resp.State.RemoveResource(ctx)
		return
	}

	// Find the specific load balancer rule by ID
	var foundLoadBalancerRule *responses.LoadBalancerRule
	for _, rule := range listResp.Data {
		if rule.ID == data.ID.ValueString() {
			foundLoadBalancerRule = &rule
			break
		}
	}

	if foundLoadBalancerRule == nil {
		log.Printf("Load balancer rule not found in list, removing from state")
		resp.State.RemoveResource(ctx)
		return
	}

	// Update the resource state with current values
	data.Name = types.StringValue(foundLoadBalancerRule.Name)
	data.Algorithm = types.StringValue(foundLoadBalancerRule.Algorithm)
	data.PublicPort = types.Int64Value(int64(foundLoadBalancerRule.PublicPort))
	data.PrivatePort = types.Int64Value(int64(foundLoadBalancerRule.PrivatePort))
	data.Status = types.StringValue(foundLoadBalancerRule.Status)

	log.Printf("Updated load balancer state:")
	log.Printf("  - ID: %s", foundLoadBalancerRule.ID)
	log.Printf("  - Name: %s", foundLoadBalancerRule.Name)
	log.Printf("  - Algorithm: %s", foundLoadBalancerRule.Algorithm)
	log.Printf("  - Public Port: %d", foundLoadBalancerRule.PublicPort)
	log.Printf("  - Private Port: %d", foundLoadBalancerRule.PrivatePort)
	log.Printf("  - Status: %s", foundLoadBalancerRule.Status)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		log.Printf("Error setting updated state: %v", resp.Diagnostics)
		return
	}

	log.Printf("=== Load Balancer Read Completed ===")
}

func (r *loadBalancerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var plan models.LoadBalancerResourceModel
	var state models.LoadBalancerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	log.Printf("=== Starting Load Balancer Update ===")
	log.Printf("Current Name: %s", state.Name.ValueString())
	log.Printf("New Name: %s", plan.Name.ValueString())

	// Load balancer rules are immutable - require replacement for changes
	resp.Diagnostics.AddError("Load Balancer Update Not Supported", "Load balancer rules cannot be updated. Please create a new load balancer rule with the desired changes.")
}

func (r *loadBalancerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var data models.LoadBalancerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	log.Printf("=== Starting Load Balancer Deletion ===")
	log.Printf("Load Balancer ID: %s", data.ID.ValueString())
	log.Printf("Name: %s", data.Name.ValueString())
	log.Printf("Zone ID: %s", data.ZoneID.ValueString())
	log.Printf("Network ID: %s", data.NetworkID.ValueString())

	// Delete the load balancer rule
	deleteResp, err := r.client.DeleteLoadBalancerRule(data.ZoneID.ValueString(), data.NetworkID.ValueString(), data.ID.ValueString())
	if err != nil {
		log.Printf("Load balancer deletion failed: %v", err)
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete load balancer, got error: %s", err))
		return
	}
	if !deleteResp.Data.Success {
		resp.Diagnostics.AddError("Load Balancer Deletion Failed", "API returned failure for load balancer deletion")
		return
	}

	log.Printf("✓ Load balancer deletion request submitted successfully")
	log.Printf("=== Load Balancer Deletion Completed ===")
}
