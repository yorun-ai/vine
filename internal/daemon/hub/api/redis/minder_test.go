package redis

import (
	"context"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"go.yorun.ai/vine/internal/app"
)

func TestBuildsRedisClientFromRedisEndpoint(t *testing.T) {
	oldFactory := newRedisClient
	t.Cleanup(func() {
		newRedisClient = oldFactory
	})

	var options *redis.Options
	newRedisClient = func(opt *redis.Options) *redis.Client {
		options = opt
		return redis.NewClient(opt)
	}

	component := &_RedisTestClient{}
	minder := initTestClient(component)
	defer minder.AfterAppStop()

	assert.NotNil(t, options)
	assert.Equal(t, "demo.local:7093", options.Addr)
	assert.Equal(t, LinkUsername, options.Username)
	assert.Equal(t, LinkPassword, options.Password)
	assert.NotNil(t, options.OnConnect)
}

type _RedisTestClient struct {
	Client
}

func (*_RedisTestClient) InitOption(option *Option) {
	option.Endpoint = "redis://demo.local:7093"
	option.Username = LinkUsername
	option.Password = LinkPassword
}

func initTestClient(component app.FrameworkComponent) *ClientMinder {
	minder := &ClientMinder{
		Context: context.Background(),
	}
	minder.InitComponent(component)
	return minder
}
