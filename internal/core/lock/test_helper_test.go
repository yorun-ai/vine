package lock

import (
	"context"
	"go.yorun.ai/vine/internal/core/rpc/client"
	"sync"
	"time"
)

type testClient struct {
	mutex    sync.Mutex
	leases   map[string]testLease
	keys     []string
	attempts []string
	ttl      time.Duration
	renewals int
	releases int
	acquire  func(context.Context, string, string, time.Duration) bool
	renew    func(context.Context, string, string, time.Duration) bool
	release  func(context.Context, string, string) bool
}

func (c *testClient) Acquire(key, token string, ttlMillis int, options ...client.InvokeOption) bool {
	ttl := time.Duration(ttlMillis) * time.Millisecond
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.keys = append(c.keys, key)
	c.attempts = append(c.attempts, token)
	c.ttl = ttl
	if c.acquire != nil {
		return c.acquire(context.Background(), key, token, ttl)
	}
	lease := c.leases[key]
	if lease.token != "" && time.Now().Before(lease.expiry) {
		return false
	}
	if c.leases == nil {
		c.leases = make(map[string]testLease)
	}
	c.leases[key] = testLease{token: token, expiry: time.Now().Add(ttl)}
	return true
}

func (c *testClient) Renew(key, token string, ttlMillis int, options ...client.InvokeOption) bool {
	ttl := time.Duration(ttlMillis) * time.Millisecond
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.renewals++
	if c.renew != nil {
		return c.renew(context.Background(), key, token, ttl)
	}
	lease := c.leases[key]
	if token != lease.token || !time.Now().Before(lease.expiry) {
		return false
	}
	c.leases[key] = testLease{token: token, expiry: time.Now().Add(ttl)}
	return true
}

func (c *testClient) Release(key, token string, options ...client.InvokeOption) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.releases++
	if c.release != nil {
		return c.release(context.Background(), key, token)
	}
	lease := c.leases[key]
	if token != lease.token || !time.Now().Before(lease.expiry) {
		return false
	}
	delete(c.leases, key)
	return true
}

type testLease struct {
	token  string
	expiry time.Time
}
