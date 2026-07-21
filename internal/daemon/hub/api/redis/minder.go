package redis

import (
	"context"
	"strings"

	"github.com/redis/go-redis/v9"
	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/util/vpre"
)

type ClientMinder struct {
	app.BaseFrameworkComponentMinder

	Context context.Context `inject:""`

	client app.FrameworkComponent
	option *Option
}

func (m *ClientMinder) InitComponent(component app.FrameworkComponent) {
	m.client = component
	m.option = &Option{}

	spec := component.(ClientSpec)
	spec.InitOption(m.option)

	client := component.(_RedisClientSetter)
	redisOptions := &redis.Options{
		Protocol:        2,
		DisableIdentity: true,
	}
	if m.option.InprocMode {
		vpre.CheckNotNil(InprocServer(), "inproc redis server missing")
		redisOptions.Addr = RedisInprocEndpoint
		redisOptions.Dialer = DialInproc
		client.setRedisClient(context.Background(), newRedisClient(redisOptions))
		return
	}

	vpre.CheckNotEmpty(m.option.Endpoint, "redis endpoint is empty")
	redisOptions.Addr = redisAddr(m.option.Endpoint)
	client.setRedisClient(m.Context, newRedisClient(redisOptions))
}

func (m *ClientMinder) Component() app.FrameworkComponent {
	return m.client
}

func (m *ClientMinder) AfterAppStop() {
	if client, ok := m.client.(interface{ closeRedisClient() }); ok {
		client.closeRedisClient()
	}
}

func (c *Client) closeRedisClient() {
	c.Close()
}

var newRedisClient = func(opt *redis.Options) *redis.Client {
	return redis.NewClient(opt)
}

func redisAddr(endpoint string) string {
	if !strings.Contains(endpoint, "://") {
		return endpoint
	}
	parts := strings.SplitN(endpoint, "://", 2)
	vpre.Check(len(parts) == 2 && parts[1] != "", "redis endpoint host is empty")
	return parts[1]
}
