// Package watchtest provides an owned, event-delivering Hub Watch fixture.
package watchtest

import (
	"context"
	"maps"
	"strings"
	"sync"
	"testing"

	"github.com/gobwas/glob"
	"go.yorun.ai/vine/internal/daemon/hub/api/watch"
)

// Client implements Watch operations without a Redis connection.
type Client struct {
	mutex         sync.Mutex
	values        map[string]string
	subscriptions []*_Subscription
	ctx           context.Context
	wait          sync.WaitGroup
}

type _Subscription struct {
	ctx    context.Context
	match  func(string) bool
	handle func(watch.Event)
	events chan watch.Event
	start  chan struct{}
	once   sync.Once
}

// New copies the initial snapshot and cancels and joins subscriptions at cleanup.
func New(t testing.TB, values map[string]string) *Client {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	c := &Client{values: maps.Clone(values), ctx: ctx}
	if c.values == nil {
		c.values = map[string]string{}
	}
	t.Cleanup(func() { cancel(); c.wait.Wait() })
	return c
}

// Load returns the current stored value.
func (c *Client) Load(key string) (string, bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	value, ok := c.values[key]
	return value, ok
}

// SetValue changes only the stored snapshot, for tests of retained values.
func (c *Client) SetValue(key string, value string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.values[key] = value
}

// Publish updates the snapshot and queues matching events until Start is called.
func (c *Client) Publish(event watch.Event) {
	c.mutex.Lock()
	if event.Kind == watch.EventKindDelete {
		delete(c.values, event.Key)
	} else {
		c.values[event.Key] = event.Value
	}
	subscriptions := append([]*_Subscription(nil), c.subscriptions...)
	c.mutex.Unlock()
	for _, sub := range subscriptions {
		if !sub.match(event.Key) {
			continue
		}
		select {
		case <-sub.ctx.Done():
		case <-c.ctx.Done():
		case sub.events <- event:
		}
	}
}

// LoadAndSubscribe atomically reads a value and registers its event handler.
func (c *Client) LoadAndSubscribe(ctx context.Context, key string, handle func(watch.Event)) (string, bool, watch.Subscription) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	value, ok := c.values[key]
	return value, ok, c.subscribe(ctx, func(candidate string) bool { return candidate == key }, handle)
}

// LoadListAndSubscribe atomically reads a prefix and registers its event handler.
func (c *Client) LoadListAndSubscribe(ctx context.Context, prefix string, handle func(watch.Event)) (map[string]string, watch.Subscription) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	pattern := glob.MustCompile(strings.TrimSuffix(prefix, ":") + ":*")
	values := map[string]string{}
	for key, value := range c.values {
		if pattern.Match(key) {
			values[key] = value
		}
	}
	return values, c.subscribe(ctx, pattern.Match, handle)
}

func (c *Client) subscribe(ctx context.Context, match func(string) bool, handle func(watch.Event)) *_Subscription {
	sub := &_Subscription{ctx: ctx, match: match, handle: handle, events: make(chan watch.Event, 64), start: make(chan struct{})}
	c.subscriptions = append(c.subscriptions, sub)
	c.wait.Add(1)
	go func() {
		defer c.wait.Done()
		select {
		case <-ctx.Done():
			return
		case <-c.ctx.Done():
			return
		case <-sub.start:
		}
		for {
			select {
			case <-ctx.Done():
				return
			case <-c.ctx.Done():
				return
			case event := <-sub.events:
				if ctx.Err() == nil {
					handle(event)
				}
			}
		}
	}()
	return sub
}

func (s *_Subscription) Start() { s.once.Do(func() { close(s.start) }) }

var _ watch.ClientOps = (*Client)(nil)
