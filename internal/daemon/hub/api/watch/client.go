package watch

import (
	"context"
	"crypto/tls"
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

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
	LoadList(prefix string) map[string]string
	LoadKeysAndSubscribe(ctx context.Context, keys []string, handle func(event Event)) (map[string]string, Subscription, error)
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
	lockLifecycle()
	unlockLifecycle()
	closeWatchers()
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

	watcherMutex  sync.Mutex
	lifecycleOnce sync.Once
	lifecycle     chan struct{}
	watchers      map[*_Watcher]struct{}
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
	c.lockLifecycle()
	defer c.unlockLifecycle()
	redisClient, _ := c.currentRedisClient()
	c.closeWatchers()
	if redisClient != nil {
		_ = redisClient.Close()
	}
}

func (c *Client) lockLifecycle() { _ = c.lockLifecycleContext(context.Background()) }
func (c *Client) lockLifecycleContext(ctx context.Context) error {
	c.lifecycleOnce.Do(func() { c.lifecycle = make(chan struct{}, 1) })
	select {
	case c.lifecycle <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (c *Client) unlockLifecycle() { <-c.lifecycle }

func (c *Client) addWatcher(watcher *_Watcher) {
	c.watcherMutex.Lock()
	defer c.watcherMutex.Unlock()

	if c.watchers == nil {
		c.watchers = map[*_Watcher]struct{}{}
	}
	c.watchers[watcher] = struct{}{}
	context.AfterFunc(watcher.ctx, func() {
		c.lockLifecycle()
		defer c.unlockLifecycle()
		watcher.stop()
		c.removeWatcher(watcher)
	})
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

const snapshotTimeout = 5 * time.Second
const snapshotBatchSize = 256

// LoadList returns a revision-consistent prefix snapshot without retaining a subscription.
func (c *Client) LoadList(prefix string) map[string]string {
	c.lockLifecycle()
	defer c.unlockLifecycle()
	_, ctx := c.currentRedisClient()
	values, _, err := c.loadStableSnapshot(ctx, nil, prefix)
	vpre.CheckNilError(err, "load redis snapshot failed")
	return values
}

// loadStableSnapshot uses bounded pipelines and a revision fence. The deadline
// also bounds retries when unrelated writes keep moving the global revision.
func (c *Client) loadStableSnapshot(ctx context.Context, keys []string, prefix string) (map[string]string, uint64, error) {
	ctx, cancel := context.WithTimeout(ctx, snapshotTimeout)
	defer cancel()
	client, _ := c.currentRedisClient()
	for {
		if err := ctx.Err(); err != nil {
			return nil, 0, err
		}
		before, err := readRevision(ctx, client)
		if err != nil {
			return nil, 0, err
		}
		selected := keys
		if prefix != "" {
			selected = nil
			var cursor uint64
			for {
				batch, next, err := client.Scan(ctx, cursor, formatRedisListPattern(prefix), 1000).Result()
				if err != nil {
					return nil, 0, err
				}
				selected = append(selected, batch...)
				cursor = next
				if cursor == 0 {
					break
				}
			}
		}
		values, err := readKeyValues(ctx, client, selected)
		if err != nil {
			return nil, 0, err
		}
		after, err := readRevision(ctx, client)
		if err != nil {
			return nil, 0, err
		}
		if before == after {
			return values, after, nil
		}
	}
}

func readKeyValues(ctx context.Context, client *redis.Client, keys []string) (map[string]string, error) {
	values := make(map[string]string, len(keys))
	for offset := 0; offset < len(keys); offset += snapshotBatchSize {
		batch := keys[offset:min(offset+snapshotBatchSize, len(keys))]
		pipeline := client.Pipeline()
		commands := make([]*redis.StringCmd, len(batch))
		for i, key := range batch {
			commands[i] = pipeline.Get(ctx, key)
		}
		_, err := pipeline.Exec(ctx)
		if err != nil && !errors.Is(err, redis.Nil) {
			return nil, err
		}
		for i, command := range commands {
			value, err := command.Result()
			if errors.Is(err, redis.Nil) {
				continue
			}
			if err != nil {
				return nil, err
			}
			values[batch[i]] = value
		}
	}
	return values, nil
}

func readRevision(ctx context.Context, client *redis.Client) (uint64, error) {
	value, err := client.Get(ctx, RevisionKey).Result()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if value == "" {
		return 0, nil
	}
	return strconv.ParseUint(value, 10, 64)
}

func formatRedisListPattern(prefix string) string {
	return strings.TrimSuffix(prefix, ":") + ":*"
}
