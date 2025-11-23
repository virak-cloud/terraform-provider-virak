package provider

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/virak-cloud/cli/pkg/http"
	"github.com/virak-cloud/terraform-provider-virak/internal/models"
)

var _ datasource.DataSource = &instanceOfferingsDataSource{}

func NewInstanceOfferingsDataSource() datasource.DataSource {
	return &instanceOfferingsDataSource{}
}

type instanceOfferingsDataSource struct {
	client *http.Client
}

type instanceOfferingsConfig struct {
	ID          types.String                              `tfsdk:"id"`
	ZoneID      types.String                              `tfsdk:"zone_id"`
	Filter      []models.InstanceOfferingFilterBlockModel `tfsdk:"filter"`
	FilterLogic types.String                              `tfsdk:"filter_logic"`
	Offerings   []models.InstanceOfferingModel            `tfsdk:"offerings"`
}

func (d *instanceOfferingsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance_service_offerings"
}

func (d *instanceOfferingsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists available Virak Cloud instance service offerings.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of this data source.",
			},
			"zone_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the zone to list offerings for.",
			},
			"filter_logic": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Logic for combining multiple filters. Valid values: 'and' (default) or 'or'. When 'and', all filters must match. When 'or', any filter can match.",
			},
			"filter": schema.ListNestedAttribute{
				Optional:            true,
				MarkdownDescription: "One or more filter blocks to narrow offerings by attribute name and values.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The attribute name to filter on. Supported: name, description, cpu_core, memory_mb, cpu_speed_mhz, root_disk_size_gb, network_rate, disk_iops, hourly_price_up, hourly_price_down, is_available, is_public.",
						},
						"match_type": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "The type of matching to perform. For string fields: 'exact', 'partial', 'prefix', 'suffix', 'regex'. For numeric fields: 'exact', 'gt', 'gte', 'lt', 'lte', 'between'. For boolean fields: 'exact' only. Default: 'exact'.",
						},
						"values": schema.ListAttribute{
							Required:            true,
							ElementType:         types.StringType,
							MarkdownDescription: "List of string values to match for the given attribute. For 'between' match type, provide exactly 2 values.",
						},
						"case_sensitive": schema.BoolAttribute{
							Optional:            true,
							MarkdownDescription: "Whether string matching should be case-sensitive. Only applies to string fields. Default: false.",
						},
					},
				},
			},
			"offerings": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The ID of the instance offering.",
						},
						"name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The name of the instance offering.",
						},
						"cpu_core": schema.Int64Attribute{
							Computed:            true,
							MarkdownDescription: "Number of CPU cores.",
						},
						"memory_mb": schema.Int64Attribute{
							Computed:            true,
							MarkdownDescription: "Memory in MB.",
						},
						"cpu_speed_mhz": schema.Int64Attribute{
							Computed:            true,
							MarkdownDescription: "CPU speed in MHz.",
						},
						"root_disk_size_gb": schema.Int64Attribute{
							Computed:            true,
							MarkdownDescription: "Root disk size in GB.",
						},
						"network_rate": schema.Int64Attribute{
							Computed:            true,
							MarkdownDescription: "Network rate.",
						},
						"disk_iops": schema.Int64Attribute{
							Computed:            true,
							MarkdownDescription: "Disk IOPS.",
						},
						"hourly_price_up": schema.Int64Attribute{
							Computed:            true,
							MarkdownDescription: "Hourly price (up).",
						},
						"hourly_price_down": schema.Int64Attribute{
							Computed:            true,
							MarkdownDescription: "Hourly price (down).",
						},
						"is_available": schema.BoolAttribute{
							Computed:            true,
							MarkdownDescription: "Availability status.",
						},
						"is_public": schema.BoolAttribute{
							Computed:            true,
							MarkdownDescription: "Public offering status.",
						},
						"description": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Description text.",
						},
					},
				},
			},
		},
	}
}

func (d *instanceOfferingsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func hasField(v interface{}, fieldName string) bool {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return false
	}
	return rv.FieldByName(fieldName).IsValid()
}

func getStringField(v interface{}, fieldName string) (string, bool) {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return "", false
	}
	field := rv.FieldByName(fieldName)
	if !field.IsValid() || !field.CanInterface() {
		return "", false
	}
	if field.Kind() == reflect.String {
		return field.String(), true
	}
	return "", false
}

func getIntField(v interface{}, fieldName string) (int64, bool) {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return 0, false
	}
	field := rv.FieldByName(fieldName)
	if !field.IsValid() || !field.CanInterface() {
		return 0, false
	}
	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return field.Int(), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int64(field.Uint()), true
	}
	return 0, false
}

func getBoolField(v interface{}, fieldName string) (bool, bool) {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return false, false
	}
	field := rv.FieldByName(fieldName)
	if !field.IsValid() || !field.CanInterface() {
		return false, false
	}
	if field.Kind() == reflect.Bool {
		return field.Bool(), true
	}
	return false, false
}

func matchString(value, pattern string, matchType string, caseSensitive bool) bool {
	if !caseSensitive {
		value = strings.ToLower(value)
		pattern = strings.ToLower(pattern)
	}

	switch matchType {
	case "exact":
		return value == pattern
	case "partial":
		return strings.Contains(value, pattern)
	case "prefix":
		return strings.HasPrefix(value, pattern)
	case "suffix":
		return strings.HasSuffix(value, pattern)
	case "regex":
		re, err := regexp.Compile(pattern)
		if err != nil {
			return false
		}
		return re.MatchString(value)
	default:
		return value == pattern
	}
}

func matchNumeric(fieldValue int64, matchType string, values []string) bool {
	if len(values) == 0 {
		return false
	}

	switch matchType {
	case "exact":
		val, err := strconv.ParseInt(values[0], 10, 64)
		if err != nil {
			return false
		}
		return fieldValue == val
	case "gt":
		val, err := strconv.ParseInt(values[0], 10, 64)
		if err != nil {
			return false
		}
		return fieldValue > val
	case "gte":
		val, err := strconv.ParseInt(values[0], 10, 64)
		if err != nil {
			return false
		}
		return fieldValue >= val
	case "lt":
		val, err := strconv.ParseInt(values[0], 10, 64)
		if err != nil {
			return false
		}
		return fieldValue < val
	case "lte":
		val, err := strconv.ParseInt(values[0], 10, 64)
		if err != nil {
			return false
		}
		return fieldValue <= val
	case "between":
		if len(values) < 2 {
			return false
		}
		minBetween, err1 := strconv.ParseInt(values[0], 10, 64)
		maxBetween, err2 := strconv.ParseInt(values[1], 10, 64)
		if err1 != nil || err2 != nil {
			return false
		}
		return fieldValue >= minBetween && fieldValue <= maxBetween
	default:
		val, err := strconv.ParseInt(values[0], 10, 64)
		if err != nil {
			return false
		}
		return fieldValue == val
	}
}

func matchBool(fieldValue bool, values []string) bool {
	if len(values) == 0 {
		return false
	}
	target, err := strconv.ParseBool(values[0])
	if err != nil {
		return false
	}
	return fieldValue == target
}

func applyFilter(offering interface{}, filter models.InstanceOfferingFilterBlockModel) bool {
	if filter.Name.IsNull() || filter.Name.IsUnknown() {
		return true
	}

	fieldName := filter.Name.ValueString()
	matchType := "exact"
	if !filter.MatchType.IsNull() && !filter.MatchType.IsUnknown() {
		matchType = filter.MatchType.ValueString()
	}
	caseSensitive := false
	if !filter.CaseSensitive.IsNull() && !filter.CaseSensitive.IsUnknown() {
		caseSensitive = filter.CaseSensitive.ValueBool()
	}

	var values []string
	for _, v := range filter.Values {
		if !v.IsNull() && !v.IsUnknown() {
			values = append(values, v.ValueString())
		}
	}
	if len(values) == 0 {
		return true
	}

	fieldMap := map[string]string{
		"name":              "Name",
		"description":       "Description",
		"cpu_core":          "CPUCore",
		"memory_mb":         "MemoryMB",
		"cpu_speed_mhz":     "CPUSpeedMHz",
		"root_disk_size_gb": "RootDiskSizeGB",
		"network_rate":      "NetworkRate",
		"disk_iops":         "DiskIOPS",
		"hourly_price_up":   "HourlyPriceUp",
		"hourly_price_down": "HourlyPriceDown",
		"is_available":      "IsAvailable",
		"is_public":         "IsPublic",
	}

	apiFieldName, exists := fieldMap[fieldName]
	if !exists {
		return true
	}

	if !hasField(offering, apiFieldName) {
		return true
	}

	switch fieldName {
	case "name", "description":
		fieldValue, ok := getStringField(offering, apiFieldName)
		if !ok {
			return true
		}
		for _, pattern := range values {
			if matchString(fieldValue, pattern, matchType, caseSensitive) {
				return true
			}
		}
		return false

	case "cpu_core", "memory_mb", "cpu_speed_mhz", "root_disk_size_gb", "network_rate", "disk_iops", "hourly_price_up", "hourly_price_down":
		fieldValue, ok := getIntField(offering, apiFieldName)
		if !ok {
			return true
		}
		return matchNumeric(fieldValue, matchType, values)

	case "is_available", "is_public":
		fieldValue, ok := getBoolField(offering, apiFieldName)
		if !ok {
			return true
		}
		return matchBool(fieldValue, values)

	default:
		return true
	}
}

func (d *instanceOfferingsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var (
		cfg  instanceOfferingsConfig
		data models.InstanceOfferingsDataSourceModel
	)

	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	offeringsResp, err := d.client.ListInstanceServiceOfferings(cfg.ZoneID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list instance service offerings, got error: %s", err))
		return
	}

	filterLogic := "and"
	if !cfg.FilterLogic.IsNull() && !cfg.FilterLogic.IsUnknown() {
		filterLogic = strings.ToLower(cfg.FilterLogic.ValueString())
		if filterLogic != "and" && filterLogic != "or" {
			resp.Diagnostics.AddWarning("Invalid Filter Logic", fmt.Sprintf("Invalid filter_logic value '%s', defaulting to 'and'", cfg.FilterLogic.ValueString()))
			filterLogic = "and"
		}
	}

	var filteredOfferings []interface{}
	for _, offering := range offeringsResp.Data {
		if len(cfg.Filter) == 0 {
			filteredOfferings = append(filteredOfferings, offering)
			continue
		}

		matches := make([]bool, len(cfg.Filter))
		for i, filter := range cfg.Filter {
			matches[i] = applyFilter(offering, filter)
		}

		shouldInclude := false
		if filterLogic == "or" {
			for _, match := range matches {
				if match {
					shouldInclude = true
					break
				}
			}
		} else {
			shouldInclude = true
			for _, match := range matches {
				if !match {
					shouldInclude = false
					break
				}
			}
		}

		if shouldInclude {
			filteredOfferings = append(filteredOfferings, offering)
		}
	}

	for _, offering := range filteredOfferings {
		offeringModel := models.InstanceOfferingModel{}

		if id, ok := getStringField(offering, "ID"); ok {
			offeringModel.ID = types.StringValue(id)
		}
		if name, ok := getStringField(offering, "Name"); ok {
			offeringModel.Name = types.StringValue(name)
		}
		if cpuCore, ok := getIntField(offering, "CPUCore"); ok {
			offeringModel.CPUCore = types.Int64Value(cpuCore)
		}
		if memoryMB, ok := getIntField(offering, "MemoryMB"); ok {
			offeringModel.MemoryMB = types.Int64Value(memoryMB)
		}
		if cpuSpeedMHz, ok := getIntField(offering, "CPUSpeedMHz"); ok {
			offeringModel.CPUSpeedMHz = types.Int64Value(cpuSpeedMHz)
		}
		if rootDiskSizeGB, ok := getIntField(offering, "RootDiskSizeGB"); ok {
			offeringModel.RootDiskSizeGB = types.Int64Value(rootDiskSizeGB)
		}
		if networkRate, ok := getIntField(offering, "NetworkRate"); ok {
			offeringModel.NetworkRate = types.Int64Value(networkRate)
		}
		if diskIOPS, ok := getIntField(offering, "DiskIOPS"); ok {
			offeringModel.DiskIOPS = types.Int64Value(diskIOPS)
		}
		if hourlyPriceUp, ok := getIntField(offering, "HourlyPriceUp"); ok {
			offeringModel.HourlyPriceUp = types.Int64Value(hourlyPriceUp)
		}
		if hourlyPriceDown, ok := getIntField(offering, "HourlyPriceDown"); ok {
			offeringModel.HourlyPriceDown = types.Int64Value(hourlyPriceDown)
		}
		if isAvailable, ok := getBoolField(offering, "IsAvailable"); ok {
			offeringModel.IsAvailable = types.BoolValue(isAvailable)
		}
		if isPublic, ok := getBoolField(offering, "IsPublic"); ok {
			offeringModel.IsPublic = types.BoolValue(isPublic)
		}
		if description, ok := getStringField(offering, "Description"); ok {
			offeringModel.Description = types.StringValue(description)
		}

		data.Offerings = append(data.Offerings, offeringModel)
	}

	data.ZoneID = types.StringValue(cfg.ZoneID.ValueString())
	data.ID = types.StringValue(cfg.ZoneID.ValueString() + "_offerings")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
