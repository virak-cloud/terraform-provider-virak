package helpers

import (
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/virak-cloud/cli/pkg/http"
	"github.com/virak-cloud/cli/pkg/http/responses"
)

type CheckFunc func() (bool, error)

func PollUntilCondition(checkFunc CheckFunc, maxRetries int, interval time.Duration, errorMsg string) error {
	for i := 0; i < maxRetries; i++ {
		done, err := checkFunc()
		if err != nil {
			return err
		}
		if done {
			return nil
		}
		time.Sleep(interval)
	}
	return fmt.Errorf("%s", errorMsg)
}

func WaitForInstanceStatus(client *http.Client, zoneID, instanceID string, targetStatuses []string, maxRetries int, interval time.Duration) (string, error) {
	var currentStatus string
	checkFunc := func() (bool, error) {
		readResp, err := client.ShowInstance(zoneID, instanceID)
		if err != nil {
			return false, err
		}
		currentStatus = readResp.Data.Status
		for _, target := range targetStatuses {
			if currentStatus == target {
				return true, nil
			}
		}
		return false, nil
	}

	err := PollUntilCondition(checkFunc, maxRetries, interval, fmt.Sprintf("Instance '%s' did not reach status %v within timeout", instanceID, targetStatuses))
	return currentStatus, err
}

func WaitForVolumeAttachment(client *http.Client, zoneID, instanceID, volumeID string, maxRetries int, interval time.Duration) error {
	checkFunc := func() (bool, error) {
		readResp, err := client.ShowInstance(zoneID, instanceID)
		if err != nil {
			return false, err
		}
		for _, v := range readResp.Data.DataVolumes {
			if s, ok := v.(string); ok && s == volumeID {
				return true, nil
			}
		}
		return false, nil
	}

	return PollUntilCondition(checkFunc, maxRetries, interval, fmt.Sprintf("Volume '%s' attachment to instance '%s' did not complete within timeout", volumeID, instanceID))
}

func WaitForVolumeDetachment(client *http.Client, zoneID, instanceID, volumeID string, maxRetries int, interval time.Duration) error {
	checkFunc := func() (bool, error) {
		readResp, err := client.ShowInstance(zoneID, instanceID)
		if err != nil {
			return false, err
		}
		for _, v := range readResp.Data.DataVolumes {
			if s, ok := v.(string); ok && s == volumeID {
				return false, nil
			}
		}
		return true, nil
	}

	return PollUntilCondition(checkFunc, maxRetries, interval, fmt.Sprintf("Volume '%s' detachment from instance '%s' did not complete within timeout", volumeID, instanceID))
}

func WaitForNetworkConnection(client *http.Client, zoneID, networkID, instanceID string, maxRetries int, interval time.Duration) (*responses.InstanceNetwork, error) {
	var foundAttachment *responses.InstanceNetwork
	checkFunc := func() (bool, error) {
		networkInstancesResp, err := client.ListNetworkInstances(zoneID, networkID, instanceID)
		if err != nil {
			return false, err
		}

		for _, ni := range networkInstancesResp.Data {
			if ni.InstanceID == instanceID && ni.Network.ID == networkID {
				foundAttachment = &ni
				return true, nil
			}
		}
		return false, nil
	}

	err := PollUntilCondition(checkFunc, maxRetries, interval, fmt.Sprintf("Instance '%s' connection to network '%s' was not established within timeout", instanceID, networkID))
	if err != nil {
		return nil, err
	}
	return foundAttachment, nil
}

func WaitForNetworkDisconnection(client *http.Client, zoneID, networkID, instanceID, attachmentID string, maxRetries int, interval time.Duration) error {
	checkFunc := func() (bool, error) {
		verifyResp, err := client.ListNetworkInstances(zoneID, networkID, instanceID)
		if err != nil {
			return false, err
		}

		for _, ni := range verifyResp.Data {
			if ni.ID == attachmentID && ni.InstanceID == instanceID {
				return false, nil
			}
		}
		return true, nil
	}

	return PollUntilCondition(checkFunc, maxRetries, interval, fmt.Sprintf("Instance '%s' disconnection from network '%s' was not verified within timeout", instanceID, networkID))
}

func WaitForResourceDeletion(client *http.Client, zoneID, resourceID string, listFunc func(string) (interface{}, error), checkFunc func(interface{}, string) bool, maxRetries int, interval time.Duration) error {
	check := func() (bool, error) {
		listResp, err := listFunc(zoneID)
		if err != nil {
			return false, err
		}
		if checkFunc(listResp, resourceID) {
			return true, nil
		}
		return false, nil
	}

	return PollUntilCondition(check, maxRetries, interval, fmt.Sprintf("Resource '%s' was not deleted successfully within timeout", resourceID))
}

func FindNewInstanceID(client *http.Client, zoneID string, existingIDs map[string]struct{}, instanceName string, maxRetries int, interval time.Duration) (string, error) {
	var newInstanceID string
	checkFunc := func() (bool, error) {
		latestInstances, err := client.ListInstances(zoneID)
		if err != nil {
			return false, err
		}

		for _, instance := range latestInstances.Data {
			if _, exists := existingIDs[instance.ID]; !exists {
				if instanceName == "" || instance.Name == instanceName {
					newInstanceID = instance.ID
					return true, nil
				}
			}
		}
		return false, nil
	}

	err := PollUntilCondition(checkFunc, maxRetries, interval, "New instance could not be found after creation")
	return newInstanceID, err
}

func FindNewNetworkID(client *http.Client, zoneID string, existingIDs map[string]struct{}, networkName string, maxRetries int, interval time.Duration) (string, error) {
	var newNetworkID string
	checkFunc := func() (bool, error) {
		latestNetworks, err := client.ListNetworks(zoneID)
		if err != nil {
			return false, err
		}

		for _, network := range latestNetworks.Data {
			if _, exists := existingIDs[network.ID]; !exists {
				if networkName == "" || network.Name == networkName {
					newNetworkID = network.ID
					return true, nil
				}
			}
		}
		return false, nil
	}

	err := PollUntilCondition(checkFunc, maxRetries, interval, fmt.Sprintf("New network '%s' could not be found after creation", networkName))
	return newNetworkID, err
}

func FindNewVolumeID(client *http.Client, zoneID string, existingIDs map[string]struct{}, volumeName string, maxRetries int, interval time.Duration) (string, error) {
	var newVolumeID string
	checkFunc := func() (bool, error) {
		latestVolumes, err := client.ListInstanceVolumes(zoneID)
		if err != nil {
			return false, err
		}

		for _, vol := range latestVolumes.Data {
			if _, exists := existingIDs[vol.ID]; !exists {
				if volumeName == "" || vol.Name == volumeName {
					newVolumeID = vol.ID
					return true, nil
				}
			}
		}
		return false, nil
	}

	err := PollUntilCondition(checkFunc, maxRetries, interval, fmt.Sprintf("New volume '%s' could not be found after creation", volumeName))
	return newVolumeID, err
}

func FindNewBucketID(client *http.Client, zoneID string, existingIDs map[string]struct{}, bucketName string, maxRetries int, interval time.Duration) (string, error) {
	var newBucketID string
	checkFunc := func() (bool, error) {
		latestBuckets, err := client.GetObjectStorageBuckets(zoneID)
		if err != nil {
			return false, err
		}

		for _, bucket := range latestBuckets.Data {
			if _, exists := existingIDs[bucket.ID]; !exists {
				// Check if name matches if provided, otherwise just take the first new one (legacy behavior)
				// Assuming bucket struct has Name field. If not, we might need to fetch details.
				// Based on other resources, list response usually has Name.
				if bucketName == "" || bucket.Name == bucketName {
					newBucketID = bucket.ID
					return true, nil
				}
			}
		}
		return false, nil
	}

	err := PollUntilCondition(checkFunc, maxRetries, interval, fmt.Sprintf("New bucket '%s' could not be found after creation", bucketName))
	return newBucketID, err
}

func FindNewKubernetesClusterID(client *http.Client, zoneID string, clusterName string, maxRetries int, interval time.Duration) (string, error) {
	var newClusterID string
	checkFunc := func() (bool, error) {
		clustersResp, err := client.GetKubernetesClusters(zoneID)
		if err != nil {
			return false, err
		}

		for _, cluster := range clustersResp.Data {
			if cluster.Name == clusterName {
				newClusterID = cluster.ID
				return true, nil
			}
		}
		return false, nil
	}

	err := PollUntilCondition(checkFunc, maxRetries, interval, fmt.Sprintf("New kubernetes cluster '%s' could not be found after creation", clusterName))
	return newClusterID, err
}

func AddDiagnosticError(diags *diag.Diagnostics, summary, detail string) {
	diags.AddError(summary, detail)
}

func AddDiagnosticWarning(diags *diag.Diagnostics, summary, detail string) {
	diags.AddWarning(summary, detail)
}
