package sweeper

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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

func (r *_SweeperRegistryRepo) PopExpiredAppLeases() []core.AppHeartbeat {
	leases := r.leases
	r.leases = nil
	return leases
}

func (r *_SweeperRegistryRepo) GetAppStatus(string, string) (*core.AppStatus, bool) {
	return r.status, r.statusOK
}

func (r *_SweeperRegistryRepo) RemoveAppStatus(appName string, instanceId string) {
	r.removedApp = append(r.removedApp, appName+":"+instanceId)
}

type _SweeperDescriptorRepo struct {
	core.DescriptorRepo
	released []string
	views    []core.DomainDescriptorView
}

func (r *_SweeperDescriptorRepo) ReleaseDomainDescriptors(ownerName string, ownerId string) {
	r.released = append(r.released, ownerName+":"+ownerId)
}

func (*_SweeperDescriptorRepo) SaveDomainDescriptorsJSON(string, string, []skeltype.JSON) {}

func (r *_SweeperDescriptorRepo) ListDomainDescriptorViews() []core.DomainDescriptorView {
	return r.views
}

type _SweeperPortalSiteRepo struct {
	core.PortalSiteRepo
	entries []*core.PortalSite
}

func (r *_SweeperPortalSiteRepo) List() []*core.PortalSite {
	return r.entries
}

func TestSweeperSkipsLiveLeaseStatus(t *testing.T) {
	registryRepo := &_SweeperRegistryRepo{
		leases: []core.AppHeartbeat{{Name: "demo.app", InstanceId: "instance-1"}},
		status: &core.AppStatus{
			Name:       "demo.app",
			InstanceId: "instance-1",
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
		leases: []core.AppHeartbeat{{Name: "demo.app", InstanceId: "instance-1"}},
		status: &core.AppStatus{
			Name:       "demo.app",
			InstanceId: "instance-1",
			ExpiresAt:  time.Now().Add(-time.Minute),
		},
		statusOK: true,
	}
	descriptorRepo := &_SweeperDescriptorRepo{views: []core.DomainDescriptorView{{
		DomainVersion: core.DomainDescriptorVersion{
			Descriptor: &skeldesc.Domain{
				Services: []*skeldesc.Service{{
					SkelName: "demo.Service",
					Hash:     "service-main",
					Audiences: []*skeldesc.ActorAudience{{
						SkelName: "demo.Actor",
					}}, AuthMode: skeldesc.AuthModeRequired,
				}}, Generated: &skeldesc.GeneratedInfo{CompilerVersion: "v99.0.0"},
			},
			Main: true,
		},
		Services: []core.DescriptorVersion[*skeldesc.Service]{{
			Descriptor: &skeldesc.Service{
				SkelName: "demo.Service",
				Hash:     "service-main", AuthMode: skeldesc.AuthModeRequired,
			},
			SkelName:       "demo.Service",
			DescriptorHash: "service-main",
			Main:           true,
		}},
	}}}
	portalSiteRepo := &_SweeperPortalSiteRepo{entries: []*core.PortalSite{{
		Id:            1,
		Name:          "demo-rpc",
		Type:          core.PortalSiteTypeRPCGW,
		ActorSkelName: "demo.Actor",
		RpcgwServices: []string{"demo.Service"},
		Enabled:       true,
	}}}
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

	assert.Equal(t, []string{"demo.app:instance-1"}, registryRepo.removedApp)
	assert.Equal(t, []string{"demo.app:instance-1"}, descriptorRepo.released)
	value, ok := watchServer.Get(watched.FormatPortalSiteKey("demo-rpc"))
	assert.True(t, ok)
	assert.JSONEq(t, `{
		"name": "demo-rpc",
		"type": "RPCGW",
		"actorVia": {
			"actorSkelName": "demo.Actor",
			"actorVia": ""
		},
		"cors": {
			"mode": "",
			"allowedOrigins": []
		},
		"rpcgwConfig": {
			"services": [
				{"skelName": "demo.Service"}
			]
		}
	}`, value)
}

type _SweeperPortalInstanceRepo struct {
	core.PortalInstanceRepo

	leases  []string
	removed []string
}

func (r *_SweeperPortalInstanceRepo) PopExpiredLeases() []string {
	leases := r.leases
	r.leases = nil
	return leases
}

func (r *_SweeperPortalInstanceRepo) Remove(instanceId string) {
	r.removed = append(r.removed, instanceId)
}

func TestSweeperRemovesExpiredPortalInstances(t *testing.T) {
	portalRepo := &_SweeperPortalInstanceRepo{leases: []string{"instance-1", "instance-2"}}
	target := &Sweeper{
		PortalInstanceCore: &core.PortalInstanceCore{PortalInstanceRepo: portalRepo},
		RegistryCore: &core.RegistryCore{
			RegistryRepo:   &_SweeperRegistryRepo{},
			DescriptorRepo: &_SweeperDescriptorRepo{},
		},
	}

	target.sweepExpiredLeases()

	assert.Equal(t, []string{"instance-1", "instance-2"}, portalRepo.removed)
}
