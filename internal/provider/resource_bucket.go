package provider

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/virak-cloud/terraform-provider-virak/internal/models"
	"github.com/virak-cloud/terraform-provider-virak/internal/provider/helpers"
)

var _ resource.Resource = &bucketResource{}

func NewBucketResource(mutex *sync.Mutex) resource.Resource {
	r := &bucketResource{}
	r.setMutex(mutex)
	return r
}

type bucketResource struct {
	baseResource
}

func stringValueOrNull(value string) types.String {
	if value == "" {
		return types.StringNull()
	}
	return types.StringValue(value)
}

func (r *bucketResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket"
}

func (r *bucketResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Virak Cloud object storage bucket.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Bucket ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the bucket.",
			},
			"zone_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the zone where the bucket is located.",
			},
			"policy": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The access policy for the bucket.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("Private", "Public"),
				},
			},
			"url": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The URL of the bucket.",
			},
			"access_key": schema.StringAttribute{
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "The access key for the bucket.",
			},
			"secret_key": schema.StringAttribute{
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "The secret key for the bucket.",
			},
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The status of the bucket.",
			},
			"size": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "The size of the bucket in bytes.",
			},
		},
	}
}

func (r *bucketResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var data models.BucketResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// 1. Get existing buckets
	initialBuckets, err := r.client.GetObjectStorageBuckets(data.ZoneID.ValueString())
	if err != nil {
		helpers.HandleAPIError(&resp.Diagnostics, "Client Error", fmt.Errorf("unable to list buckets before creation: %w", err))
		return
	}
	existingIDs := make(map[string]struct{})
	for _, bucket := range initialBuckets.Data {
		existingIDs[bucket.ID] = struct{}{}
	}

	// 2. Create the new bucket
	_, err = r.client.CreateObjectStorageBucket(data.ZoneID.ValueString(), data.Name.ValueString(), data.Policy.ValueString())
	if err != nil {
		helpers.HandleAPIError(&resp.Diagnostics, "Client Error", fmt.Errorf("unable to create bucket: %w", err))
		return
	}

	// 3. Find the new bucket ID
	newBucketID, err := helpers.FindNewBucketID(r.client, data.ZoneID.ValueString(), existingIDs, data.Name.ValueString(), 10, 5*time.Second)
	if err != nil {
		helpers.HandleAPIError(&resp.Diagnostics, "Client Error", fmt.Errorf("could not find the newly created bucket: %w", err))
		return
	}

	data.ID = types.StringValue(newBucketID)

	// 4. Read the bucket details
	readResp, err := r.client.GetObjectStorageBucket(data.ZoneID.ValueString(), data.ID.ValueString())
	if err != nil {
		helpers.HandleAPIError(&resp.Diagnostics, "Client Error", fmt.Errorf("unable to read bucket after creation: %w", err))
		return
	}

	data.Policy = stringValueOrNull(readResp.Data.Policy)
	data.URL = types.StringValue(readResp.Data.URL)
	data.AccessKey = types.StringValue(readResp.Data.AccessKey)
	data.SecretKey = types.StringValue(readResp.Data.SecretKey)
	data.Status = types.StringValue(readResp.Data.Status)
	data.Size = types.Int64Value(int64(readResp.Data.Size))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *bucketResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data models.BucketResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readResp, err := r.client.GetObjectStorageBucket(data.ZoneID.ValueString(), data.ID.ValueString())
	if err != nil {
		// Handle not found error
		resp.State.RemoveResource(ctx)
		return
	}

	data.Policy = stringValueOrNull(readResp.Data.Policy)
	data.URL = types.StringValue(readResp.Data.URL)
	data.AccessKey = types.StringValue(readResp.Data.AccessKey)
	data.SecretKey = types.StringValue(readResp.Data.SecretKey)
	data.Status = types.StringValue(readResp.Data.Status)
	data.Size = types.Int64Value(int64(readResp.Data.Size))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
func (r *bucketResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var plan, state models.BucketResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check if policy has changed
	if !plan.Policy.Equal(state.Policy) {
		_, err := r.client.UpdateObjectStorageBucket(plan.ZoneID.ValueString(), plan.ID.ValueString(), plan.Policy.ValueString())
		if err != nil {
			helpers.HandleAPIError(&resp.Diagnostics, "Client Error", fmt.Errorf("unable to update bucket policy: %w", err))
			return
		}
	}

	// Read the updated bucket
	readResp, err := r.client.GetObjectStorageBucket(plan.ZoneID.ValueString(), plan.ID.ValueString())
	if err != nil {
		helpers.HandleAPIError(&resp.Diagnostics, "Client Error", fmt.Errorf("unable to read bucket after update: %w", err))
		return
	}

	plan.Policy = stringValueOrNull(readResp.Data.Policy)
	plan.URL = types.StringValue(readResp.Data.URL)
	plan.AccessKey = types.StringValue(readResp.Data.AccessKey)
	plan.SecretKey = types.StringValue(readResp.Data.SecretKey)
	plan.Status = types.StringValue(readResp.Data.Status)
	plan.Size = types.Int64Value(int64(readResp.Data.Size))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *bucketResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var data models.BucketResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteObjectStorageBucket(data.ZoneID.ValueString(), data.ID.ValueString())
	if err != nil {
		helpers.HandleAPIError(&resp.Diagnostics, "Client Error", fmt.Errorf("unable to delete bucket: %w", err))
		return
	}

	// Wait for the bucket to be deleted
	for i := 0; i < 10; i++ { // Retry for up to 50 seconds
		buckets, err := r.client.GetObjectStorageBuckets(data.ZoneID.ValueString())
		if err != nil {
			helpers.HandleAPIError(&resp.Diagnostics, "Client Error", fmt.Errorf("unable to list buckets after deletion: %w", err))
			return
		}

		bucketExists := false
		for _, bucket := range buckets.Data {
			if bucket.ID == data.ID.ValueString() {
				bucketExists = true
				break
			}
		}

		if !bucketExists {
			break
		}

		if i == 9 {
			helpers.HandleAPIError(&resp.Diagnostics, "Client Error", fmt.Errorf("bucket was not deleted after waiting"))
			return
		}

		time.Sleep(5 * time.Second)
	}
}

func (r *bucketResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data models.BucketResourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

}
