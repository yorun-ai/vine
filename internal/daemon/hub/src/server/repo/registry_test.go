package repo

import (
	"context"
	"encoding/json"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	internalapp "go.yorun.ai/vine/internal/app"
	hubredis "go.yorun.ai/vine/internal/daemon/hub/api/redis"
	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/redisserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
	"go.yorun.ai/vine/util/vcode"
)

func newTestRegistryRepo(t *testing.T, inproc bool) (*RedisRegistryRepo, *redisserver.Server) {
	t.Helper()

	redisServer := &redisserver.Server{
		Context:    t.Context(),
		Option:     &flag.Flag{},
		InprocFlag: &internalapp.InternalInprocFlag{Enabled: true},
	}
	redisServer.DIInit()
	repo := &RedisRegistryRepo{
		RedisServer: redisServer,
		InprocFlag: &internalapp.InternalInprocFlag{
			Enabled: inproc,
		},
	}

	t.Cleanup(func() {
		redisServer.AfterAppStop()
	})

	return repo, redisServer
}

func formatTestRedisListPattern(prefix string) string {
	return strings.TrimSuffix(prefix, ":") + ":*"
}

func TestRegistryRepoSaveAndGetAppStatus(t *testing.T) {
	repo, redisServer := newTestRegistryRepo(t, false)
	now := time.Now().Round(0)
	setTimeNowForTest(t, func() time.Time { return now })

	status := &core.AppStatus{
		InstanceId:      "instance-1",
		Name:            "demo.app",
		Version:         "1.2.3",
		Endpoint:        "http://127.0.0.1:23001/rpc",
		ServiceHandlers: []core.ServiceHandlerRegistration{{ServiceSkelName: "svc.alpha"}, {ServiceSkelName: "svc.beta"}},
		WebHandlers:     []core.WebHandlerRegistration{{WebSkelName: "default@demo.app"}},
	}

	repo.SaveAppStatus(status)

	key := redised.FormatAppStatusKey(status.Name, status.InstanceId)
	raw, ok := redisServer.Get(key)
	assert.True(t, ok)
	gotRaw := vcode.MustUnmarshalJsonS[*_AppStatus](raw)
	assert.Equal(t, &_AppStatus{
		InstanceId:      status.InstanceId,
		Name:            status.Name,
		Version:         status.Version,
		Endpoint:        status.Endpoint,
		ExpiresAt:       now.Add(hubRegistryLeaseTTL),
		ServiceHandlers: status.ServiceHandlers,
		WebHandlers:     status.WebHandlers,
	}, gotRaw)

	got, ok := repo.GetAppStatus(status.Name, status.InstanceId)
	assert.True(t, ok)
	status.ExpiresAt = now.Add(hubRegistryLeaseTTL)
	assert.Equal(t, status, got)

	ttl := redisServer.TTL(key)
	assert.True(t, ttl > 0)
	assert.True(t, ttl <= int(hubRegistryEphemeralTTL/time.Second))
}

func TestRegistryRepoGetAppStatusReturnsFalseWhenMissing(t *testing.T) {
	repo, _ := newTestRegistryRepo(t, false)

	got, ok := repo.GetAppStatus("", "missing")
	assert.False(t, ok)
	assert.Nil(t, got)
}

func TestRegistryRepoListAppStatuses(t *testing.T) {
	repo, _ := newTestRegistryRepo(t, false)

	repo.SaveAppStatus(&core.AppStatus{
		InstanceId: "instance-2",
		Name:       "demo.beta",
		Version:    "2.0.0",
	})
	repo.SaveAppStatus(&core.AppStatus{
		InstanceId: "instance-1",
		Name:       "demo.alpha",
		Version:    "1.0.0",
	})

	items := repo.ListAppStatuses()

	assert.Len(t, items, 2)
	assert.Equal(t, "demo.alpha", items[0].Name)
	assert.Equal(t, "instance-1", items[0].InstanceId)
	assert.Equal(t, "1.0.0", items[0].Version)
	assert.Equal(t, "demo.beta", items[1].Name)
}

func TestRegistryRepoKeepAndRemoveAppStatus(t *testing.T) {
	repo, redisServer := newTestRegistryRepo(t, false)

	status := &core.AppStatus{InstanceId: "instance-1", Name: "demo.app"}
	repo.SaveAppStatus(status)

	key := redised.FormatAppStatusKey(status.Name, status.InstanceId)
	assert.True(t, redisServer.Expire(key, 1))
	repo.KeepAppStatus(status.Name, status.InstanceId)

	got, ok := repo.GetAppStatus(status.Name, status.InstanceId)
	assert.True(t, ok)
	assert.Equal(t, status.InstanceId, got.InstanceId)

	ttlAfter := redisServer.TTL(key)
	assert.True(t, ttlAfter > 1)

	repo.RemoveAppStatus(status.Name, status.InstanceId)

	got, ok = repo.GetAppStatus(status.Name, status.InstanceId)
	assert.False(t, ok)
	assert.Nil(t, got)
}

func TestRegistryRepoPopExpiredAppLeases(t *testing.T) {
	repo, _ := newTestRegistryRepo(t, false)
	now := time.Now().Round(0)
	setTimeNowForTest(t, func() time.Time { return now })
	status := &core.AppStatus{InstanceId: "instance-1", Name: "demo.app"}
	repo.SaveAppStatus(status)

	now = now.Add(hubRegistryLeaseTTL - time.Second)
	assert.Empty(t, repo.PopExpiredAppLeases())

	now = now.Add(2 * time.Second)
	leases := repo.PopExpiredAppLeases()

	assert.Equal(t, []core.AppHeartbeat{{Name: "demo.app", InstanceId: "instance-1"}}, leases)
	assert.Empty(t, repo.PopExpiredAppLeases())
}

func TestRegistryRepoKeepAppStatusRefreshesAppLease(t *testing.T) {
	repo, _ := newTestRegistryRepo(t, false)
	now := time.Now().Round(0)
	setTimeNowForTest(t, func() time.Time { return now })
	status := &core.AppStatus{InstanceId: "instance-1", Name: "demo.app"}
	repo.SaveAppStatus(status)

	now = now.Add(hubRegistryLeaseTTL / 2)
	repo.KeepAppStatus(status.Name, status.InstanceId)

	now = now.Add(hubRegistryLeaseTTL - time.Second)
	assert.Empty(t, repo.PopExpiredAppLeases())
}

func TestRegistryRepoPopExpiredAppLeasesSkipsRenewedStatus(t *testing.T) {
	repo, redisServer := newTestRegistryRepo(t, false)
	now := time.Now().Round(0)
	setTimeNowForTest(t, func() time.Time { return now })
	status := &core.AppStatus{InstanceId: "instance-1", Name: "demo.app"}
	repo.SaveAppStatus(status)

	key := redised.FormatAppStatusKey(status.Name, status.InstanceId)
	raw, ok := repo.getAppStatus(status.Name, status.InstanceId)
	assert.True(t, ok)
	raw.ExpiresAt = now.Add(2 * hubRegistryLeaseTTL)
	redisServer.SetEphemeral(key, vcode.MustMarshalJsonS(raw), hubRegistryEphemeralTTL)

	now = now.Add(hubRegistryLeaseTTL + time.Second)

	assert.Empty(t, repo.PopExpiredAppLeases())
}

func TestRegistryRepoInprocDoesNotSaveAppLease(t *testing.T) {
	repo, _ := newTestRegistryRepo(t, true)

	repo.SaveAppStatus(&core.AppStatus{InstanceId: "instance-1", Name: "demo.app"})

	assert.Empty(t, repo.PopExpiredAppLeases())
}

func TestRegistryRepoSaveAndGetRpcServiceRegistration(t *testing.T) {
	repo, redisServer := newTestRegistryRepo(t, false)

	registration := &core.RpcServiceRegistration{
		Endpoint:      "http://127.0.0.1:23001/rpc",
		ServiceName:   "svc.alpha",
		AppName:       "demo.app",
		AppVersion:    "1.2.3",
		AppInstanceId: "instance-1",
	}

	repo.SaveRpcServiceRegistration(registration)

	key := redised.FormatRpcServiceRegistrationKey(registration.ServiceName, registration.AppName, registration.AppInstanceId)
	raw, ok := redisServer.Get(key)
	assert.True(t, ok)
	gotRaw := vcode.MustUnmarshalJsonS[*redised.RpcServiceRegistration](raw)
	assert.Equal(t, toRpcServiceRegistration(registration), gotRaw)

	got, ok := repo.GetRpcServiceRegistration(registration.ServiceName, registration.AppName, registration.AppInstanceId)
	assert.True(t, ok)
	assert.Equal(t, registration, got)

	ttl := redisServer.TTL(key)
	assert.True(t, ttl > 0)
	assert.True(t, ttl <= int(hubRegistryEphemeralTTL/time.Second))
}

func TestRegistryRepoSaveAndGetWebRegistration(t *testing.T) {
	repo, redisServer := newTestRegistryRepo(t, false)

	registration := &core.WebRegistration{
		Endpoint:      "http://127.0.0.1:23001",
		WebSkelName:   "default@demo.app",
		AppName:       "demo.app",
		AppVersion:    "1.2.3",
		AppInstanceId: "instance-1",
	}

	repo.SaveWebRegistration(registration)

	key := redised.FormatWebRegistrationKey(registration.WebSkelName, registration.AppName, registration.AppInstanceId)
	raw, ok := redisServer.Get(key)
	assert.True(t, ok)
	gotRaw := vcode.MustUnmarshalJsonS[*redised.WebRegistration](raw)
	assert.Equal(t, toWebRegistration(registration), gotRaw)

	got, ok := repo.GetWebRegistration(registration.WebSkelName, registration.AppName, registration.AppInstanceId)
	assert.True(t, ok)
	assert.Equal(t, registration, got)

	ttl := redisServer.TTL(key)
	assert.True(t, ttl > 0)
	assert.True(t, ttl <= int(hubRegistryEphemeralTTL/time.Second))
}

func TestRegistryRepoGetRpcServiceRegistrationReturnsFalseWhenMissing(t *testing.T) {
	repo, _ := newTestRegistryRepo(t, false)

	got, ok := repo.GetRpcServiceRegistration("svc.alpha", "demo.app", "missing")
	assert.False(t, ok)
	assert.Nil(t, got)
}

func TestRegistryRepoGetWebRegistrationReturnsFalseWhenMissing(t *testing.T) {
	repo, _ := newTestRegistryRepo(t, false)

	got, ok := repo.GetWebRegistration("default@demo.app", "demo.app", "missing")
	assert.False(t, ok)
	assert.Nil(t, got)
}

func TestRegistryRepoKeepAndRemoveRpcServiceRegistration(t *testing.T) {
	repo, redisServer := newTestRegistryRepo(t, false)

	registration := &core.RpcServiceRegistration{
		Endpoint:      "http://127.0.0.1:23001/rpc",
		ServiceName:   "svc.alpha",
		AppName:       "demo.app",
		AppVersion:    "1.2.3",
		AppInstanceId: "instance-1",
	}
	repo.SaveRpcServiceRegistration(registration)

	key := redised.FormatRpcServiceRegistrationKey(registration.ServiceName, registration.AppName, registration.AppInstanceId)

	assert.True(t, redisServer.Expire(key, 1))
	repo.KeepRpcServiceRegistration(registration.ServiceName, registration.AppName, registration.AppInstanceId)

	got, ok := repo.GetRpcServiceRegistration(registration.ServiceName, registration.AppName, registration.AppInstanceId)
	assert.True(t, ok)
	assert.Equal(t, registration, got)

	ttlAfter := redisServer.TTL(key)
	assert.True(t, ttlAfter > 1)

	repo.RemoveRpcServiceRegistration(registration.ServiceName, registration.AppName, registration.AppInstanceId)

	got, ok = repo.GetRpcServiceRegistration(registration.ServiceName, registration.AppName, registration.AppInstanceId)
	assert.False(t, ok)
	assert.Nil(t, got)
}

func TestRegistryRepoKeepAndRemoveWebRegistration(t *testing.T) {
	repo, redisServer := newTestRegistryRepo(t, false)

	registration := &core.WebRegistration{
		Endpoint:      "http://127.0.0.1:23001",
		WebSkelName:   "default@demo.app",
		AppName:       "demo.app",
		AppVersion:    "1.2.3",
		AppInstanceId: "instance-1",
	}
	repo.SaveWebRegistration(registration)

	key := redised.FormatWebRegistrationKey(registration.WebSkelName, registration.AppName, registration.AppInstanceId)

	assert.True(t, redisServer.Expire(key, 1))
	repo.KeepWebRegistration(registration.WebSkelName, registration.AppName, registration.AppInstanceId)

	got, ok := repo.GetWebRegistration(registration.WebSkelName, registration.AppName, registration.AppInstanceId)
	assert.True(t, ok)
	assert.Equal(t, registration, got)

	ttlAfter := redisServer.TTL(key)
	assert.True(t, ttlAfter > 1)

	repo.RemoveWebRegistration(registration.WebSkelName, registration.AppName, registration.AppInstanceId)

	got, ok = repo.GetWebRegistration(registration.WebSkelName, registration.AppName, registration.AppInstanceId)
	assert.False(t, ok)
	assert.Nil(t, got)
}

func TestRegistryRepoInprocSavesWithoutTTL(t *testing.T) {
	repo, redisServer := newTestRegistryRepo(t, true)

	status := &core.AppStatus{
		InstanceId: "instance-1",
		Name:       "demo.app",
	}
	repo.SaveAppStatus(status)

	key := redised.FormatAppStatusKey(status.Name, status.InstanceId)
	ttl := redisServer.TTL(key)
	assert.Equal(t, -1, ttl)
	got, ok := repo.GetAppStatus(status.Name, status.InstanceId)
	assert.True(t, ok)
	assert.True(t, got.ExpiresAt.IsZero())

	registration := &core.RpcServiceRegistration{
		Endpoint:      "app/instance-1",
		ServiceName:   "svc.alpha",
		AppName:       "demo.app",
		AppVersion:    "1.2.3",
		AppInstanceId: "instance-1",
	}
	repo.SaveRpcServiceRegistration(registration)

	key = redised.FormatRpcServiceRegistrationKey(registration.ServiceName, registration.AppName, registration.AppInstanceId)
	ttl = redisServer.TTL(key)
	assert.Equal(t, -1, ttl)

	webRegistration := &core.WebRegistration{
		Endpoint:      "app/instance-1",
		WebSkelName:   "default@demo.app",
		AppName:       "demo.app",
		AppVersion:    "1.2.3",
		AppInstanceId: "instance-1",
	}
	repo.SaveWebRegistration(webRegistration)

	key = redised.FormatWebRegistrationKey(webRegistration.WebSkelName, webRegistration.AppName, webRegistration.AppInstanceId)
	ttl = redisServer.TTL(key)
	assert.Equal(t, -1, ttl)
}

func TestRegistryRepoInprocSaveWebRegistrationNotifiesSubscribers(t *testing.T) {
	repo, redisServer := newTestRegistryRepo(t, true)
	pattern := "web:default@demo.app:endpoint:*"
	client := redis.NewClient(&redis.Options{
		Addr: hubredis.RedisInprocEndpoint,
		Dialer: func(ctx context.Context, network string, addr string) (net.Conn, error) {
			return redisServer.DialInproc(ctx)
		},
		Protocol:        2,
		DisableIdentity: true,
	})
	defer client.Close()
	pubsub := client.PSubscribe(t.Context(), pattern)
	defer pubsub.Close()
	read, err := pubsub.Receive(t.Context())
	assert.NoError(t, err)
	assert.Equal(t, &redis.Subscription{Kind: "psubscribe", Channel: pattern, Count: 1}, read)

	registration := &core.WebRegistration{
		Endpoint:      "link+inproc://demo/web/proxy/in/instance-1/default@demo.app",
		WebSkelName:   "default@demo.app",
		AppName:       "demo.app",
		AppVersion:    "1.2.3",
		AppInstanceId: "instance-1",
	}
	repo.SaveWebRegistration(registration)

	message, err := pubsub.ReceiveMessage(t.Context())
	assert.NoError(t, err)
	var event hubredis.Event
	assert.NoError(t, json.Unmarshal([]byte(message.Payload), &event))
	assert.Equal(t, hubredis.EventKindUpsert, event.Kind)
	assert.Equal(t, redised.FormatWebRegistrationKey(registration.WebSkelName, registration.AppName, registration.AppInstanceId), event.Key)
	assert.Equal(t, vcode.MustMarshalJsonS(toWebRegistration(registration)), event.Value)
}
