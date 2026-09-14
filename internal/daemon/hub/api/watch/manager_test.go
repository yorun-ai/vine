package watch

import (
	"context"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"go.yorun.ai/vine/internal/app"
)

func TestBuildsRedisClientFromWatchEndpoint(t *testing.T) {
	oldFactory := newRedisClient
	t.Cleanup(func() {
		newRedisClient = oldFactory
	})

	var options *redis.Options
	newRedisClient = func(opt *redis.Options) *redis.Client {
		options = opt
		return redis.NewClient(opt)
	}

	component := &_WatchTestClient{}
	manager := initTestClient(component)
	defer manager.AfterAppStop()

	assert.NotNil(t, options)
	assert.Equal(t, "demo.local:7093", options.Addr)
	assert.Equal(t, LinkUsername, options.Username)
	assert.Equal(t, LinkPassword, options.Password)
	assert.NotNil(t, options.OnConnect)
}

type _WatchTestClient struct {
	Client
}

func (*_WatchTestClient) InitOption(option *Option) {
	option.Endpoint = "redis://demo.local:7093"
	option.Username = LinkUsername
	option.Password = LinkPassword
}

func initTestClient(component app.ManagedComponent) *ClientManager {
	manager := &ClientManager{
		Context: context.Background(),
	}
	manager.InitComponent(component)
	return manager
}
