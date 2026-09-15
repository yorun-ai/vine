package core

import "time"

// PortalInstance is a Portal daemon instance registered with Hub. Portal has no
// processing capabilities to advertise, so Hub only tracks its identity and
// liveness.
type PortalInstance struct {
	InstanceId string
	Version    string
	ExpiresAt  time.Time
}

// PortalInstanceRepo tracks the Portal daemons registered with Hub. List and the
// lookups return entities the caller owns.
type PortalInstanceRepo interface {
	Save(instance *PortalInstance)
	List() []*PortalInstance
	GetById(instanceId string) (*PortalInstance, bool)
	Keep(instanceId string) bool
	Remove(instanceId string)
	PopExpiredLeases() []string
}

// PortalInstanceCore tracks the Portal daemons registered with Hub.
type PortalInstanceCore struct {
	PortalInstanceRepo PortalInstanceRepo `inject:""`
}

// Register records a Portal instance, replacing the record of the same identity.
func (m *PortalInstanceCore) Register(instance PortalInstance) {
	m.PortalInstanceRepo.Save(&instance)
}

// Unregister removes a Portal instance.
func (m *PortalInstanceCore) Unregister(instanceId string) {
	m.PortalInstanceRepo.Remove(instanceId)
}

// Heartbeat reports whether Hub still knows the instance.
func (m *PortalInstanceCore) Heartbeat(instanceId string) bool {
	return m.PortalInstanceRepo.Keep(instanceId)
}

// SweepExpired removes the instances whose lease expired and returns how many
// registrations were dropped.
func (m *PortalInstanceCore) SweepExpired() int {
	expired := m.PortalInstanceRepo.PopExpiredLeases()
	for _, instanceId := range expired {
		m.PortalInstanceRepo.Remove(instanceId)
	}
	return len(expired)
}
