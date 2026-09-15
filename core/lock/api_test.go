package lock_test

import (
	"testing"
	"time"

	"go.yorun.ai/vine/core/lock"
	internallock "go.yorun.ai/vine/internal/core/lock"
	rpcclient "go.yorun.ai/vine/internal/core/rpc/client"
)

type client struct{}

func (*client) Acquire(string, string, int, ...rpcclient.InvokeOption) bool {
	return true
}

func (*client) Renew(string, string, int, ...rpcclient.InvokeOption) bool {
	return true
}

func (*client) Release(string, string, ...rpcclient.InvokeOption) bool {
	return true
}

func TestPublicFacade(t *testing.T) {
	var locker *lock.Locker = internallock.NewLocker(new(client), t.Context(), "app")
	var lease *lock.Lock
	lease, acquired := locker.TryLock("job", lock.WithTTL(time.Second))
	if !acquired {
		t.Fatal("expected acquisition")
	}
	lease.Unlock()
	lease = locker.Lock("job", lock.WithTTL(time.Second))
	lease.Unlock()
}

func TestUniversalPublicFacade(t *testing.T) {
	var locker *lock.UniversalLocker = internallock.NewUniversalLocker(new(client), t.Context())
	var bound *lock.UniversalLocker = locker.WithContext(t.Context())
	lease, acquired := bound.TryLock("job", lock.WithTTL(time.Second))
	if !acquired {
		t.Fatal("expected acquisition")
	}
	lease.Unlock()
	bound.Lock("job").Unlock()
}
