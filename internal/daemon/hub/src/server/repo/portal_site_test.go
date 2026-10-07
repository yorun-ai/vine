package repo

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	skeldesc "go.yorun.ai/skel/descriptor"
	"go.yorun.ai/vine/infra/rdb"
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

// _PortalSiteDescriptorRepo supplies the registered descriptors a portal site repository
// assembles Web mount paths and Rpc services from.
type _PortalSiteDescriptorRepo struct {
	core.DescriptorRepo
	webs              []*skeldesc.Web
	views             []core.DomainDescriptorView
	configDescriptors []*skeldesc.Config
	enumDescriptors   []*skeldesc.Enum
	dataDescriptors   []*skeldesc.Data
}

func (r *_PortalSiteDescriptorRepo) ListAppConfigDescriptors() []*skeldesc.Config {
	return r.configDescriptors
}

func (r *_PortalSiteDescriptorRepo) ListEnumDescriptors() []*skeldesc.Enum {
	return r.enumDescriptors
}

func (r *_PortalSiteDescriptorRepo) GetWebDescriptor(skelName string) *skeldesc.Web {
	for _, descriptor := range r.webs {
		if descriptor.SkelName == skelName {
			return descriptor
		}
	}
	return nil
}

func (r *_PortalSiteDescriptorRepo) ListWebDescriptors() []*skeldesc.Web {
	return r.webs
}

func (r *_PortalSiteDescriptorRepo) ListDomainDescriptorViews() []core.DomainDescriptorView {
	return r.views
}

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
		DescriptorRepo: new(_PortalSiteDescriptorRepo),
		Syncer:         testSyncer(watchServer),
		Access:         new(configaccess.Access),
	}
	repo.Dao.EnsureSchema()
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
	descriptors := repo.DescriptorRepo.(*_PortalSiteDescriptorRepo)
	descriptors.webs = []*skeldesc.Web{{Name: "Web", SkelName: "demo.Web", MountPath: "/demo", AuthMode: skeldesc.AuthModeRequired}}
	descriptors.views = []core.DomainDescriptorView{{
		DomainVersion: core.DomainDescriptorVersion{
			Main: true,
			Descriptor: &skeldesc.Domain{
				Name: "demo",
				Services: []*skeldesc.Service{{
					SkelName:  "demo.Service",
					Audiences: []*skeldesc.ActorAudience{{SkelName: "demo.Actor", Via: skeldesc.ActorViaClient}}, AuthMode: skeldesc.AuthModeRequired,
				}}, Generated: &skeldesc.GeneratedInfo{CompilerVersion: "v99.0.0"},
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

func TestPortalSiteRepoLeavesDerivedValuesEmptyWithoutDescriptors(t *testing.T) {
	_, repo, _ := newTestPortalSiteRepo(t)

	entry := testPortalSite("demo-web")
	entry.Type, entry.WebName = core.PortalSiteTypeWEBGW, "demo.Web"
	repo.Save(entry)

	got, ok := repo.GetByName("demo-web")
	require.True(t, ok)
	assert.Empty(t, got.WebMountPath)
	assert.Empty(t, got.RpcgwServices)
}

func (r *_PortalSiteDescriptorRepo) ListAppConfigTypeDescriptors() ([]*skeldesc.Config, []*skeldesc.Enum, []*skeldesc.Data) {
	return r.ListAppConfigDescriptors(), r.ListEnumDescriptors(), r.dataDescriptors
}
