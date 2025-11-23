package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/virak-cloud/cli/pkg/http"
)

// Ensure interface implementation
var _ datasource.DataSource = &vmImagesDataSource{}

// NewInstanceImagesDataSource returns a new data source instance.
func NewVMImagesDataSource() datasource.DataSource {
	return &vmImagesDataSource{}
}

type vmImagesDataSource struct {
	client *http.Client
}

func (d *vmImagesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm_image"
}

func (d *vmImagesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Select a single Virak Cloud VM image by applying optional filters (name, zone_id).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "ID of the selected VM image.",
			},
			"name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Name of the selected VM image.",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Description of the selected VM image.",
			},
			// filter blocks allow narrowing the results by attribute name and values (e.g., name, zone)
			"filter": schema.ListNestedAttribute{
				Optional:            true,
				MarkdownDescription: "One or more filter blocks to narrow images by attribute name and values.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The attribute name to filter on. Supported: 'name', 'zone_id'.",
						},
						"values": schema.ListAttribute{
							Required:            true,
							ElementType:         types.StringType,
							MarkdownDescription: "List of string values to match for the given attribute.",
						},
					},
				},
			},
		},
	}
}

func (d *vmImagesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*http.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	d.client = client
}

// Local types used for config parsing and response state
type vmImageFilterBlock struct {
	Name   types.String   `tfsdk:"name"`
	Values []types.String `tfsdk:"values"`
}

type vmImagesConfig struct {
	Filter []vmImageFilterBlock `tfsdk:"filter"`
}

type vmImageDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

func (d *vmImagesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var (
		cfg  vmImagesConfig
		data vmImageDataSourceModel
	)

	// Read config (optional filter blocks)
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Parse filters (preserve zone order if provided)
	nameSet := map[string]bool{}
	var zoneValues []string
	for _, fb := range cfg.Filter {
		if fb.Name.IsNull() || fb.Name.IsUnknown() {
			continue
		}
		switch fb.Name.ValueString() {
		case "name":
			for _, v := range fb.Values {
				if v.IsNull() || v.IsUnknown() {
					continue
				}
				nameSet[v.ValueString()] = true
			}
		case "zone_id":
			for _, v := range fb.Values {
				if v.IsNull() || v.IsUnknown() {
					continue
				}
				zoneValues = append(zoneValues, v.ValueString())
			}
		default:
			// ignore unknown filter names defensively
		}
	}

	// Resolve provider-level zone fallback (provider stores this at runtime; use env var as reasonable fallback)
	providerZone := os.Getenv("VIRAKCLOUD_ZONE_ID")

	// Helper to find first matching image in a given zone (zone may be empty to list across zones)
	findInZone := func(zone string) (string, string, bool) {
		imagesResp, err := d.client.ListInstanceVMImages(zone)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list VM images for zone '%s': %s", zone, err))
			return "", "", false
		}
		// Iterate returned images (other data sources use imagesResp.Data with ID and Name)
		for _, image := range imagesResp.Data {
			if image.ID == "" {
				continue
			}
			// name filter exact-match (if provided)
			if len(nameSet) > 0 {
				if _, ok := nameSet[image.Name]; !ok {
					continue
				}
			}
			// first deterministic match wins
			return image.ID, image.Name, true
		}
		return "", "", false
	}

	// If user provided zone filters, try each in order
	var chosenID, chosenName string
	found := false
	for _, z := range zoneValues {
		if id, nm, ok := findInZone(z); ok {
			chosenID = id
			chosenName = nm
			found = true
			break
		}
	}

	// If not found and we have a provider-level zone, try it
	if !found {
		if providerZone != "" {
			if id, nm, ok := findInZone(providerZone); ok {
				chosenID = id
				chosenName = nm
				found = true
			}
		}
	}

	// If still not found, try without zone restriction (list across zones if API allows)
	if !found {
		if id, nm, ok := findInZone(""); ok {
			chosenID = id
			chosenName = nm
			found = true
		}
	}

	if !found {
		resp.Diagnostics.AddError("No VM image matched", "no VM image matched the supplied filters")
		return
	}

	data.ID = types.StringValue(chosenID)
	data.Name = types.StringValue(chosenName)
	// Description is not returned by the client's image type in many places; return empty string for now.
	data.Description = types.StringValue("")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
