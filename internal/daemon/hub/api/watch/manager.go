package watch

import (
	"context"
	"strings"
	"sync"

	"github.com/redis/go-redis/v9"
	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/util/vpre"
)

// ClientManager configures and closes a Hub Watch client.
type ClientManager struct {
	app.BaseComponentManager

	Context context.Context `inject:""`

	repairMutex sync.Mutex
	client      app.ManagedComponent
	option      *Option
	addr        string
}

func (m *ClientManager) InitComponent(component app.ManagedComponent) {
	m.client = component
	m.option = &Option{}

	spec := component.(ClientSpec)
	spec.InitOption(m.option)

	client := component.(_RedisClientSetter)
	if m.option.InprocMode {
		vpre.CheckNotNil(InprocServer(), "inproc watch server missing")
		m.addr = WatchInprocEndpoint
		client.setRedisClient(context.Background(), newRedisClient(redisOptionsFor(m.option, m.addr)))
	} else {
		vpre.CheckNotEmpty(m.option.Endpoint, "watch endpoint is empty")
		m.addr = redisAddr(m.option.Endpoint)
		client.setRedisClient(m.Context, newRedisClient(redisOptionsFor(m.option, m.addr)))
	}

	component.(_ClientRepairer).setRepair(m.repairEndpoint)
}

// repairEndpoint reconnects the watch client after Hub advertised a different
// watch endpoint and re-subscribes every active watcher. An unchanged endpoint
// keeps the existing connections, so a Hub restart without a watch endpoint
// change reconnects nothing.
func (m *ClientManager) repairEndpoint(option *Option) bool {
	m.repairMutex.Lock()
	defer m.repairMutex.Unlock()
	repairer := m.client.(_ClientRepairer)
	repairer.lockLifecycle()
	defer repairer.unlockLifecycle()

	if option.InprocMode || m.option.InprocMode {
		return false
	}
	vpre.CheckNotEmpty(option.Endpoint, "watch endpoint is empty")

	addr := redisAddr(option.Endpoint)
	if addr == m.addr {
		return false
	}

	client := m.client.(_RedisClientSetter)
	next := newRedisClient(redisOptionsFor(option, addr))
	repairer.closeWatchers()
	previous := client.swapRedisClient(m.Context, next)
	succeeded := false
	defer func() {
		if succeeded {
			return
		}
		client.swapRedisClient(m.Context, previous)
		_ = next.Close()
		// Restore the subscriptions that were stopped before the replacement
		// endpoint failed. A later Hub refresh can then retry the replacement.
		m.client.(_ClientRepairer).restartWatchers()
	}()
	m.client.(_ClientRepairer).restartWatchers()
	m.option = option
	m.addr = addr
	succeeded = true
	if previous != nil {
		_ = previous.Close()
	}
	return true
}

func (m *ClientManager) Component() app.ManagedComponent {
	return m.client
}

func (m *ClientManager) AfterAppStop() {
	if client, ok := m.client.(interface{ closeRedisClient() }); ok {
		client.closeRedisClient()
	}
}

func (c *Client) closeRedisClient() {
	c.Close()
}

func redisOptionsFor(option *Option, addr string) *redis.Options {
	redisOptions := &redis.Options{
		Protocol:        2,
		DisableIdentity: true,
		Username:        option.Username,
		Password:        option.Password,
		TLSConfig:       option.TLSConfig,
		Addr:            addr,
	}
	if option.Username != "" && option.Password == "" {
		// go-redis omits HELLO AUTH when the password is empty. Link and Portal
		// temporarily use empty passwords, so authenticate each newly opened
		// connection explicitly. OnConnect is also used for PubSub and replacement
		// pool connections, preventing reconnects from silently becoming anonymous.
		username := option.Username
		redisOptions.OnConnect = func(ctx context.Context, conn *redis.Conn) error {
			return conn.AuthACL(ctx, username, "").Err()
		}
	}
	if option.InprocMode {
		redisOptions.Addr = WatchInprocEndpoint
		redisOptions.Dialer = DialInproc
	}
	return redisOptions
}

var newRedisClient = func(opt *redis.Options) *redis.Client {
	return redis.NewClient(opt)
}

func redisAddr(endpoint string) string {
	if !strings.Contains(endpoint, "://") {
		return endpoint
	}
	parts := strings.SplitN(endpoint, "://", 2)
	vpre.Check(len(parts) == 2 && parts[1] != "", "watch endpoint host is empty")
	return parts[1]
}
