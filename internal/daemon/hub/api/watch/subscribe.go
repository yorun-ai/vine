package watch

import (
	"context"
	"encoding/json/v2"
	"sync"

	"github.com/redis/go-redis/v9"
	"go.yorun.ai/vine/util/vmap"
	"go.yorun.ai/vine/util/vpre"
)

func (c *Client) LoadAndSubscribe(ctx context.Context, key string, handle func(event Event)) (string, bool, Subscription) {
	value, ok, _ := c.loadStableValue(key)
	watcher := &_Watcher{
		client:       c,
		ctx:          ctx,
		key:          key,
		handle:       handle,
		snapshot:     &_WatchKeySnapshot{client: c, key: key, value: value, hasValue: ok},
		subscription: newSubscription(ctx, handle),
	}
	c.subscribeWatcher(watcher)
	return value, ok, watcher.subscription
}

func (c *Client) LoadListAndSubscribe(ctx context.Context, prefix string, handle func(event Event)) (map[string]string, Subscription) {
	valuesByKey, _ := c.loadStableScanKeyValues(prefix)
	watcher := &_Watcher{
		client:       c,
		ctx:          ctx,
		prefix:       prefix,
		handle:       handle,
		snapshot:     &_WatchListSnapshot{client: c, prefix: prefix, valuesByKey: vmap.Clone(valuesByKey)},
		subscription: newSubscription(ctx, handle),
	}
	c.subscribeWatcher(watcher)
	return valuesByKey, watcher.subscription
}

func (c *Client) subscribeWatcher(watcher *_Watcher) {
	redisClient, _ := c.currentRedisClient()
	c.addWatcher(watcher)
	watcher.start(redisClient)
}

// _Watcher owns one subscription to a key or key prefix. It outlives a
// reconnect: restart re-creates the PubSub on the current connection and
// reconciles the snapshot against Hub again, so keys that changed while the
// endpoint was stale are reported as regular events.
type _Watcher struct {
	client   *Client
	ctx      context.Context
	key      string
	prefix   string
	handle   func(event Event)
	snapshot _Snapshot

	subscription *_Subscription

	mutex  sync.Mutex
	cancel context.CancelFunc
	pubsub *redis.PubSub
	done   chan struct{}
}

// _Snapshot reconciles watched values with the events delivered for them.
type _Snapshot interface {
	handleSubscription(handle func(event Event)) uint64
	handleEvent(handle func(event Event)) func(event Event)
}

func (w *_Watcher) start(redisClient *redis.Client) {
	if redisClient == nil || w.ctx.Err() != nil {
		w.client.removeWatcher(w)
		return
	}

	consumeCtx, cancel := context.WithCancel(w.ctx)
	pubsub := w.subscribe(consumeCtx, redisClient)
	if _, err := pubsub.Receive(consumeCtx); err != nil {
		_ = pubsub.Close()
		cancel()
		vpre.Panic(err)
	}

	// Reconcile before consuming: the reload covers everything that changed
	// while no subscription was attached, and later events are filtered by the
	// resulting revision.
	handleEvent := w.snapshot.handleEvent(w.enqueue)
	revision := w.snapshot.handleSubscription(handleEvent)

	w.mutex.Lock()
	w.cancel = cancel
	w.pubsub = pubsub
	w.done = make(chan struct{})
	done := w.done
	w.mutex.Unlock()

	go func() {
		defer close(done)
		w.client.consumePatternMessages(consumeCtx, pubsub.ChannelWithSubscriptions(), revision, w.snapshot.handleSubscription, handleEvent, pubsub.Close)
	}()
}

func (w *_Watcher) subscribe(ctx context.Context, redisClient *redis.Client) *redis.PubSub {
	if w.key != "" {
		return redisClient.Subscribe(ctx, w.key)
	}
	return redisClient.PSubscribe(ctx, formatRedisListPattern(w.prefix))
}

func (w *_Watcher) enqueue(event Event) {
	w.subscription.enqueue(event)
}

func (w *_Watcher) restart(redisClient *redis.Client) {
	w.stop()
	w.start(redisClient)
}

func (w *_Watcher) stop() {
	w.mutex.Lock()
	cancel := w.cancel
	pubsub := w.pubsub
	done := w.done
	w.cancel = nil
	w.pubsub = nil
	w.done = nil
	w.mutex.Unlock()

	if cancel != nil {
		cancel()
	}
	if pubsub != nil {
		_ = pubsub.Close()
	}
	if done != nil {
		<-done
	}
}

type _Subscription struct {
	ctx    context.Context
	handle func(event Event)

	mutex       sync.Mutex
	started     bool
	dispatching bool
	events      []Event
}

func newSubscription(ctx context.Context, handle func(event Event)) *_Subscription {
	return &_Subscription{
		ctx:    ctx,
		handle: handle,
		events: make([]Event, 0),
	}
}

func (s *_Subscription) Start() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.started {
		return
	}
	s.started = true
	s.startDispatcherLocked()
}

func (s *_Subscription) enqueue(event Event) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.ctx.Err() != nil {
		return
	}
	s.events = append(s.events, event)
	s.startDispatcherLocked()
}

func (s *_Subscription) startDispatcherLocked() {
	if !s.started || s.dispatching || len(s.events) == 0 {
		return
	}
	s.dispatching = true
	go s.dispatch()
}

func (s *_Subscription) dispatch() {
	for {
		s.mutex.Lock()
		if s.ctx.Err() != nil || len(s.events) == 0 {
			clear(s.events)
			s.events = nil
			s.dispatching = false
			s.mutex.Unlock()
			return
		}
		event := s.events[0]
		s.events[0] = Event{}
		s.events = s.events[1:]
		s.mutex.Unlock()

		s.handle(event)
	}
}

func (c *Client) consumePatternMessages(
	ctx context.Context,
	messageCh <-chan any,
	revision uint64,
	handleSubscription func(handle func(event Event)) uint64,
	handleEvent func(event Event),
	closeFn func() error,
) {
	defer func() {
		_ = closeFn()
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case message, ok := <-messageCh:
			if !ok {
				return
			}
			switch message := message.(type) {
			case *redis.Subscription:
				if message.Kind == "subscribe" || message.Kind == "psubscribe" {
					revision = handleSubscription(handleEvent)
				}
			case *redis.Message:
				var event Event
				if err := json.Unmarshal([]byte(message.Payload), &event); err != nil {
					continue
				}
				if event.Revision < revision {
					continue
				}
				revision = event.Revision
				handleEvent(event)
			}
		}
	}
}

type _WatchKeySnapshot struct {
	client   *Client
	key      string
	value    string
	hasValue bool
}

func (s *_WatchKeySnapshot) handleSubscription(handle func(event Event)) uint64 {
	value, ok, revision := s.client.loadStableValue(s.key)
	return s.reconcile(value, ok, revision, handle)
}

func (s *_WatchKeySnapshot) reconcile(value string, ok bool, revision uint64, handle func(event Event)) uint64 {
	if ok {
		if s.hasValue && s.value == value {
			return revision
		}
		s.value = value
		s.hasValue = true
		handle(Event{
			Revision: revision,
			Kind:     EventKindUpsert,
			Key:      s.key,
			Value:    value,
		})
		return revision
	}

	if !s.hasValue {
		return revision
	}
	s.value = ""
	s.hasValue = false
	handle(Event{
		Revision: revision,
		Kind:     EventKindDelete,
		Key:      s.key,
	})
	return revision
}

func (s *_WatchKeySnapshot) handleEvent(handle func(event Event)) func(event Event) {
	return func(event Event) {
		switch event.Kind {
		case EventKindDelete:
			s.value = ""
			s.hasValue = false
		default:
			s.value = event.Value
			s.hasValue = true
		}
		handle(event)
	}
}

type _WatchListSnapshot struct {
	client      *Client
	prefix      string
	valuesByKey map[string]string
}

func (s *_WatchListSnapshot) handleSubscription(handle func(event Event)) uint64 {
	valuesByKey, revision := s.client.loadStableScanKeyValues(s.prefix)
	return s.reconcile(valuesByKey, revision, handle)
}

func (s *_WatchListSnapshot) reconcile(valuesByKey map[string]string, revision uint64, handle func(event Event)) uint64 {
	for key, value := range valuesByKey {
		oldValue, exists := s.valuesByKey[key]
		if exists && oldValue == value {
			continue
		}
		s.valuesByKey[key] = value
		handle(Event{
			Revision: revision,
			Kind:     EventKindUpsert,
			Key:      key,
			Value:    value,
		})
	}
	for key := range s.valuesByKey {
		if _, ok := valuesByKey[key]; ok {
			continue
		}
		delete(s.valuesByKey, key)
		handle(Event{
			Revision: revision,
			Kind:     EventKindDelete,
			Key:      key,
		})
	}
	return revision
}

func (s *_WatchListSnapshot) handleEvent(handle func(event Event)) func(event Event) {
	return func(event Event) {
		switch event.Kind {
		case EventKindDelete:
			delete(s.valuesByKey, event.Key)
		default:
			s.valuesByKey[event.Key] = event.Value
		}
		handle(event)
	}
}
