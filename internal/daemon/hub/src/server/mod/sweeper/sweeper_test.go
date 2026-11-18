package sweeper

import (
	"testing"
	"time"
	"uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	skeldesc "go.yorun.ai/skel/descriptor"
	skeltype "go.yorun.ai/skel/types"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/watchserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/mod/syncer"
)

type _SweeperRegistryRepo struct {
	core.RegistryRepo

	leases     []core.AppHeartbeat
	status     *core.AppStatus
	statusOK   bool
	removedApp []string
}

func (r *_SweeperRegistryRepo) ListExpiredAppLeases() []core.AppHeartbeat {
	leases := r.leases
	r.leases = nil
	return leases
}

func (r *_SweeperRegistryRepo) GetAppStatus(string, uuid.UUID) (*core.AppStatus, bool) {
	return r.status, r.statusOK
}

func (r *_SweeperRegistryRepo) RemoveAppStatus(appName string, instanceId uuid.UUID) {
	r.removedApp = append(r.removedApp, appName+":"+instanceId.String())
}

type _SweeperDescriptorRepo struct {
	core.DescriptorRepo
	released []string
	views    []core.DomainDescriptorView
	failList bool
}

func (r *_SweeperDescriptorRepo) ReleaseDomainDescriptors(ownerName string, ownerId string) {
	r.released = append(r.released, ownerName+":"+ownerId)
}

func (*_SweeperDescriptorRepo) SaveDomainDescriptorsJSON(string, string, []skeltype.JSON) {}

func (r *_SweeperDescriptorRepo) ListDomainDescriptorViews() []core.DomainDescriptorView {
	if r.failList {
		panic("descriptor snapshot unavailable")
	}
	return r.views
}

type _SweeperPortalSiteRepo struct {
	core.PortalSiteRepo
	entries   []*core.PortalSite
	failList  bool
	listCalls int
}

func (r *_SweeperPortalSiteRepo) List() []*core.PortalSite {
	r.listCalls++
	if r.failList {
		panic("portal sites unavailable")
	}
	return r.entries
}

func TestSweeperSkipsLiveLeaseStatus(t *testing.T) {
	registryRepo := &_SweeperRegistryRepo{
		leases: []core.AppHeartbeat{{Name: "demo.app", InstanceId: uuid.MustParse("00000000-0000-0000-0000-000000000001")}},
		status: &core.AppStatus{
			Name:       "demo.app",
			InstanceId: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			ExpiresAt:  time.Now().Add(time.Minute),
		},
		statusOK: true,
	}
	descriptorRepo := &_SweeperDescriptorRepo{}
	target := &Sweeper{
		PortalInstanceCore: &core.PortalInstanceCore{PortalInstanceRepo: &_SweeperPortalInstanceRepo{}},
		RegistryCore: &core.RegistryCore{
			RegistryRepo:   registryRepo,
			DescriptorRepo: descriptorRepo,
		},
	}

	target.sweepExpiredLeases()

	assert.Empty(t, registryRepo.removedApp)
	assert.Empty(t, descriptorRepo.released)
}

func TestSweeperUnregistersExpiredLeaseStatus(t *testing.T) {
	watchServer := watchserver.NewServerForTest()
	defer watchServer.AfterAppStop()
	syncerModule := &syncer.Syncer{WatchServer: watchServer}
	syncerModule.DIInit()
	registryRepo := &_SweeperRegistryRepo{
		leases: []core.AppHeartbeat{{Name: "demo.app", InstanceId: uuid.MustParse("00000000-0000-0000-0000-000000000001")}},
		status: &core.AppStatus{
			Name:       "demo.app",
			InstanceId: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			ExpiresAt:  time.Now().Add(-time.Minute),
		},
		statusOK: true,
	}
	descriptorRepo := &_SweeperDescriptorRepo{
		views: []core.DomainDescriptorView{{
			DomainVersion: core.DomainDescriptorVersion{
				Descriptor: &skeldesc.Domain{
					Services: []*skeldesc.Service{{
						SkelName: "demo.Service",
						Hash:     "service-main",
						Audiences: []*skeldesc.ActorAudience{{
							SkelName: "demo.Actor",
						}}, AuthMode: skeldesc.AuthModeRequired,
					}},
					Generated: &skeldesc.GeneratedInfo{CompilerVersion: "v99.0.0"},
				},
				Main: true,
			},
			Services: []core.DescriptorVersion[*skeldesc.Service]{{
				Descriptor: &skeldesc.Service{
					SkelName: "demo.Service",
					Hash:     "service-main",
					AuthMode: skeldesc.AuthModeRequired,
				},
				SkelName:       "demo.Service",
				DescriptorHash: "service-main",
				Main:           true,
			}},
		}},
	}
	portalSiteRepo := &_SweeperPortalSiteRepo{
		entries: []*core.PortalSite{{
			Id:            1,
			Name:          "demo-rpc",
			Type:          core.PortalSiteTypeRPCGW,
			ActorSkelName: "demo.Actor",
			RpcgwServices: []string{"demo.Service"},
			Enabled:       true,
		}},
	}
	target := &Sweeper{
		PortalInstanceCore: &core.PortalInstanceCore{PortalInstanceRepo: &_SweeperPortalInstanceRepo{}},
		DescriptorRepo:     descriptorRepo,
		PortalSiteCore:     &core.PortalSiteCore{PortalSiteRepo: portalSiteRepo, DescriptorRepo: descriptorRepo},
		Syncer:             syncerModule,
		RegistryCore: &core.RegistryCore{
			RegistryRepo:   registryRepo,
			DescriptorRepo: descriptorRepo,
		},
	}

	target.sweepExpiredLeases()

	assert.Equal(t, []string{"demo.app:00000000-0000-0000-0000-000000000001"}, registryRepo.removedApp)
	assert.Equal(t, []string{"demo.app:00000000-0000-0000-0000-000000000001"}, descriptorRepo.released)
	value, ok := watchServer.Get(watched.FormatPortalSiteKey("demo-rpc"))
	assert.True(t, ok)
	assert.JSONEq(t, `{
		"name": "demo-rpc",
		"type": "RPCGW",
		"actorSkelName": "demo.Actor",
		"cors": {
			"mode": "",
			"allowedOrigins": []
		},
		"rpcgwConfig": {
            "inspectEnabled": false,
            "mcpEnabled": false,
			"services": [
				{"skelName": "demo.Service"}
			]
		}
	}`, value)
}

type _SweeperPortalInstanceRepo struct {
	core.PortalInstanceRepo

	leases  []uuid.UUID
	removed []uuid.UUID
}

func (r *_SweeperPortalInstanceRepo) ListExpiredLeases() []uuid.UUID {
	leases := r.leases
	r.leases = nil
	return leases
}

func (r *_SweeperPortalInstanceRepo) Remove(instanceId uuid.UUID) {
	r.removed = append(r.removed, instanceId)
}

func TestSweeperRemovesExpiredPortalInstances(t *testing.T) {
	portalRepo := &_SweeperPortalInstanceRepo{leases: []uuid.UUID{uuid.MustParse("00000000-0000-0000-0000-000000000001"), uuid.MustParse("00000000-0000-0000-0000-000000000002")}}
	target := &Sweeper{
		PortalInstanceCore: &core.PortalInstanceCore{PortalInstanceRepo: portalRepo},
		RegistryCore: &core.RegistryCore{
			RegistryRepo:   &_SweeperRegistryRepo{},
			DescriptorRepo: &_SweeperDescriptorRepo{},
		},
	}

	target.sweepExpiredLeases()

	assert.Equal(t, []uuid.UUID{uuid.MustParse("00000000-0000-0000-0000-000000000001"), uuid.MustParse("00000000-0000-0000-0000-000000000002")}, portalRepo.removed)
}

func (r *_SweeperPortalInstanceRepo) GetById(instanceId uuid.UUID) (*core.PortalInstance, bool) {
	return &core.PortalInstance{InstanceId: instanceId, ExpiresAt: time.Now().Add(-time.Minute)}, true
}

func TestSweeperRetriesPublicationAfterCleanup(t *testing.T) {
	for _, stage := range []string{"descriptors", "portal sites"} {
		t.Run(stage, func(t *testing.T) {
			server := watchserver.NewServerForTest()
			t.Cleanup(server.AfterAppStop)
			publisher := &syncer.Syncer{WatchServer: server}
			publisher.DIInit()
			publisher.SyncDescriptors([]core.DomainDescriptorView{{DomainVersion: core.DomainDescriptorVersion{
				Descriptor: &skeldesc.Domain{Name: "demo", Hash: "00000000-0000-0000-0000-0000000000d5"},
				Main:       true,
			}}})
			registryRepo := &_SweeperRegistryRepo{
				leases:   []core.AppHeartbeat{{Name: "demo.app", InstanceId: uuid.MustParse("00000000-0000-0000-0000-0000000000d5")}},
				status:   &core.AppStatus{Name: "demo.app", InstanceId: uuid.MustParse("00000000-0000-0000-0000-0000000000d5"), ExpiresAt: time.Now().Add(-time.Minute)},
				statusOK: true,
			}
			descriptors := &_SweeperDescriptorRepo{failList: stage == "descriptors"}
			sites := &_SweeperPortalSiteRepo{failList: stage == "portal sites"}
			target := &Sweeper{
				PortalInstanceCore: &core.PortalInstanceCore{PortalInstanceRepo: &_SweeperPortalInstanceRepo{}},
				RegistryCore:       &core.RegistryCore{RegistryRepo: registryRepo, DescriptorRepo: descriptors},
				DescriptorRepo:     descriptors,
				PortalSiteCore:     &core.PortalSiteCore{PortalSiteRepo: sites, DescriptorRepo: descriptors},
				Syncer:             publisher,
			}
			require.Panics(t, target.sweepExpiredLeases)
			require.Equal(t, []string{"demo.app:00000000-0000-0000-0000-0000000000d5"}, registryRepo.removedApp)
			require.True(t, target.syncPending)
			descriptors.failList = false
			sites.failList = false
			target.sweepExpiredLeases()
			require.False(t, target.syncPending)
			_, ok := server.Get(watched.FormatDescriptorDomainKey("demo"))
			require.False(t, ok, "stale descriptor must be withdrawn despite no newly expired leases")
			calls := sites.listCalls
			target.sweepExpiredLeases()
			require.Equal(t, calls, sites.listCalls, "successful publication does not need another retry")
		})
	}
}
