package provider

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/virak-cloud/terraform-provider-virak/internal/models"
	"github.com/virak-cloud/terraform-provider-virak/internal/provider/helpers"
)

var _ resource.Resource = &kubernetesClusterResource{}

func NewKubernetesClusterResource(mutex *sync.Mutex) resource.Resource {
	r := &kubernetesClusterResource{}
	r.setMutex(mutex)
	return r
}

type kubernetesClusterResource struct {
	baseResource
}

func (r *kubernetesClusterResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kubernetes_cluster"
}

func (r *kubernetesClusterResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Virak Cloud Kubernetes cluster.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Cluster ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the Kubernetes cluster.",
			},
			"zone_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the zone where the cluster is located.",
			},
			"kubernetes_version_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the Kubernetes version to use.",
			},
			"service_offering_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the service offering for the cluster.",
			},
			"ssh_key_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the SSH key for accessing the cluster.",
			},
			"network_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the Isolated network for the cluster. The network must be of type 'Isolated'.",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "A description of the cluster.",
			},
			"ha_enabled": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Whether high availability is enabled for the cluster.",
			},
			"cluster_size": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "The size of the cluster. Must be at least 1.",
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The status of the cluster.",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The creation timestamp of the cluster.",
			},
			"updated_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The last update timestamp of the cluster.",
			},
			"kubeconfig": schema.StringAttribute{
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "The kubeconfig for accessing the cluster.",
			},
		},
	}
}

func (r *kubernetesClusterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var data models.KubernetesClusterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate network type
	networkResp, err := r.client.ShowNetwork(data.ZoneID.ValueString(), data.NetworkID.ValueString())
	if err != nil {
		helpers.HandleAPIError(&resp.Diagnostics, "Client Error", fmt.Errorf("unable to get network: %w", err))
		return
	}
	if networkResp.Data.NetworkOffering.Type != "Isolated" {
		helpers.HandleValidationError(&resp.Diagnostics, "Validation Error", "The network must be of type 'Isolated' for Kubernetes clusters")
		return
	}

	// Validate cluster size
	if data.ClusterSize.ValueInt64() < 1 {
		helpers.HandleValidationError(&resp.Diagnostics, "Validation Error", "Cluster size must be at least 1")
		return
	}

	_, err = r.client.CreateKubernetesCluster(data.ZoneID.ValueString(), data.Name.ValueString(), data.KubernetesVersionID.ValueString(), data.ServiceOfferingID.ValueString(), data.SshKeyID.ValueString(), data.NetworkID.ValueString(), data.HaEnabled.ValueBool(), int(data.ClusterSize.ValueInt64()), data.Description.ValueString(), "", "", "", 0, "")
	if err != nil {
		helpers.HandleAPIError(&resp.Diagnostics, "Client Error", fmt.Errorf("unable to create kubernetes cluster: %w", err))
		return
	}

	// Find the new cluster ID by listing
	newClusterID, err := helpers.FindNewKubernetesClusterID(r.client, data.ZoneID.ValueString(), data.Name.ValueString(), 10, 5*time.Second)
	if err != nil {
		helpers.HandleAPIError(&resp.Diagnostics, "Client Error", fmt.Errorf("could not find the newly created cluster: %w", err))
		return
	}

	data.ID = types.StringValue(newClusterID)

	// Read to get status
	readResp, err := r.client.GetKubernetesCluster(data.ZoneID.ValueString(), data.ID.ValueString())
	if err != nil {
		helpers.HandleAPIError(&resp.Diagnostics, "Client Error", fmt.Errorf("unable to read cluster after creation: %w", err))
		return
	}

	data.Status = types.StringValue(readResp.Data.Status)
	data.CreatedAt = types.StringValue(fmt.Sprintf("%d", readResp.Data.CreatedAt))
	data.UpdatedAt = types.StringValue(fmt.Sprintf("%d", readResp.Data.UpdatedAt))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *kubernetesClusterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data models.KubernetesClusterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readResp, err := r.client.GetKubernetesCluster(data.ZoneID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data.Name = types.StringValue(readResp.Data.Name)
	data.Status = types.StringValue(readResp.Data.Status)
	data.CreatedAt = types.StringValue(fmt.Sprintf("%d", readResp.Data.CreatedAt))
	data.UpdatedAt = types.StringValue(fmt.Sprintf("%d", readResp.Data.UpdatedAt))
	// Kubeconfig might be available in some responses

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *kubernetesClusterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var data models.KubernetesClusterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.UpdateKubernetesClusterDetails(data.ZoneID.ValueString(), data.ID.ValueString(), data.Name.ValueString(), data.Description.ValueString())
	if err != nil {
		helpers.HandleAPIError(&resp.Diagnostics, "Client Error", fmt.Errorf("unable to update kubernetes cluster: %w", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *kubernetesClusterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var data models.KubernetesClusterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteKubernetesCluster(data.ZoneID.ValueString(), data.ID.ValueString())
	if err != nil {
		helpers.HandleAPIError(&resp.Diagnostics, "Client Error", fmt.Errorf("unable to delete kubernetes cluster: %w", err))
		return
	}
}
