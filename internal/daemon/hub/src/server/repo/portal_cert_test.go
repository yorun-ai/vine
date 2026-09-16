package repo

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/infra/rdb"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/configaccess"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/watchserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/mod/syncer"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/repo/db/model"
	"go.yorun.ai/vine/util/vcode"
	"gorm.io/gorm"
)

var (
	testPortalCertRepoDB     *gorm.DB
	testPortalCertRepoDBOnce sync.Once
)

func TestPortalCertRepoSaveCertCreate(t *testing.T) {
	_, repo, watchServer := newTestPortalCertRepo(t)

	cert := testPortalCert("demo-cert")
	repo.Save(cert)

	got, ok := repo.GetById(cert.Id)
	require.True(t, ok)
	assert.Equal(t, cert, got)

	key := watched.FormatPortalCertKey("demo-cert")
	raw, ok := watchServer.Get(key)
	require.True(t, ok)
	assert.Equal(t, syncer.ToWatchedPortalCert(cert), vcode.MustUnmarshalJsonS[*watched.PortalCert](raw))
}

func TestPortalCertRepoSaveCertUpdate(t *testing.T) {
	_, repo, _ := newTestPortalCertRepo(t)

	cert := testPortalCert("demo-cert")
	repo.Save(cert)
	cert.Issuer = "manual"
	cert.Domains = []string{"next.local"}
	repo.Save(cert)

	got, ok := repo.GetById(cert.Id)
	require.True(t, ok)
	assert.Equal(t, "manual", got.Issuer)
	assert.Equal(t, []string{"next.local"}, got.Domains)
}

func TestPortalCertRepoSaveCertRename(t *testing.T) {
	_, repo, watchServer := newTestPortalCertRepo(t)

	cert := testPortalCert("demo-cert")
	repo.Save(cert)
	cert.Name = "next-cert"
	repo.Save(cert)

	got, ok := repo.GetById(cert.Id)
	require.True(t, ok)
	assert.Equal(t, "next-cert", got.Name)

	_, ok = watchServer.Get(watched.FormatPortalCertKey("demo-cert"))
	assert.False(t, ok)

	raw, ok := watchServer.Get(watched.FormatPortalCertKey("next-cert"))
	require.True(t, ok)
	assert.Equal(t, syncer.ToWatchedPortalCert(cert), vcode.MustUnmarshalJsonS[*watched.PortalCert](raw))
}

func TestPortalCertRepoRemoveCert(t *testing.T) {
	_, repo, watchServer := newTestPortalCertRepo(t)

	cert := testPortalCert("demo-cert")
	repo.Save(cert)
	assert.True(t, repo.Remove(cert.Id))

	got, ok := repo.GetById(cert.Id)
	assert.False(t, ok)
	assert.Nil(t, got)

	key := watched.FormatPortalCertKey("demo-cert")
	_, ok = watchServer.Get(key)
	assert.False(t, ok)
	assert.False(t, repo.Remove(cert.Id))
}

func newTestPortalCertRepo(t *testing.T) (*gorm.DB, *PortalCertRepo, *watchserver.Server) {
	t.Helper()

	db := sharedTestPortalCertRepoDB(t)
	watchServer := watchserver.NewServerForTest()
	t.Cleanup(watchServer.AfterAppStop)

	repo := &PortalCertRepo{
		Dao: &model.PortalCertDao{
			Dao: rdb.NewDao[*model.PortalCert](db),
		},
		Syncer: testSyncer(watchServer),
		Access: new(configaccess.Access),
	}
	repo.Dao.InitSchema()
	require.NoError(t, db.Exec("DELETE FROM portal_cert").Error)

	return db, repo, watchServer
}

func sharedTestPortalCertRepoDB(t *testing.T) *gorm.DB {
	t.Helper()

	testPortalCertRepoDBOnce.Do(func() {
		root, err := os.MkdirTemp("", "vine-hub-portal-cert-repo-*")
		require.NoError(t, err)
		db, err := gorm.Open(sqlite.Open(filepath.Join(root, "portal_cert.sqlite")), &gorm.Config{})
		require.NoError(t, err)
		testPortalCertRepoDB = db
	})
	return testPortalCertRepoDB
}

func testPortalCert(name string) *core.PortalCert {
	return &core.PortalCert{
		Name:             name,
		Issuer:           "letsencrypt",
		Domains:          []string{"demo.local", "*.demo.local"},
		PublicKeyBase64:  "pub",
		PrivateKeyBase64: "pri",
		ValidFrom:        time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		ValidTo:          time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

func TestPortalCertRepoRejectsReadOnlyWrites(t *testing.T) {
	access := new(configaccess.Access)
	access.Lock()
	repo := &PortalCertRepo{Access: access}
	require.PanicsWithError(t, "Configuration is read-only; update the configuration source and restart Hub. type=APPLICATION code=PERMISSION_DENIED", func() { repo.Save(new(core.PortalCert)) })
	require.PanicsWithError(t, "Configuration is read-only; update the configuration source and restart Hub. type=APPLICATION code=PERMISSION_DENIED", func() { repo.Remove(1) })
}
