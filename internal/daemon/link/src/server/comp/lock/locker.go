package lock

import (
	"time"

	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/meta"
	hublock "go.yorun.ai/vine/internal/daemon/hub/api/lock"
	hubskeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/control"
	"go.yorun.ai/vine/internal/daemon/link/src/server/comp/hubinfo"
	"go.yorun.ai/vine/util/vpre"
)

type _Locker interface {
	Acquire(ctx meta.Context, key, token string, ttlMillis int) bool
	Renew(ctx meta.Context, key, token string, ttlMillis int) bool
	Release(ctx meta.Context, key, token string) bool
	Close()
}

// Locker routes lock requests to Hub or a Link-owned Redis connection.
type Locker struct {
	app.BaseComponent

	HubInfo *hubinfo.HubInfo            `inject:""`
	Hub     hubskeled.LockServiceClient `inject:""`

	impl _Locker
}

func (l *Locker) DIInit() {
	mode := l.HubInfo.LockMode()
	switch mode {
	case hublock.ModeEmbedded:
		l.impl = new(_HubLocker{client: l.Hub})
	case hublock.ModeDisable:
	case hublock.ModeRedis:
		l.impl = newRedisLocker(l.HubInfo.LockRedisEndpoint())
	default:
		vpre.Panicf("unsupported Hub lock mode %q", mode)
	}
}

func (l *Locker) AfterAppStop() {
	if l.impl != nil {
		l.impl.Close()
	}
}

func (l *Locker) checkRequest(key, token string) {
	ex.PanicNewIfNot(l.impl != nil, ex.ServiceUnavailable, "lock service is disabled")
	ex.PanicNewIfNot(key != "" && token != "", ex.InvalidRequest, "lock key and token must not be empty")
}

func checkTTL(ttlMillis int) {
	ex.PanicNewIfNot(ttlMillis > 0 && int64(ttlMillis) <= int64((1<<63-1)/time.Millisecond), ex.InvalidRequest, "invalid lock lease duration")
}

func (l *Locker) Acquire(ctx meta.Context, key, token string, ttlMillis int) bool {
	l.checkRequest(key, token)
	checkTTL(ttlMillis)
	return l.impl.Acquire(ctx, key, token, ttlMillis)
}

func (l *Locker) Renew(ctx meta.Context, key, token string, ttlMillis int) bool {
	l.checkRequest(key, token)
	checkTTL(ttlMillis)
	return l.impl.Renew(ctx, key, token, ttlMillis)
}

func (l *Locker) Release(ctx meta.Context, key, token string) bool {
	l.checkRequest(key, token)
	return l.impl.Release(ctx, key, token)
}
