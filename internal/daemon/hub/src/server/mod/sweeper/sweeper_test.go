package sweeper

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/redisserver"
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

type _SweeperSchemaRepo struct {
	core.SchemaRepo
	released []string
	views    []core.DomainSchemaView
}

func (r *_SweeperSchemaRepo) ReleaseDomainSchemas(ownerName string, ownerId string) {
	r.released = append(r.released, ownerName+":"+ownerId)
}

func (*_SweeperSchemaRepo) SaveDomainSchemasJSON(string, string, []skel.JSON) {}

func (r *_SweeperSchemaRepo) ListDomainSchemaViews() []core.DomainSchemaView {
	return r.views
}

type _SweeperPortalSiteRepo struct {
	core.PortalSiteRepo
	entries []core.PortalSite
}

func (r *_SweeperPortalSiteRepo) ListEntries() []core.PortalSite {
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
	schemaRepo := &_SweeperSchemaRepo{}
	target := &Sweeper{
		RegistryRepo: registryRepo,
		RegistryCore: &core.RegistryCore{
			RegistryRepo: registryRepo,
			SchemaRepo:   schemaRepo,
		},
	}

	target.sweepExpiredLeases()

	assert.Empty(t, registryRepo.removedApp)
	assert.Empty(t, schemaRepo.released)
}

func TestSweeperUnregistersExpiredLeaseStatus(t *testing.T) {
	redisServer := redisserver.NewServerForTest()
	defer redisServer.AfterAppStop()
	syncerModule := &syncer.Syncer{RedisServer: redisServer}
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
	schemaRepo := &_SweeperSchemaRepo{views: []core.DomainSchemaView{{
		DomainVersion: core.DomainSchemaVersion{
			Schema: &skel.DomainSchema{
				Services: []*skel.ServiceSchema{{
					SkelName: "demo.Service",
					Hash:     "service-main",
					Audiences: []*skel.ActorAudienceSchema{{
						SkelName: "demo.Actor",
					}},
				}},
			},
			Main: true,
		},
		Services: []core.SchemaVersion[*skel.ServiceSchema]{{
			Schema: &skel.ServiceSchema{
				SkelName: "demo.Service",
				Hash:     "service-main",
			},
			SkelName:   "demo.Service",
			SchemaHash: "service-main",
			Main:       true,
		}},
	}}}
	portalSiteRepo := &_SweeperPortalSiteRepo{entries: []core.PortalSite{{
		Id:            1,
		Name:          "demo-rpc",
		Type:          core.PortalSiteTypeRPCGW,
		ActorSkelName: "demo.Actor",
	}}}
	target := &Sweeper{
		RegistryRepo:   registryRepo,
		SchemaRepo:     schemaRepo,
		PortalSiteRepo: portalSiteRepo,
		Syncer:         syncerModule,
		RegistryCore: &core.RegistryCore{
			RegistryRepo: registryRepo,
			SchemaRepo:   schemaRepo,
		},
	}

	target.sweepExpiredLeases()

	assert.Equal(t, []string{"demo.app:instance-1"}, registryRepo.removedApp)
	assert.Equal(t, []string{"demo.app:instance-1"}, schemaRepo.released)
	value, ok := redisServer.Get(redised.FormatPortalSiteKey("demo-rpc"))
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
