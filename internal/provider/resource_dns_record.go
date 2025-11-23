package provider

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/virak-cloud/cli/pkg/http"
	"github.com/virak-cloud/cli/pkg/http/responses"
	"github.com/virak-cloud/terraform-provider-virak/internal/models"
)

var _ resource.Resource = &dnsRecordResource{}

func NewDnsRecordResource(mutex *sync.Mutex) resource.Resource {
	return &dnsRecordResource{mutex: mutex}
}

type dnsRecordResource struct {
	client *http.Client
	mutex  *sync.Mutex
}

func (r *dnsRecordResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_record"
}

func (r *dnsRecordResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Virak Cloud DNS record.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Unique record ID (domain/record/type/contentId).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"domain": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The parent domain name.",
			},
			"record": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The record name (e.g., 'www').",
			},
			"type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The record type (A, MX, SRV, etc.).",
			},
			"content": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The record content.",
			},
			"ttl": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(3600),
				MarkdownDescription: "Time to live (TTL) for the record.",
			},
			"priority": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Priority for MX and SRV records.",
			},
			"weight": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Weight for SRV records.",
			},
			"port": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Port for SRV records.",
			},
			"flags": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Flags for CAA records.",
			},
			"tag": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Tag for CAA records.",
			},
			"license": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "License for TLSA records.",
			},
			"choicer": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Choicer for TLSA records.",
			},
			"match": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Match for TLSA records.",
			},
			"content_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "API-assigned content ID for the record.",
			},
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The status of the DNS record.",
			},
			"is_protected": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the DNS record is protected.",
			},
		},
	}
}

func (r *dnsRecordResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *dnsRecordResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var data models.DnsRecordResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Extract values
	domain := data.Domain.ValueString()
	record := data.Record.ValueString()
	recordType := data.Type.ValueString()
	content := data.Content.ValueString()
	ttl := int(data.TTL.ValueInt64())
	priority := int(data.Priority.ValueInt64())
	weight := int(data.Weight.ValueInt64())
	port := int(data.Port.ValueInt64())
	flags := int(data.Flags.ValueInt64())
	tag := data.Tag.ValueString()
	license := int(data.License.ValueInt64())
	choicer := int(data.Choicer.ValueInt64())
	match := int(data.Match.ValueInt64())

	// Check if record already exists
	recordsResp, err := r.client.GetRecords(domain)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to fetch existing records, got error: %s", err))
		return
	}

	// Look for existing record with same name and type
	var existingContentID string
	for _, rec := range recordsResp.Data {
		recordFull := record
		if !strings.Contains(record, domain) {
			recordFull = record + "." + domain
		}
		if (rec.Name == record || rec.Name == recordFull) && rec.Type == recordType {
			if len(rec.Content) > 0 {
				existingContentID = rec.Content[0].ID
			}
			break
		}
	}

	if existingContentID != "" {
		// Record exists, update it instead
		_, err := r.client.UpdateRecord(domain, record, recordType, existingContentID, content, ttl, priority, weight, port, flags, tag, license, choicer, match)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update existing DNS record, got error: %s", err))
			return
		}
		data.ContentID = types.StringValue(existingContentID)
	} else {
		// Record doesn't exist, create it
		_, err := r.client.CreateRecord(domain, record, recordType, content, ttl, priority, weight, port, flags, tag, license, choicer, match)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create DNS record, got error: %s", err))
			return
		}

		// Fetch records again to get content_id
		recordsResp, err := r.client.GetRecords(domain)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to fetch records after create, got error: %s", err))
			return
		}

		// Find the newly created record
		var contentID string
		for _, rec := range recordsResp.Data {
			recordFull := record
			if !strings.Contains(record, domain) {
				recordFull = record + "." + domain
			}
			if (rec.Name == record || rec.Name == recordFull) && rec.Type == recordType {
				if len(rec.Content) > 0 {
					contentID = rec.Content[0].ID
				}
				break
			}
		}
		if contentID == "" {
			resp.Diagnostics.AddError("Client Error", "Unable to find the created record's content_id")
			return
		}
		data.ContentID = types.StringValue(contentID)
	}

	// Set ID
	data.ID = types.StringValue(fmt.Sprintf("%s/%s/%s/%s", domain, record, recordType, data.ContentID.ValueString()))

	// Fetch the record to get status and is_protected
	latestRecordsResp, err := r.client.GetRecords(domain)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to fetch records after operation, got error: %s", err))
		return
	}

	// Find the record
	for _, rec := range latestRecordsResp.Data {
		if rec.Name == record && rec.Type == recordType {
			for _, c := range rec.Content {
				if c.ID == data.ContentID.ValueString() {
					data.Status = types.StringValue(rec.Status)
					data.IsProtected = types.BoolValue(rec.IsProtected)
					break
				}
			}
			break
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *dnsRecordResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data models.DnsRecordResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Parse ID to get domain, record, type, contentID
	parts := strings.Split(data.ID.ValueString(), "/")
	if len(parts) != 4 {
		resp.Diagnostics.AddError("Invalid ID", "ID format is invalid")
		return
	}
	domain := parts[0]
	record := parts[1]
	recordType := parts[2]
	contentID := parts[3]

	// Fetch records
	recordsResp, err := r.client.GetRecords(domain)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read DNS records, got error: %s", err))
		return
	}

	// Find the record
	var foundRecord *responses.Record
	var foundContent *responses.Content
	for _, rec := range recordsResp.Data {
		recordFull := record
		if !strings.Contains(record, domain) {
			recordFull = record + "." + domain
		}
		if (rec.Name == record || rec.Name == recordFull) && rec.Type == recordType {
			for _, c := range rec.Content {
				if c.ID == contentID {
					foundRecord = &rec
					foundContent = &c
					break
				}
			}
			if foundContent != nil {
				break
			}
		}
	}
	if foundContent == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// Populate data
	data.Domain = types.StringValue(domain)
	data.Record = types.StringValue(foundRecord.Name)
	data.Type = types.StringValue(foundRecord.Type)
	data.Content = types.StringValue(foundContent.ContentRaw)
	data.TTL = types.Int64Value(int64(foundRecord.TTL))
	// Other fields like Priority, Weight, etc., are not in the provided struct, so not populated in Read
	data.ContentID = types.StringValue(foundContent.ID)
	data.Status = types.StringValue(foundRecord.Status)
	data.IsProtected = types.BoolValue(foundRecord.IsProtected)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *dnsRecordResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var data models.DnsRecordResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Parse ID
	parts := strings.Split(data.ID.ValueString(), "/")
	if len(parts) != 4 {
		resp.Diagnostics.AddError("Invalid ID", "ID format is invalid")
		return
	}
	domain := parts[0]
	record := parts[1]
	recordType := parts[2]
	contentID := parts[3]

	// Extract values
	content := data.Content.ValueString()
	ttl := int(data.TTL.ValueInt64())
	priority := int(data.Priority.ValueInt64())
	weight := int(data.Weight.ValueInt64())
	port := int(data.Port.ValueInt64())
	flags := int(data.Flags.ValueInt64())
	tag := data.Tag.ValueString()
	license := int(data.License.ValueInt64())
	choicer := int(data.Choicer.ValueInt64())
	match := int(data.Match.ValueInt64())

	// Update the record
	_, err := r.client.UpdateRecord(domain, record, recordType, contentID, content, ttl, priority, weight, port, flags, tag, license, choicer, match)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update DNS record, got error: %s", err))
		return
	}

	// Fetch the record to get status and is_protected
	latestRecordsResp, err := r.client.GetRecords(domain)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to fetch records after operation, got error: %s", err))
		return
	}

	// Find the record
	for _, rec := range latestRecordsResp.Data {
		recordFull := record
		if !strings.Contains(record, domain) {
			recordFull = record + "." + domain
		}
		if (rec.Name == record || rec.Name == recordFull) && rec.Type == recordType {
			for _, c := range rec.Content {
				if c.ID == contentID {
					data.Status = types.StringValue(rec.Status)
					data.IsProtected = types.BoolValue(rec.IsProtected)
					break
				}
			}
			break
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *dnsRecordResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var data models.DnsRecordResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Parse ID
	parts := strings.Split(data.ID.ValueString(), "/")
	if len(parts) != 4 {
		resp.Diagnostics.AddError("Invalid ID", "ID format is invalid")
		return
	}
	domain := parts[0]
	record := parts[1]
	recordType := parts[2]
	contentID := parts[3]

	// Delete the record
	_, err := r.client.DeleteRecord(domain, record, recordType, contentID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete DNS record, got error: %s", err))
		return
	}
}
