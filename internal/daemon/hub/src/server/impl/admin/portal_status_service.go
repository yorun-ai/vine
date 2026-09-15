package admin

import (
	"cmp"

	skeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/admin"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/util/vslice"
)

type PortalStatusApiServiceServerImpl struct {
	skeled.DefaultPortalStatusApiServiceServer

	PortalInstanceRepo core.PortalInstanceRepo `inject:""`
}

func (s *PortalStatusApiServiceServerImpl) List() []skeled.PortalStatusView {
	instances := s.PortalInstanceRepo.ListPortalInstances()
	items := make([]skeled.PortalStatusView, 0, len(instances))
	for _, instance := range instances {
		items = append(items, skeled.PortalStatusView{
			InstanceId: instance.InstanceId,
			Version:    instance.Version,
		})
	}
	return vslice.SortBy(items, func(a skeled.PortalStatusView, b skeled.PortalStatusView) bool {
		return cmp.Compare(a.InstanceId, b.InstanceId) < 0
	})
}
