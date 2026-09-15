package control

import (
	"go.yorun.ai/vine/internal/core/skel"
	skeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/control"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
)

// PortalRegistryServiceServerImpl tracks Portal daemon instances for the Hub
// Dashboard. Portal registers, heartbeats and unregisters itself here.
type PortalRegistryServiceServerImpl struct {
	skeled.DefaultPortalRegistryServiceServer

	PortalInstanceCore *core.PortalInstanceCore `inject:""`
}

func (s *PortalRegistryServiceServerImpl) Register(registration skeled.PortalRegistration) {
	s.PortalInstanceCore.Register(core.PortalInstance{
		InstanceId: registration.InstanceId.String(),
		Version:    registration.Version,
	})
}

func (s *PortalRegistryServiceServerImpl) Unregister(instanceId skel.UUID) {
	s.PortalInstanceCore.Unregister(instanceId.String())
}

func (s *PortalRegistryServiceServerImpl) Heartbeat(status skeled.PortalStatus) bool {
	return s.PortalInstanceCore.Heartbeat(status.InstanceId.String())
}
