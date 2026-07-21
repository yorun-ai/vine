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

	Context        context.Context         `inject:""`
	InprocFlag     *app.InternalInprocFlag `inject:""`
	RegistryCore   *core.RegistryCore      `inject:""`
	RegistryRepo   core.RegistryRepo       `inject:""`
	PortalSiteRepo core.PortalSiteRepo     `inject:""`
	SchemaRepo     core.SchemaRepo         `inject:""`
	Syncer         *syncer.Syncer          `inject:""`

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
	refreshed := false
	for {
		leases := s.RegistryRepo.PopExpiredAppLeases()
		if len(leases) == 0 {
			break
		}
		for _, lease := range leases {
			status, ok := s.RegistryRepo.GetAppStatus(lease.Name, lease.InstanceId)
			if !ok || time.Now().Before(status.ExpiresAt) {
				continue
			}
			s.RegistryCore.Unregister(lease.Name, lease.InstanceId)
			refreshed = true
		}
	}
	if !refreshed {
		return
	}
	s.refreshSchemas()
	s.refreshPortalSiteRpcgwServices()
}

func (s *Sweeper) refreshSchemas() {
	s.Syncer.SyncSchemas(s.SchemaRepo.ListDomainSchemaViews())
}

func (s *Sweeper) refreshPortalSiteRpcgwServices() {
	domainViews := s.SchemaRepo.ListDomainSchemaViews()
	for _, site := range s.PortalSiteRepo.ListEntries() {
		if site.BuiltIn {
			continue
		}
		s.Syncer.SyncPortalSiteWithRpcgwServices(&site, core.MatchPortalSiteRpcgwServicesInDomainViews(site, domainViews))
	}
}
