package hubinfo

import (
	"net"
	"strconv"
	"sync"

	"go.yorun.ai/vine/internal/daemon/hub/api/skeled/control"
	"go.yorun.ai/vine/util/vslice"
)

// Cache keeps the Hub information used by connection owners in Link and Portal.
type Cache struct {
	// Serialize fetch, listener repairs, and rollback. Readers can still see
	// the candidate endpoints while listeners are repairing their connections.
	refreshMutex  sync.Mutex
	repairPending bool

	mutex     sync.RWMutex
	info      control.Info
	listeners []func()
}

// Init stores the Hub information fetched at startup.
func (c *Cache) Init(info control.Info) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.info = info
}

// Info returns the currently advertised endpoints and capabilities.
func (c *Cache) Info() control.Info {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.info
}

// WatchEndpoint returns the dial address for the Hub's watch service.
func (c *Cache) WatchEndpoint(host string) string {
	return net.JoinHostPort(host, strconv.Itoa(c.Info().WatchPort))
}

// OnRefresh registers a listener that repairs its connections after a change.
func (c *Cache) OnRefresh(listener func()) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.listeners = append(c.listeners, listener)
}

// Refresh fetches the latest Hub information and runs all pending repairs.
// A failed repair rolls back the snapshot and is retried on the next refresh,
// even if the Hub has switched back to the previous endpoints.
func (c *Cache) Refresh(client control.InfoServiceClient) {
	c.refreshMutex.Lock()
	defer c.refreshMutex.Unlock()
	info := client.GetInfo()

	c.mutex.Lock()
	previous := c.info
	if previous == info && !c.repairPending {
		c.mutex.Unlock()
		return
	}
	c.info = info
	listeners := vslice.Clone(c.listeners)
	c.mutex.Unlock()

	var firstFailure any
	for _, listener := range listeners {
		func() {
			defer func() {
				if recovered := recover(); recovered != nil && firstFailure == nil {
					firstFailure = recovered
				}
			}()
			listener()
		}()
	}
	if firstFailure != nil {
		c.mutex.Lock()
		c.info = previous
		c.mutex.Unlock()
		c.repairPending = true
		panic(firstFailure)
	}
	c.repairPending = false
}
