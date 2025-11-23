package helpers

import (
	"encoding/json"
	"fmt"
	"log"
)

func LogVolumeOperation(operation, volumeID, instanceID, message string) {
	log.Printf("VOLUME_LOG: %s - volume_id=%s instance_id=%s: %s", operation, volumeID, instanceID, message)
}

func LogVolumeError(operation, volumeID, instanceID string, err error) {
	log.Printf("VOLUME_LOG: ERROR - %s failed for volume_id=%s instance_id=%s: %s", operation, volumeID, instanceID, err)
}

func LogNetworkOperation(operation, networkID, instanceID, message string) {
	log.Printf("NETWORK_LOG: %s - network_id=%s instance_id=%s: %s", operation, networkID, instanceID, message)
}

func LogNetworkError(operation, networkID, instanceID string, err error) {
	log.Printf("NETWORK_LOG: ERROR - %s failed for network_id=%s instance_id=%s: %s", operation, networkID, instanceID, err)
}

func LogInstanceOperation(operation, instanceID, message string) {
	log.Printf("INSTANCE_LOG: %s - instance_id=%s: %s", operation, instanceID, message)
}

func LogInstanceError(operation, instanceID string, err error) {
	log.Printf("INSTANCE_LOG: ERROR - %s failed for instance_id=%s: %s", operation, instanceID, err)
}

func LogInfo(message string, args ...interface{}) {
	if len(args) > 0 {
		log.Printf(message, args...)
	} else {
		log.Print(message)
	}
}

func LogWarning(message string, args ...interface{}) {
	msg := fmt.Sprintf("WARNING: %s", message)
	if len(args) > 0 {
		log.Printf(msg, args...)
	} else {
		log.Print(msg)
	}
}

func LogError(message string, args ...interface{}) {
	msg := fmt.Sprintf("ERROR: %s", message)
	if len(args) > 0 {
		log.Printf(msg, args...)
	} else {
		log.Print(msg)
	}
}

func LogNetworks(message string, networks interface{}) {
	log.Printf("=== %s ===", message)

	networksJSON, err := json.MarshalIndent(networks, "", "  ")
	if err != nil {
		log.Printf("Error marshaling networks for logging: %v", err)
		return
	}

	log.Printf("Networks data: %s", string(networksJSON))

	if listResp, ok := networks.(*struct {
		Data []struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			Status string `json:"status"`
			Type   string `json:"type"`
		} `json:"data"`
	}); ok {
		log.Printf("Total networks found: %d", len(listResp.Data))
		for _, network := range listResp.Data {
			log.Printf("  - Network: %s (ID: %s, Status: %s, Type: %s)",
				network.Name, network.ID, network.Status, network.Type)
		}
	}
	log.Printf("=== End %s ===", message)
}

func LogAPIResponse(operation string, response interface{}, err error) {
	log.Printf("=== API Response: %s ===", operation)
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		responseJSON, marshalErr := json.MarshalIndent(response, "", "  ")
		if marshalErr != nil {
			log.Printf("Error marshaling response for logging: %v", marshalErr)
		} else {
			log.Printf("Response: %s", string(responseJSON))
		}
	}
	log.Printf("=== End API Response: %s ===", operation)
}
