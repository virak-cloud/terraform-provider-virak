package helpers

import "time"

const (
	DefaultPollInterval        = 5 * time.Second
	DefaultNetworkPollInterval = 1 * time.Second
	DefaultVolumePollInterval  = 5 * time.Second

	MaxInstanceStatusRetries    = 120
	MaxNetworkConnectionRetries = 30
	MaxVolumeAttachmentRetries  = 120
	MaxVolumeStatusRetries      = 10
	MaxNetworkDeletionRetries   = 100
	MaxInstanceDeletionRetries  = 100

	InstanceStatusRunning = "Running"
	InstanceStatusUP      = "UP"
	InstanceStatusStopped = "Stopped"
	InstanceStatusSTOPPED = "STOPPED"
	InstanceStatusDown    = "DOWN"

	VolumeStatusAllocated = "ALLOCATED"
	VolumeStatusAttaching = "ATTACHING"

	NetworkStatusActive = "Active"
)
