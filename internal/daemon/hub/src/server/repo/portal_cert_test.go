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
	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/redisserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/mod/syncer"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/repo/db/model"
	"go.yorun.ai/vine/internal/infra/rdb"
	"go.yorun.ai/vine/util/vcode"
	"gorm.io/gorm"
)

var (
	testPortalCertRepoDB     *gorm.DB
	testPortalCertRepoDBOnce sync.Once
)

func TestDBPortalCertRepoSaveCertCreate(t *testing.T) {
	_, repo, redisServer := newTestDBPortalCertRepo(t)

	cert := testPortalCert("demo-cert")
	repo.SaveCert(cert)

	got, ok := repo.GetCertById(cert.Id)
	require.True(t, ok)
	assert.Equal(t, cert, got)

	key := redised.FormatPortalCertKey("demo-cert")
	raw, ok := redisServer.Get(key)
	require.True(t, ok)
	assert.Equal(t, syncer.ToRedisedPortalCert(cert), vcode.MustUnmarshalJsonS[*redised.PortalCert](raw))
}

func TestDBPortalCertRepoSaveCertUpdate(t *testing.T) {
	_, repo, _ := newTestDBPortalCertRepo(t)

	cert := testPortalCert("demo-cert")
	repo.SaveCert(cert)
	cert.Issuer = "manual"
	cert.Domains = []string{"next.local"}
	repo.SaveCert(cert)

	got, ok := repo.GetCertById(cert.Id)
	require.True(t, ok)
	assert.Equal(t, "manual", got.Issuer)
	assert.Equal(t, []string{"next.local"}, got.Domains)
}

func TestDBPortalCertRepoSaveCertRename(t *testing.T) {
	_, repo, redisServer := newTestDBPortalCertRepo(t)

	cert := testPortalCert("demo-cert")
	repo.SaveCert(cert)
	cert.Name = "next-cert"
	repo.SaveCert(cert)

	got, ok := repo.GetCertById(cert.Id)
	require.True(t, ok)
	assert.Equal(t, "next-cert", got.Name)

	_, ok = redisServer.Get(redised.FormatPortalCertKey("demo-cert"))
	assert.False(t, ok)

	raw, ok := redisServer.Get(redised.FormatPortalCertKey("next-cert"))
	require.True(t, ok)
	assert.Equal(t, syncer.ToRedisedPortalCert(cert), vcode.MustUnmarshalJsonS[*redised.PortalCert](raw))
}

func TestDBPortalCertRepoRemoveCert(t *testing.T) {
	_, repo, redisServer := newTestDBPortalCertRepo(t)

	cert := testPortalCert("demo-cert")
	repo.SaveCert(cert)
	assert.True(t, repo.RemoveCert(cert.Id))

	got, ok := repo.GetCertById(cert.Id)
	assert.False(t, ok)
	assert.Nil(t, got)

	key := redised.FormatPortalCertKey("demo-cert")
	_, ok = redisServer.Get(key)
	assert.False(t, ok)
	assert.False(t, repo.RemoveCert(cert.Id))
}

func newTestDBPortalCertRepo(t *testing.T) (*gorm.DB, *DBPortalCertRepo, *redisserver.Server) {
	t.Helper()

	db := sharedTestPortalCertRepoDB(t)
	redisServer := redisserver.NewServerForTest()
	t.Cleanup(redisServer.AfterAppStop)

	repo := &DBPortalCertRepo{
		Dao: &model.PortalCertDao{
			Dao: rdb.NewDao[*model.PortalCert](db),
		},
		Syncer: testSyncer(redisServer),
	}
	repo.Dao.DIInit()
	require.NoError(t, db.Exec("DELETE FROM portal_cert").Error)

	return db, repo, redisServer
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
