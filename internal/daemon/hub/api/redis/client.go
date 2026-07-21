package redis

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/redis/go-redis/v9"
	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/util/vpre"
)

type Option struct {
	InprocMode bool
	Endpoint   string
}

type ClientSpec interface {
	InitOption(option *Option)

	mustBeClient()
}

type ClientOps interface {
	Load(key string) (string, bool)
	LoadAndSubscribe(ctx context.Context, key string, handle func(event Event)) (string, bool)
	LoadListAndSubscribe(ctx context.Context, prefix string, handle func(event Event)) map[string]string
}

type _RedisClientSetter interface {
	setRedisClient(ctx context.Context, redisClient *redis.Client)
}

type Client struct {
	app.BaseFrameworkComponent[*ClientMinder]

	ctx         context.Context
	redisClient *redis.Client
}

func (*Client) InitOption(*Option) {}

func (*Client) mustBeClient() {}

func (c *Client) setRedisClient(ctx context.Context, redisClient *redis.Client) {
	c.ctx = ctx
	c.redisClient = redisClient
}

func (c *Client) Close() {
	if c.redisClient != nil {
		_ = c.redisClient.Close()
	}
}

func (c *Client) Load(key string) (string, bool) {
	value, err := c.redisClient.Get(c.ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", false
	}
	vpre.CheckNilError(err, "load redis key value failed")
	return value, true
}

func (c *Client) loadScanKeyValues(prefix string) map[string]string {
	pattern := formatRedisListPattern(prefix)
	keys := make([]string, 0)
	var cursor uint64
	for {
		batch, nextCursor, err := c.redisClient.Scan(c.ctx, cursor, pattern, 1000).Result()
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
