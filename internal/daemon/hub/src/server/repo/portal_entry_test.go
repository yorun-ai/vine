package repo

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/infra/rdb"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/configaccess"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/watchserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/repo/db/model"
	"go.yorun.ai/vine/util/vcode"
	"gorm.io/gorm"
)

func TestPortalEntryRepoStoresAccessAndName(t *testing.T) {
	repo := newTestPortalEntryRepoDB(t)

	entry := &core.PortalEntry{Name: "web", Protocol: "http", Http: &core.PortalEntryHTTP{Host: "demo.local", Http: "DISABLED", Https: "ENABLED", HttpsPort: 8443}}
	repo.Save(entry)
	require.NotZero(t, entry.Id)

	got, ok := repo.GetById(entry.Id)
	require.True(t, ok)
	assert.Equal(t, entry.Id, got.Id)
	// An entry keeps the name it was stored with.
	assert.Equal(t, "web", got.Name)
	byName, ok := repo.GetByName("web")
	require.True(t, ok)
	assert.Equal(t, entry.Id, byName.Id)
	_, ok = repo.GetByName("other")
	assert.False(t, ok)

	_, ok = repo.GetById(entry.Id)
	assert.True(t, ok)
}

func TestPortalEntryRepoRejectsReadOnlyWrites(t *testing.T) {
	access := new(configaccess.Access)
	access.Lock()
	repo := &PortalEntryRepo{Access: access}

	require.PanicsWithError(t, "Configuration is read-only; update the configuration source and restart Hub. type=APPLICATION code=PERMISSION_DENIED", func() {
		repo.Save(new(core.PortalEntry))
	})
	require.PanicsWithError(t, "Configuration is read-only; update the configuration source and restart Hub. type=APPLICATION code=PERMISSION_DENIED", func() {
		repo.Remove(1)
	})
}

func newTestPortalEntryRepoDB(t *testing.T) *PortalEntryRepo {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "portal-entry.sqlite")), &gorm.Config{})
	require.NoError(t, err)
	connection, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = connection.Close() })
	watchServer := watchserver.NewServerForTest()
	t.Cleanup(watchServer.AfterAppStop)
	repo := &PortalEntryRepo{
		CertDao: &model.PortalCertDao{Dao: rdb.NewDao[*model.PortalCert](db)},
		Dao:     &model.PortalEntryDao{Dao: rdb.NewDao[*model.PortalEntry](db)},
		Syncer:  testSyncer(watchServer),
		Access:  new(configaccess.Access),
	}
	repo.Dao.EnsureSchema()
	repo.CertDao.EnsureSchema()
	return repo
}

func TestPortalEntryRepoListenIPsRoundTrip(t *testing.T) {
	repo := newTestPortalEntryRepoDB(t)
	entry := &core.PortalEntry{Name: "local", ListenIPs: []string{"127.0.0.1", "::1"}, Protocol: "http", Http: &core.PortalEntryHTTP{Http: "ENABLED", Https: "DISABLED", HttpPort: 8080}}
	repo.Save(entry)
	got, ok := repo.GetById(entry.Id)
	require.True(t, ok)
	assert.Equal(t, entry.ListenIPs, got.ListenIPs)
	got.ListenIPs[0] = "127.0.0.2"
	again, _ := repo.GetById(entry.Id)
	assert.Equal(t, "127.0.0.1", again.ListenIPs[0])
	entry.ListenIPs = []string{}
	repo.Save(entry)
	got, _ = repo.GetById(entry.Id)
	assert.Empty(t, got.ListenIPs)
}

func TestPortalEntryRepoStoresHTTPHost(t *testing.T) {
	repo := newTestPortalEntryRepoDB(t)
	row := &model.PortalEntry{
		Name: "legacy", Protocol: "http", ListenIPs: "[]", Enabled: true,
		HTTPConfig: `{"host":"demo.local","http":"ENABLED","httpPort":80,"https":"ENABLED","httpsPort":443,"autohttps":"ENABLED"}`,
	}
	repo.Dao.Save(row)
	entry, ok := repo.GetById(row.Id)
	require.True(t, ok)
	require.Equal(t, "demo.local", entry.Http.Host)
	entry.Http.Host = "new.local"
	repo.Save(entry)
	got, ok := repo.GetById(row.Id)
	require.True(t, ok)
	require.Equal(t, "new.local", got.Http.Host)
}

func TestPortalEntryRepoPersistsNormalizedHost(t *testing.T) {
	repo := newTestPortalEntryRepoDB(t)
	for _, host := range []string{"APP.EXAMPLE.COM.", "*.EXAMPLE.COM."} {
		entry := new(core.PortalEntry{Name: host, Protocol: "http", Http: new(core.PortalEntryHTTP{Host: " " + host + " "})})
		repo.Save(entry)
		row, ok := repo.Dao.ById(entry.Id)
		require.True(t, ok)
		stored := vcode.MustUnmarshalJsonS[core.PortalEntryHTTP](row.HTTPConfig)
		want := "app.example.com"
		if host[0] == '*' {
			want = "*.example.com"
		}
		require.Equal(t, want, stored.Host)
		loaded, ok := repo.GetByName(host)
		require.True(t, ok)
		require.Equal(t, want, loaded.Http.Host)
		require.Equal(t, want, loaded.EffectiveHTTP.Host)
	}
}

func TestPortalEntryRepoPersistsIntentAndEnrichesEffectiveHTTP(t *testing.T) {
	repo := newTestPortalEntryRepoDB(t)
	entry := &core.PortalEntry{Name: "web", Protocol: "http", Enabled: true, Http: &core.PortalEntryHTTP{Host: "app.example.com"}}
	repo.Save(entry)
	require.EqualValues(t, "AUTO", entry.Http.Https)
	require.False(t, entry.EffectiveHTTP.HttpsEnabled)
	// Certificate metadata is normally parsed by PortalCertCore before persistence.
	cert := &model.PortalCert{Name: "wildcard", Domains: `["*.example.com"]`, Enabled: true}
	repo.CertDao.Save(cert)
	loaded, ok := repo.GetById(entry.Id)
	require.True(t, ok)
	require.EqualValues(t, "AUTO", loaded.Http.Https)
	require.True(t, loaded.EffectiveHTTP.HttpsEnabled)
	require.True(t, loaded.EffectiveHTTP.AutoHTTPSEnabled)
	cert.Enabled = false
	repo.CertDao.Save(cert)
	require.False(t, repo.List()[0].EffectiveHTTP.HttpsEnabled)
	loaded.Http.Https = "DISABLED"
	repo.Save(loaded)
	loaded, ok = repo.GetByName("web")
	require.True(t, ok)
	require.EqualValues(t, "DISABLED", loaded.Http.Https)
	loaded.Http.Https = "AUTO"
	repo.Save(loaded)
	loaded, ok = repo.GetById(entry.Id)
	require.True(t, ok)
	require.EqualValues(t, "AUTO", loaded.Http.Https)
}

func TestPortalEntryRepoRejectsInvalidStoredHTTPFields(t *testing.T) {
	repo := newTestPortalEntryRepoDB(t)
	for _, config := range []string{
		`{"host":"demo.local","httpEnabled":false}`,
		`{"host":"demo.local","httpsEnabled":true}`,
		`{"host":"demo.local","autoHTTPS":false}`,
		`{"host":"demo.local","typo":"AUTO"}`,
		`null`,
	} {
		row := new(model.PortalEntry{Name: config, Protocol: "http", HTTPConfig: config, ListenIPs: "[]", Enabled: true})
		repo.Dao.Save(row)
		require.Panics(t, func() { repo.GetById(row.Id) }, config)
	}
}
