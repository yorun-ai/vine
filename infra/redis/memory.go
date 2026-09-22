package redis

import (
	"context"
	"net"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"go.yorun.ai/vine/util/vpre"
)

const memoryCleanupInterval = 60 * time.Second

var (
	memoryEndpointPattern = regexp.MustCompile(`^redis\+memory://([A-Za-z0-9_.-]+)(?:/([0-9]+))?$`)
	memoryMu              sync.Mutex
	memoryRedises         = map[string]*_MemoryRedis{}
)

// acquireMemoryClient returns the process-shared client for redis+memory://name[/dbIndex] and a release
// function. Every acquisition must be released. Do not call client.Close directly.
// The last release closes the client and instance and discards all cached data.
// Different processes never share data or locks, even with identical URLs.
func acquireMemoryClient(endpoint string) (*goredis.Client, func()) {
	parts := memoryEndpointPattern.FindStringSubmatch(endpoint)
	vpre.Check(parts != nil, "memory Redis endpoint must be redis+memory://{name}[/{dbIndex}]; name may contain only letters, digits, _, -, and .")
	dbIndex := 0
	if parts[2] != "" {
		var err error
		dbIndex, err = strconv.Atoi(parts[2])
		vpre.Check(err == nil && dbIndex < 16, "memory Redis database index must be between 0 and 15")
	}
	instanceEndpoint := "redis+memory://" + parts[1]

	memoryMu.Lock()
	defer memoryMu.Unlock()

	redis := memoryRedises[instanceEndpoint]
	if redis == nil {
		redis = newMemoryRedis(instanceEndpoint)
		memoryRedises[instanceEndpoint] = redis
	}
	return redis.Acquire(dbIndex)
}

// Serializing Dial and close prevents miniredis.Dial from restarting a closed server.
type _MemoryRedis struct {
	endpoint    string
	mutex       sync.Mutex
	server      *miniredis.Miniredis
	clients     map[int]*goredis.Client // Protected by memoryMu.
	refs        map[int]int             // Protected by memoryMu.
	cleanupStop chan struct{}
	cleanupDone chan struct{}
	closed      bool
}

func newMemoryRedis(endpoint string) *_MemoryRedis {
	server := miniredis.NewMiniRedis()
	server.ClockTTL(true)
	instance := new(_MemoryRedis{
		endpoint: endpoint, server: server,
		clients: make(map[int]*goredis.Client), refs: make(map[int]int),
		cleanupStop: make(chan struct{}), cleanupDone: make(chan struct{}),
	})
	go instance.cleanupLoop()
	return instance
}

// Acquire runs under memoryMu so lookup and reference acquisition are atomic.
func (i *_MemoryRedis) Acquire(dbIndex int) (*goredis.Client, func()) {
	client := i.clients[dbIndex]
	if client == nil {
		client = i.newClient(dbIndex)
		i.clients[dbIndex] = client
	}
	i.refs[dbIndex]++
	return client, sync.OnceFunc(func() { i.release(dbIndex) })
}

func (i *_MemoryRedis) release(dbIndex int) {
	memoryMu.Lock()
	defer memoryMu.Unlock()
	i.refs[dbIndex]--
	if i.refs[dbIndex] == 0 {
		_ = i.clients[dbIndex].Close()
		delete(i.clients, dbIndex)
		delete(i.refs, dbIndex)
	}
	if len(i.refs) == 0 {
		i.close()
		delete(memoryRedises, i.endpoint)
	}
}

func (i *_MemoryRedis) newClient(dbIndex int) *goredis.Client {
	return goredis.NewClient(&goredis.Options{
		// A numeric address avoids client-side DNS; Dialer uses only the memory pipe.
		Addr:            "127.0.0.1:0",
		Protocol:        2,
		DB:              dbIndex,
		DisableIdentity: true,
		Dialer: func(ctx context.Context, network string, addr string) (net.Conn, error) {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			return i.dial()
		},
	})
}

func (i *_MemoryRedis) dial() (net.Conn, error) {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	if i.closed {
		return nil, net.ErrClosed
	}
	return i.server.Dial()
}

// cleanupLoop reclaims expired data even when clients are idle. ClockTTL(true)
// already advances TTLs using elapsed wall time on database access. In the pinned
// miniredis version, source inspection shows that DB(0) calls db(), which invokes
// tickLocked() and expires keys and hash fields across ALL databases, not just DB 0.
// It neither sends SELECT nor changes clients. This is an implementation-dependent
// trick discovered in the source, NOT a cleanup guarantee of the public DB API.
// Replace it with the official cleanup API if miniredis provides one in the future.
// Until then, recheck the upstream source and keep TestMemoryPeriodicCleanup
// passing whenever upgrading the dependency.
// Do not call FastForward(interval): that would deduct time a second time and
// expire live keys early. The 60-second interval controls idle memory reclamation,
// not TTL precision on reads. This loop never takes memoryMu, so the last release
// can hold that lock while waiting for cleanup to stop.
func (i *_MemoryRedis) cleanupLoop() {
	defer close(i.cleanupDone)
	ticker := time.NewTicker(memoryCleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-i.cleanupStop:
			return
		case <-ticker.C:
			i.server.DB(0)
		}
	}
}

func (i *_MemoryRedis) close() {
	close(i.cleanupStop)
	<-i.cleanupDone

	i.mutex.Lock()
	defer i.mutex.Unlock()
	i.closed = true
	i.server.Close()
}
