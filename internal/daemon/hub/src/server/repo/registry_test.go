package repo

import (
	"context"
	"encoding/json/v2"
	"net"
	"testing"
	"time"
	"uuid"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	internalapp "go.yorun.ai/vine/internal/app"
	hubwatch "go.yorun.ai/vine/internal/daemon/hub/api/watch"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/watchserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/repo/descriptor"
	"go.yorun.ai/vine/util/vcode"
)

func newTestRegistryRepo(t *testing.T, inproc bool) (*RegistryRepo, *watchserver.Server) {
	t.Helper()

	watchServer := &watchserver.Server{
		Context:    t.Context(),
		Option:     &flag.Flag{},
		InprocFlag: &internalapp.InternalInprocFlag{Enabled: true},
	}
	watchServer.DIInit()
	repo := &RegistryRepo{
		WatchServer: watchServer,
		InprocFlag:  &internalapp.InternalInprocFlag{Enabled: inproc},
	}

	t.Cleanup(func() {
		watchServer.AfterAppStop()
	})

	return repo, watchServer
}

func TestRegistryRepoSaveAndGetAppStatus(t *testing.T) {
	repo, watchServer := newTestRegistryRepo(t, false)
	now := time.Now().UTC().Round(0)
	setTimeNowForTest(t, func() time.Time { return now })

	status := &core.AppStatus{
		InstanceId:      uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		Name:            "demo.app",
		Version:         "1.2.3",
		Endpoint:        "http://127.0.0.1:23001/rpc",
		ServiceHandlers: []core.ServiceHandlerRegistration{{ServiceSkelName: "svc.alpha"}, {ServiceSkelName: "svc.beta"}},
		WebHandlers:     []core.WebHandlerRegistration{{WebSkelName: "default@demo.app"}},
		EventListeners:  []core.EventListenerRegistration{},
		TaskRunners:     []core.TaskRunnerRegistration{},
	}

	repo.SaveAppStatus(status)

	key := watched.FormatAppStatusKey(status.Name, status.InstanceId)
	raw, ok := watchServer.Get(key)
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
		EventListeners:  []core.EventListenerRegistration{},
		TaskRunners:     []core.TaskRunnerRegistration{},
	}, gotRaw)

	got, ok := repo.GetAppStatus(status.Name, status.InstanceId)
	assert.True(t, ok)
	status.ExpiresAt = now.Add(hubRegistryLeaseTTL)
	assert.Equal(t, status, got)

	ttl := watchServer.TTL(key)
	assert.Equal(t, -1, ttl)
}

func TestRegistryRepoGetAppStatusReturnsFalseWhenMissing(t *testing.T) {
	repo, _ := newTestRegistryRepo(t, false)

	got, ok := repo.GetAppStatus("", uuid.MustParse("00000000-0000-0000-0000-0000000000d8"))
	assert.False(t, ok)
	assert.Nil(t, got)
}

func TestRegistryRepoListAppStatuses(t *testing.T) {
	repo, _ := newTestRegistryRepo(t, false)

	repo.SaveAppStatus(&core.AppStatus{
		InstanceId: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		Name:       "demo.beta",
		Version:    "2.0.0",
	})
	repo.SaveAppStatus(&core.AppStatus{
		InstanceId: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		Name:       "demo.alpha",
		Version:    "1.0.0",
	})

	items := repo.ListAppStatuses()

	assert.Len(t, items, 2)
	assert.Equal(t, "demo.alpha", items[0].Name)
	assert.Equal(t, uuid.MustParse("00000000-0000-0000-0000-000000000001"), items[0].InstanceId)
	assert.Equal(t, "1.0.0", items[0].Version)
	assert.Equal(t, "demo.beta", items[1].Name)
}

func TestRegistryRepoKeepAndRemoveAppStatus(t *testing.T) {
	repo, watchServer := newTestRegistryRepo(t, false)

	status := &core.AppStatus{InstanceId: uuid.MustParse("00000000-0000-0000-0000-000000000001"), Name: "demo.app"}
	repo.SaveAppStatus(status)

	key := watched.FormatAppStatusKey(status.Name, status.InstanceId)
	repo.KeepAppStatus(status.Name, status.InstanceId)

	got, ok := repo.GetAppStatus(status.Name, status.InstanceId)
	assert.True(t, ok)
	assert.Equal(t, status.InstanceId, got.InstanceId)

	ttlAfter := watchServer.TTL(key)
	assert.Equal(t, -1, ttlAfter)

	repo.RemoveAppStatus(status.Name, status.InstanceId)

	got, ok = repo.GetAppStatus(status.Name, status.InstanceId)
	assert.False(t, ok)
	assert.Nil(t, got)
}

func TestRegistryRepoListExpiredAppLeases(t *testing.T) {
	repo, _ := newTestRegistryRepo(t, false)
	now := time.Now().Round(0)
	setTimeNowForTest(t, func() time.Time { return now })
	status := &core.AppStatus{InstanceId: uuid.MustParse("00000000-0000-0000-0000-000000000001"), Name: "demo.app"}
	repo.SaveAppStatus(status)

	now = now.Add(hubRegistryLeaseTTL - time.Second)
	assert.Empty(t, repo.ListExpiredAppLeases())

	now = now.Add(2 * time.Second)
	leases := repo.ListExpiredAppLeases()

	assert.Equal(t, []core.AppHeartbeat{{Name: "demo.app", InstanceId: uuid.MustParse("00000000-0000-0000-0000-000000000001")}}, leases)
	assert.Equal(t, leases, repo.ListExpiredAppLeases())
	repo.RemoveAppStatus(status.Name, status.InstanceId)
	assert.Empty(t, repo.ListExpiredAppLeases())
}

func TestRegistryRepoKeepAppStatusRefreshesAppLease(t *testing.T) {
	repo, _ := newTestRegistryRepo(t, false)
	now := time.Now().Round(0)
	setTimeNowForTest(t, func() time.Time { return now })
	status := &core.AppStatus{InstanceId: uuid.MustParse("00000000-0000-0000-0000-000000000001"), Name: "demo.app"}
	repo.SaveAppStatus(status)

	now = now.Add(hubRegistryLeaseTTL / 2)
	repo.KeepAppStatus(status.Name, status.InstanceId)

	now = now.Add(hubRegistryLeaseTTL - time.Second)
	assert.Empty(t, repo.ListExpiredAppLeases())
}

func TestRegistryRepoListExpiredAppLeasesLeavesRenewalCheckToCore(t *testing.T) {
	repo, watchServer := newTestRegistryRepo(t, false)
	now := time.Now().Round(0)
	setTimeNowForTest(t, func() time.Time { return now })
	status := &core.AppStatus{InstanceId: uuid.MustParse("00000000-0000-0000-0000-000000000001"), Name: "demo.app"}
	repo.SaveAppStatus(status)

	key := watched.FormatAppStatusKey(status.Name, status.InstanceId)
	raw, ok := repo.getAppStatus(status.Name, status.InstanceId)
	assert.True(t, ok)
	raw.ExpiresAt = now.Add(2 * hubRegistryLeaseTTL)
	watchServer.Set(key, vcode.MustMarshalJsonS(raw))

	now = now.Add(hubRegistryLeaseTTL + time.Second)

	assert.Equal(t, []core.AppHeartbeat{{Name: status.Name, InstanceId: status.InstanceId}}, repo.ListExpiredAppLeases())
}

func TestRegistryRepoInprocDoesNotSaveAppLease(t *testing.T) {
	repo, _ := newTestRegistryRepo(t, true)

	repo.SaveAppStatus(&core.AppStatus{InstanceId: uuid.MustParse("00000000-0000-0000-0000-000000000001"), Name: "demo.app"})

	assert.Empty(t, repo.ListExpiredAppLeases())
}

func TestRegistryRepoSaveAndGetRpcServiceRegistration(t *testing.T) {
	repo, watchServer := newTestRegistryRepo(t, false)

	registration := &core.RpcServiceRegistration{
		Endpoint:      "http://127.0.0.1:23001/rpc",
		ServiceName:   "svc.alpha",
		Api:           true,
		AppName:       "demo.app",
		AppVersion:    "1.2.3",
		AppInstanceId: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
	}

	repo.SaveRpcServiceRegistration(registration)

	key := watched.FormatRpcServiceRegistrationKey(registration.ServiceName, registration.AppName, registration.AppInstanceId)
	raw, ok := watchServer.Get(key)
	assert.True(t, ok)
	gotRaw := vcode.MustUnmarshalJsonS[*watched.RpcServiceRegistration](raw)
	assert.Equal(t, toRpcServiceRegistration(registration), gotRaw)

	got, ok := repo.GetRpcServiceRegistration(registration.ServiceName, registration.AppName, registration.AppInstanceId)
	assert.True(t, ok)
	assert.Equal(t, registration, got)

	ttl := watchServer.TTL(key)
	assert.Equal(t, -1, ttl)
}

func TestRegistryRepoSaveAndGetWebRegistration(t *testing.T) {
	repo, watchServer := newTestRegistryRepo(t, false)

	registration := &core.WebRegistration{
		Endpoint:      "http://127.0.0.1:23001",
		WebSkelName:   "default@demo.app",
		AppName:       "demo.app",
		AppVersion:    "1.2.3",
		AppInstanceId: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
	}

	repo.SaveWebRegistration(registration)

	key := watched.FormatWebRegistrationKey(registration.WebSkelName, registration.AppName, registration.AppInstanceId)
	raw, ok := watchServer.Get(key)
	assert.True(t, ok)
	gotRaw := vcode.MustUnmarshalJsonS[*watched.WebRegistration](raw)
	assert.Equal(t, toWebRegistration(registration), gotRaw)

	got, ok := repo.GetWebRegistration(registration.WebSkelName, registration.AppName, registration.AppInstanceId)
	assert.True(t, ok)
	assert.Equal(t, registration, got)

	ttl := watchServer.TTL(key)
	assert.Equal(t, -1, ttl)
}

func TestRegistryRepoGetRpcServiceRegistrationReturnsFalseWhenMissing(t *testing.T) {
	repo, _ := newTestRegistryRepo(t, false)

	got, ok := repo.GetRpcServiceRegistration("svc.alpha", "demo.app", uuid.MustParse("00000000-0000-0000-0000-0000000000d8"))
	assert.False(t, ok)
	assert.Nil(t, got)
}

func TestRegistryRepoGetWebRegistrationReturnsFalseWhenMissing(t *testing.T) {
	repo, _ := newTestRegistryRepo(t, false)

	got, ok := repo.GetWebRegistration("default@demo.app", "demo.app", uuid.MustParse("00000000-0000-0000-0000-0000000000d8"))
	assert.False(t, ok)
	assert.Nil(t, got)
}

func TestRegistryRepoKeepAndRemoveRpcServiceRegistration(t *testing.T) {
	repo, watchServer := newTestRegistryRepo(t, false)

	registration := &core.RpcServiceRegistration{
		Endpoint:      "http://127.0.0.1:23001/rpc",
		ServiceName:   "svc.alpha",
		AppName:       "demo.app",
		AppVersion:    "1.2.3",
		AppInstanceId: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
	}
	repo.SaveRpcServiceRegistration(registration)

	key := watched.FormatRpcServiceRegistrationKey(registration.ServiceName, registration.AppName, registration.AppInstanceId)

	repo.KeepRpcServiceRegistration(registration.ServiceName, registration.AppName, registration.AppInstanceId)

	got, ok := repo.GetRpcServiceRegistration(registration.ServiceName, registration.AppName, registration.AppInstanceId)
	assert.True(t, ok)
	assert.Equal(t, registration, got)

	ttlAfter := watchServer.TTL(key)
	assert.Equal(t, -1, ttlAfter)

	repo.RemoveRpcServiceRegistration(registration.ServiceName, registration.AppName, registration.AppInstanceId)

	got, ok = repo.GetRpcServiceRegistration(registration.ServiceName, registration.AppName, registration.AppInstanceId)
	assert.False(t, ok)
	assert.Nil(t, got)
}

func TestRegistryRepoKeepAndRemoveWebRegistration(t *testing.T) {
	repo, watchServer := newTestRegistryRepo(t, false)

	registration := &core.WebRegistration{
		Endpoint:      "http://127.0.0.1:23001",
		WebSkelName:   "default@demo.app",
		AppName:       "demo.app",
		AppVersion:    "1.2.3",
		AppInstanceId: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
	}
	repo.SaveWebRegistration(registration)

	key := watched.FormatWebRegistrationKey(registration.WebSkelName, registration.AppName, registration.AppInstanceId)

	repo.KeepWebRegistration(registration.WebSkelName, registration.AppName, registration.AppInstanceId)

	got, ok := repo.GetWebRegistration(registration.WebSkelName, registration.AppName, registration.AppInstanceId)
	assert.True(t, ok)
	assert.Equal(t, registration, got)

	ttlAfter := watchServer.TTL(key)
	assert.Equal(t, -1, ttlAfter)

	repo.RemoveWebRegistration(registration.WebSkelName, registration.AppName, registration.AppInstanceId)

	got, ok = repo.GetWebRegistration(registration.WebSkelName, registration.AppName, registration.AppInstanceId)
	assert.False(t, ok)
	assert.Nil(t, got)
}

func TestRegistryRepoInprocSavesWithoutTTL(t *testing.T) {
	repo, watchServer := newTestRegistryRepo(t, true)

	status := &core.AppStatus{InstanceId: uuid.MustParse("00000000-0000-0000-0000-000000000001"), Name: "demo.app"}
	repo.SaveAppStatus(status)

	key := watched.FormatAppStatusKey(status.Name, status.InstanceId)
	ttl := watchServer.TTL(key)
	assert.Equal(t, -1, ttl)
	got, ok := repo.GetAppStatus(status.Name, status.InstanceId)
	assert.True(t, ok)
	assert.True(t, got.ExpiresAt.IsZero())

	registration := &core.RpcServiceRegistration{
		Endpoint:      "app/instance-1",
		ServiceName:   "svc.alpha",
		AppName:       "demo.app",
		AppVersion:    "1.2.3",
		AppInstanceId: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
	}
	repo.SaveRpcServiceRegistration(registration)

	key = watched.FormatRpcServiceRegistrationKey(registration.ServiceName, registration.AppName, registration.AppInstanceId)
	ttl = watchServer.TTL(key)
	assert.Equal(t, -1, ttl)

	webRegistration := &core.WebRegistration{
		Endpoint:      "app/instance-1",
		WebSkelName:   "default@demo.app",
		AppName:       "demo.app",
		AppVersion:    "1.2.3",
		AppInstanceId: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
	}
	repo.SaveWebRegistration(webRegistration)

	key = watched.FormatWebRegistrationKey(webRegistration.WebSkelName, webRegistration.AppName, webRegistration.AppInstanceId)
	ttl = watchServer.TTL(key)
	assert.Equal(t, -1, ttl)
}

func TestRegistryRepoInprocSaveWebRegistrationNotifiesSubscribers(t *testing.T) {
	repo, watchServer := newTestRegistryRepo(t, true)
	pattern := "web:default@demo.app:endpoint:*"
	client := redis.NewClient(&redis.Options{
		Addr: hubwatch.WatchInprocEndpoint,
		Dialer: func(ctx context.Context, network string, addr string) (net.Conn, error) {
			return watchServer.DialInproc(ctx)
		},
		Username: hubwatch.PortalUsername,
		Password: hubwatch.PortalPassword,
		OnConnect: func(ctx context.Context, conn *redis.Conn) error {
			return conn.AuthACL(ctx, hubwatch.PortalUsername, hubwatch.PortalPassword).Err()
		},
		Protocol:        2,
		DisableIdentity: true,
	})
	defer client.Close()
	pubsub := client.PSubscribe(t.Context(), pattern)
	defer pubsub.Close()
	read, err := pubsub.Receive(t.Context())
	require.NoError(t, err)
	assert.Equal(t, &redis.Subscription{Kind: "psubscribe", Channel: pattern, Count: 1}, read)

	registration := &core.WebRegistration{
		Endpoint:      "link+inproc://demo/web/proxy/in/instance-1/default@demo.app",
		WebSkelName:   "default@demo.app",
		AppName:       "demo.app",
		AppVersion:    "1.2.3",
		AppInstanceId: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
	}
	repo.SaveWebRegistration(registration)

	message, err := pubsub.ReceiveMessage(t.Context())
	require.NoError(t, err)
	var event hubwatch.Event
	assert.NoError(t, json.Unmarshal([]byte(message.Payload), &event))
	assert.Equal(t, hubwatch.EventKindUpsert, event.Kind)
	assert.Equal(t, watched.FormatWebRegistrationKey(registration.WebSkelName, registration.AppName, registration.AppInstanceId), event.Key)
	assert.Equal(t, vcode.MustMarshalJsonS(toWebRegistration(registration)), event.Value)
}

type cleanupDescriptorRepo struct {
	core.DescriptorRepo
	failOwner string
	released  []string
}

func (r *cleanupDescriptorRepo) ReleaseDomainDescriptors(owner string, instanceId string) {
	if owner == r.failOwner {
		panic("descriptor store unavailable")
	}
	r.released = append(r.released, owner+":"+instanceId)
}

func TestRegistryRetainsExpiredStateUntilCompleteCleanup(t *testing.T) {
	repo, server := newTestRegistryRepo(t, false)
	now := time.Now().Add(-4 * time.Minute)
	setTimeNowForTest(t, func() time.Time { return now })
	status := &core.AppStatus{
		Name:            "demo.app",
		InstanceId:      uuid.MustParse("00000000-0000-0000-0000-0000000000d5"),
		ServiceHandlers: []core.ServiceHandlerRegistration{{ServiceSkelName: "demo.Service"}},
		WebHandlers:     []core.WebHandlerRegistration{{WebSkelName: "demo.Web"}},
	}
	repo.SaveAppStatus(status)
	repo.SaveRpcServiceRegistration(&core.RpcServiceRegistration{ServiceName: "demo.Service", AppName: status.Name, AppInstanceId: status.InstanceId, Endpoint: "old-ip"})
	repo.SaveWebRegistration(&core.WebRegistration{WebSkelName: "demo.Web", AppName: status.Name, AppInstanceId: status.InstanceId, Endpoint: "old-ip"})
	now = time.Now()
	_, ok := repo.GetAppStatus(status.Name, status.InstanceId)
	require.True(t, ok, "cleanup metadata must survive a sweep delayed beyond 180 seconds")
	require.Empty(t, repo.ListAppStatuses(), "expired instances must not be scheduled as live")
	require.Equal(t, -1, server.TTL(watched.FormatAppStatusKey(status.Name, status.InstanceId)))

	client := redis.NewClient(&redis.Options{
		Addr: hubwatch.WatchInprocEndpoint,
		Dialer: func(ctx context.Context, network string, addr string) (net.Conn, error) {
			return server.DialInproc(ctx)
		},
		OnConnect: func(ctx context.Context, conn *redis.Conn) error {
			return conn.AuthACL(ctx, hubwatch.PortalUsername, hubwatch.PortalPassword).Err()
		},
		Protocol:        2,
		DisableIdentity: true,
	})
	defer client.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	subscription := client.PSubscribe(ctx, "rpc:*", "web:*")
	defer subscription.Close()
	for range 2 {
		_, err := subscription.Receive(ctx)
		require.NoError(t, err)
	}
	descriptors := new(cleanupDescriptorRepo)
	registry := &core.RegistryCore{RegistryRepo: repo, DescriptorRepo: descriptors}
	require.True(t, registry.SweepExpiredLeases())
	for range 2 {
		message, err := subscription.ReceiveMessage(ctx)
		require.NoError(t, err)
		event := vcode.MustUnmarshalJsonS[hubwatch.Event](message.Payload)
		require.Equal(t, hubwatch.EventKindDelete, event.Kind)
	}
	_, ok = repo.GetAppStatus(status.Name, status.InstanceId)
	require.False(t, ok)
	require.Empty(t, repo.ListExpiredAppLeases())
	require.Equal(t, []string{"demo.app:00000000-0000-0000-0000-0000000000d5"}, descriptors.released)
	require.False(t, registry.SweepExpiredLeases())
}

func TestRegistryRetriesFailedCleanupWithoutBlockingOtherInstances(t *testing.T) {
	repo, _ := newTestRegistryRepo(t, false)
	now := time.Now().Add(-time.Minute)
	setTimeNowForTest(t, func() time.Time { return now })
	for _, name := range []string{"failed", "healthy"} {
		repo.SaveAppStatus(&core.AppStatus{
			Name:            name,
			InstanceId:      uuid.MustParse("00000000-0000-0000-0000-0000000000d5"),
			ServiceHandlers: []core.ServiceHandlerRegistration{{ServiceSkelName: "demo.Service"}},
			WebHandlers:     []core.WebHandlerRegistration{{WebSkelName: "demo.Web"}},
		})
		repo.SaveRpcServiceRegistration(&core.RpcServiceRegistration{ServiceName: "demo.Service", AppName: name, AppInstanceId: uuid.MustParse("00000000-0000-0000-0000-0000000000d5")})
		repo.SaveWebRegistration(&core.WebRegistration{WebSkelName: "demo.Web", AppName: name, AppInstanceId: uuid.MustParse("00000000-0000-0000-0000-0000000000d5")})
	}
	now = time.Now()
	descriptors := &cleanupDescriptorRepo{failOwner: "failed"}
	registry := &core.RegistryCore{RegistryRepo: repo, DescriptorRepo: descriptors}
	require.True(t, registry.SweepExpiredLeases())
	require.Equal(t, []core.AppHeartbeat{{Name: "failed", InstanceId: uuid.MustParse("00000000-0000-0000-0000-0000000000d5")}}, repo.ListExpiredAppLeases())
	_, ok := repo.GetAppStatus("failed", uuid.MustParse("00000000-0000-0000-0000-0000000000d5"))
	require.True(t, ok)
	_, ok = repo.GetAppStatus("healthy", uuid.MustParse("00000000-0000-0000-0000-0000000000d5"))
	require.False(t, ok)
	_, ok = repo.GetRpcServiceRegistration("demo.Service", "failed", uuid.MustParse("00000000-0000-0000-0000-0000000000d5"))
	require.False(t, ok, "endpoint removal already completed before descriptor cleanup failed")
	_, ok = repo.GetWebRegistration("demo.Web", "failed", uuid.MustParse("00000000-0000-0000-0000-0000000000d5"))
	require.False(t, ok)
	descriptors.failOwner = ""
	require.True(t, registry.SweepExpiredLeases())
	require.Empty(t, repo.ListExpiredAppLeases())
	require.Equal(t, []string{"healthy:00000000-0000-0000-0000-0000000000d5", "failed:00000000-0000-0000-0000-0000000000d5"}, descriptors.released)
}

func TestRegistryReregisterRemovesWithdrawnEndpoints(t *testing.T) {
	repo, server := newTestRegistryRepo(t, false)
	registry := &core.RegistryCore{RegistryRepo: repo, DescriptorRepo: new(descriptor.DescriptorRepo)}
	reg := core.AppRegistration{
		Name:       "demo.app",
		InstanceId: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		ServiceHandlers: []core.ServiceHandlerRegistration{
			{ServiceSkelName: "demo.RemovedService", Endpoint: "old-ip"},
			{ServiceSkelName: "demo.KeptService", Endpoint: "old-ip"},
		},
		WebHandlers: []core.WebHandlerRegistration{
			{WebSkelName: "demo.RemovedWeb", Endpoint: "old-ip"},
			{WebSkelName: "demo.KeptWeb", Endpoint: "old-ip"},
		},
	}
	registry.Register(reg)
	reg.ServiceHandlers = []core.ServiceHandlerRegistration{{ServiceSkelName: "demo.KeptService", Endpoint: "new-ip"}}
	reg.WebHandlers = []core.WebHandlerRegistration{{WebSkelName: "demo.KeptWeb", Endpoint: "new-ip"}}
	registry.Register(reg)

	_, ok := repo.GetRpcServiceRegistration("demo.RemovedService", reg.Name, reg.InstanceId)
	require.False(t, ok)
	_, ok = repo.GetWebRegistration("demo.RemovedWeb", reg.Name, reg.InstanceId)
	require.False(t, ok)
	rpc, ok := repo.GetRpcServiceRegistration("demo.KeptService", reg.Name, reg.InstanceId)
	require.True(t, ok)
	require.Equal(t, "new-ip", rpc.Endpoint)
	web, ok := repo.GetWebRegistration("demo.KeptWeb", reg.Name, reg.InstanceId)
	require.True(t, ok)
	require.Equal(t, "new-ip", web.Endpoint)
	registry.Unregister(reg.Name, reg.InstanceId)
	require.Empty(t, server.Scan("rpc:*"))
	require.Empty(t, server.Scan("web:*"))
}
