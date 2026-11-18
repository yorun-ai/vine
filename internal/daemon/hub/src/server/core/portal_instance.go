package core

import (
	"time"
	"uuid"

	"go.yorun.ai/vine/internal/core/di"
	"go.yorun.ai/vine/internal/util/goutil"
)

// PortalInstance is a Portal daemon instance registered with Hub. Portal has no
// processing capabilities to advertise, so Hub only tracks its identity and
// liveness.
type PortalInstance struct {
	InstanceId uuid.UUID
	Version    string
	ExpiresAt  time.Time
}

// PortalInstanceRepo tracks the Portal daemons registered with Hub. List and the
// lookups return entities the caller owns.
type PortalInstanceRepo interface {
	Save(instance *PortalInstance)
	List() []*PortalInstance
	GetById(instanceId uuid.UUID) (*PortalInstance, bool)
	Keep(instanceId uuid.UUID) bool
	Remove(instanceId uuid.UUID)
	ListExpiredLeases() []uuid.UUID
}

// PortalInstanceCore tracks the Portal daemons registered with Hub.
type PortalInstanceCore struct {
	di.SingletonScoped
	locks              _InstanceLocks
	PortalInstanceRepo PortalInstanceRepo `inject:""`
}

// Register records a Portal instance, replacing the record of the same identity.
func (m *PortalInstanceCore) Register(instance PortalInstance) {
	unlock := m.locks.lock("", instance.InstanceId.String())
	defer unlock()
	m.PortalInstanceRepo.Save(&instance)
}

// Unregister removes a Portal instance.
func (m *PortalInstanceCore) Unregister(instanceId uuid.UUID) {
	unlock := m.locks.lock("", instanceId.String())
	defer unlock()
	m.PortalInstanceRepo.Remove(instanceId)
}

// Heartbeat reports whether Hub still knows the instance.
func (m *PortalInstanceCore) Heartbeat(instanceId uuid.UUID) bool {
	unlock := m.locks.lock("", instanceId.String())
	defer unlock()
	return m.PortalInstanceRepo.Keep(instanceId)
}

// SweepExpired removes the instances whose lease expired and returns how many
// registrations were dropped.
func (m *PortalInstanceCore) SweepExpired() int {
	removed := 0
	for _, instanceId := range m.PortalInstanceRepo.ListExpiredLeases() {
		goutil.RunWithRecover(func(err any) {
			registryLogger.Error("expired portal cleanup failed", "instanceId", instanceId, "error", err)
		}, func() {
			unlock := m.locks.lock("", instanceId.String())
			defer unlock()
			instance, ok := m.PortalInstanceRepo.GetById(instanceId)
			if ok && (instance.ExpiresAt.IsZero() || time.Now().Before(instance.ExpiresAt)) {
				return
			}
			m.PortalInstanceRepo.Remove(instanceId)
			removed++
		})
	}
	return removed
}
