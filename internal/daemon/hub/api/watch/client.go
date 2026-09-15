package watch

import (
	"context"
	"crypto/tls"
	"errors"
	"strconv"
	"strings"
	"sync"

	"github.com/redis/go-redis/v9"
	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/util/vpre"
)

type Option struct {
	InprocMode bool
	Endpoint   string
	Username   string
	Password   string
	TLSConfig  *tls.Config
}

type ClientSpec interface {
	InitOption(option *Option)

	mustBeClient()
}

// Subscription delays event delivery until Start releases events buffered while
// the caller publishes the loaded snapshot.
type Subscription interface {
	Start()
}

type ClientOps interface {
	Load(key string) (string, bool)
	LoadAndSubscribe(ctx context.Context, key string, handle func(event Event)) (string, bool, Subscription)
	LoadListAndSubscribe(ctx context.Context, prefix string, handle func(event Event)) (map[string]string, Subscription)
}

type _RedisClientSetter interface {
	setRedisClient(ctx context.Context, redisClient *redis.Client)
	swapRedisClient(ctx context.Context, redisClient *redis.Client) *redis.Client
}

type _ClientRepairer interface {
	setRepair(repair func(option *Option) bool)
	restartWatchers()
}

// Client watches Hub-published keys over Hub's Redis protocol watch service.
// A restarted Hub can advertise a different watch endpoint, so the owning
// manager can replace the connection and re-subscribe every active watcher.
type Client struct {
	app.BaseManagedComponent[*ClientManager]

	clientMutex sync.RWMutex
	ctx         context.Context
	redisClient *redis.Client
	repair      func(option *Option) bool

	watcherMutex sync.Mutex
	watchers     map[*_Watcher]struct{}
}

func (*Client) InitOption(*Option) {}

func (*Client) mustBeClient() {}

func (c *Client) setRedisClient(ctx context.Context, redisClient *redis.Client) {
	c.clientMutex.Lock()
	defer c.clientMutex.Unlock()

	c.ctx = ctx
	c.redisClient = redisClient
}

func (c *Client) swapRedisClient(ctx context.Context, redisClient *redis.Client) *redis.Client {
	c.clientMutex.Lock()
	defer c.clientMutex.Unlock()

	previous := c.redisClient
	c.ctx = ctx
	c.redisClient = redisClient
	return previous
}

func (c *Client) currentRedisClient() (*redis.Client, context.Context) {
	c.clientMutex.RLock()
	defer c.clientMutex.RUnlock()

	return c.redisClient, c.ctx
}

func (c *Client) setRepair(repair func(option *Option) bool) {
	c.clientMutex.Lock()
	defer c.clientMutex.Unlock()

	c.repair = repair
}

// RepairEndpoint reconnects the watch client after Hub advertised a different
// watch endpoint and re-subscribes every active watcher. It reports whether the
// endpoint moved; an unchanged endpoint keeps the existing connections.
func (c *Client) RepairEndpoint(option *Option) bool {
	c.clientMutex.RLock()
	repair := c.repair
	c.clientMutex.RUnlock()

	if repair == nil {
		return false
	}
	return repair(option)
}

func (c *Client) Close() {
	redisClient, _ := c.currentRedisClient()
	c.closeWatchers()
	if redisClient != nil {
		_ = redisClient.Close()
	}
}

func (c *Client) addWatcher(watcher *_Watcher) {
	c.watcherMutex.Lock()
	defer c.watcherMutex.Unlock()

	if c.watchers == nil {
		c.watchers = map[*_Watcher]struct{}{}
	}
	c.watchers[watcher] = struct{}{}
}

func (c *Client) removeWatcher(watcher *_Watcher) {
	c.watcherMutex.Lock()
	defer c.watcherMutex.Unlock()

	delete(c.watchers, watcher)
}

func (c *Client) watcherSnapshot() []*_Watcher {
	c.watcherMutex.Lock()
	defer c.watcherMutex.Unlock()

	watchers := make([]*_Watcher, 0, len(c.watchers))
	for watcher := range c.watchers {
		watchers = append(watchers, watcher)
	}
	return watchers
}

func (c *Client) closeWatchers() {
	for _, watcher := range c.watcherSnapshot() {
		watcher.stop()
	}
}

// restartWatchers re-subscribes every active watcher on the current connection.
// Each watcher reloads its snapshot and reconciles against the last known state,
// which is what recovers keys that changed while the watch endpoint was stale.
func (c *Client) restartWatchers() {
	redisClient, _ := c.currentRedisClient()
	for _, watcher := range c.watcherSnapshot() {
		watcher.restart(redisClient)
	}
}

func (c *Client) Load(key string) (string, bool) {
	redisClient, ctx := c.currentRedisClient()

	value, err := redisClient.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", false
	}
	vpre.CheckNilError(err, "load redis key value failed")
	return value, true
}

func (c *Client) loadScanKeyValues(prefix string) map[string]string {
	redisClient, ctx := c.currentRedisClient()

	pattern := formatRedisListPattern(prefix)
	keys := make([]string, 0)
	var cursor uint64
	for {
		batch, nextCursor, err := redisClient.Scan(ctx, cursor, pattern, 1000).Result()
		vpre.CheckNilError(err, "scan redis keys failed")
		keys = append(keys, batch...)
		if nextCursor == 0 {
			break
		}
		cursor = nextCursor
	}

	valuesByKey := map[string]string{}
	for _, key := range keys {
		value, ok := c.Load(key)
		if !ok {
			continue
		}
		valuesByKey[key] = value
	}
	return valuesByKey
}

func (c *Client) loadStableValue(key string) (string, bool, uint64) {
	for {
		revision1 := c.loadRevision()
		value, ok := c.Load(key)
		revision2 := c.loadRevision()
		if revision1 == revision2 {
			return value, ok, revision2
		}
	}
}

func (c *Client) loadStableScanKeyValues(prefix string) (map[string]string, uint64) {
	for {
		revision1 := c.loadRevision()
		valuesByKey := c.loadScanKeyValues(prefix)
		revision2 := c.loadRevision()
		if revision1 == revision2 {
			return valuesByKey, revision2
		}
	}
}

func (c *Client) loadRevision() uint64 {
	value, ok := c.Load(RevisionKey)
	if !ok || value == "" {
		return 0
	}
	revision, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0
	}
	return revision
}

func formatRedisListPattern(prefix string) string {
	return strings.TrimSuffix(prefix, ":") + ":*"
}
