package provider

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/virak-cloud/cli/pkg/http"
	"github.com/virak-cloud/terraform-provider-virak/internal/models"
)

var _ resource.Resource = &dnsDomainResource{}

func NewDnsDomainResource(mutex *sync.Mutex) resource.Resource {
	return &dnsDomainResource{mutex: mutex}
}

type dnsDomainResource struct {
	client *http.Client
	mutex  *sync.Mutex
}

func (r *dnsDomainResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_domain"
}

func (r *dnsDomainResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Virak Cloud DNS domain.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Domain ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"domain": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The domain name.",
			},
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The status of the DNS domain.",
			},
			"dns_info": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "DNS information for the domain.",
			},
		},
	}
}

func (r *dnsDomainResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*http.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData))
		return
	}
	r.client = client
}

func (r *dnsDomainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var data models.DomainResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// First, get the list of domains to check if it already exists
	domainsResp, err := r.client.GetDomains()
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list domains, got error: %s", err))
		return
	}

	// Check if the domain already exists
	for _, domain := range domainsResp.Data {
		if domain.Domain == data.Domain.ValueString() {
			// Domain exists, use the existing one
			data.ID = types.StringValue(domain.Domain)
			data.Status = types.StringValue(domain.Status)
			data.DNSInfo = types.StringValue("{}") // Placeholder for DNS info
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
	}

	// Domain does not exist, create it
	_, err = r.client.CreateDomain(data.Domain.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create domain, got error: %s", err))
		return
	}

	data.ID = types.StringValue(data.Domain.ValueString())
	data.Status = types.StringValue("active") // Assume active after create
	data.DNSInfo = types.StringValue("{}")    // Placeholder for DNS info

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *dnsDomainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data models.DomainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainResp, err := r.client.GetDomain(data.Domain.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "cannot unmarshal array into Go struct field DomainShow.data of type responses.Domain") {
			resp.State.RemoveResource(ctx)
			return
		}
		if strings.Contains(err.Error(), "invalid") {
			// Domain is invalid or not found, remove from state
			resp.State.RemoveResource(ctx)
			return
		}
		// Add verbose error details
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to read domain '%s', got error: %s. This may indicate the domain does not exist, is not accessible, or there is an API issue with the zone/domain validation.", data.Domain.ValueString(), err),
		)
		return
	}

	data.Status = types.StringValue(domainResp.Data.Status)
	data.DNSInfo = types.StringValue("{}") // Placeholder for DNS info
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *dnsDomainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Domains may not support updates
}

func (r *dnsDomainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var data models.DomainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteDomain(data.Domain.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete domain, got error: %s", err))
		return
	}
}
