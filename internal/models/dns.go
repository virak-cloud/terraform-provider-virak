package models

import "github.com/hashicorp/terraform-plugin-framework/types"

type DomainResourceModel struct {
	ID      types.String `tfsdk:"id"`
	Domain  types.String `tfsdk:"domain"`
	Status  types.String `tfsdk:"status"`
	DNSInfo types.String `tfsdk:"dns_info"`
}

type DnsRecordResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Domain      types.String `tfsdk:"domain"`
	Record      types.String `tfsdk:"record"`
	Type        types.String `tfsdk:"type"`
	Content     types.String `tfsdk:"content"`
	TTL         types.Int64  `tfsdk:"ttl"`
	Priority    types.Int64  `tfsdk:"priority"`
	Weight      types.Int64  `tfsdk:"weight"`
	Port        types.Int64  `tfsdk:"port"`
	Flags       types.Int64  `tfsdk:"flags"`
	Tag         types.String `tfsdk:"tag"`
	License     types.Int64  `tfsdk:"license"`
	Choicer     types.Int64  `tfsdk:"choicer"`
	Match       types.Int64  `tfsdk:"match"`
	ContentID   types.String `tfsdk:"content_id"`
	Status      types.String `tfsdk:"status"`
	IsProtected types.Bool   `tfsdk:"is_protected"`
}
