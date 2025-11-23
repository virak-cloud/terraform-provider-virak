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
	"github.com/virak-cloud/cli/pkg/http/responses"
	"github.com/virak-cloud/terraform-provider-virak/internal/models"
	"github.com/virak-cloud/terraform-provider-virak/internal/provider/helpers"
)

// Ensure the implementation satisfies the resource.Resource interface.
var _ resource.Resource = &snapshotResource{}

func NewSnapshotResource(mutex *sync.Mutex) resource.Resource {
	return &snapshotResource{mutex: mutex}
}

type snapshotResource struct {
	client *http.Client
	mutex  *sync.Mutex
}

func (r *snapshotResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_snapshot"
}

func (r *snapshotResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Virak Cloud instance snapshot.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Snapshot ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"zone_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the zone where the instance resides.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"instance_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the instance to snapshot.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the snapshot.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The status of the snapshot.",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The creation timestamp of the snapshot.",
			},
			"revert": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Set to true to revert the instance to this snapshot. This will trigger a revert operation on the next apply.",
			},
		},
	}
}

func (r *snapshotResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *snapshotResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var data models.SnapshotResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create the snapshot
	createResp, err := r.client.CreateInstanceSnapshot(
		data.ZoneID.ValueString(),
		data.InstanceID.ValueString(),
		data.Name.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Snapshot Creation Failed",
			fmt.Sprintf("Failed to create snapshot '%s' for instance %s in zone %s. Error: %s", data.Name.ValueString(), data.InstanceID.ValueString(), data.ZoneID.ValueString(), err),
		)
		return
	}

	if !createResp.Data.Success {
		resp.Diagnostics.AddError(
			"Snapshot Creation Failed",
			fmt.Sprintf("API returned failure for snapshot creation '%s'", data.Name.ValueString()),
		)
		return
	}

	// Find the new snapshot ID by listing snapshots for the instance
	var snapshotID string
	var snapshotStatus string
	var createdAt string

	// Poll for the snapshot to appear and be ready
	for i := 0; i < 120; i++ { // Retry for up to 600 seconds
		instanceResp, err := r.client.ShowInstance(data.ZoneID.ValueString(), data.InstanceID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read instance after snapshot creation, got error: %s", err))
			return
		}

		// Find the snapshot with matching name
		for _, snap := range instanceResp.Data.Snapshot {
			if snap.Name == data.Name.ValueString() {
				snapshotID = snap.ID
				snapshotStatus = snap.Status
				createdAt = fmt.Sprintf("%d", snap.CreatedAt)
				break
			}
		}

		if snapshotID != "" {
			// If status is READY, we can proceed
			if snapshotStatus == "READY" {
				break
			}
			// If status is WAITING, continue polling
		}
		time.Sleep(5 * time.Second)
	}

	if snapshotID == "" {
		resp.Diagnostics.AddError(
			"Snapshot Creation Timeout",
			fmt.Sprintf("Snapshot '%s' was submitted for creation but could not be found in the API after 600 seconds.", data.Name.ValueString()),
		)
		return
	}

	if snapshotStatus != "READY" {
		resp.Diagnostics.AddError(
			"Snapshot Not Ready",
			fmt.Sprintf("Snapshot '%s' was created but did not reach 'READY' status within 600 seconds. Current status: %s.", data.Name.ValueString(), snapshotStatus),
		)
		return
	}

	data.ID = types.StringValue(snapshotID)
	data.Status = types.StringValue(snapshotStatus)
	data.CreatedAt = types.StringValue(createdAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *snapshotResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data models.SnapshotResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get instance details to find the snapshot
	instanceResp, err := r.client.ShowInstance(data.ZoneID.ValueString(), data.InstanceID.ValueString())
	if err != nil {
		// Handle not found error
		resp.State.RemoveResource(ctx)
		return
	}

	// Find the snapshot
	var foundSnapshot *responses.InstanceSnapshot
	for _, snap := range instanceResp.Data.Snapshot {
		if snap.ID == data.ID.ValueString() {
			foundSnapshot = &snap
			break
		}
	}

	if foundSnapshot == nil {
		// Snapshot not found
		resp.State.RemoveResource(ctx)
		return
	}

	data.Name = types.StringValue(foundSnapshot.Name)
	data.Status = types.StringValue(foundSnapshot.Status)
	data.CreatedAt = types.StringValue(fmt.Sprintf("%d", foundSnapshot.CreatedAt))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *snapshotResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var plan models.SnapshotResourceModel
	var state models.SnapshotResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Handle revert operation
	if !plan.Revert.IsNull() && plan.Revert.ValueBool() && (!state.Revert.ValueBool() || state.Revert.IsNull()) {
		revertResp, err := r.client.RevertInstanceSnapshot(
			plan.ZoneID.ValueString(),
			plan.InstanceID.ValueString(),
			plan.ID.ValueString(),
		)
		if err != nil {
			resp.Diagnostics.AddError(
				"Snapshot Revert Failed",
				fmt.Sprintf("Failed to revert instance %s to snapshot %s in zone %s. Error: %s", plan.InstanceID.ValueString(), plan.ID.ValueString(), plan.ZoneID.ValueString(), err),
			)
			return
		}

		if !revertResp.Data.Success {
			resp.Diagnostics.AddError(
				"Snapshot Revert Failed",
				fmt.Sprintf("API returned failure for snapshot revert '%s'", plan.ID.ValueString()),
			)
			return
		}

		for i := 0; i < 120; i++ {
			instanceResp, err := r.client.ShowInstance(plan.ZoneID.ValueString(), plan.InstanceID.ValueString())
			if err != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read instance after revert, got error: %s", err))
				return
			}

			if instanceResp.Data.Status == helpers.InstanceStatusUP || instanceResp.Data.Status == helpers.InstanceStatusRunning {
				break
			}
			time.Sleep(5 * time.Second)
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *snapshotResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var data models.SnapshotResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the snapshot
	deleteResp, err := r.client.DeleteInstanceSnapshot(
		data.ZoneID.ValueString(),
		data.InstanceID.ValueString(),
		data.ID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Snapshot Deletion Failed",
			fmt.Sprintf("Failed to delete snapshot '%s' for instance %s in zone %s. Error: %s", data.ID.ValueString(), data.InstanceID.ValueString(), data.ZoneID.ValueString(), err),
		)
		return
	}

	if !deleteResp.Data.Success {
		resp.Diagnostics.AddError(
			"Snapshot Deletion Failed",
			fmt.Sprintf("API returned failure for snapshot deletion '%s'", data.ID.ValueString()),
		)
		return
	}
}
