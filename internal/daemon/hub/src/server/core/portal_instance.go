package core

import "time"

// PortalInstance is a Portal daemon instance registered with Hub. Portal has no
// processing capabilities to advertise, so Hub only tracks its identity,
// process start time and liveness.
type PortalInstance struct {
	InstanceId string
	Version    string
	StartedAt  time.Time
	ExpiresAt  time.Time
}

type PortalInstanceRepo interface {
	SavePortalInstance(instance *PortalInstance)
	ListPortalInstances() []*PortalInstance
	GetPortalInstance(instanceId string) (*PortalInstance, bool)
	KeepPortalInstance(instanceId string) bool
	RemovePortalInstance(instanceId string)
	PopExpiredPortalLeases() []string
}
