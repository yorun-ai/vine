package sweeper

import (
	"context"
	"time"

	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/mod/syncer"
	"go.yorun.ai/vine/internal/util/goutil"
)

const registrySweepInterval = 5 * time.Second

type Sweeper struct {
	app.BaseModule

	Context            context.Context          `inject:""`
	InprocFlag         *app.InternalInprocFlag  `inject:""`
	RegistryCore       *core.RegistryCore       `inject:""`
	PortalInstanceCore *core.PortalInstanceCore `inject:""`
	PortalSiteCore     *core.PortalSiteCore     `inject:""`
	DescriptorRepo     core.DescriptorRepo      `inject:""`
	Syncer             *syncer.Syncer           `inject:""`

	stop        context.CancelFunc
	syncPending bool
}

func (s *Sweeper) AfterAppStart() {
	if s.InprocFlag.Enabled {
		return
	}

	ctx, cancel := context.WithCancel(s.Context)
	s.stop = cancel
	s.sweepExpiredLeases()
	goutil.NewSafeTicker(ctx, registrySweepInterval, nil).Go(s.sweepExpiredLeases)
}

func (s *Sweeper) AfterAppStop() {
	if s.stop != nil {
		s.stop()
		s.stop = nil
	}
}

func (s *Sweeper) sweepExpiredLeases() {
	s.sweepExpiredPortalInstances()
	s.syncPending = s.RegistryCore.SweepExpiredLeases() || s.syncPending
	if !s.syncPending {
		return
	}
	s.refreshDescriptors()
	s.refreshPortalSiteRpcgwServices()
	s.syncPending = false
}

// sweepExpiredPortalInstances drops Portal instances that stopped heartbeating,
// for example a Portal that was terminated without unregistering.
func (s *Sweeper) sweepExpiredPortalInstances() {
	s.PortalInstanceCore.SweepExpired()
}

func (s *Sweeper) refreshDescriptors() {
	s.Syncer.SyncDescriptors(s.DescriptorRepo.ListDomainDescriptorViews())
}

func (s *Sweeper) refreshPortalSiteRpcgwServices() {
	s.Syncer.RefreshPortalSites(s.PortalSiteCore.List)
}
