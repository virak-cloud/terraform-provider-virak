package helpers

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/virak-cloud/cli/pkg/http"
	"github.com/virak-cloud/cli/pkg/http/responses"
)

func GetInstanceDetails(client *http.Client, zoneID, instanceID string) (*responses.InstanceShowResponse, error) {
	return client.ShowInstance(zoneID, instanceID)
}

func EnsureInstanceRunning(client *http.Client, zoneID, instanceID string, diags *diag.Diagnostics) error {
	readResp, err := client.ShowInstance(zoneID, instanceID)
	if err != nil {
		diags.AddError("Client Error", fmt.Sprintf("Unable to read instance before ensuring running state, got error: %s", err))
		return err
	}

	instanceStatus := readResp.Data.Status
	if !IsInstanceRunning(instanceStatus) {
		_, err = client.StartInstance(zoneID, instanceID)
		if err != nil {
			diags.AddError("Client Error", fmt.Sprintf("Unable to start instance, got error: %s", err))
			return err
		}

		status, err := WaitForInstanceStatus(client, zoneID, instanceID, []string{InstanceStatusRunning, InstanceStatusUP}, MaxInstanceStatusRetries, DefaultPollInterval)
		if err != nil {
			diags.AddError("Instance Not Running", fmt.Sprintf("Instance '%s' could not be started. Current status: %s. Error: %s", instanceID, status, err))
			return err
		}
	}

	return nil
}

func EnsureInstanceStopped(client *http.Client, zoneID, instanceID string, diags *diag.Diagnostics) error {
	readResp, err := client.ShowInstance(zoneID, instanceID)
	if err != nil {
		diags.AddError("Client Error", fmt.Sprintf("Unable to read instance before ensuring stopped state, got error: %s", err))
		return err
	}

	instanceStatus := readResp.Data.Status
	if IsInstanceRunning(instanceStatus) {
		_, err = client.StopInstance(zoneID, instanceID, false)
		if err != nil {
			diags.AddError("Client Error", fmt.Sprintf("Unable to stop instance, got error: %s", err))
			return err
		}
	}

	status, err := WaitForInstanceStatus(client, zoneID, instanceID, []string{InstanceStatusStopped, InstanceStatusSTOPPED, InstanceStatusDown}, MaxInstanceStatusRetries, DefaultPollInterval)
	if err != nil {
		diags.AddError("Instance Not Stopped", fmt.Sprintf("Instance '%s' did not reach stopped state. Current status: %s. Error: %s", instanceID, status, err))
		return err
	}

	return nil
}

type LifecycleResult struct {
	Status types.String
	Diags  diag.Diagnostics
}

func HandleInstanceLifecycle(client *http.Client, zoneID, instanceID, desiredState, currentStatus string) LifecycleResult {
	result := LifecycleResult{}

	if desiredState == "reboot" {
		if IsInstanceRunning(currentStatus) {
			_, err := client.RebootInstance(zoneID, instanceID)
			if err != nil {
				result.Diags.AddError("Client Error", fmt.Sprintf("Unable to reboot instance, got error: %s", err))
				return result
			}

			status, err := WaitForInstanceStatus(client, zoneID, instanceID, []string{InstanceStatusRunning, InstanceStatusUP}, MaxInstanceStatusRetries, DefaultPollInterval)
			if err != nil {
				result.Diags.AddError("Client Error", fmt.Sprintf("Unable to read instance status after reboot, got error: %s", err))
				return result
			}
			result.Status = types.StringValue(status)
		} else {
			result.Diags.AddWarning("Reboot Skipped", fmt.Sprintf("Instance is not running (status: %s), cannot reboot", currentStatus))
			result.Status = types.StringValue(currentStatus)
		}
	} else if desiredState == "running" {
		if !IsInstanceRunning(currentStatus) {
			_, err := client.StartInstance(zoneID, instanceID)
			if err != nil {
				result.Diags.AddError("Client Error", fmt.Sprintf("Unable to start instance, got error: %s", err))
				return result
			}

			status, err := WaitForInstanceStatus(client, zoneID, instanceID, []string{InstanceStatusRunning, InstanceStatusUP}, MaxInstanceStatusRetries, DefaultPollInterval)
			if err != nil {
				result.Diags.AddError("Client Error", fmt.Sprintf("Unable to read instance status after start, got error: %s", err))
				return result
			}
			result.Status = types.StringValue(status)
		} else {
			result.Status = types.StringValue(currentStatus)
		}
	} else if desiredState == "stopped" {
		if IsInstanceRunning(currentStatus) {
			_, err := client.StopInstance(zoneID, instanceID, false)
			if err != nil {
				result.Diags.AddError("Client Error", fmt.Sprintf("Unable to stop instance, got error: %s", err))
				return result
			}
		}

		status, err := WaitForInstanceStatus(client, zoneID, instanceID, []string{InstanceStatusStopped, InstanceStatusSTOPPED, InstanceStatusDown}, MaxInstanceStatusRetries, DefaultPollInterval)
		if err != nil {
			result.Diags.AddError("Client Error", fmt.Sprintf("Unable to read instance status after stop, got error: %s", err))
			return result
		}
		result.Status = types.StringValue(status)
	} else {
		result.Status = types.StringValue(currentStatus)
	}

	return result
}

func IsInstanceRunning(status string) bool {
	return status == InstanceStatusRunning || status == InstanceStatusUP
}

func IsInstanceStopped(status string) bool {
	return status == InstanceStatusStopped || status == InstanceStatusSTOPPED || status == InstanceStatusDown
}
