package embedded

import (
	"context"
	"crypto/tls"
	"testing"

	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/mtls"
	hubredis "go.yorun.ai/vine/internal/daemon/hub/api/redis"
	"go.yorun.ai/vine/internal/testutil/mtlstest"
)

func TestStoreUsesMTLSIdentityForRedisRole(t *testing.T) {
	ca := mtlstest.NewCA(t)
	hubIdentity := ca.Identity(t, mtls.HubIdentity)
	linkIdentity := ca.Identity(t, mtls.LinkIdentity)
	portalIdentity := ca.Identity(t, mtls.PortalIdentity)
	store := NewStore("127.0.0.1:0", false, "test-server-password", hubIdentity)
	store.Start()
	t.Cleanup(store.Stop)
	store.Set(hubredis.RevisionKey, "42")

	linkClient := newMTLSRedisClient(store.ListenAddr(), hubredis.LinkUsername, linkIdentity.ClientConfig(mtls.HubIdentity))
	t.Cleanup(func() { _ = linkClient.Close() })
	value, err := linkClient.Get(context.Background(), hubredis.RevisionKey).Result()
	require.NoError(t, err)
	assert.Equal(t, "42", value)

	impersonatingClient := newMTLSRedisClient(store.ListenAddr(), hubredis.LinkUsername, portalIdentity.ClientConfig(mtls.HubIdentity))
	t.Cleanup(func() { _ = impersonatingClient.Close() })
	_, err = impersonatingClient.Get(context.Background(), hubredis.RevisionKey).Result()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "username does not match mTLS client identity")
}

func newMTLSRedisClient(addr string, username string, tlsConfig *tls.Config) *goredis.Client {
	client := goredis.NewClient(&goredis.Options{
		Addr:            addr,
		Protocol:        2,
		DisableIdentity: true,
		TLSConfig:       tlsConfig,
		OnConnect: func(ctx context.Context, conn *goredis.Conn) error {
			return conn.AuthACL(ctx, username, "").Err()
		},
	})
	return client
}

func TestStoreZSetPopRangeByScoreReturnsSortedMembers(t *testing.T) {
	store := NewStore("", false, "test-server-password", nil)

	store.ZAdd("registry:leases", 30, "app-c:instance-3")
	store.ZAdd("registry:leases", 10, "app-b:instance-2")
	store.ZAdd("registry:leases", 10, "app-a:instance-1")
	store.ZAdd("registry:leases", 50, "app-d:instance-4")

	members := store.ZPopRangeByScore("registry:leases", 0, 30, 0)

	assert.Equal(t, []string{
		"app-a:instance-1",
		"app-b:instance-2",
		"app-c:instance-3",
	}, members)
}

func TestStoreZSetUpdatesScoreAndLimits(t *testing.T) {
	store := NewStore("", false, "test-server-password", nil)

	store.ZAdd("registry:leases", 30, "app-a:instance-1")
	store.ZAdd("registry:leases", 5, "app-a:instance-1")
	store.ZAdd("registry:leases", 10, "app-b:instance-2")

	members := store.ZPopRangeByScore("registry:leases", 0, 30, 1)

	assert.Equal(t, []string{"app-a:instance-1"}, members)
}

func TestStoreZSetRemovesMembers(t *testing.T) {
	store := NewStore("", false, "test-server-password", nil)

	store.ZAdd("registry:leases", 10, "app-a:instance-1")

	assert.True(t, store.ZRem("registry:leases", "app-a:instance-1"))
	assert.False(t, store.ZRem("registry:leases", "app-a:instance-1"))
	assert.Empty(t, store.ZPopRangeByScore("registry:leases", 0, 30, 0))
}

func TestStoreZSetPopsRangeByScore(t *testing.T) {
	store := NewStore("", false, "test-server-password", nil)

	store.ZAdd("registry:leases", 30, "app-c:instance-3")
	store.ZAdd("registry:leases", 10, "app-b:instance-2")
	store.ZAdd("registry:leases", 10, "app-a:instance-1")
	store.ZAdd("registry:leases", 50, "app-d:instance-4")

	members := store.ZPopRangeByScore("registry:leases", 0, 30, 2)

	assert.Equal(t, []string{"app-a:instance-1", "app-b:instance-2"}, members)
	assert.Equal(t, []string{"app-c:instance-3", "app-d:instance-4"}, store.ZPopRangeByScore("registry:leases", 0, 100, 0))
}
