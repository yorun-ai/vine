package hubredis

import (
	"context"
	"strings"
	"sync"

	"github.com/gobwas/glob"
	hubapiredis "go.yorun.ai/vine/internal/daemon/hub/api/redis"
)

var testClientByClient sync.Map

type _TestClient struct {
	valuesByKey map[string]string
}

func NewClientForTest(valuesByKey map[string]string) *Client {
	client := new(Client)
	copied := map[string]string{}
	for key, value := range valuesByKey {
		copied[key] = value
	}
	testClientByClient.Store(client, &_TestClient{
		valuesByKey: copied,
	})
	return client
}

func (c *Client) Load(key string) (string, bool) {
	if testClient, ok := testClientByClient.Load(c); ok {
		return testClient.(*_TestClient).Load(key)
	}
	return c.Client.Load(key)
}

func (c *Client) LoadAndSubscribe(ctx context.Context, key string, handle func(event hubapiredis.Event)) (string, bool) {
	if testClient, ok := testClientByClient.Load(c); ok {
		return testClient.(*_TestClient).LoadAndSubscribe(ctx, key, handle)
	}
	return c.Client.LoadAndSubscribe(ctx, key, handle)
}

func (c *Client) LoadListAndSubscribe(ctx context.Context, prefix string, handle func(event hubapiredis.Event)) map[string]string {
	if testClient, ok := testClientByClient.Load(c); ok {
		return testClient.(*_TestClient).LoadListAndSubscribe(ctx, prefix, handle)
	}
	return c.Client.LoadListAndSubscribe(ctx, prefix, handle)
}

func (c *Client) SetValue(key string, value string) {
	if testClient, ok := testClientByClient.Load(c); ok {
		testClient.(*_TestClient).SetValue(key, value)
	}
}

func (c *_TestClient) Load(key string) (string, bool) {
	value, ok := c.valuesByKey[key]
	return value, ok
}

func (c *_TestClient) loadScanKeyValues(prefix string) map[string]string {
	pattern := glob.MustCompile(formatRedisListPattern(prefix))
	valuesByKey := map[string]string{}
	for key, value := range c.valuesByKey {
		if !pattern.Match(key) {
			continue
		}
		valuesByKey[key] = value
	}
	return valuesByKey
}

func formatRedisListPattern(prefix string) string {
	return strings.TrimSuffix(prefix, ":") + ":*"
}

func (c *_TestClient) SetValue(key string, value string) {
	c.valuesByKey[key] = value
}

func (c *_TestClient) LoadAndSubscribe(_ context.Context, key string, _ func(hubapiredis.Event)) (string, bool) {
	return c.Load(key)
}

func (c *_TestClient) LoadListAndSubscribe(_ context.Context, prefix string, _ func(hubapiredis.Event)) map[string]string {
	return c.loadScanKeyValues(prefix)
}
