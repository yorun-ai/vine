package redis

import (
	"strings"
	"sync"

	goredis "github.com/redis/go-redis/v9"
)

var (
	connectionsMu sync.Mutex
	connections   = map[string]*_RedisConnection{}
)

type _RedisConnection struct {
	endpoint string
	client   *goredis.Client
	refs     int // Protected by connectionsMu.
}

// acquireRedisClient shares external clients by the exact endpoint string.
// Memory clients are shared by instance name and parsed database index.
func acquireRedisClient(option *Option) (*goredis.Client, func()) {
	if strings.HasPrefix(strings.ToLower(option.Endpoint), "redis+memory:") {
		return acquireMemoryClient(option.Endpoint)
	}
	connectionsMu.Lock()
	defer connectionsMu.Unlock()
	connection := connections[option.Endpoint]
	if connection == nil {
		connection = newRedisConnection(option)
		connections[option.Endpoint] = connection
	}
	return connection.acquire()
}

func newRedisConnection(option *Option) *_RedisConnection {
	client := newRedisClient(option)
	client.AddHook(_SelectHook{dbIndex: client.Options().DB})
	return new(_RedisConnection{endpoint: option.Endpoint, client: client})
}

// acquire requires connectionsMu to keep lookup and reference acquisition atomic.
func (c *_RedisConnection) acquire() (*goredis.Client, func()) {
	c.refs++
	return c.client, sync.OnceFunc(c.release)
}

func (c *_RedisConnection) release() {
	connectionsMu.Lock()
	defer connectionsMu.Unlock()
	c.refs--
	if c.refs == 0 {
		_ = c.client.Close()
		delete(connections, c.endpoint)
	}
}
