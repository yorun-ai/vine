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
	SchemaRepo         core.SchemaRepo          `inject:""`
	Syncer             *syncer.Syncer           `inject:""`

	stop context.CancelFunc
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
	if !s.RegistryCore.SweepExpiredLeases() {
		return
	}
	s.refreshSchemas()
	s.refreshPortalSiteRpcgwServices()
}

// sweepExpiredPortalInstances drops Portal instances that stopped heartbeating,
// for example a Portal that was terminated without unregistering.
func (s *Sweeper) sweepExpiredPortalInstances() {
	s.PortalInstanceCore.SweepExpired()
}

func (s *Sweeper) refreshSchemas() {
	s.Syncer.SyncSchemas(s.SchemaRepo.ListDomainSchemaViews())
}

func (s *Sweeper) refreshPortalSiteRpcgwServices() {
	for _, site := range s.PortalSiteCore.List() {
		s.Syncer.SyncPortalSite(site)
	}
}
