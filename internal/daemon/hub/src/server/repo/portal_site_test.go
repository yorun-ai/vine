package repo

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/infra/rdb"
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/configaccess"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/watchserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/repo/db/model"
	"go.yorun.ai/vine/util/vcode"
	"gorm.io/gorm"
)

var (
	testPortalSiteRepoDB     *gorm.DB
	testPortalSiteRepoDBOnce sync.Once
)

// _PortalSiteSchemaRepo supplies the registered schemas a portal site repository
// assembles Web mount paths and Rpc services from.
type _PortalSiteSchemaRepo struct {
	core.SchemaRepo
	webs          []*skel.WebSchema
	views         []core.DomainSchemaView
	configSchemas []*skel.ConfigSchema
	enumSchemas   []*skel.EnumSchema
}

func (r *_PortalSiteSchemaRepo) ListAppConfigSchemas() []*skel.ConfigSchema { return r.configSchemas }

func (r *_PortalSiteSchemaRepo) ListEnumSchemas() []*skel.EnumSchema { return r.enumSchemas }

func (r *_PortalSiteSchemaRepo) GetWebSchema(skelName string) *skel.WebSchema {
	for _, schema := range r.webs {
		if schema.SkelName == skelName {
			return schema
		}
	}
	return nil
}

func (r *_PortalSiteSchemaRepo) ListWebSchemas() []*skel.WebSchema { return r.webs }

func (r *_PortalSiteSchemaRepo) ListDomainSchemaViews() []core.DomainSchemaView { return r.views }

func TestPortalSiteRepoSaveCreate(t *testing.T) {
	_, repo, watchServer := newTestPortalSiteRepo(t)

	entry := testPortalSite("demo-entry")
	repo.Save(entry)

	got, ok := repo.GetById(entry.Id)
	require.True(t, ok)
	assert.Equal(t, entry, got)

	raw, ok := watchServer.Get(watched.FormatPortalSiteKey("demo-entry"))
	require.True(t, ok)
	assertWatchedPortalSite(t, entry, vcode.MustUnmarshalJsonS[*watched.PortalSite](raw))
}

func TestPortalSiteRepoSaveRename(t *testing.T) {
	_, repo, watchServer := newTestPortalSiteRepo(t)

	entry := testPortalSite("demo-entry")
	repo.Save(entry)
	entry.Name = "next-entry"
	repo.Save(entry)

	_, ok := watchServer.Get(watched.FormatPortalSiteKey("demo-entry"))
	assert.False(t, ok)

	raw, ok := watchServer.Get(watched.FormatPortalSiteKey("next-entry"))
	require.True(t, ok)
	assertWatchedPortalSite(t, entry, vcode.MustUnmarshalJsonS[*watched.PortalSite](raw))
}

func TestPortalSiteRepoRemove(t *testing.T) {
	_, repo, watchServer := newTestPortalSiteRepo(t)

	entry := testPortalSite("demo-entry")
	repo.Save(entry)
	assert.True(t, repo.Remove(entry.Id))

	got, ok := repo.GetById(entry.Id)
	assert.False(t, ok)
	assert.Nil(t, got)

	_, ok = watchServer.Get(watched.FormatPortalSiteKey("demo-entry"))
	assert.False(t, ok)
	assert.False(t, repo.Remove(entry.Id))
}

func newTestPortalSiteRepo(t *testing.T) (*gorm.DB, *PortalSiteRepo, *watchserver.Server) {
	t.Helper()

	db := sharedTestPortalSiteRepoDB(t)
	watchServer := watchserver.NewServerForTest()
	t.Cleanup(watchServer.AfterAppStop)

	repo := &PortalSiteRepo{
		Dao: &model.PortalSiteDao{
			Dao: rdb.NewDao[*model.PortalSite](db),
		},
		SchemaRepo: new(_PortalSiteSchemaRepo),
		Syncer:     testSyncer(watchServer),
		Access:     new(configaccess.Access),
	}
	repo.Dao.InitSchema()
	require.NoError(t, db.Exec("DELETE FROM portal_site").Error)

	return db, repo, watchServer
}

func sharedTestPortalSiteRepoDB(t *testing.T) *gorm.DB {
	t.Helper()

	testPortalSiteRepoDBOnce.Do(func() {
		root, err := os.MkdirTemp("", "vine-hub-portal-site-repo-*")
		require.NoError(t, err)
		db, err := gorm.Open(sqlite.Open(filepath.Join(root, "portal_site.sqlite")), &gorm.Config{})
		require.NoError(t, err)
		testPortalSiteRepoDB = db
	})
	return testPortalSiteRepoDB
}

func testPortalSite(name string) *core.PortalSite {
	return &core.PortalSite{
		Name:          name,
		Type:          core.PortalSiteTypeRPCGW,
		ActorSkelName: "demo.Actor",
		ActorVia:      "client",
		Cors: core.PortalCors{
			Mode:           core.PortalCorsModeStrict,
			AllowedOrigins: []string{"https://console.example.com"},
		},
		Enabled: true,
	}
}

func assertWatchedPortalSite(t *testing.T, expected *core.PortalSite, actual *watched.PortalSite) {
	t.Helper()

	require.NotNil(t, actual)
	assert.Equal(t, expected.Name, actual.Name)
	assert.Equal(t, string(expected.Type), actual.Type)
	assert.Equal(t, expected.ActorSkelName, actual.ActorVia.ActorSkelName)
	assert.Equal(t, expected.ActorVia, actual.ActorVia.ActorVia)
	assert.Equal(t, watched.PortalCorsMode(expected.Cors.Mode), actual.Cors.Mode)
	assert.Equal(t, expected.Cors.AllowedOrigins, actual.Cors.AllowedOrigins)
	require.NotNil(t, actual.RpcgwConfig)
	assert.Empty(t, actual.RpcgwConfig.Services)
}

func TestPortalSiteRepoRejectsReadOnlyWrites(t *testing.T) {
	access := new(configaccess.Access)
	access.Lock()
	repo := &PortalSiteRepo{Access: access}
	require.PanicsWithError(t, "Configuration is read-only; update the configuration source and restart Hub. type=APPLICATION code=PERMISSION_DENIED", func() { repo.Save(new(core.PortalSite)) })
	require.PanicsWithError(t, "Configuration is read-only; update the configuration source and restart Hub. type=APPLICATION code=PERMISSION_DENIED", func() { repo.Remove(1) })
}

func TestPortalSiteRepoAssemblesDerivedValues(t *testing.T) {
	_, repo, watchServer := newTestPortalSiteRepo(t)
	schemas := repo.SchemaRepo.(*_PortalSiteSchemaRepo)
	schemas.webs = []*skel.WebSchema{{Name: "Web", SkelName: "demo.Web", MountPath: "/demo"}}
	schemas.views = []core.DomainSchemaView{{
		DomainVersion: core.DomainSchemaVersion{
			Main: true,
			Schema: &skel.DomainSchema{
				Domain: "demo",
				Services: []*skel.ServiceSchema{{
					SkelName:  "demo.Service",
					Audiences: []*skel.ActorAudienceSchema{{SkelName: "demo.Actor", Via: skel.ActorViaClient}},
				}},
			},
		},
	}}

	web := testPortalSite("demo-web")
	web.Type, web.WebName = core.PortalSiteTypeWEBGW, "demo.Web"
	repo.Save(web)
	rpc := testPortalSite("demo-rpc")
	rpc.Type, rpc.ActorSkelName, rpc.ActorVia = core.PortalSiteTypeRPCGW, "demo.Actor", "client"
	repo.Save(rpc)

	// The repository reconstitutes aggregates: the saved entity and every read
	// path carry the Web mount path and the Rpc services of the site.
	assert.Equal(t, "/demo", web.WebMountPath)
	assert.Equal(t, []string{"demo.Service"}, rpc.RpcgwServices)

	got, ok := repo.GetByName("demo-web")
	require.True(t, ok)
	assert.Equal(t, "/demo", got.WebMountPath)

	entries := repo.List()
	require.Len(t, entries, 2)
	mountPaths := map[string]string{}
	rpcgwServices := map[string][]string{}
	for _, entry := range entries {
		mountPaths[entry.Name] = entry.WebMountPath
		rpcgwServices[entry.Name] = entry.RpcgwServices
	}
	assert.Equal(t, map[string]string{"demo-rpc": "", "demo-web": "/demo"}, mountPaths)
	assert.Equal(t, map[string][]string{"demo-rpc": {"demo.Service"}, "demo-web": {}}, rpcgwServices)

	raw, ok := watchServer.Get(watched.FormatPortalSiteKey("demo-rpc"))
	require.True(t, ok)
	watchedRpc := vcode.MustUnmarshalJsonS[*watched.PortalSite](raw)
	require.NotNil(t, watchedRpc.RpcgwConfig)
	assert.Equal(t, []watched.PortalRpcgwService{{SkelName: "demo.Service"}}, watchedRpc.RpcgwConfig.Services)
}

func TestPortalSiteRepoLeavesDerivedValuesEmptyWithoutSchemas(t *testing.T) {
	_, repo, _ := newTestPortalSiteRepo(t)

	entry := testPortalSite("demo-web")
	entry.Type, entry.WebName = core.PortalSiteTypeWEBGW, "demo.Web"
	repo.Save(entry)

	got, ok := repo.GetByName("demo-web")
	require.True(t, ok)
	assert.Empty(t, got.WebMountPath)
	assert.Empty(t, got.RpcgwServices)
}
