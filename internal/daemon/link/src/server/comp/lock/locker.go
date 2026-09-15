package lock

import (
	"sync"
	"time"

	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/meta"
	hublock "go.yorun.ai/vine/internal/daemon/hub/api/lock"
	hubskeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/control"
	"go.yorun.ai/vine/internal/daemon/link/src/server/comp/hubinfo"
	"go.yorun.ai/vine/util/vpre"
)

var lockLogger = logger.New("daemon:link:lock")

type _Locker interface {
	Acquire(ctx meta.Context, key string, token string, ttlMillis int) bool
	Renew(ctx meta.Context, key string, token string, ttlMillis int) bool
	Release(ctx meta.Context, key string, token string) bool
	Close()
}

// Locker routes lock requests to Hub or a Link-owned Redis connection.
type Locker struct {
	app.BaseComponent

	HubInfo *hubinfo.HubInfo            `inject:""`
	Hub     hubskeled.LockServiceClient `inject:""`

	mutex        sync.RWMutex
	impl         _Locker
	lockMode     string
	lockEndpoint string
}

func (l *Locker) DIInit() {
	l.applyHubLockConfig()
	l.HubInfo.OnRefresh(l.onHubInfoRefresh)
}

// onHubInfoRefresh rebuilds the lock client after Hub advertised different lock
// configuration, for example a Redis endpoint that moved. Leases held through the
// previous client are abandoned rather than migrated: holders observe the loss
// when their next renewal fails and stop, and only acquisitions made afterwards
// use the new connection. An unchanged mode and endpoint keep the existing
// client, so a Hub restart without a lock configuration change reconnects
// nothing.
func (l *Locker) onHubInfoRefresh() {
	if l.applyHubLockConfig() {
		lockLogger.Info("link replaced the lock client after Hub advertised new lock configuration", "mode", l.HubInfo.LockMode())
	}
}

// applyHubLockConfig installs the locker advertised by Hub, dropping leases the
// previous client held, and reports whether the effective configuration changed.
func (l *Locker) applyHubLockConfig() bool {
	mode := l.HubInfo.LockMode()
	endpoint := ""
	if mode == hublock.ModeRedis {
		endpoint = l.HubInfo.LockRedisEndpoint()
	}

	l.mutex.Lock()
	if l.impl != nil && l.lockMode == mode && l.lockEndpoint == endpoint {
		l.mutex.Unlock()
		return false
	}

	var next _Locker
	switch mode {
	case hublock.ModeEmbedded:
		next = new(_HubLocker{client: l.Hub})
	case hublock.ModeDisable:
	case hublock.ModeRedis:
		next = newRedisLocker(endpoint)
	default:
		l.mutex.Unlock()
		vpre.Panicf("unsupported Hub lock mode %q", mode)
	}

	previous := l.impl
	l.impl = next
	l.lockMode = mode
	l.lockEndpoint = endpoint
	l.mutex.Unlock()

	if previous != nil {
		previous.Close()
	}
	return true
}

func (l *Locker) AfterAppStop() {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.impl != nil {
		l.impl.Close()
	}
}

func (l *Locker) currentImpl() _Locker {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	return l.impl
}

func checkRequest(impl _Locker, key string, token string) {
	ex.PanicNewIfNot(impl != nil, ex.ServiceUnavailable, "lock service is disabled")
	ex.PanicNewIfNot(key != "" && token != "", ex.InvalidRequest, "lock key and token must not be empty")
}

func checkTTL(ttlMillis int) {
	ex.PanicNewIfNot(ttlMillis > 0 && int64(ttlMillis) <= int64((1<<63-1)/time.Millisecond), ex.InvalidRequest, "invalid lock lease duration")
}

func (l *Locker) Acquire(ctx meta.Context, key string, token string, ttlMillis int) bool {
	impl := l.currentImpl()
	checkRequest(impl, key, token)
	checkTTL(ttlMillis)
	return impl.Acquire(ctx, key, token, ttlMillis)
}

func (l *Locker) Renew(ctx meta.Context, key string, token string, ttlMillis int) bool {
	impl := l.currentImpl()
	checkRequest(impl, key, token)
	checkTTL(ttlMillis)
	return impl.Renew(ctx, key, token, ttlMillis)
}

func (l *Locker) Release(ctx meta.Context, key string, token string) bool {
	impl := l.currentImpl()
	checkRequest(impl, key, token)
	return impl.Release(ctx, key, token)
}
