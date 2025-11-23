package models

import "github.com/hashicorp/terraform-plugin-framework/types"

type BucketResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	ZoneID    types.String `tfsdk:"zone_id"`
	Policy    types.String `tfsdk:"policy"`
	URL       types.String `tfsdk:"url"`
	AccessKey types.String `tfsdk:"access_key"`
	SecretKey types.String `tfsdk:"secret_key"`
	Status    types.String `tfsdk:"status"`
	Size      types.Int64  `tfsdk:"size"`
}
