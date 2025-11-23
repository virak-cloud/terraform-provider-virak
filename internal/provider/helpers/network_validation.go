package helpers

import (
	"log"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// CompareNetworkAttributes compares plan and state attributes and returns changed attributes
func CompareNetworkAttributes(plan, state struct {
	Name              types.String
	ZoneID            types.String
	NetworkOfferingID types.String
	Type              types.String
	Gateway           types.String
	Netmask           types.String
}) []string {
	changedAttributes := make([]string, 0)

	if !plan.Name.Equal(state.Name) {
		log.Printf("Name changed: '%s' -> '%s'", state.Name.ValueString(), plan.Name.ValueString())
		changedAttributes = append(changedAttributes, "name")
	}
	if !plan.ZoneID.Equal(state.ZoneID) {
		log.Printf("Zone ID changed: '%s' -> '%s'", state.ZoneID.ValueString(), plan.ZoneID.ValueString())
		changedAttributes = append(changedAttributes, "zone_id")
	}
	if !plan.NetworkOfferingID.Equal(state.NetworkOfferingID) {
		log.Printf("Network Offering ID changed: '%s' -> '%s'", state.NetworkOfferingID.ValueString(), plan.NetworkOfferingID.ValueString())
		changedAttributes = append(changedAttributes, "network_offering_id")
	}
	if !plan.Type.Equal(state.Type) {
		log.Printf("Type changed: '%s' -> '%s'", state.Type.ValueString(), plan.Type.ValueString())
		changedAttributes = append(changedAttributes, "type")
	}
	if !plan.Gateway.Equal(state.Gateway) {
		if !plan.Gateway.IsUnknown() && !state.Gateway.IsUnknown() {
			if plan.Gateway.ValueString() != state.Gateway.ValueString() {
				log.Printf("Gateway changed: '%s' -> '%s'", state.Gateway.ValueString(), plan.Gateway.ValueString())
				changedAttributes = append(changedAttributes, "gateway")
			}
		}
	}
	if !plan.Netmask.Equal(state.Netmask) {
		if !plan.Netmask.IsUnknown() && !state.Netmask.IsUnknown() {
			if plan.Netmask.ValueString() != state.Netmask.ValueString() {
				log.Printf("Netmask changed: '%s' -> '%s'", state.Netmask.ValueString(), plan.Netmask.ValueString())
				changedAttributes = append(changedAttributes, "netmask")
			}
		}
	}

	return changedAttributes
}
