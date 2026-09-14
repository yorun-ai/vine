package hubwatch

import (
	"context"
	"maps"
	"strings"
	"sync"

	"github.com/gobwas/glob"
	hubapiwatch "go.yorun.ai/vine/internal/daemon/hub/api/watch"
)

var testClientByClient sync.Map

type _TestClient struct {
	valuesByKey map[string]string
}

func NewClientForTest(valuesByKey map[string]string) *Client {
	client := new(Client)
	copied := map[string]string{}
	maps.Copy(copied, valuesByKey)
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

func (c *Client) LoadAndSubscribe(ctx context.Context, key string, handle func(event hubapiwatch.Event)) (string, bool, hubapiwatch.Subscription) {
	if testClient, ok := testClientByClient.Load(c); ok {
		return testClient.(*_TestClient).LoadAndSubscribe(ctx, key, handle)
	}
	return c.Client.LoadAndSubscribe(ctx, key, handle)
}

func (c *Client) LoadListAndSubscribe(ctx context.Context, prefix string, handle func(event hubapiwatch.Event)) (map[string]string, hubapiwatch.Subscription) {
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
	pattern := glob.MustCompile(formatWatchListPattern(prefix))
	valuesByKey := map[string]string{}
	for key, value := range c.valuesByKey {
		if !pattern.Match(key) {
			continue
		}
		valuesByKey[key] = value
	}
	return valuesByKey
}

func formatWatchListPattern(prefix string) string {
	return strings.TrimSuffix(prefix, ":") + ":*"
}

func (c *_TestClient) SetValue(key string, value string) {
	c.valuesByKey[key] = value
}

func (c *_TestClient) LoadAndSubscribe(_ context.Context, key string, _ func(hubapiwatch.Event)) (string, bool, hubapiwatch.Subscription) {
	value, ok := c.Load(key)
	return value, ok, _TestSubscription{}
}

func (c *_TestClient) LoadListAndSubscribe(_ context.Context, prefix string, _ func(hubapiwatch.Event)) (map[string]string, hubapiwatch.Subscription) {
	return c.loadScanKeyValues(prefix), _TestSubscription{}
}

type _TestSubscription struct{}

func (_TestSubscription) Start() {}
